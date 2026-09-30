package samsung

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
	"bfr-webui-go/internal/modem"
)

// DeviceStatus holds Samsung-specific hardware and software environment info.
type DeviceStatus struct {
	IsSamsung      bool   `json:"is_samsung"`
	Manufacturer   string `json:"manufacturer"`
	Model          string `json:"model"`
	Device         string `json:"device"`
	OneUIVersion   string `json:"oneui_version"`
	DexExists      bool   `json:"dex_exists"`
	CliExists      bool   `json:"cli_exists"`
	CliPath        string `json:"cli_path"`
	SupportedSlot  int    `json:"supported_slots"`
	ActiveDataSlot int    `json:"active_data_slot"`
}

// BandItem represents an LTE/NR carrier frequency band supported by the modem.
type BandItem struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

// BandStatus represents the current band locking state and active bands for a SIM slot.
type BandStatus struct {
	Slot        int        `json:"slot"`
	Mode        string     `json:"mode"` // "AUTOMATIC" or "LOCKED"
	ActiveBands []int      `json:"active_bands"`
	Bands       []BandItem `json:"bands"`
	Error       string     `json:"error,omitempty"`
}

// jsonResponse matches the JSON output emitted by `BandManager <slot> status` and lock/auto.
type jsonResponse struct {
	Slot        int        `json:"slot"`
	Mode        string     `json:"mode"`
	ActiveBands []BandItem `json:"active_bands"`
	Status      string     `json:"status"`
	Message     string     `json:"message"`
}

// Type aliases to modem universal structures
type SimSwitchResult = modem.SimSwitchResult
type ThermalSensors = modem.ThermalSensors
type ThermalThrottlers = modem.ThermalThrottlers
type CriticalThresholds = modem.CriticalThresholds
type Thermal5GStatus = modem.Thermal5GStatus

// getSystemProp reads an Android system property using `getprop`.
func getSystemProp(key string) string {
	out, err := config.ExecSuTimeout(2*time.Second, "getprop "+key)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// findBandManagerDex locates the compiled band_manager.dex payload.
func findBandManagerDex() string {
	candidates := []string{
		filepath.Join(config.ModuleDir, "bin", "band_manager.dex"),
		"/data/adb/modules/bfr_webui_go/bin/band_manager.dex",
		"./bin/band_manager.dex",
		"bin/band_manager.dex",
	}

	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	return candidates[0]
}

// dexFileExists checks if the band_manager.dex payload is present on disk.
func dexFileExists() bool {
	p := findBandManagerDex()
	if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
		return true
	}
	return false
}

// execBandManager executes BandManager.dex directly via app_process as UID 1001 (radio).
func execBandManager(cmd string, slot int, extraArgs ...string) ([]byte, error) {
	tmpDir := "/data/local/tmp/radiotmp"
	targetDex := filepath.Join(tmpDir, "band_manager.dex")
	srcDex := findBandManagerDex()

	// 1. Prepare directory with radio UID ownership
	setupCmd := fmt.Sprintf("mkdir -p %q/dalvik-cache/arm64 && chown -R 1001:1001 %q && chmod -R 777 %q", tmpDir, tmpDir, tmpDir)
	_, _ = config.ExecSuTimeout(2*time.Second, setupCmd)

	if srcDex != "" {
		copyCmd := fmt.Sprintf("cp -f %q %q && chown 1001:1001 %q && chmod 444 %q", srcDex, targetDex, targetDex, targetDex)
		_, _ = config.ExecSuTimeout(2*time.Second, copyCmd)
	}

	// 2. Execute via app_process as radio user (1001)
	argStr := ""
	if len(extraArgs) > 0 {
		argStr = " " + strings.Join(extraArgs, " ")
	}

	runCmd := fmt.Sprintf(
		"su 1001 -c 'ANDROID_DATA=%s CLASSPATH=%s app_process / com.qmanager.tools.BandManager %s %d%s'",
		tmpDir, targetDex, cmd, slot, argStr,
	)
	return config.ExecSuTimeout(10*time.Second, runCmd)
}

// GetActiveDataSlot returns the SIM slot currently used for mobile data (0 or 1).
func GetActiveDataSlot() int {
	return modem.GetActiveDataSlot()
}

// GetDeviceStatus checks whether the device is running Samsung firmware and verifies tools.
func GetDeviceStatus() *DeviceStatus {
	mfg := getSystemProp("ro.product.manufacturer")
	model := getSystemProp("ro.product.model")
	device := getSystemProp("ro.product.device")
	oneui := getSystemProp("ro.build.version.oneui")
	if oneui == "" {
		oneui = getSystemProp("ro.build.version.sep")
	}

	isSamsung := strings.EqualFold(mfg, "samsung") || strings.Contains(strings.ToLower(mfg), "samsung")
	dexExists := dexFileExists()
	activeDataSlot := GetActiveDataSlot()

	return &DeviceStatus{
		IsSamsung:      isSamsung,
		Manufacturer:   mfg,
		Model:          model,
		Device:         device,
		OneUIVersion:   oneui,
		DexExists:      dexExists,
		CliExists:      dexExists,
		CliPath:        "band_manager.dex (native runner)",
		SupportedSlot:  2,
		ActiveDataSlot: activeDataSlot,
	}
}

// parseBandList parses the output of `BandManager <slot> list`.
var listLineRegex = regexp.MustCompile(`(?i)ID:\s*(\d+)\s*\|\s*([^|]+)\|\s*Active:\s*(YES|NO|true|false)`)

func parseBandList(output string) []BandItem {
	var items []BandItem
	lines := strings.Split(output, "\n")
	for _, l := range lines {
		matches := listLineRegex.FindStringSubmatch(l)
		if len(matches) == 4 {
			id, _ := strconv.Atoi(strings.TrimSpace(matches[1]))
			name := strings.TrimSpace(matches[2])
			val := strings.ToUpper(strings.TrimSpace(matches[3]))
			active := val == "YES" || val == "TRUE"
			items = append(items, BandItem{
				ID:     id,
				Name:   name,
				Active: active,
			})
		}
	}
	return items
}

// GetBands fetches current band configuration and supported bands for the given SIM slot.
func GetBands(slot int) (*BandStatus, error) {
	if slot < 0 || slot > 1 {
		return nil, fmt.Errorf("invalid slot %d, must be 0 or 1", slot)
	}

	status := &BandStatus{
		Slot:        slot,
		Mode:        "AUTOMATIC",
		ActiveBands: []int{},
		Bands:       []BandItem{},
	}

	// 1. Query status: provides mode (AUTOMATIC / LOCKED) and active_bands
	outStatus, err := execBandManager("status", slot)
	if err == nil && len(outStatus) > 0 {
		var parsed jsonResponse
		trimmed := strings.TrimSpace(string(outStatus))
		if idx := strings.Index(trimmed, "{"); idx != -1 {
			trimmed = trimmed[idx:]
		}
		if jsonErr := json.Unmarshal([]byte(trimmed), &parsed); jsonErr == nil {
			if parsed.Mode != "" {
				status.Mode = parsed.Mode
			}
			for _, b := range parsed.ActiveBands {
				status.ActiveBands = append(status.ActiveBands, b.ID)
			}
		}
	}

	// 2. Query full bands list via json (returns JSON array of all BandItems)
	outJSON, err := execBandManager("json", slot)
	if err == nil && len(outJSON) > 0 {
		var jsonBands []BandItem
		trimmed := strings.TrimSpace(string(outJSON))
		if idx := strings.Index(trimmed, "["); idx != -1 {
			trimmed = trimmed[idx:]
		}
		if jsonErr := json.Unmarshal([]byte(trimmed), &jsonBands); jsonErr == nil && len(jsonBands) > 0 {
			status.Bands = jsonBands
		}
	}

	// 3. Fallback to `list` if json didn't return any bands
	if len(status.Bands) == 0 {
		outList, err := execBandManager("list", slot)
		if err == nil && len(outList) > 0 {
			status.Bands = parseBandList(string(outList))
		} else if err != nil && len(status.ActiveBands) == 0 {
			status.Error = fmt.Sprintf("Failed to query band list: %v", err)
			logger.Get().Errorf("Samsung", "Failed to query bands for slot %d: %v", slot, err)
			return status, err
		}
	}

	// Sync active status from active_bands list if mode is LOCKED
	if status.Mode == "LOCKED" && len(status.ActiveBands) > 0 {
		activeMap := make(map[int]bool)
		for _, b := range status.ActiveBands {
			activeMap[b] = true
		}
		for i := range status.Bands {
			if activeMap[status.Bands[i].ID] {
				status.Bands[i].Active = true
			} else {
				status.Bands[i].Active = false
			}
		}
	}

	return status, nil
}

// LockBands locks the modem to the specified frequency bands for the given SIM slot.
func LockBands(slot int, bands []int) error {
	if slot < 0 || slot > 1 {
		return fmt.Errorf("invalid slot %d, must be 0 or 1", slot)
	}
	if len(bands) == 0 {
		return fmt.Errorf("at least one band must be specified")
	}

	// Query supported bands to evaluate NR NSA anchoring
	currentStatus, _ := GetBands(slot)
	if currentStatus != nil && len(currentStatus.Bands) > 0 {
		hasLTE := false
		hasNR := false
		bandMap := make(map[int]bool)
		idToName := make(map[int]string)
		for _, b := range currentStatus.Bands {
			idToName[b.ID] = b.Name
		}
		for _, b := range bands {
			bandMap[b] = true
			name := idToName[b]
			if strings.HasPrefix(name, "LTE") || strings.HasPrefix(name, "B") {
				hasLTE = true
			} else if strings.HasPrefix(name, "NR") || strings.HasPrefix(name, "N") {
				hasNR = true
			}
		}

		if hasNR && !hasLTE {
			addedAnchors := false
			for _, b := range currentStatus.Bands {
				if (strings.HasPrefix(b.Name, "LTE") || strings.HasPrefix(b.Name, "B")) && b.Active {
					if !bandMap[b.ID] {
						bandMap[b.ID] = true
						bands = append(bands, b.ID)
						addedAnchors = true
					}
				}
			}
			if !addedAnchors {
				defaultAnchors := map[string]bool{
					"LTE B1": true, "LTE B3": true, "LTE B5": true, "LTE B8": true, "LTE B40": true,
				}
				for _, b := range currentStatus.Bands {
					if defaultAnchors[b.Name] && !bandMap[b.ID] {
						bandMap[b.ID] = true
						bands = append(bands, b.ID)
					}
				}
			}
		}
	}

	bandStrList := make([]string, len(bands))
	for i, b := range bands {
		if b <= 0 || b > 1000 {
			return fmt.Errorf("invalid band ID %d", b)
		}
		bandStrList[i] = strconv.Itoa(b)
	}
	bandArg := strings.Join(bandStrList, ",")

	logger.Get().Infof("Samsung", "Executing native band lock on slot %d: %s", slot, bandArg)
	out, err := execBandManager("lock", slot, bandArg)
	if err != nil {
		return fmt.Errorf("band lock failed: %v (output: %s)", err, strings.TrimSpace(string(out)))
	}

	trimmed := strings.TrimSpace(string(out))
	if idx := strings.Index(trimmed, "{"); idx != -1 {
		var res jsonResponse
		if jsonErr := json.Unmarshal([]byte(trimmed[idx:]), &res); jsonErr == nil {
			if res.Status == "error" {
				return fmt.Errorf("%s", res.Message)
			}
		}
	}

	return nil
}

// ResetAuto restores automatic band selection on the given SIM slot.
func ResetAuto(slot int) error {
	if slot < 0 || slot > 1 {
		return fmt.Errorf("invalid slot %d, must be 0 or 1", slot)
	}
	logger.Get().Infof("Samsung", "Resetting band selection to auto on slot %d", slot)

	out, err := execBandManager("auto", slot)
	if err != nil {
		return fmt.Errorf("reset band auto failed: %v (output: %s)", err, strings.TrimSpace(string(out)))
	}

	return nil
}

// SwitchDataSim delegates to the universal native Go SIM switcher.
func SwitchDataSim(slot int) (*SimSwitchResult, error) {
	return modem.SwitchDataSim(slot)
}

// Get5GThermalStatus delegates to the universal native Go thermal reader.
func Get5GThermalStatus() (*Thermal5GStatus, error) {
	return modem.Get5GThermalStatus()
}
