package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"bfr-webui-go/internal/logger"
	"bfr-webui-go/internal/power"
)

type powerRequest struct {
	Action string `json:"action"`
}

func HandlePower(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req powerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request payload"})
		return
	}

	actionStr := strings.ToLower(req.Action)

	// Validate action against whitelist before spawning execution goroutine
	validActions := map[string]bool{
		"reboot":      true,
		"shutdown":    true,
		"poweroff":    true,
		"soft_reboot": true,
		"recovery":    true,
		"bootloader":  true,
	}
	if !validActions[actionStr] {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "invalid action; must be one of: reboot, shutdown, soft_reboot, recovery, bootloader",
		})
		return
	}

	// Normalize shutdown/poweroff to ActionPoweroff
	var act power.Action
	switch actionStr {
	case "shutdown", "poweroff":
		act = power.ActionPoweroff
	case "reboot":
		act = power.ActionReboot
	case "soft_reboot":
		act = power.ActionSoftReboot
	case "recovery":
		act = power.ActionRebootRecovery
	case "bootloader":
		act = power.ActionRebootBootloader
	}

	logger.Get().Infof("Power", "Executing power action: %s", act)

	go func() {
		time.Sleep(1 * time.Second)
		if err := power.Execute(act); err != nil {
			logger.Get().Errorf("Power", "Failed to execute %s: %v", act, err)
			log.Printf("Failed to execute %s: %v", act, err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"action":  actionStr,
		"message": fmt.Sprintf("Power action '%s' initiated successfully", actionStr),
	})
}

func HandlePowerSchedule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	scheduler := power.GetScheduler()

	switch r.Method {
	case http.MethodGet:
		status := scheduler.GetStatus()
		_ = json.NewEncoder(w).Encode(status)

	case http.MethodPost:
		var cfg power.ScheduleConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request payload"})
			return
		}

		if err := scheduler.UpdateConfig(cfg); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}

		status := scheduler.GetStatus()
		_ = json.NewEncoder(w).Encode(status)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": "Method not allowed"})
	}
}
