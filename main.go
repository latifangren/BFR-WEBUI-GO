package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bfr-webui-go/internal/auth"
	"bfr-webui-go/internal/charger"
	"bfr-webui-go/internal/handlers"
	"bfr-webui-go/internal/hotspot"
	"bfr-webui-go/internal/kernel"
	"bfr-webui-go/internal/logger"
	"bfr-webui-go/internal/modem"
	"bfr-webui-go/internal/nas"
	"bfr-webui-go/internal/network"
	"bfr-webui-go/internal/power"
	"bfr-webui-go/internal/qos"
	"bfr-webui-go/internal/ssh"
	"bfr-webui-go/internal/telegram"
	"bfr-webui-go/internal/tlsgen"
	_ "bfr-webui-go/internal/vnstat"
)

func main() {
	logger.SetupGlobalLogHook()

	port := flag.String("port", "80", "HTTP server port")
	httpsPort := flag.String("https-port", "443", "HTTPS server port (0 to disable)")
	applyTweaks := flag.Bool("apply-tweaks", false, "Apply all optimized network tweaks and exit")
	flag.Parse()

	if *applyTweaks {
		log.Println("Applying system & network optimizations from network_config.json...")
		if err := network.ApplyAllTweaks(); err != nil {
			log.Fatalf("Failed to apply tweaks: %v", err)
		}
		log.Println("Applying saved kernel parameters & governor policies...")
		if err := kernel.GetManager().ApplySavedConfig(); err != nil {
			log.Printf("Warning: applying saved kernel config: %v", err)
		}
		log.Println("Applying saved cell lock configuration...")
		if err := modem.ApplySavedCellLock(); err != nil {
			log.Printf("Notice: applying saved cell lock: %v", err)
		}
		log.Println("Optimizations applied successfully.")
		os.Exit(0)
	}

	if envPort := os.Getenv("PORT"); envPort != "" {
		*port = envPort
	}
	if envHttpsPort := os.Getenv("HTTPS_PORT"); envHttpsPort != "" {
		*httpsPort = envHttpsPort
	}

	authPass := os.Getenv("BFR_PASSWORD")
	authMgr := auth.NewManager(authPass)

	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux, authMgr)

	go func() {
		tgMgr := telegram.GetManager()
		if tgMgr.IsEnabled() {
			if err := tgMgr.Start(); err != nil {
				log.Printf("Telegram bot startup failed: %v", err)
			}
		}
	}()

	go func() {
		sshMgr := ssh.GetManager()
		if sshMgr.IsEnabled() {
			if err := sshMgr.Start(); err != nil {
				log.Printf("SSH daemon startup failed: %v", err)
			}
		}
	}()

	go func() {
		qosMgr := qos.GetManager()
		if cfg, err := qosMgr.LoadConfig(); err == nil && cfg.Enabled {
			if err := qosMgr.ApplyQoS(cfg); err != nil {
				log.Printf("QoS startup initialization failed: %v", err)
			}
		}
	}()

	go func() {
		power.GetScheduler().Start()
	}()

	go func() {
		_ = charger.GetManager()
	}()

	go func() {
		time.Sleep(3 * time.Second)
		_ = modem.ApplySavedCellLock()
	}()

	go func() {
		if err := kernel.GetManager().ApplySavedConfig(); err != nil {
			log.Printf("Kernel config restoration: %v", err)
		}
	}()

	go func() {
		nasMgr := nas.GetManager()
		if cfg, err := nasMgr.LoadConfig(); err == nil && cfg.Enabled {
			if err := nasMgr.StartNAS(*cfg); err != nil {
				log.Printf("NAS startup initialization failed: %v", err)
			}
		}
	}()

	// Hook Tethering state change to QoS auto re-apply
	hotspot.SetTetheringStateChangeHook(func(activeIface string) {
		qos.GetManager().OnTetheringChanged(activeIface)
	})

	go func() {
		time.Sleep(2 * time.Second)
		if err := hotspot.ApplySavedMACFilter(); err != nil {
			log.Printf("Hotspot MAC filter restoration: %v", err)
		}
	}()

	go func() {
		// Reconcile persisted router settings with the live tethering state
		// after Android networking has settled, then keep it in sync.
		time.Sleep(8 * time.Second)
		if err := hotspot.ReconcileRouterSettings(); err != nil {
			log.Printf("Router settings reconciliation: %v", err)
		}
		hotspot.GetReconciler().Start()
	}()

	addr := ":" + *port
	server := &http.Server{
		Addr:              addr,
		Handler:           handlers.GzipMiddleware(mux),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	var httpsServer *http.Server
	if *httpsPort != "" && *httpsPort != "0" {
		certPath, keyPath, err := tlsgen.EnsureCertificates("")
		if err != nil {
			log.Printf("[TLS] Warning: Failed to prepare certificates: %v", err)
		} else {
			httpsAddr := ":" + *httpsPort
			httpsServer = &http.Server{
				Addr:              httpsAddr,
				Handler:           handlers.GzipMiddleware(mux),
				TLSConfig:         &tls.Config{MinVersion: tls.VersionTLS12},
				ReadHeaderTimeout: 10 * time.Second,
				IdleTimeout:       120 * time.Second,
			}
			go func() {
				log.Printf("BFR WEBUI GO HTTPS starting on %s...", httpsAddr)
				if err := httpsServer.ListenAndServeTLS(certPath, keyPath); err != nil && err != http.ErrServerClosed {
					log.Printf("[TLS] HTTPS server error: %v", err)
				}
			}()
		}
	}

	// L-5: Graceful shutdown on SIGINT / SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("BFR WEBUI GO starting on %s...", addr)
		var err error
		for attempt := 1; attempt <= 3; attempt++ {
			err = server.ListenAndServe()
			if err == nil || err == http.ErrServerClosed {
				return
			}
			log.Printf("HTTP server error (attempt %d/3): %v", attempt, err)
			if attempt < 3 {
				time.Sleep(time.Duration(attempt*2) * time.Second)
			}
		}
		log.Printf("HTTP server failed after 3 attempts, exiting: %v", err)
		os.Exit(1)
	}()

	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
	if httpsServer != nil {
		if err := httpsServer.Shutdown(ctx); err != nil {
			log.Printf("HTTPS server shutdown error: %v", err)
		}
	}

	log.Println("Server exited cleanly.")
}
