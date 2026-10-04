//go:build !nomodem

package handlers

import (
	"encoding/json"
	"net/http"

	"bfr-webui-go/internal/logger"
	"bfr-webui-go/internal/shannon"
)

// HandleShannonStatus returns current Google Tensor Shannon modem capability and state.
func HandleShannonStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	st := shannon.GetStatus()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(st)
}

// HandleShannonLock applies manual frequency band locks on Google Tensor modem.
func HandleShannonLock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req shannon.LockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request body"})
		return
	}

	err := shannon.ApplyShannonSettings(req)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		logger.Get().Warnf("ShannonHandler", "Failed to lock bands: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"status":  shannon.GetStatus(),
	})
}

// HandleShannonReset restores factory default band lists on Google Tensor modem.
func HandleShannonReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Slot int `json:"slot"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	err := shannon.ResetBands(req.Slot)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		logger.Get().Warnf("ShannonHandler", "Failed to reset bands: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"status":  shannon.GetStatus(),
	})
}
