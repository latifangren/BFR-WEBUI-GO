package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"bfr-webui-go/internal/auth"
	"bfr-webui-go/internal/handlers"
)

func main() {
	port := flag.String("port", "8080", "HTTP server port")
	flag.Parse()

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
