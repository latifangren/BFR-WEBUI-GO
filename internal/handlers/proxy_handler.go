package handlers

import (
	"encoding/json"
	"net/http"
	"os"

	"bfr-webui-go/internal/logger"
	"bfr-webui-go/internal/proxy"
)

type proxyControlRequest struct {
	Action string `json:"action"` // "start", "stop", "restart", "switch_core"
	Core   string `json:"core"`   // "mihomo", "sing-box", "clash"
	Mode   string `json:"mode"`   // "rule", "global", "direct"
}

type watchdogRequest struct {
	Enable bool `json:"enable"`
}

type configRequest struct {
	Content string `json:"content"`
}

func HandleProxyStatus(w http.ResponseWriter, r *http.Request) {
	cores := proxy.DetectCores()
	mode := proxy.GetMode()
	wd := proxy.GetWatchdog()
	boxSettings, _ := proxy.LoadBoxSettings()
	version, _ := proxy.GetClashVersion()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"cores":        cores,
		"mode":         mode,
		"watchdog":     wd,
		"box_settings": boxSettings,
		"version":      version,
		"config_path":  proxy.DetectConfigPath(),
	})
}

func HandleProxyControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req proxyControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid payload"})
		return
	}

	// 1. Switch active core binary (Mihomo vs Sing-box)
	if req.Action == "switch_core" && req.Core != "" {
		settings, _ := proxy.LoadBoxSettings()
		settings.BinName = req.Core
		if err := proxy.SaveBoxSettings(settings); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
		_ = proxy.ControlService("restart")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Core switched to " + req.Core})
		return
	}

	// 2. Routing Mode (Rule / Global / Direct)
	if req.Mode != "" {
		validModes := map[string]bool{
			"rule":   true,
			"global": true,
			"direct": true,
			"script": true,
		}
		if !validModes[req.Mode] {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid mode; must be one of: rule, global, direct, script"})
			return
		}
		err := proxy.SetMode(req.Mode)
		if err == nil {
			logger.Get().Infof("proxy", "Proxy mode set to %s", req.Mode)
		}
	}

	// 3. Service Lifecycle Control (start, stop, restart)
	if req.Action != "" && req.Action != "switch_core" {
		err := proxy.ControlService(req.Action)
		if err != nil {
			logger.Get().Errorf("proxy", "Control action %s failed: %v", req.Action, err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
		logger.Get().Infof("proxy", "Proxy service action %s completed", req.Action)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"action":  req.Action,
		"mode":    req.Mode,
	})
}

func HandleProxySettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		settings, err := proxy.LoadBoxSettings()
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"settings": settings,
			"error":    errString(err),
		})
		return
	}

	if r.Method == http.MethodPost {
		var req proxy.BoxSettings
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid settings payload"})
			return
		}

		err := proxy.SaveBoxSettings(req)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": err == nil,
			"error":   errString(err),
		})
	}
}

func HandleProxyApps(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		cfg, _ := proxy.LoadAppProxyConfig()
		apps, err := proxy.GetInstalledApps()
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"mode":     cfg.Mode,
			"selected": cfg.Packages,
			"apps":     apps,
			"error":    errString(err),
		})
		return
	}

	if r.Method == http.MethodPost {
		var req proxy.AppProxyConfig
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid apps payload"})
			return
		}

		err := proxy.SaveAppProxyConfig(req)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": err == nil,
			"error":   errString(err),
		})
	}
}

func HandleProxyHotspot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		cfg, err := proxy.LoadHotspotProxyConfig()
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"config": cfg,
			"error":  errString(err),
		})
		return
	}

	if r.Method == http.MethodPost {
		var req proxy.HotspotProxyConfig
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid hotspot payload"})
			return
		}

		err := proxy.SaveHotspotProxyConfig(req)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": err == nil,
			"error":   errString(err),
		})
	}
}

func HandleProxySubs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		cfg := proxy.GetSubsConfig()
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"subs": cfg,
		})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Action string `json:"action"` // "update_sub", "update_geox"
			URL    string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid payload"})
			return
		}

		if req.Action == "update_geox" {
			msg, err := proxy.UpdateGeoXNow()
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": err == nil,
				"message": msg,
				"error":   errString(err),
			})
			return
		}

		// Update Subscription
		msg, err := proxy.UpdateSubscriptionNow(req.URL)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": err == nil,
			"message": msg,
			"error":   errString(err),
		})
	}
}

func HandleProxyNodes(w http.ResponseWriter, r *http.Request) {
	groups, err := proxy.GetProxyGroups()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"groups": groups,
		"error":  errString(err),
	})
}

func HandleProxyNodeSelect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Group string `json:"group"`
		Node  string `json:"node"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Group == "" || req.Node == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Missing group or node name"})
		return
	}

	err := proxy.SelectProxyNode(req.Group, req.Node)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"group":   req.Group,
		"node":    req.Node,
		"error":   errString(err),
	})
}

func HandleProxyDelay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Node string `json:"node"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Node == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Missing node name"})
		return
	}

	delay, err := proxy.TestNodeDelay(req.Node)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"node":    req.Node,
		"delay":   delay,
		"error":   errString(err),
	})
}

func HandleProxyLogs(w http.ResponseWriter, r *http.Request) {
	proxy.StreamLogs(w, r)
}

func HandleProxyWatchdog(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"enabled": proxy.GetWatchdog(),
		})
		return
	}

	if r.Method == http.MethodPost {
		var req watchdogRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid payload"})
			return
		}
		proxy.SetWatchdog(req.Enable)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"enabled": proxy.GetWatchdog(),
		})
	}
}

func HandleProxyConfig(w http.ResponseWriter, r *http.Request) {
	configPath := proxy.DetectConfigPath()
	if configPath == "" {
		configPath = "/data/adb/box/clash/config.yaml"
	}

	if r.Method == http.MethodGet {
		data, err := os.ReadFile(configPath)
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"path":    configPath,
				"content": "",
				"error":   err.Error(),
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"path":    configPath,
			"content": string(data),
		})
		return
	}

	if r.Method == http.MethodPost {
		var req configRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid payload"})
			return
		}

		err := os.WriteFile(configPath, []byte(req.Content), 0644)
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"path":    configPath,
		})
	}
}
