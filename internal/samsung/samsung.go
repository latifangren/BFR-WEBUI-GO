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
)

// DeviceStatus holds Samsung-specific hardware and software environment info.
type DeviceStatus struct {
	IsSamsung     bool   `json:"is_samsung"`
	Manufacturer  string `json:"manufacturer"`
	Model         string `json:"model"`
	Device        string `json:"device"`
	OneUIVersion  string `json:"oneui_version"`
	DexExists     bool   `json:"dex_exists"`
	CliExists     bool   `json:"cli_exists"`
	CliPath       string `json:"cli_path"`
	SupportedSlot int    `json:"supported_slots"`
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

// jsonResponse matches the JSON output emitted by `band_manager <slot> json`.
type jsonResponse struct {
	Mode        string `json:"mode"`
	ActiveBands []int  `json:"active_bands"`
	Status      string `json:"status"`
	Message     string `json:"message"`
}

// getSystemProp reads an Android system property using `getprop`.
func getSystemProp(key string) string {
	out, err := config.ExecSuTimeout(2*time.Second, "getprop "+key)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// findBandManagerCLI searches for the band_manager shell wrapper or executable.
func findBandManagerCLI() (string, bool) {
	candidates := []string{
		filepath.Join(config.ModuleDir, "bin", "band_manager"),
		"/data/adb/modules/bfr_webui_go/bin/band_manager",
		"/data/adb/modules/qmanager-go/bin/band_manager",
		"./bin/band_manager",
		"bin/band_manager",
		"/system/bin/band_manager",
	}

	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, true
		}
	}
	return candidates[0], false
}

// dexFileExists checks if the band_manager.dex payload is present on disk.
func dexFileExists() bool {
	candidates := []string{
		filepath.Join(config.ModuleDir, "bin", "band_manager.dex"),
		"/data/adb/modules/bfr_webui_go/bin/band_manager.dex",
		"/data/local/tmp/radiotmp/band_manager.dex",
		"/data/local/tmp/band_manager.dex",
		"/data/adb/modules/qmanager-go/bin/band_manager.dex",
		"./bin/band_manager.dex",
		"bin/band_manager.dex",
	}

	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return true
		}
	}
	return false
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

	cliPath, cliExists := findBandManagerCLI()
	dexExists := dexFileExists()

	return &DeviceStatus{
		IsSamsung:     isSamsung,
		Manufacturer:  mfg,
		Model:         model,
		Device:        device,
		OneUIVersion:  oneui,
		DexExists:     dexExists,
		CliExists:     cliExists,
		CliPath:       cliPath,
		SupportedSlot: 2,
	}
}

// parseBandList parses the output of `band_manager <slot> list`.
// Output pattern: `ID: %3d | %-22s | Active: %s`
var listLineRegex = regexp.MustCompile(`ID:\s*(\d+)\s*\|\s*([^|]+)\|\s*Active:\s*(true|false)`)

func parseBandList(output string) []BandItem {
	var items []BandItem
	lines := strings.Split(output, "\n")
	for _, l := range lines {
		matches := listLineRegex.FindStringSubmatch(l)
		if len(matches) == 4 {
			id, _ := strconv.Atoi(strings.TrimSpace(matches[1]))
			name := strings.TrimSpace(matches[2])
			active := strings.EqualFold(strings.TrimSpace(matches[3]), "true")
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

	cli, exists := findBandManagerCLI()
	if !exists {
		// Even if binary file stat fails on PC/host, construct command with fallback
		cli = filepath.Join(config.ModuleDir, "bin", "band_manager")
	}

	status := &BandStatus{
		Slot:        slot,
		Mode:        "AUTOMATIC",
		ActiveBands: []int{},
		Bands:       []BandItem{},
	}

	// 1. Query JSON mode & active bands
	jsonCmd := fmt.Sprintf("%q %d json", cli, slot)
	outJson, err := config.ExecSuTimeout(8*time.Second, jsonCmd)
	if err == nil && len(outJson) > 0 {
		var parsed jsonResponse
		trimmed := strings.TrimSpace(string(outJson))
		// Locate opening brace '{' if any shell warnings preceded it
		if idx := strings.Index(trimmed, "{"); idx != -1 {
			trimmed = trimmed[idx:]
		}
		if jsonErr := json.Unmarshal([]byte(trimmed), &parsed); jsonErr == nil {
			if parsed.Mode != "" {
				status.Mode = parsed.Mode
			}
			if len(parsed.ActiveBands) > 0 {
				status.ActiveBands = parsed.ActiveBands
			}
		}
	}

	// 2. Query available band list
	listCmd := fmt.Sprintf("%q %d list", cli, slot)
	outList, err := config.ExecSuTimeout(8*time.Second, listCmd)
	if err != nil {
		status.Error = fmt.Sprintf("Failed to query band list: %v", err)
		logger.Get().Errorf("Samsung", "Failed to query bands for slot %d: %v", slot, err)
		return status, err
	}

	status.Bands = parseBandList(string(outList))

	// Sync active status from active_bands list if mode is LOCKED
	if status.Mode == "LOCKED" && len(status.ActiveBands) > 0 {
		activeMap := make(map[int]bool)
		for _, b := range status.ActiveBands {
			activeMap[b] = true
		}
		for i := range status.Bands {
			if activeMap[status.Bands[i].ID] {
				status.Bands[i].Active = true
			}
		}
	}

	return status, nil
}

// LockBands locks the selected SIM slot to the given band IDs.
func LockBands(slot int, bands []int) error {
	if slot < 0 || slot > 1 {
		return fmt.Errorf("invalid slot %d, must be 0 or 1", slot)
	}
	if len(bands) == 0 {
		return fmt.Errorf("no bands selected for locking")
	}
	if len(bands) > 64 {
		return fmt.Errorf("too many bands specified (max 64)")
	}

	cli, _ := findBandManagerCLI()

	bandStrList := make([]string, len(bands))
	for i, b := range bands {
		if b <= 0 || b > 1000 {
			return fmt.Errorf("invalid band ID %d", b)
		}
		bandStrList[i] = strconv.Itoa(b)
	}
	bandArg := strings.Join(bandStrList, ",")

	cmd := fmt.Sprintf("%q %d lock %s", cli, slot, bandArg)
	logger.Get().Infof("Samsung", "Executing band lock on slot %d: %s", slot, bandArg)

	out, err := config.ExecSuTimeout(10*time.Second, cmd)
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
	cli, _ := findBandManagerCLI()
	cmd := fmt.Sprintf("%q %d auto", cli, slot)
	logger.Get().Infof("Samsung", "Resetting band selection to auto on slot %d", slot)

	out, err := config.ExecSuTimeout(10*time.Second, cmd)
	if err != nil {
		return fmt.Errorf("reset band auto failed: %v (output: %s)", err, strings.TrimSpace(string(out)))
	}

	return nil
}
