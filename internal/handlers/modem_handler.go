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
