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
	"bfr-webui-go/internal/logger"
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
		log.Println("Applying system & network optimizations from tweaks.json...")
		if err := network.ApplyAllTweaks(); err != nil {
			log.Fatalf("Failed to apply tweaks: %v", err)
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
		nasMgr := nas.GetManager()
		if cfg, err := nasMgr.LoadConfig(); err == nil && cfg.Enabled {
			if err := nasMgr.StartNAS(*cfg); err != nil {
				log.Printf("NAS startup initialization failed: %v", err)
			}
		}
	}()

	addr := ":" + *port
	server := &http.Server{
		Addr:    addr,
		Handler: handlers.GzipMiddleware(mux),
	}

	var httpsServer *http.Server
	if *httpsPort != "" && *httpsPort != "0" {
		certPath, keyPath, err := tlsgen.EnsureCertificates("")
		if err != nil {
			log.Printf("[TLS] Warning: Failed to prepare certificates: %v", err)
		} else {
			httpsAddr := ":" + *httpsPort
			httpsServer = &http.Server{
				Addr:      httpsAddr,
				Handler:   handlers.GzipMiddleware(mux),
				TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12},
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
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
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
