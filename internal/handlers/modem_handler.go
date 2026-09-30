package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

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
		targetSlot := -1
		slotStr := r.URL.Query().Get("slot")
		if slotStr == "" {
			slotStr = r.URL.Query().Get("sim_slot")
		}
		if s, err := strconv.Atoi(slotStr); err == nil && (s == 0 || s == 1) {
			targetSlot = s
		}

		cfg, err := mgr.LoadConfigForSlot(targetSlot)
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

		res, err := mgr.ApplyBandLock(cfg)
		if err == nil {
			logger.Get().Infof("Modem", "Band lock applied successfully: engine=%s, RAT=%s", cfg.Engine, cfg.PreferredRAT)
		} else {
			logger.Get().Warnf("Modem", "Band lock failed: %v", err)
		}

		updatedCfg, _ := mgr.LoadConfig()

		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
		}
		resp := map[string]interface{}{
			"success": err == nil,
			"config":  updatedCfg,
			"result":  res,
		}
		if err != nil {
			resp["error"] = err.Error()
		}
		_ = json.NewEncoder(w).Encode(resp)
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

	updatedCfg, _ := mgr.LoadConfig()

	w.Header().Set("Content-Type", "application/json")
	resp := map[string]interface{}{
		"success": err == nil,
		"config":  updatedCfg,
	}
	if err != nil {
		resp["error"] = err.Error()
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// HandleModemSimStatus returns current multi-SIM status and active data slot.
func HandleModemSimStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	st := modem.GetSimStatus()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(st)
}

// HandleModemSimSwitch switches the active mobile data SIM slot.
func HandleModemSimSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Slot int `json:"slot"`
	}
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

	res, err := modem.SwitchDataSim(req.Slot)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		logger.Get().Errorf("Modem", "Failed to switch data SIM: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(res)
}

// HandleModemThermal returns real-time modem thermal sensors and throttling telemetry.
func HandleModemThermal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	st, err := modem.Get5GThermalStatus()
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		logger.Get().Errorf("Modem", "Failed to get thermal status: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(st)
}

// HandleModemCellLock gets or sets physical cell and EARFCN lock.
func HandleModemCellLock(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		targetSlot := 0
		slotStr := r.URL.Query().Get("slot")
		if slotStr == "" {
			slotStr = r.URL.Query().Get("sim_slot")
		}
		if s, err := strconv.Atoi(slotStr); err == nil && (s == 0 || s == 1) {
			targetSlot = s
		}

		active, earfcn, pci, err := modem.GetLTECellLockStatus(targetSlot)
		resp := map[string]interface{}{
			"supported":   true,
			"active_lock": active,
			"earfcn":      earfcn,
			"pci":         pci,
			"slot":        targetSlot,
		}
		if err != nil {
			resp["message"] = err.Error()
		}
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Slot   int    `json:"slot"`
			Type   string `json:"type"`
			EARFCN uint32 `json:"earfcn"`
			PCI    uint16 `json:"pci"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Invalid cell lock payload",
			})
			return
		}

		if req.EARFCN == 0 || req.PCI > 503 {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Invalid EARFCN or PCI (PCI must be 0-503, EARFCN > 0)",
			})
			return
		}

		err := modem.ApplyLTECellLock(req.Slot, req.EARFCN, req.PCI)
		resp := map[string]interface{}{
			"success": err == nil,
			"slot":    req.Slot,
			"earfcn":  req.EARFCN,
			"pci":     req.PCI,
		}
		if err != nil {
			logger.Get().Warnf("Modem", "Cell lock rejected: slot=%d, earfcn=%d, pci=%d, err=%v", req.Slot, req.EARFCN, req.PCI, err)
			w.WriteHeader(http.StatusBadRequest)
			resp["error"] = err.Error()
			resp["message"] = "Modem rejected cell lock command: " + err.Error()
		} else {
			logger.Get().Infof("Modem", "Cell lock applied successfully: slot=%d, earfcn=%d, pci=%d", req.Slot, req.EARFCN, req.PCI)
			resp["message"] = "LTE Cell Lock applied successfully"
		}
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// HandleModemCellLockClear clears any active cell lock on the modem.
func HandleModemCellLockClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	var req struct {
		Slot int `json:"slot"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	err := modem.ClearLTECellLock(req.Slot)
	resp := map[string]interface{}{
		"success": err == nil,
		"slot":    req.Slot,
	}
	if err != nil {
		logger.Get().Warnf("Modem", "Failed to clear cell lock on slot %d: %v", req.Slot, err)
		w.WriteHeader(http.StatusBadRequest)
		resp["error"] = err.Error()
		resp["message"] = "Modem failed to clear cell lock: " + err.Error()
	} else {
		logger.Get().Infof("Modem", "Cell lock cleared successfully on slot %d", req.Slot)
		resp["message"] = "LTE Cell Lock cleared successfully"
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// HandleModemPlmnLock handles automatic or manual PLMN carrier locking.
func HandleModemPlmnLock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	var req struct {
		Slot int    `json:"slot"`
		Mode string `json:"mode"`
		MCC  uint16 `json:"mcc"`
		MNC  uint16 `json:"mnc"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid PLMN lock payload",
		})
		return
	}

	err := modem.ApplyPLMNLock(req.Slot, req.Mode, req.MCC, req.MNC)
	resp := map[string]interface{}{
		"success": err == nil,
		"slot":    req.Slot,
		"mode":    req.Mode,
	}
	if err != nil {
		logger.Get().Warnf("Modem", "Failed to apply PLMN lock (%s): %v", req.Mode, err)
		w.WriteHeader(http.StatusBadRequest)
		resp["error"] = err.Error()
		resp["message"] = "Modem rejected PLMN lock: " + err.Error()
	} else {
		logger.Get().Infof("Modem", "PLMN lock applied successfully: mode=%s, mcc=%d, mnc=%d", req.Mode, req.MCC, req.MNC)
		resp["message"] = "PLMN network selection updated successfully"
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// HandleModemHardwareBands queries factory hardware-supported frequency bands.
func HandleModemHardwareBands(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	targetSlot := 0
	slotStr := r.URL.Query().Get("slot")
	if slotStr == "" {
		slotStr = r.URL.Query().Get("sim_slot")
	}
	if s, err := strconv.Atoi(slotStr); err == nil && (s == 0 || s == 1) {
		targetSlot = s
	}

	bands, err := modem.GetHardwareBandsInfo(targetSlot)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(bands)
}

// HandleModemCellScan scans and lists all detected serving, CA secondary, and neighbor towers.
func HandleModemCellScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	targetSlot := 0
	slotStr := r.URL.Query().Get("slot")
	if slotStr == "" {
		slotStr = r.URL.Query().Get("sim_slot")
	}
	if s, err := strconv.Atoi(slotStr); err == nil && (s == 0 || s == 1) {
		targetSlot = s
	} else {
		// Use currently active mobile data slot by default
		simSt := modem.GetSimStatus()
		targetSlot = simSt.ActiveSlot
	}

	res, err := modem.ScanNearbyCells(targetSlot)
	if err != nil {
		logger.Get().Warnf("Modem", "Cell scan failed on slot %d: %v", targetSlot, err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success":       false,
			"error_message": err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(res)
}

