package handlers

import (
	"encoding/json"
	"net/http"

	"bfr-webui-go/internal/logger"
	"bfr-webui-go/internal/network"
	"bfr-webui-go/internal/sysinfo"
)

type ttlRequest struct {
	Enable bool `json:"enable"`
	TTL    int  `json:"ttl"`
}

type dnsRequest struct {
	Primary   string `json:"primary"`
	Secondary string `json:"secondary"`
}

type pingRequest struct {
	Host  string `json:"host"`
	Count int    `json:"count"`
}

func HandlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req pingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Host == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid host"})
		return
	}
	out, err := sysinfo.RunPing(req.Host, req.Count)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"output":  out,
		"success": err == nil,
	})
}

func HandleDNS(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		d1, d2 := network.GetActiveDNS()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"primary":   d1,
			"secondary": d2,
			"presets":   network.PresetDNS,
		})
		return
	}

	if r.Method == http.MethodPost {
		var req dnsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Primary == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid DNS request"})
			return
		}
		logger.Get().Infof("network", "DNS change requested: primary=%s, secondary=%s", req.Primary, req.Secondary)
		err := network.SetDNS(req.Primary, req.Secondary)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": err == nil, "error": errString(err)})
	}
}

func HandleDNSReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	err := network.ResetDNS()
	d1, d2 := network.GetActiveDNS()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   err == nil,
		"primary":   d1,
		"secondary": d2,
		"error":     errString(err),
	})
}
func HandleTTL(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		status := network.GetTTLSpoofStatus()
		defaultTTL := network.GetDefaultTTL()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ttl_spoof":   status,
			"current_ttl": defaultTTL,
		})
		return
	}

	if r.Method == http.MethodPost {
		var req ttlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid TTL request"})
			return
		}
		err := network.SetTTLSpoofSDK(req.Enable, req.TTL)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": err == nil, "error": errString(err)})
	}
}

func errString(err error) interface{} {
	if err != nil {
		return err.Error()
	}
	return nil
}
