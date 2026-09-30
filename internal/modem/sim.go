package modem

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

// SimSlotInfo contains information about a physical or eSIM slot.
type SimSlotInfo struct {
	SlotIndex int    `json:"slot_index"`
	SubID     int    `json:"sub_id"`
	State     string `json:"state"` // "READY", "ABSENT", "LOADED", "UNKNOWN"
	IsActive  bool   `json:"is_active"`
	Label     string `json:"label"`
}

// SimStatus contains the multi-SIM data state across slots.
type SimStatus struct {
	ActiveSlot  int           `json:"active_slot"`
	ActiveSubID int           `json:"active_sub_id"`
	Slots       []SimSlotInfo `json:"slots"`
}

// SimSwitchResult contains the outcome of switching the active data SIM slot.
type SimSwitchResult struct {
	Success bool   `json:"success"`
	Slot    int    `json:"slot"`
	SubID   int    `json:"sub_id"`
	Message string `json:"message"`
}

// GetActiveDataSlot returns the SIM slot currently used for mobile data (0 or 1).
func GetActiveDataSlot() int {
	out, err := config.ExecSuTimeout(2*time.Second, "settings get global multi_sim_data_call_slot")
	if err == nil {
		str := strings.TrimSpace(string(out))
		if slot, err := strconv.Atoi(str); err == nil && (slot == 0 || slot == 1) {
			return slot
		}
	}

	// Fallback to multi_sim_data_call
	out, err = config.ExecSuTimeout(2*time.Second, "settings get global multi_sim_data_call")
	if err == nil {
		str := strings.TrimSpace(string(out))
		if subID, err := strconv.Atoi(str); err == nil {
			if subID >= 2 {
				return 1
			}
			return 0
		}
	}

	return 0
}

// GetSimStatus gathers the current SIM states and active data route.
func GetSimStatus() *SimStatus {
	activeSlot := GetActiveDataSlot()
	subIDStr, _ := config.ExecSuTimeout(2*time.Second, "settings get global multi_sim_data_call")
	activeSubID, _ := strconv.Atoi(strings.TrimSpace(string(subIDStr)))
	if activeSubID <= 0 {
		activeSubID = activeSlot + 1
	}

	// Read gsm.sim.state prop (e.g. "READY,READY" or "ABSENT,READY")
	simStatesProp, _ := config.ExecSuTimeout(2*time.Second, "getprop gsm.sim.state")
	states := strings.Split(strings.TrimSpace(string(simStatesProp)), ",")

	slot0State := "UNKNOWN"
	slot1State := "UNKNOWN"
	if len(states) > 0 && strings.TrimSpace(states[0]) != "" {
		slot0State = strings.TrimSpace(states[0])
	}
	if len(states) > 1 && strings.TrimSpace(states[1]) != "" {
		slot1State = strings.TrimSpace(states[1])
	}

	return &SimStatus{
		ActiveSlot:  activeSlot,
		ActiveSubID: activeSubID,
		Slots: []SimSlotInfo{
			{
				SlotIndex: 0,
				SubID:     1,
				State:     slot0State,
				IsActive:  activeSlot == 0,
				Label:     "SIM 1 (Primary / pSIM)",
			},
			{
				SlotIndex: 1,
				SubID:     2,
				State:     slot1State,
				IsActive:  activeSlot == 1,
				Label:     "SIM 2 (Secondary / eSIM)",
			},
		},
	}
}

func findSimSwitcherDex() string {
	candidates := []string{
		filepath.Join(config.ModuleDir, "bin", "sim_switcher.dex"),
		"/data/adb/modules/bfr_webui_go/bin/sim_switcher.dex",
		"./bin/sim_switcher.dex",
		"bin/sim_switcher.dex",
	}
	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	return candidates[0]
}

// SwitchDataSim executes a SIM switch to slot 0 or 1 without needing a bash wrapper.
func SwitchDataSim(slot int) (*SimSwitchResult, error) {
	if slot < 0 || slot > 1 {
		return nil, fmt.Errorf("invalid slot %d, must be 0 or 1", slot)
	}

	subID := slot + 1
	tmpDir := "/data/local/tmp/radiotmp"
	targetDex := filepath.Join(tmpDir, "sim_switcher.dex")
	srcDex := findSimSwitcherDex()

	// 1. Prepare tmp directory and copy DEX if needed
	setupCmd := fmt.Sprintf(
		"mkdir -p %q && chown -R 1001:1001 %q && chmod 770 %q",
		tmpDir, tmpDir, tmpDir,
	)
	_, _ = config.ExecSuTimeout(2*time.Second, setupCmd)

	if srcDex != "" {
		copyCmd := fmt.Sprintf("cp -f %q %q && chown 1001:1001 %q && chmod 444 %q", srcDex, targetDex, targetDex, targetDex)
		_, _ = config.ExecSuTimeout(2*time.Second, copyCmd)
	}

	// 2. Execute Java SimSwitcher via app_process (AOSP ISub reflection)
	appCmd := fmt.Sprintf(
		"ANDROID_DATA=%q CLASSPATH=%q app_process / com.qmanager.tools.SimSwitcher %d %d",
		tmpDir, targetDex, subID, slot,
	)
	_, _ = config.ExecSuTimeout(5*time.Second, appCmd)

	// 3. Apply standard AOSP ISub service call transaction & settings updates
	serviceCmd := fmt.Sprintf(
		"service call isub 31 i32 %d ; "+
			"settings put global multi_sim_data_call %d ; "+
			"settings put global multi_sim_data_call_slot %d ; "+
			"settings put global user_preferred_data_sub %d ; "+
			"am broadcast -a android.intent.action.ACTION_DEFAULT_DATA_SUBSCRIPTION_CHANGED --ei subscription %d",
		subID, subID, slot, subID, subID,
	)
	_, _ = config.ExecSuTimeout(5*time.Second, serviceCmd)

	logger.Get().Infof("Modem", "Data SIM successfully switched to slot %d (SubID: %d)", slot, subID)

	return &SimSwitchResult{
		Success: true,
		Slot:    slot,
		SubID:   subID,
		Message: fmt.Sprintf("Active data SIM switched to slot %d (SubID: %d)", slot, subID),
	}, nil
}
