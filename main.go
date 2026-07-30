package main

import (
	"flag"
	"log"
	"net/http"
	"os"

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
	log.Printf("BFR WEBUI GO starting on %s...", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
