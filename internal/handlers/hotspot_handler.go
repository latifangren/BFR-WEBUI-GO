package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"bfr-webui-go/internal/hotspot"
	"bfr-webui-go/internal/logger"
)

type hotspotControlRequest struct {
	Enable bool   `json:"enable"`
	SSID   string `json:"ssid"`
	Pass   string `json:"password"`
}

func HandleHotspotStatus(w http.ResponseWriter, r *http.Request) {
	status := hotspot.GetHotspotStatus()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

func HandleHotspotControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req hotspotControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request payload"})
		return
	}

	err := hotspot.ToggleHotspot(req.Enable, req.SSID, req.Pass)
	if err == nil {
		logger.Get().Infof("hotspot", "Hotspot toggle executed: enable=%v, ssid=%s", req.Enable, req.SSID)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"error":   fmt.Sprintf("%v", err),
	})
}

func HandleHotspotClients(w http.ResponseWriter, r *http.Request) {
	clients, err := hotspot.GetConnectedClients()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(clients)
}
