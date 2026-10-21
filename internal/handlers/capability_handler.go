//go:build !nomodem

package handlers

import (
	"encoding/json"
	"net/http"

	"bfr-webui-go/internal/efs"
	"bfr-webui-go/internal/logger"
)

// HandleEfsCapability handles GET and POST requests for RRC Capabilities (Fitur A).
func HandleEfsCapability(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	sess, err := efs.GetManager().GetSession()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	switch r.Method {
	case http.MethodGet:
		state, err := sess.ReadCapability()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":           true,
			"capability":   state,
			"token_string": state.TokenString,
			"files":        state.Files,
		})

	case http.MethodPost:
		var req struct {
			State  efs.CapabilityState `json:"state"`
			Backup *bool               `json:"backup,omitempty"`
			Verify *bool               `json:"verify,omitempty"`
			SSR    bool                `json:"ssr"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "invalid JSON body"})
			return
		}

		backup := true
		if req.Backup != nil {
			backup = *req.Backup
		}
		verify := true
		if req.Verify != nil {
			verify = *req.Verify
		}

		opts := efs.WriteOptions{
			ForceItem: -1,
			Backup:    backup,
			Verify:    verify,
		}

		if err := sess.ApplyCapability(&req.State, opts); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
			return
		}

		ssrTriggered := false
		if req.SSR {
			if err := efs.GetManager().RestartModem(); err != nil {
				logger.Get().Warnf("EFS", "Apply capability succeeded but SSR restart failed: %v", err)
			} else {
				ssrTriggered = true
			}
		}

		tokens := efs.BuildTokenString(&req.State)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":            true,
			"token_string":  tokens,
			"ssr_triggered": ssrTriggered,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
