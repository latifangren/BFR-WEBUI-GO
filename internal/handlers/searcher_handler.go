//go:build !nomodem

package handlers

import (
	"encoding/json"
	"net/http"

	"bfr-webui-go/internal/efs"
	"bfr-webui-go/internal/logger"
)

// HandleEfsSearcher handles GET and POST requests for RAT Searcher Preferences.
func HandleEfsSearcher(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	sess, err := efs.GetManager().GetSession()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	switch r.Method {
	case http.MethodGet:
		prefs, err := sess.ReadSearcherPreferences()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":          true,
			"preferences": prefs,
		})

	case http.MethodPost:
		var req struct {
			Preferences efs.SearcherPreferences `json:"preferences"`
			Backup      *bool                   `json:"backup,omitempty"`
			Verify      *bool                   `json:"verify,omitempty"`
			SSR         bool                    `json:"ssr"`
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

		if err := sess.WriteSearcherPreferences(&req.Preferences, opts); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
			return
		}

		ssrTriggered := false
		if req.SSR {
			if err := efs.GetManager().RestartModem(); err != nil {
				logger.Get().Warnf("EFS", "Searcher write succeeded but SSR restart failed: %v", err)
			} else {
				ssrTriggered = true
			}
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":            true,
			"ssr_triggered": ssrTriggered,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleEfsExtraLocks handles GET and POST requests for extra GERAN, WCDMA, and CSP locks.
func HandleEfsExtraLocks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	sess, err := efs.GetManager().GetSession()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}

	switch r.Method {
	case http.MethodGet:
		locks, err := sess.ReadExtraLocks()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":    true,
			"locks": locks,
		})

	case http.MethodPost:
		var req struct {
			Locks  efs.ExtraLocksConfig `json:"locks"`
			Backup *bool                `json:"backup,omitempty"`
			Verify *bool                `json:"verify,omitempty"`
			SSR    bool                 `json:"ssr"`
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

		if err := sess.WriteExtraLocks(&req.Locks, opts); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
			return
		}

		ssrTriggered := false
		if req.SSR {
			if err := efs.GetManager().RestartModem(); err != nil {
				logger.Get().Warnf("EFS", "Locks write succeeded but SSR restart failed: %v", err)
			} else {
				ssrTriggered = true
			}
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":            true,
			"ssr_triggered": ssrTriggered,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
