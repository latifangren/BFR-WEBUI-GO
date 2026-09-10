package handlers

import (
	"encoding/json"
	"net/http"

	"bfr-webui-go/internal/logger"
	"bfr-webui-go/internal/nas"
)

func formatNASStatus(st nas.NASStatus) map[string]interface{} {
	return map[string]interface{}{
		"active":        st.Active,
		"running":       st.Active,
		"share_path":    st.SharePath,
		"url":           st.URL,
		"protocol":      st.Protocol,
		"storage_used":  st.StorageUsed,
		"storage_total": st.StorageTotal,
	}
}

func HandleNASStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mgr := nas.GetManager()
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
		"status": formatNASStatus(status),
	})
}

func HandleNASStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var cfg nas.NASConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid NAS request payload"})
		return
	}

	mgr := nas.GetManager()
	err := mgr.StartNAS(cfg)
	if err == nil {
		logger.Get().Infof("NAS", "NAS File Server started on port %d (path: %s)", cfg.Port, cfg.SharePath)
	} else {
		logger.Get().Warnf("NAS", "NAS File Server start failed: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"error":   errString(err),
		"status":  formatNASStatus(mgr.GetStatus()),
	})
}

func HandleNASStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mgr := nas.GetManager()
	err := mgr.StopNAS()
	if err == nil {
		logger.Get().Infof("NAS", "NAS File Server stopped")
	} else {
		logger.Get().Warnf("NAS", "NAS File Server stop failed: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"error":   errString(err),
		"status":  formatNASStatus(mgr.GetStatus()),
	})
}
