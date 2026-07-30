package charger

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Enabled      bool `json:"enabled"`
	StartPercent int  `json:"start_percent"`
	StopPercent  int  `json:"stop_percent"`
}

type StatusResponse struct {
	Config           Config   `json:"config"`
	DetectedPath     string   `json:"detected_path"`
	DetectedType     string   `json:"detected_type"`
	ChargingDisabled bool     `json:"charging_disabled"`
	CurrentLevel     int      `json:"current_level"`
	Logs             []string `json:"logs"`
}

type Manager struct {
	mu               sync.RWMutex
	config           Config
	dataPath         string
	detectedPath     string
	detectedType     string
	chargingDisabled bool
	logs             []string
}

var candidatePaths = []struct {
	path string
	typ  string
}{
	{"/sys/class/power_supply/battery/charging_enabled", "charging_enabled"},
	{"/sys/class/power_supply/battery/input_suspend", "input_suspend"},
	{"/sys/class/power_supply/battery/charge_control_limit_max", "charge_control_limit_max"},
	{"/sys/class/power_supply/battery/charge_control_limit", "charge_control_limit"},
	{"/sys/class/power_supply/battery/batt_slate_mode", "batt_slate_mode"},
	{"/sys/class/power_supply/battery/store_mode", "store_mode"},
	{"/sys/class/power_supply/main/charging_enabled", "charging_enabled"},
	{"/sys/class/power_supply/bms/charging_enabled", "charging_enabled"},
	{"/sys/class/power_supply/charger/charging_enabled", "charging_enabled"},
	{"/sys/class/power_supply/usb/charging_enabled", "charging_enabled"},
	{"/sys/class/power_supply/battery/mmi_charging_enable", "mmi_charging_enable"},
	{"/sys/class/power_supply/battery/op_disable_charge", "op_disable_charge"},
	{"/sys/class/power_supply/battery/charging_switch", "charging_switch"},
}

var (
	globalManager *Manager
	once          sync.Once
)

func getStoragePath() string {
	magiskDir := "/data/adb/modules/bfr_webui_go"
	magiskPath := filepath.Join(magiskDir, "charger_config.json")
	if _, err := os.Stat(magiskDir); err == nil {
		return magiskPath
	}
	if _, err := os.Stat(magiskPath); err == nil {
		return magiskPath
	}
	if _, err := os.Stat("charger_config.json"); err == nil {
		return "charger_config.json"
	}
	return magiskPath
}

func newManager() *Manager {
	m := &Manager{
		dataPath: getStoragePath(),
		config: Config{
			Enabled:      false,
			StartPercent: 70,
			StopPercent:  80,
		},
		logs: make([]string, 0),
	}
	m.loadConfig()
	m.autoScan()
	return m
}

func GetManager() *Manager {
	once.Do(func() {
		globalManager = newManager()
		globalManager.start()
	})
	return globalManager
}

func init() {
	GetManager()
}

func (m *Manager) log(msg string) {
	entry := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg)
	m.logs = append(m.logs, entry)
	if len(m.logs) > 50 {
		m.logs = m.logs[len(m.logs)-50:]
	}
}

func (m *Manager) loadConfig() {
	buf, err := os.ReadFile(m.dataPath)
	if err != nil && m.dataPath != "charger_config.json" {
		buf, err = os.ReadFile("charger_config.json")
		if err == nil {
			m.dataPath = "charger_config.json"
		}
	}
	if err == nil {
		var cfg Config
		if err := json.Unmarshal(buf, &cfg); err == nil {
			if cfg.StartPercent <= 0 {
				cfg.StartPercent = 70
			}
			if cfg.StopPercent <= 0 {
				cfg.StopPercent = 80
			}
			m.config = cfg
		}
	}
}

func (m *Manager) saveConfigLocked() {
	buf, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return
	}
	dir := filepath.Dir(m.dataPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		m.dataPath = "charger_config.json"
	}
	if err := os.WriteFile(m.dataPath, buf, 0644); err != nil {
		if m.dataPath != "charger_config.json" {
			m.dataPath = "charger_config.json"
			_ = os.WriteFile(m.dataPath, buf, 0644)
		}
	}
}

func fileExistsAndWritable(path string) bool {
	// Directly run the shell command checking existence and writability
	cmd := exec.Command("su", "-c", fmt.Sprintf("[ -f %s ] && [ -w %s ] && echo 1 || echo 0", path, path))
	out, err := cmd.Output()
	if err == nil && strings.TrimSpace(string(out)) == "1" {
		return true
	}
	// Fallback to direct os package open check
	file, err := os.OpenFile(path, os.O_WRONLY, 0644)
	if err == nil {
		file.Close()
		return true
	}
	return false
}

func (m *Manager) autoScan() {
	for _, cand := range candidatePaths {
		if fileExistsAndWritable(cand.path) {
			m.detectedPath = cand.path
			m.detectedType = cand.typ
			m.log(fmt.Sprintf("Auto-scanned charging control path: %s (%s)", cand.path, cand.typ))
			return
		}
	}
	m.log("No writable charging control path found in sysfs scan")
}

func writeSysfs(path string, val string) error {
	err := os.WriteFile(path, []byte(val+"\n"), 0644)
	if err == nil {
		return nil
	}
	cmd := exec.Command("su", "-c", fmt.Sprintf("echo %s > %s", val, path))
	return cmd.Run()
}

func getBatteryCapacity() int {
	paths := []string{
		"/sys/class/power_supply/battery/capacity",
		"/sys/class/power_supply/bms/capacity",
	}
	for _, p := range paths {
		file, err := os.Open(p)
		if err == nil {
			scanner := bufio.NewScanner(file)
			if scanner.Scan() {
				text := strings.TrimSpace(scanner.Text())
				if cap, err := strconv.Atoi(text); err == nil {
					file.Close()
					return cap
				}
			}
			file.Close()
		}
	}
	return -1
}

func (m *Manager) setChargingStateLocked(disable bool) {
	if m.detectedPath == "" {
		return
	}

	var enableVal, disableVal string
	switch m.detectedType {
	case "charge_control_limit", "charge_control_limit_max":
		enableVal = "100"
		disableVal = strconv.Itoa(m.config.StopPercent)
	case "input_suspend", "op_disable_charge":
		enableVal = "0"
		disableVal = "1"
	default:
		enableVal = "1"
		disableVal = "0"
	}

	targetVal := enableVal
	if disable {
		targetVal = disableVal
	}

	if err := writeSysfs(m.detectedPath, targetVal); err != nil {
		m.log(fmt.Sprintf("Failed writing '%s' to %s: %v", targetVal, m.detectedPath, err))
	} else {
		m.chargingDisabled = disable
		if disable {
			m.log(fmt.Sprintf("Charging disabled (wrote %s to %s)", targetVal, m.detectedPath))
		} else {
			m.log(fmt.Sprintf("Charging enabled (wrote %s to %s)", targetVal, m.detectedPath))
		}
	}
}

func (m *Manager) evaluateLocked() {
	if m.detectedPath == "" {
		m.autoScan()
		if m.detectedPath == "" {
			return
		}
	}

	level := getBatteryCapacity()

	if !m.config.Enabled {
		if m.chargingDisabled {
			m.setChargingStateLocked(false)
			m.log("Limiter disabled: Charge state restored to normal (enabled)")
		}
		return
	}

	if level < 0 {
		return
	}

	// For percentage threshold kernels: write directly when enabled
	if m.detectedType == "charge_control_limit" || m.detectedType == "charge_control_limit_max" {
		expectedVal := strconv.Itoa(m.config.StopPercent)
		if !m.chargingDisabled {
			m.setChargingStateLocked(true)
		} else {
			// If already set, verify we write the updated stop percent if config changed
			_ = writeSysfs(m.detectedPath, expectedVal)
		}
		return
	}

	// For binary switches: handle StopPercent and StartPercent threshold cycles
	if level >= m.config.StopPercent {
		if !m.chargingDisabled {
			m.log(fmt.Sprintf("Battery level (%d%%) >= stop limit (%d%%). Stopping charge.", level, m.config.StopPercent))
			m.setChargingStateLocked(true)
		}
	} else if level <= m.config.StartPercent {
		if m.chargingDisabled {
			m.log(fmt.Sprintf("Battery level (%d%%) <= start limit (%d%%). Resuming charge.", level, m.config.StartPercent))
			m.setChargingStateLocked(false)
		}
	}
}

func (m *Manager) start() {
	m.mu.Lock()
	m.evaluateLocked()
	m.mu.Unlock()

	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			m.mu.Lock()
			m.evaluateLocked()
			m.mu.Unlock()
		}
	}()
}

func (m *Manager) GetStatus() StatusResponse {
	m.mu.Lock()
	m.evaluateLocked()
	resp := StatusResponse{
		Config:           m.config,
		DetectedPath:     m.detectedPath,
		DetectedType:     m.detectedType,
		ChargingDisabled: m.chargingDisabled,
		CurrentLevel:     getBatteryCapacity(),
		Logs:             make([]string, len(m.logs)),
	}
	copy(resp.Logs, m.logs)
	m.mu.Unlock()
	return resp
}

func (m *Manager) UpdateConfig(cfg Config) StatusResponse {
	m.mu.Lock()
	if cfg.StartPercent <= 0 {
		cfg.StartPercent = 70
	}
	if cfg.StopPercent <= 0 {
		cfg.StopPercent = 80
	}
	if cfg.StartPercent >= cfg.StopPercent {
		cfg.StartPercent = cfg.StopPercent - 5
	}
	m.config = cfg
	m.saveConfigLocked()
	m.log(fmt.Sprintf("Config updated: Enabled=%v, Start=%d%%, Stop=%d%%", cfg.Enabled, cfg.StartPercent, cfg.StopPercent))
	m.evaluateLocked()
	m.mu.Unlock()

	return m.GetStatus()
}

func GetStatus() StatusResponse {
	return GetManager().GetStatus()
}

func UpdateConfig(cfg Config) StatusResponse {
	return GetManager().UpdateConfig(cfg)
}
