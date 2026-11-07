package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"bfr-webui-go/internal/hotspot"
	"bfr-webui-go/internal/logger"
)

type hotspotControlRequest struct {
	Type     string `json:"type"` // "wifi", "usb", "ethernet" (default "wifi")
	Enable   bool   `json:"enable"`
	SSID     string `json:"ssid"`
	Pass     string `json:"password"`
	Security string `json:"security"` // "wpa2" or "open"
	Band     string `json:"band"`     // "any", "2", "5"
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

	var err error
	reqType := req.Type
	if reqType == "" || reqType == "wifi" || reqType == "softap" {
		if req.Band != "" && hotspot.ValidateBand(req.Band) == nil {
			if cfg, cfgErr := hotspot.LoadRouterConfig(); cfgErr == nil && cfg != nil && cfg.Band != req.Band {
				cfg.Band = req.Band
				_ = hotspot.SaveRouterConfig(cfg)
			}
		}
		err = hotspot.ToggleHotspot(req.Enable, req.SSID, req.Pass, req.Security)
		if err == nil {
			logger.Get().Infof("hotspot", "Wi-Fi Hotspot toggle executed: enable=%v, ssid=%s, sec=%s", req.Enable, req.SSID, req.Security)
		}
	} else {
		err = hotspot.ToggleTethering(reqType, req.Enable)
		if err == nil {
			logger.Get().Infof("hotspot", "Tethering toggle executed: type=%s, enable=%v", reqType, req.Enable)
		}
	}

	var errStr string
	if err != nil {
		errStr = err.Error()
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"error":   errStr,
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

func HandleHotspotMACFilter(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		cfg, err := hotspot.LoadMACFilterConfig()
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
		status := hotspot.GetMACFilterStatus()
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"config": cfg,
			"status": status,
		})
		return
	}

	if r.Method == http.MethodPost {
		var cfg hotspot.MACFilterConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid MAC filter request payload"})
			return
		}
		err := hotspot.ApplyMACFilter(&cfg)
		if err == nil {
			logger.Get().Infof("hotspot", "MAC filter updated and applied: mode=%s", cfg.Mode)
		}

		var errStr string
		if err != nil {
			errStr = err.Error()
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": err == nil,
			"error":   errStr,
			"status":  hotspot.GetMACFilterStatus(),
		})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func HandleHotspotSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		cfg, err := hotspot.LoadRouterConfig()
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
		status := hotspot.GetRouterStatus()
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"config": cfg,
			"status": status,
		})
		return
	}

	if r.Method == http.MethodPost {
		var cfg hotspot.RouterConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid router settings payload"})
			return
		}

		// Strict validation: every field below reaches a root shell.
		hotspot.NormalizeRouterConfig(&cfg)
		if err := hotspot.ValidateRouterConfig(&cfg); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   err.Error(),
				"field":   validationField(err),
			})
			return
		}

		// 1. Persist desired state and sync SoftAP credentials to AOSP XML if provided.
		saveErr := hotspot.SaveRouterConfig(&cfg)
		if saveErr == nil && cfg.SSID != "" {
			_ = hotspot.UpdateSavedSoftApCredentials(cfg.SSID, cfg.Password, cfg.SecurityType)
		}

		// 2. Apply runtime state and report the result separately.
		applyErr := error(nil)
		applied := false
		if saveErr == nil {
			names := hotspot.GetActiveInterfaceNames()
			switch {
			case cfg.GatewayMode != "static":
				// Android owns the gateway, DHCP and DNS. BFR only makes sure
				// no leftover static state of its own remains.
				hotspot.RemoveStaticGateway("")
				applied = true
			case len(names) == 0:
				// No tether interface is up: nothing can be applied to the
				// kernel right now, so this is not a success.
				applyErr = errors.New("no active tether interface: settings saved but not applied")
			default:
				applyErr = hotspot.ApplyStaticGateway(names[0], &cfg)
				applied = applyErr == nil
			}
		}
		logger.Get().Infof("hotspot", "Router settings saved: mode=%s, gateway=%s, band=%s, applyErr=%v", cfg.GatewayMode, cfg.GatewayIP, cfg.Band, applyErr)

		errStr := ""
		if saveErr != nil {
			errStr = saveErr.Error()
		} else if applyErr != nil {
			errStr = applyErr.Error()
		}

		status := hotspot.GetRouterStatus()
		// The applied flag is derived from the verified kernel state, so a
		// save that could not be applied is never reported as applied.
		applied = applied && status.Applied

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"saved":   saveErr == nil,
			"applied": applied,
			"success": saveErr == nil && applyErr == nil,
			"error":   errStr,
			"status":  status,
		})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// validationField extracts the offending field name from a validation error.
func validationField(err error) string {
	var ve *hotspot.ValidationError
	if errors.As(err, &ve) {
		return ve.Field
	}
	return ""
}
