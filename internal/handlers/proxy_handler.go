package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"bfr-webui-go/internal/logger"
	"bfr-webui-go/internal/proxy"
)

type proxyControlRequest struct {
	Action string `json:"action"`
	Mode   string `json:"mode"`
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
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"cores":    cores,
		"mode":     mode,
		"watchdog": wd,
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

	if req.Mode != "" {
		// M-3: validate mode against a strict whitelist before forwarding.
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
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": err == nil, "error": fmt.Sprintf("%v", err)})
		return
	}

	if req.Action != "" {
		validActions := map[string]bool{
			"start":   true,
			"stop":    true,
			"restart": true,
		}
		if !validActions[req.Action] {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid action; must be one of: start, stop, restart"})
			return
		}
		err := proxy.ControlService(req.Action)
		if err == nil {
			logger.Get().Infof("proxy", "Proxy service action executed: %s", req.Action)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": err == nil, "error": fmt.Sprintf("%v", err)})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "No action specified"})
}

func HandleProxyLogs(w http.ResponseWriter, r *http.Request) {
	proxy.StreamLogs(w, r)
}

func HandleProxyWatchdog(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"watchdog": proxy.GetWatchdog(),
		})
		return
	}

	if r.Method == http.MethodPost {
		var req watchdogRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid watchdog configuration"})
			return
		}
		proxy.SetWatchdog(req.Enable)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "watchdog": proxy.GetWatchdog()})
	}
}

func HandleProxyConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		path, content, err := proxy.ReadConfig()
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"path":  path,
				"error": err.Error(),
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"path":    path,
			"content": content,
		})
		return
	}

	if r.Method == http.MethodPost {
		var req configRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid config body"})
			return
		}
		err := proxy.SaveConfig(req.Content)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": err == nil,
			"error":   fmt.Sprintf("%v", err),
		})
	}
}
