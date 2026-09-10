package handlers

import (
	"encoding/json"
	"net/http"

	"bfr-webui-go/internal/logger"
	"bfr-webui-go/internal/tunnel"
)

func formatTunnelStatus(st tunnel.TunnelStatus) map[string]interface{} {
	return map[string]interface{}{
		"engine":       st.Engine,
		"active":       st.Active,
		"running":      st.Active,
		"public_url":   st.PublicURL,
		"ip_address":   st.IPAddress,
		"logs":         st.Logs,
		"binary_found": st.BinaryFound,
		"binary_path":  st.BinaryPath,
	}
}

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
		"status": formatTunnelStatus(status),
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
		"error":   errString(err),
		"status":  formatTunnelStatus(mgr.GetStatus()),
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
	} else {
		logger.Get().Warnf("Tunnel", "Tunnel stop failed: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"error":   errString(err),
		"status":  formatTunnelStatus(mgr.GetStatus()),
	})
}

func HandleTunnelUploadBinary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	engine := r.URL.Query().Get("engine")
	if engine == "" {
		engine = "cloudflare"
	}

	err := r.ParseMultipartForm(64 << 20) // 64MB max
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to parse multipart form"})
		return
	}

	file, header, err := r.FormFile("binary")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "No binary file uploaded"})
		return
	}
	defer file.Close()

	binPath, err := tunnel.DownloadBinary(engine, file, header.Filename)
	if err == nil {
		logger.Get().Infof("Tunnel", "Binary uploaded and installed: %s (%s)", header.Filename, binPath)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"path":    binPath,
		"error":   errString(err),
	})
}
