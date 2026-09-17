package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"bfr-webui-go/internal/logger"
	"bfr-webui-go/internal/samsung"
)

type lockBandRequest struct {
	Slot  int   `json:"slot"`
	Bands []int `json:"bands"`
}

type autoBandRequest struct {
	Slot int `json:"slot"`
}

// HandleSamsungStatus returns hardware status, OEM verification, and tool availability.
func HandleSamsungStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	st := samsung.GetDeviceStatus()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(st)
}

// HandleSamsungBands returns supported and active frequency bands for the requested SIM slot.
func HandleSamsungBands(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	slotStr := r.URL.Query().Get("slot")
	slot := 0
	if slotStr != "" {
		s, err := strconv.Atoi(slotStr)
		if err != nil || s < 0 || s > 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid slot parameter, must be 0 or 1"})
			return
		}
		slot = s
	}

	bands, err := samsung.GetBands(slot)
	w.Header().Set("Content-Type", "application/json")
	if err != nil && (bands == nil || len(bands.Bands) == 0) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
			"data":  bands,
		})
		return
	}

	_ = json.NewEncoder(w).Encode(bands)
}

// HandleSamsungLock applies band locking to specified bands on a SIM slot.
func HandleSamsungLock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req lockBandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request payload"})
		return
	}

	if req.Slot < 0 || req.Slot > 1 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid slot, must be 0 or 1"})
		return
	}

	if len(req.Bands) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "At least one band must be specified"})
		return
	}

	if len(req.Bands) > 64 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Too many bands specified (max 64)"})
		return
	}

	for _, b := range req.Bands {
		if b <= 0 || b > 1000 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid band ID"})
			return
		}
	}

	if err := samsung.LockBands(req.Slot, req.Bands); err != nil {
		logger.Get().Errorf("Samsung", "Failed to lock bands: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Samsung band lock applied successfully",
		"slot":    req.Slot,
		"bands":   req.Bands,
	})
}

// HandleSamsungAuto restores automatic band selection on the given SIM slot.
func HandleSamsungAuto(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req autoBandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Slot = 0
	}

	if req.Slot < 0 || req.Slot > 1 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid slot, must be 0 or 1"})
		return
	}

	if err := samsung.ResetAuto(req.Slot); err != nil {
		logger.Get().Errorf("Samsung", "Failed to reset bands to auto: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Restored automatic band selection",
		"slot":    req.Slot,
	})
}

type simSwitchRequest struct {
	Slot int `json:"slot"`
}

// HandleSamsungSimSwitch switches the active data SIM to the specified slot.
func HandleSamsungSimSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req simSwitchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request payload"})
		return
	}

	if req.Slot < 0 || req.Slot > 1 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid slot, must be 0 or 1"})
		return
	}

	res, err := samsung.SwitchDataSim(req.Slot)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		logger.Get().Errorf("Samsung", "Failed to switch data SIM: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(res)
}

// HandleSamsungThermal returns 5G thermal diagnostics and throttling metrics.
func HandleSamsungThermal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	st, err := samsung.Get5GThermalStatus()
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		logger.Get().Errorf("Samsung", "Failed to get 5G thermal status: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(st)
}
