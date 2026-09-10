package handlers

import (
	"encoding/json"
	"net/http"

	"bfr-webui-go/internal/logger"
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

		d1, d2 := network.GetActiveDNS()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"interfaces":     ifaces,
			"ttl_spoof":      ttlStatus,
			"tcp_congestion": tcpCongestion,
			"tcp_fastopen":   tcpFastOpen,
			"preset_dns":     network.PresetDNS,
			"active_dns1":    d1,
			"active_dns2":    d2,
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
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": err == nil, "error": errString(err)})
				return
			}
		case "ttl":
			var req ttlRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
				err := network.SetTTLSpoofSDK(req.Enable, req.TTL)
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": err == nil, "error": errString(err)})
				return
			}
		case "interface":
			var req interfaceRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
				err := network.SetInterfaceConfig(req.Interface, req.MTU, req.TxQueueLen)
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": err == nil, "error": errString(err)})
				return
			}
		case "dns":
			var req dnsRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
				logger.Get().Infof("network", "DNS change requested: primary=%s, secondary=%s", req.Primary, req.Secondary)
				err := network.SetDNS(req.Primary, req.Secondary)
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": err == nil, "error": errString(err)})
				return
			}
		case "save_tweaks":
			var req network.TweaksConfig
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
				errSave := network.SaveTweaks(req)
				_ = network.ApplyAllTweaks() // Apply immediately
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": errSave == nil, "error": errString(errSave)})
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
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid DNS request"})
			return
		}
		logger.Get().Infof("network", "DNS preset update requested: primary=%s, secondary=%s", req.Primary, req.Secondary)
		err := network.SetDNS(req.Primary, req.Secondary)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": err == nil, "error": errString(err)})
	}
}

func HandleRPS(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		configs, err := network.GetRPSConfigs()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"configs": configs, "error": errString(err)})
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
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": err == nil, "error": errString(err)})
	}
}

func HandleTTL(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		status := network.GetTTLSpoofStatus()
		defaultTTL := network.GetDefaultTTL()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ttl_spoof":   status,
			"current_ttl": defaultTTL,
			"ttl":         defaultTTL,
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

func HandleNetworkTweaksRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	err := network.RestoreSysctlDefaults()
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Sysctl defaults restored successfully"})
}

func errString(err error) interface{} {
	if err != nil {
		return err.Error()
	}
	return nil
}
