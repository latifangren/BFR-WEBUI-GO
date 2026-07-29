package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"bfr-webui-go/internal/network"
	"bfr-webui-go/internal/sysinfo"
)

type sysctlRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ttlRequest struct {
	Enable bool `json:"enable"`
	TTL    int  `json:"ttl"`
}

type interfaceRequest struct {
	Interface  string `json:"interface"`
	MTU        int    `json:"mtu"`
	TxQueueLen int    `json:"txqueuelen"`
}

type dnsRequest struct {
	Primary   string `json:"primary"`
	Secondary string `json:"secondary"`
}

type pingRequest struct {
	Host  string `json:"host"`
	Count int    `json:"count"`
}

type rpsRequest struct {
	Interface string `json:"interface"`
	Bitmask   string `json:"bitmask"`
}

func HandleNetworkTweaks(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		ifaces, _ := network.GetInterfaces()
		ttlStatus := network.GetTTLSpoofStatus()
		tcpCongestion, _ := network.GetSysctl("net.ipv4.tcp_congestion_control")
		tcpFastOpen, _ := network.GetSysctl("net.ipv4.tcp_fastopen")
		tweaksJson, _ := network.LoadTweaks()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"interfaces":     ifaces,
			"ttl_spoof":      ttlStatus,
			"tcp_congestion": tcpCongestion,
			"tcp_fastopen":   tcpFastOpen,
			"preset_dns":     network.PresetDNS,
			"tweaks_json":    tweaksJson,
		})
		return
	}

	if r.Method == http.MethodPost {
		action := r.URL.Query().Get("action")
		switch action {
		case "sysctl":
			var req sysctlRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
				err := network.SetSysctl(req.Key, req.Value)
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": err == nil, "error": fmt.Sprintf("%v", err)})
				return
			}
		case "ttl":
			var req ttlRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
				err := network.SetTTLSpoofSDK(req.Enable, req.TTL)
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": err == nil, "error": fmt.Sprintf("%v", err)})
				return
			}
		case "interface":
			var req interfaceRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
				err := network.SetInterfaceConfig(req.Interface, req.MTU, req.TxQueueLen)
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": err == nil, "error": fmt.Sprintf("%v", err)})
				return
			}
		case "dns":
			var req dnsRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
				err := network.SetDNS(req.Primary, req.Secondary)
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": err == nil, "error": fmt.Sprintf("%v", err)})
				return
			}
		case "save_tweaks":
			var req network.TweaksConfig
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
				errSave := network.SaveTweaks(req)
				_ = network.ApplyAllTweaks() // Apply immediately
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": errSave == nil, "error": fmt.Sprintf("%v", errSave)})
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid action"})
	}
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
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(network.PresetDNS)
		return
	}

	if r.Method == http.MethodPost {
		var req dnsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid DNS request"})
			return
		}
		err := network.SetDNS(req.Primary, req.Secondary)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": err == nil, "error": fmt.Sprintf("%v", err)})
	}
}

func HandleRPS(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		configs, err := network.GetRPSConfigs()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"configs": configs, "error": fmt.Sprintf("%v", err)})
		return
	}

	if r.Method == http.MethodPost {
		var req rpsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Interface == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid RPS request"})
			return
		}
		err := network.ConfigureRPS(req.Interface, req.Bitmask)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": err == nil, "error": fmt.Sprintf("%v", err)})
	}
}

func HandleTTL(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		status := network.GetTTLSpoofStatus()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ttl_spoof": status})
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
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": err == nil, "error": fmt.Sprintf("%v", err)})
	}
}
