package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"bfr-webui-go/internal/logger"
	"bfr-webui-go/internal/tunnel"
)

func HandleTunnelStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mgr := tunnel.GetManager()
	cfg, err := mgr.LoadConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	status := mgr.GetStatus()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"config": cfg,
		"status": status,
	})
}

func HandleTunnelStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var cfg tunnel.TunnelConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid tunnel request payload"})
		return
	}

	mgr := tunnel.GetManager()
	err := mgr.StartTunnel(cfg)
	if err == nil {
		logger.Get().Infof("Tunnel", "Tunnel started successfully using engine %s", cfg.Engine)
	} else {
		logger.Get().Warnf("Tunnel", "Tunnel start failed using engine %s: %v", cfg.Engine, err)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"error":   fmt.Sprintf("%v", err),
		"status":  mgr.GetStatus(),
	})
}

func HandleTunnelStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mgr := tunnel.GetManager()
	err := mgr.StopTunnel()
	if err == nil {
		logger.Get().Infof("Tunnel", "Tunnel stopped successfully")
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"error":   fmt.Sprintf("%v", err),
		"status":  mgr.GetStatus(),
	})
}
