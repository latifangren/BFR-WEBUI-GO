package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bfr-webui-go/internal/auth"
	_ "bfr-webui-go/internal/charger"
	"bfr-webui-go/internal/handlers"
	"bfr-webui-go/internal/network"
	_ "bfr-webui-go/internal/ssh"
	_ "bfr-webui-go/internal/vnstat"
)

func main() {
	port := flag.String("port", "8080", "HTTP server port")
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

	authPass := os.Getenv("BFR_PASSWORD")
	authMgr := auth.NewManager(authPass)

	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux, authMgr)

	addr := ":" + *port
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
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
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited cleanly.")
}
