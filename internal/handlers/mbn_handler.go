//go:build !nomodem

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"bfr-webui-go/internal/efs"
	"bfr-webui-go/internal/logger"
	"bfr-webui-go/internal/modem"
)

// HandleQualcommMbnOverview returns available SW/HW MBN configs, active profile, and autoselect status.
func HandleQualcommMbnOverview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	slot := 0
	if s := r.URL.Query().Get("slot"); s != "" {
		if val, err := strconv.Atoi(s); err == nil && (val == 0 || val == 1) {
			slot = val
		}
	}

	overview := efs.MbnOverview{
		PreviousProfile: modem.GetPreviousMbnProfile(),
		SwConfigs:       make([]efs.MbnConfigItem, 0),
		HwConfigs:       make([]efs.MbnConfigItem, 0),
	}

	sess, _ := efs.GetManager().GetSession()
	if sess != nil {
		if autoStatus, err := sess.ReadMbnAutoselect(); err == nil {
			overview.Autoselect = autoStatus
		}
	}
	overview.SwConfigs = efs.DiscoverMbnConfigs(sess, "sw")
	overview.HwConfigs = efs.DiscoverMbnConfigs(sess, "hw")

	// 1. Try querying active profile over QMI PDC
	matchedActive := false
	if active, err := modem.QueryActiveMbnProfile(slot); err == nil && active != "" {
		overview.ActiveProfile = active
		for i := range overview.SwConfigs {
			if overview.SwConfigs[i].Name == active || overview.SwConfigs[i].Path == active {
				overview.SwConfigs[i].Active = true
				matchedActive = true
			}
		}
	}

	// 2. Active operator matching fallback
	if !matchedActive {
		simStatus := modem.GetSimStatus()
		activeCarrier := ""
		if simStatus != nil && slot < len(simStatus.Slots) {
			activeCarrier = simStatus.Slots[slot].Carrier
		}
		if activeCarrier == "" {
			sig, _ := modem.GetSignalInfo()
			activeCarrier = sig.Operator
		}

		if activeCarrier != "" && activeCarrier != "Unknown" {
			carrierClean := strings.ToLower(activeCarrier)
			keywords := []string{"smartfren", "telkomsel", "indosat", "xl", "tri", "three", "axis", "by.u"}
			matchedKeyword := ""
			for _, kw := range keywords {
				if strings.Contains(carrierClean, kw) {
					matchedKeyword = kw
					break
				}
			}

			if matchedKeyword != "" {
				for i := range overview.SwConfigs {
					cfgLower := strings.ToLower(overview.SwConfigs[i].Path)
					if strings.Contains(cfgLower, matchedKeyword) {
						overview.SwConfigs[i].Active = true
						overview.ActiveProfile = fmt.Sprintf("%s (Auto by IMSI)", overview.SwConfigs[i].Name)
						matchedActive = true
						break
					}
				}
			}
		}
	}

	// 3. PixelParts index check (Evolution X / Custom ROM feature)
	if _, err := os.Stat("/system_ext/etc/init/init.pixelparts_mbn.rc"); err == nil {
		currentMode := "stock"
		if out, err := exec.Command("getprop", "persist.sys.pixelparts.mbn").Output(); err == nil {
			trimmed := strings.TrimSpace(string(out))
			if trimmed != "" {
				currentMode = trimmed
			}
		}
		overview.PixelParts = &efs.PixelPartsMbnMode{
			Supported: true,
			Current:   currentMode,
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":       true,
		"overview": overview,
	})
}

// HandleQualcommMbnActivate selects and activates an MBN carrier profile on the modem.
func HandleQualcommMbnActivate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Profile string `json:"profile"`
		Slot    int    `json:"slot"`
		SSR     bool   `json:"ssr"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "invalid JSON body"})
		return
	}

	if req.Profile == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "profile is required"})
		return
	}

	if err := modem.ApplyMbnProfile(req.Profile, req.Slot); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	ssrTriggered := false
	if req.SSR {
		if err := efs.GetManager().RestartModem(); err != nil {
			logger.Get().Warnf("PDC", "MBN activated but SSR failed: %v", err)
		} else {
			ssrTriggered = true
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":            true,
		"profile":       req.Profile,
		"slot":          req.Slot,
		"ssr_triggered": ssrTriggered,
	})
}

// HandleQualcommMbnAutoselect updates the MCFG autoselect by UIM mode.
func HandleQualcommMbnAutoselect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Slot int  `json:"slot"` // 0 for SIM1, 1 for SIM2
		Mode int  `json:"mode"` // 0: disabled, 1: auto, 2: with MBN update only
		SSR  bool `json:"ssr"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "invalid JSON body"})
		return
	}

	sess, err := efs.GetManager().GetSession()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	opts := efs.WriteOptions{
		ForceItem: -1,
		Backup:    true,
		Verify:    true,
	}

	if err := sess.WriteMbnAutoselect(req.Slot, req.Mode, opts); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	ssrTriggered := false
	if req.SSR {
		if err := efs.GetManager().RestartModem(); err != nil {
			logger.Get().Warnf("PDC", "Autoselect updated but SSR failed: %v", err)
		} else {
			ssrTriggered = true
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":            true,
		"slot":          req.Slot,
		"mode":          req.Mode,
		"ssr_triggered": ssrTriggered,
	})
}

// HandleQualcommMbnRevert restores the previous active MBN profile in one click.
func HandleQualcommMbnRevert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Slot int  `json:"slot"`
		SSR  bool `json:"ssr"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Slot = 0
		req.SSR = true
	}

	prev := modem.GetPreviousMbnProfile()
	if prev == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "no previous profile recorded for rollback"})
		return
	}

	if err := modem.ApplyMbnProfile(prev, req.Slot); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	ssrTriggered := false
	if req.SSR {
		if err := efs.GetManager().RestartModem(); err != nil {
			logger.Get().Warnf("PDC", "Profile reverted but SSR failed: %v", err)
		} else {
			ssrTriggered = true
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":               true,
		"restored_profile": prev,
		"ssr_triggered":    ssrTriggered,
	})
}
// HandleQualcommMbnPixelParts toggles the PixelParts / Evolution X mbn index mode.
func HandleQualcommMbnPixelParts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Mode string `json:"mode"` // "stock", "merged", "module"
		SSR  bool   `json:"ssr"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "invalid JSON body"})
		return
	}

	validModes := map[string]bool{"stock": true, "merged": true, "module": true}
	if !validModes[req.Mode] {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "invalid mode (must be stock, merged, or module)"})
		return
	}

	if err := exec.Command("setprop", "persist.sys.pixelparts.mbn", req.Mode).Run(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	ssrTriggered := false
	if req.SSR {
		if err := efs.GetManager().RestartModem(); err != nil {
			logger.Get().Warnf("PDC", "PixelParts mode updated but SSR failed: %v", err)
		} else {
			ssrTriggered = true
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":            true,
		"mode":          req.Mode,
		"ssr_triggered": ssrTriggered,
	})
}
