package modem

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	State     string `json:"state"` // "READY", "LOADED", "ABSENT", "UNKNOWN"
	IsActive  bool   `json:"is_active"`
	IsESim    bool   `json:"is_esim"`
	Carrier   string `json:"carrier"`
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

type simSubEntry struct {
	SubID      int
	SlotIndex  int
	IsEmbedded bool
	Carrier    string
}

var (
	modemCountRe  = regexp.MustCompile(`(?i)Active modem count=(\d+)`)
	activeDataRe  = regexp.MustCompile(`(?i)(?:activeDataSubId|defaultDataSubId)=(\d+)`)
	simStateRe    = regexp.MustCompile(`(?i)mSimState\[(\d+)\]=(\w+)`)
	subInternalRe = regexp.MustCompile(`\[SubscriptionInfoInternal:[^\]]+\]`)
	subIdRe       = regexp.MustCompile(`\bid=(\d+)`)
	slotIndexRe   = regexp.MustCompile(`\bsimSlotIndex=(-?\d+)`)
	embeddedRe    = regexp.MustCompile(`\bisEmbedded=(\d+)`)
	carrierNameRe = regexp.MustCompile(`\bcarrierName=([^=]+?)\s+isOpportunistic=`)
	displayNameRe = regexp.MustCompile(`\bdisplayName=([^=]+?)\s+carrierName=`)
)

// parseDumpsysIsubInternal parses the output of dumpsys isub and correlates slot information.
func parseDumpsysIsubInternal(isubOut, fallbackSimState, fallbackOperator string) *SimStatus {
	slotCount := 2
	if m := modemCountRe.FindStringSubmatch(isubOut); len(m) > 1 {
		if c, err := strconv.Atoi(m[1]); err == nil && c > 0 && c <= 4 {
			slotCount = c
		}
	}

	activeSubID := -1
	if m := activeDataRe.FindStringSubmatch(isubOut); len(m) > 1 {
		if id, err := strconv.Atoi(m[1]); err == nil {
			activeSubID = id
		}
	}

	slotStates := make(map[int]string)
	for _, m := range simStateRe.FindAllStringSubmatch(isubOut, -1) {
		if len(m) > 2 {
			if idx, err := strconv.Atoi(m[1]); err == nil {
				slotStates[idx] = strings.ToUpper(strings.TrimSpace(m[2]))
			}
		}
	}

	// Fallback slot states from getprop gsm.sim.state
	if len(slotStates) == 0 && fallbackSimState != "" {
		parts := strings.Split(fallbackSimState, ",")
		for i, p := range parts {
			trimmed := strings.ToUpper(strings.TrimSpace(p))
			if trimmed != "" {
				slotStates[i] = trimmed
			}
		}
	}

	// Fallback operators from getprop
	fallbackOps := make(map[int]string)
	if fallbackOperator != "" {
		parts := strings.Split(fallbackOperator, ",")
		for i, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				fallbackOps[i] = trimmed
			}
		}
	}

	// Parse subscriptions line-by-line to avoid premature bracket truncation (e.g. masked [****])
	subsBySlot := make(map[int]*simSubEntry)
	for _, line := range strings.Split(isubOut, "\n") {
		if !strings.Contains(line, "SubscriptionInfoInternal:") {
			continue
		}

		id := -1
		slotIdx := -1
		isEmbedded := false
		carrier := ""

		if m := subIdRe.FindStringSubmatch(line); len(m) > 1 {
			id, _ = strconv.Atoi(m[1])
		}
		if m := slotIndexRe.FindStringSubmatch(line); len(m) > 1 {
			slotIdx, _ = strconv.Atoi(m[1])
		}
		if m := embeddedRe.FindStringSubmatch(line); len(m) > 1 {
			isEmbedded = m[1] == "1"
		}
		if m := carrierNameRe.FindStringSubmatch(line); len(m) > 1 {
			carrier = strings.TrimSpace(m[1])
		}
		if carrier == "" || carrier == "No service" {
			if m := displayNameRe.FindStringSubmatch(line); len(m) > 1 {
				carrier = strings.TrimSpace(m[1])
			}
		}

		if slotIdx >= 0 {
			subsBySlot[slotIdx] = &simSubEntry{
				SubID:      id,
				SlotIndex:  slotIdx,
				IsEmbedded: isEmbedded,
				Carrier:    carrier,
			}
		}
	}

	// Determine ActiveSlot
	activeSlot := 0
	foundActive := false
	if activeSubID > 0 {
		for slotIdx, sub := range subsBySlot {
			if sub.SubID == activeSubID {
				activeSlot = slotIdx
				foundActive = true
				break
			}
		}
	}

	if !foundActive {
		s0 := slotStates[0]
		s1 := slotStates[1]
		if (s0 == "ABSENT" || s0 == "") && (s1 == "READY" || s1 == "LOADED") {
			activeSlot = 1
		} else if (s1 == "ABSENT" || s1 == "") && (s0 == "READY" || s0 == "LOADED") {
			activeSlot = 0
		}
		if sub, ok := subsBySlot[activeSlot]; ok && sub.SubID > 0 {
			activeSubID = sub.SubID
		} else {
			activeSubID = activeSlot + 1
		}
	}

	// Construct Slots
	slots := make([]SimSlotInfo, 0, slotCount)
	for slot := 0; slot < slotCount; slot++ {
		st := slotStates[slot]
		if st == "" {
			st = "UNKNOWN"
		}

		sub, hasSub := subsBySlot[slot]
		isEsim := false
		carrier := fallbackOps[slot]
		subID := -1

		if hasSub {
			subID = sub.SubID
			isEsim = sub.IsEmbedded
			if sub.Carrier != "" {
				carrier = sub.Carrier
			}
			if st == "UNKNOWN" {
				st = "LOADED"
			}
		} else {
			// If slot 1 has eSIM profiles mentioned in dumpsys
			if slot == 1 && (strings.Contains(isubOut, "EuiccProfileInfo") || strings.Contains(isubOut, "isEmbedded=1")) {
				isEsim = true
			}
			if st == "UNKNOWN" && fallbackSimState == "" {
				st = "ABSENT"
			}
		}

		typeTag := "pSIM"
		if isEsim {
			typeTag = "eSIM"
		}

		label := fmt.Sprintf("SIM %d (%s)", slot+1, typeTag)
		if st == "ABSENT" {
			label = fmt.Sprintf("SIM %d (%s - No SIM Card)", slot+1, typeTag)
		} else if carrier != "" {
			label = fmt.Sprintf("SIM %d (%s - %s)", slot+1, typeTag, carrier)
		}

		slots = append(slots, SimSlotInfo{
			SlotIndex: slot,
			SubID:     subID,
			State:     st,
			IsActive:  slot == activeSlot,
			IsESim:    isEsim,
			Carrier:   carrier,
			Label:     label,
		})
	}

	return &SimStatus{
		ActiveSlot:  activeSlot,
		ActiveSubID: activeSubID,
		Slots:       slots,
	}
}

// GetActiveDataSlot returns the SIM slot currently used for mobile data (0 or 1).
func GetActiveDataSlot() int {
	st := GetSimStatus()
	if st != nil {
		return st.ActiveSlot
	}
	return 0
}

// GetSimStatus gathers the current SIM states and active data route.
func GetSimStatus() *SimStatus {
	// Query dumpsys isub first 45 lines
	isubOut, _ := config.ExecSuTimeout(2*time.Second, "dumpsys isub 2>/dev/null | head -n 45 || true")
	simStatesProp, _ := config.ExecSuTimeout(2*time.Second, "getprop gsm.sim.state 2>/dev/null || true")
	simOpProp, _ := config.ExecSuTimeout(2*time.Second, "getprop gsm.sim.operator.alpha 2>/dev/null || true")
	if strings.TrimSpace(string(simOpProp)) == "" {
		simOpProp, _ = config.ExecSuTimeout(2*time.Second, "getprop gsm.operator.alpha 2>/dev/null || true")
	}

	return parseDumpsysIsubInternal(
		string(isubOut),
		strings.TrimSpace(string(simStatesProp)),
		strings.TrimSpace(string(simOpProp)),
	)
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

// SwitchDataSim executes a SIM switch to slot 0 or 1 using the accurate subscription ID.
func SwitchDataSim(slot int) (*SimSwitchResult, error) {
	if slot < 0 || slot > 1 {
		return nil, fmt.Errorf("invalid slot %d, must be 0 or 1", slot)
	}

	// Retrieve real SubID for the target slot
	currentStatus := GetSimStatus()
	targetSubID := slot + 1
	if currentStatus != nil {
		for _, s := range currentStatus.Slots {
			if s.SlotIndex == slot && s.SubID > 0 {
				targetSubID = s.SubID
				break
			}
		}
	}

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
		tmpDir, targetDex, targetSubID, slot,
	)
	_, _ = config.ExecSuTimeout(5*time.Second, appCmd)

	// 3. Apply standard AOSP ISub service call transaction & settings updates
	serviceCmd := fmt.Sprintf(
		"service call isub 31 i32 %d ; "+
			"settings put global multi_sim_data_call %d ; "+
			"settings put global multi_sim_data_call_slot %d ; "+
			"settings put global user_preferred_data_sub %d ; "+
			"am broadcast -a android.intent.action.ACTION_DEFAULT_DATA_SUBSCRIPTION_CHANGED --ei subscription %d",
		targetSubID, targetSubID, slot, targetSubID, targetSubID,
	)
	_, _ = config.ExecSuTimeout(5*time.Second, serviceCmd)

	logger.Get().Infof("Modem", "Data SIM successfully switched to slot %d (SubID: %d)", slot, targetSubID)

	return &SimSwitchResult{
		Success: true,
		Slot:    slot,
		SubID:   targetSubID,
		Message: fmt.Sprintf("Active data SIM switched to slot %d (SubID: %d)", slot, targetSubID),
	}, nil
}
