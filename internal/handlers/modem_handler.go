package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"bfr-webui-go/internal/logger"
	"bfr-webui-go/internal/modem"
)

type atRequest struct {
	Command string `json:"command"`
}

func HandleModemSignal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sig, err := modem.GetSignalInfo()
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(sig)
}

func HandleModemBands(w http.ResponseWriter, r *http.Request) {
	mgr := modem.GetManager()

	if r.Method == http.MethodGet {
		cfg, err := mgr.LoadConfig()
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(cfg)
		return
	}

	if r.Method == http.MethodPost {
		var cfg modem.BandConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid band config request payload"})
			return
		}

		err := mgr.ApplyBandLock(cfg)
		if err == nil {
			logger.Get().Infof("Modem", "Band lock applied successfully: engine=%s, RAT=%s", cfg.Engine, cfg.PreferredRAT)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": err == nil,
			"error":   fmt.Sprintf("%v", err),
		})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func HandleModemAT(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req atRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Command == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid AT command request payload"})
		return
	}

	resp := modem.ExecuteATCommand(req.Command)
	if resp.Success {
		logger.Get().Infof("Modem", "AT Command executed successfully: %s", req.Command)
	} else {
		logger.Get().Warnf("Modem", "AT Command execution failed: %s (%s)", req.Command, resp.Response)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func HandleModemReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mgr := modem.GetManager()
	err := mgr.ResetBandLock()
	if err == nil {
		logger.Get().Infof("Modem", "Modem band lock settings reset to default")
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"error":   fmt.Sprintf("%v", err),
	})
}
