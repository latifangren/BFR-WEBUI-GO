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
		if subId, err := strconv.Atoi(str); err == nil {
			if subId >= 2 {
				return 1
			}
			return 0
		}
	}

	return 0
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
	activeDataSlot := GetActiveDataSlot()

	return &DeviceStatus{
		IsSamsung:      isSamsung,
		Manufacturer:   mfg,
		Model:          model,
		Device:         device,
		OneUIVersion:   oneui,
		DexExists:      dexExists,
		CliExists:      cliExists,
		CliPath:        cliPath,
		SupportedSlot:  2,
		ActiveDataSlot: activeDataSlot,
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

// findSimSwitcherCLI searches for the sim_switcher shell script or executable.
func findSimSwitcherCLI() (string, bool) {
	candidates := []string{
		filepath.Join(config.ModuleDir, "bin", "sim_switcher"),
		"/data/adb/modules/bfr_webui_go/bin/sim_switcher",
		"./bin/sim_switcher",
		"bin/sim_switcher",
		"/system/bin/sim_switcher",
	}

	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, true
		}
	}
	return candidates[0], false
}

// SimSwitchResult contains the outcome of switching the active data SIM slot.
type SimSwitchResult struct {
	Success bool   `json:"success"`
	Slot    int    `json:"slot"`
	SubID   int    `json:"sub_id"`
	Message string `json:"message"`
}

// SwitchDataSim switches mobile data to the specified SIM slot (0 or 1).
func SwitchDataSim(slot int) (*SimSwitchResult, error) {
	if slot < 0 || slot > 1 {
		return nil, fmt.Errorf("invalid slot %d, must be 0 or 1", slot)
	}

	cli, _ := findSimSwitcherCLI()
	cmd := fmt.Sprintf("%q %d", cli, slot)
	logger.Get().Infof("Samsung", "Switching data SIM to slot %d using %s", slot, cli)

	out, err := config.ExecSuTimeout(10*time.Second, cmd)
	if err != nil {
		return nil, fmt.Errorf("sim switch failed: %v (output: %s)", err, strings.TrimSpace(string(out)))
	}

	res := &SimSwitchResult{
		Success: true,
		Slot:    slot,
		SubID:   slot + 1,
		Message: fmt.Sprintf("Successfully switched active data SIM to slot %d", slot),
	}

	trimmed := strings.TrimSpace(string(out))
	if idx := strings.Index(trimmed, "{"); idx != -1 {
		var raw struct {
			Status string `json:"status"`
			Slot   int    `json:"slot"`
			SubID  int    `json:"subId"`
		}
		if jsonErr := json.Unmarshal([]byte(trimmed[idx:]), &raw); jsonErr == nil {
			if raw.Status == "ok" {
				res.Success = true
				res.Slot = raw.Slot
				if raw.SubID > 0 {
					res.SubID = raw.SubID
				}
				res.Message = fmt.Sprintf("Active data SIM switched to slot %d (SubID: %d)", res.Slot, res.SubID)
			}
		}
	}

	return res, nil
}

// ThermalSensors represents temperature sensor readings in degrees Celsius.
type ThermalSensors struct {
	PA1Sub6C       float64 `json:"pa1_sub6_c"`
	PA2Sub6C       float64 `json:"pa2_sub6_c"`
	CellFrontCFC   float64 `json:"cell_front_cf_c"`
	ModemSkinC     float64 `json:"modem_skin_c"`
	RfPmicC        float64 `json:"rf_pmic_c"`
	ModemBasebandC float64 `json:"modem_baseband_c"`
	ApSocC         float64 `json:"ap_soc_c"`
	BatteryC       float64 `json:"battery_c"`
}

// ThermalThrottlers represents cooling device throttling state counters.
type ThermalThrottlers struct {
	ModemPaFr1Dsc  int `json:"modem_pa_fr1_dsc"`
	ModemSkinNrDsc int `json:"modem_skin_nr_dsc"`
	ModemPaFr1     int `json:"modem_pa_fr1"`
}

// CriticalThresholds represents hardware safety temperature limits.
type CriticalThresholds struct {
	ModemSkinLimitC float64 `json:"modem_skin_limit_c"`
	PA1CutoffLimitC float64 `json:"pa1_cutoff_limit_c"`
}

// Thermal5GStatus holds the complete thermal diagnostic report and root cause.
type Thermal5GStatus struct {
	NetworkType        string             `json:"network_type"`
	Is5GDisconnect     bool               `json:"is_5g_disconnect"`
	NRBand             string             `json:"nr_band"`
	NRRSRP             int                `json:"nr_rsrp"`
	TrafficMbps        float64            `json:"traffic_mbps"`
	ThermalHALStatus   int                `json:"thermal_hal_status"`
	ThermalHALDesc     string             `json:"thermal_hal_desc"`
	Sensors            ThermalSensors     `json:"sensors"`
	Throttlers         ThermalThrottlers  `json:"throttlers"`
	CriticalThresholds CriticalThresholds `json:"critical_thresholds"`
	IsThrottled        bool               `json:"is_throttled"`
	Culprit            string             `json:"culprit"`
	CulpritDetail      string             `json:"culprit_detail"`
}

// find5GThermalCLI searches for the qm-5g-thermal script or executable.
func find5GThermalCLI() (string, bool) {
	candidates := []string{
		filepath.Join(config.ModuleDir, "bin", "qm-5g-thermal"),
		"/data/adb/modules/bfr_webui_go/bin/qm-5g-thermal",
		"./bin/qm-5g-thermal",
		"bin/qm-5g-thermal",
		"/system/bin/qm-5g-thermal",
	}

	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, true
		}
	}
	return candidates[0], false
}

func getThermalHALDesc(status int) string {
	switch status {
	case 0:
		return "NONE (Normal)"
	case 1:
		return "LIGHT (Hangat Ringan)"
	case 2:
		return "MODERATE (Siaga Panas)"
	case 3:
		return "SEVERE (Throttling 5G Aktif!)"
	case 4:
		return "CRITICAL (Bahaya)"
	default:
		if status > 4 {
			return "EMERGENCY"
		}
		return fmt.Sprintf("Level %d", status)
	}
}

func evaluate5GThermalCulprit(status *Thermal5GStatus) {
	status.Culprit = "TIDAK ADA (SUHU NORMAL)"
	status.CulpritDetail = "Semua sensor dalam batas aman. 5G tidak mengalami pembatasan suhu."
	status.IsThrottled = false

	if status.ThermalHALStatus >= 3 {
		status.IsThrottled = true
		status.Culprit = "Modem Skin Board (SKIN) / sec-cf"
		status.CulpritDetail = fmt.Sprintf("Suhu permukaan board/skin mencapai %.1f°C (limit: 42.0°C). Android Thermal HAL masuk status SEVERE (Level %d). Sistem memerintahkan pelepasan 5G NR Dual Connectivity!", status.Sensors.ModemSkinC, status.ThermalHALStatus)
	} else if status.Throttlers.ModemSkinNrDsc > 0 {
		status.IsThrottled = true
		status.Culprit = fmt.Sprintf("Modem Skin NR Controller (modem_skin_nr_dsc = %d)", status.Throttlers.ModemSkinNrDsc)
		status.CulpritDetail = fmt.Sprintf("Qualcomm Thermal Engine membatasi 5G NR karena suhu skin modem (%.1f°C) melebihi ambang batas.", status.Sensors.ModemSkinC)
	} else if status.Throttlers.ModemPaFr1Dsc > 0 {
		status.IsThrottled = true
		status.Culprit = fmt.Sprintf("5G Sub-6 Power Amplifier (modem_pa_fr1_dsc = %d)", status.Throttlers.ModemPaFr1Dsc)
		status.CulpritDetail = fmt.Sprintf("Chip Power Amplifier 5G (%.1f°C) melewati limit proteksi RF. Modem memotong daya transmisi 5G NR!", status.Sensors.PA1Sub6C)
	} else if status.Sensors.PA1Sub6C >= 48.0 {
		status.IsThrottled = true
		status.Culprit = fmt.Sprintf("5G Sub-6 Power Amplifier 1 (PA1 = %.1f°C)", status.Sensors.PA1Sub6C)
		status.CulpritDetail = fmt.Sprintf("Suhu PA1 sangat panas (%.1f°C). Transceiver 5G bersiap memutus SCG 5G untuk perlindungan hardware.", status.Sensors.PA1Sub6C)
	} else if status.Sensors.ModemSkinC >= 43.0 {
		status.IsThrottled = true
		status.Culprit = fmt.Sprintf("Modem Skin Board (SKIN = %.1f°C)", status.Sensors.ModemSkinC)
		status.CulpritDetail = fmt.Sprintf("Sensor bodi modem (%.1f°C) melewati batas 42.0°C. Ini adalah sensor utama pemicu 5G Disconnect di Samsung.", status.Sensors.ModemSkinC)
	} else if status.Sensors.CellFrontCFC >= 45.0 {
		status.IsThrottled = true
		status.Culprit = fmt.Sprintf("Cellular Front-End Thermistor (sec-cf = %.1f°C)", status.Sensors.CellFrontCFC)
		status.CulpritDetail = fmt.Sprintf("Sensor RF depan (%.1f°C) terlalu panas akibat traffic bandwidth besar.", status.Sensors.CellFrontCFC)
	}
}

// Get5GThermalStatus executes the 5G thermal diagnostic utility and parses the telemetry report.
func Get5GThermalStatus() (*Thermal5GStatus, error) {
	cli, _ := find5GThermalCLI()
	cmd := fmt.Sprintf("%q -j", cli)
	out, err := config.ExecSuTimeout(5*time.Second, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to run 5g thermal diagnostics: %v (output: %s)", err, strings.TrimSpace(string(out)))
	}

	trimmed := strings.TrimSpace(string(out))
	idx := strings.Index(trimmed, "{")
	if idx == -1 {
		return nil, fmt.Errorf("invalid json output from 5g thermal tool: %s", trimmed)
	}

	var status Thermal5GStatus
	if jsonErr := json.Unmarshal([]byte(trimmed[idx:]), &status); jsonErr != nil {
		return nil, fmt.Errorf("failed to parse 5g thermal json: %v", jsonErr)
	}

	status.ThermalHALDesc = getThermalHALDesc(status.ThermalHALStatus)
	evaluate5GThermalCulprit(&status)

	return &status, nil
}
