package charger

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/config"
)

type Config struct {
	Enabled      bool   `json:"enabled"`
	StartPercent int    `json:"start_percent"`
	StopPercent  int    `json:"stop_percent"`
	CustomPath   string `json:"custom_path"`
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
	// Qualcomm PMIC SMB5 Main / Force FCC (Proven working on Pixel 5 / Snapdragon)
	{"/sys/class/power_supply/main/force_main_fcc", "force_main_fcc"},
	{"/sys/class/power_supply/main/force_main_icl", "force_main_icl"},
	// Google Pixel 5 / Tensor / Pixel series
	{"/sys/class/power_supply/battery/charge_limit", "charge_limit"},
	// Standard & Qualcomm / Xiaomi / Redmi / POCO
	{"/sys/class/power_supply/battery/charging_enabled", "charging_enabled"},
	{"/sys/class/power_supply/battery/input_suspend", "input_suspend"},
	{"/sys/class/power_supply/battery/charge_control_limit", "charge_control_limit"},
	{"/sys/class/power_supply/battery/charge_control_limit_max", "charge_control_limit_max"},
	// Samsung Knox / OneUI Battery Protection
	{"/sys/class/power_supply/battery/store_mode", "store_mode"},
	{"/sys/class/power_supply/battery/batt_slate_mode", "batt_slate_mode"},
	// Qualcomm PMIC SMB5 Main & USB
	{"/sys/class/power_supply/main/charging_enabled", "charging_enabled"},
	{"/sys/class/power_supply/main/input_suspend", "input_suspend"},
	{"/sys/class/power_supply/bms/charging_enabled", "charging_enabled"},
	{"/sys/class/power_supply/usb/charging_enabled", "charging_enabled"},
	{"/sys/class/power_supply/usb/input_suspend", "input_suspend"},
	// OnePlus / OPPO / Realme / Motorola
	{"/sys/class/power_supply/battery/mmi_charging_enable", "mmi_charging_enable"},
	{"/sys/class/power_supply/battery/op_disable_charge", "op_disable_charge"},
	{"/sys/class/power_supply/battery/charging_switch", "charging_switch"},
	{"/sys/class/power_supply/battery/bd_trickle_enable", "bd_trickle_enable"},
	// MediaTek (MTK)
	{"/sys/class/power_supply/battery/sub_charging_enabled", "sub_charging_enabled"},
}

var (
	globalManager *Manager
	once          sync.Once
)

func getStoragePath() string {
	return config.GetPersistentFilePath("charger_config.json")
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
		buf = bytes.TrimPrefix(buf, []byte("\xef\xbb\xbf"))
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
	if err := config.WriteFileAtomic(m.dataPath, buf, 0644); err != nil {
		if m.dataPath != "charger_config.json" {
			m.dataPath = "charger_config.json"
			_ = config.WriteFileAtomic(m.dataPath, buf, 0644)
		}
	}
}

func fileExistsAndWritable(path string) bool {
	if path == "" {
		return false
	}
	f, err := os.OpenFile(path, os.O_WRONLY, 0666)
	if err == nil {
		f.Close()
		return true
	}
	// Fallback check if running under root su
	out, err := exec.Command(config.SUBin, "-c", fmt.Sprintf("test -w %s && echo 1 || echo 0", path)).Output()
	if err == nil && strings.TrimSpace(string(out)) == "1" {
		return true
	}
	return false
}

func (m *Manager) autoScan() {
	if m.config.CustomPath != "" && fileExistsAndWritable(m.config.CustomPath) {
		m.detectedPath = m.config.CustomPath
		m.detectedType = detectTypeFromPath(m.config.CustomPath)
		m.log(fmt.Sprintf("Using custom charging control path: %s (%s)", m.detectedPath, m.detectedType))
		return
	}

	for _, cand := range candidatePaths {
		if fileExistsAndWritable(cand.path) {
			m.detectedPath = cand.path
			m.detectedType = cand.typ
			m.log(fmt.Sprintf("Auto-scanned charging control path: %s (%s)", cand.path, cand.typ))
			return
		}
	}

	// Dynamic sysfs fallback scanner if static candidatePaths missing
	psDir := "/sys/class/power_supply"
	if entries, err := os.ReadDir(psDir); err == nil {
		keywords := []string{"charging", "suspend", "limit", "fcc", "store_mode", "switch"}
		for _, entry := range entries {
			subPath := filepath.Join(psDir, entry.Name())
			if files, err := os.ReadDir(subPath); err == nil {
				for _, f := range files {
					name := strings.ToLower(f.Name())
					for _, kw := range keywords {
						if strings.Contains(name, kw) {
							fullPath := filepath.Join(subPath, f.Name())
							if fileExistsAndWritable(fullPath) {
								m.detectedPath = fullPath
								m.detectedType = detectTypeFromPath(fullPath)
								m.log(fmt.Sprintf("Dynamic auto-scanned charging control path: %s (%s)", fullPath, m.detectedType))
								return
							}
						}
					}
				}
			}
		}
	}

	m.log("No writable charging control path found in sysfs scan")
}

func detectTypeFromPath(path string) string {
	base := filepath.Base(path)
	switch base {
	case "force_main_fcc", "force_main_icl":
		return base
	case "charge_limit":
		return "charge_limit"
	case "charge_control_limit", "charge_control_limit_max":
		return "charge_control_limit"
	case "input_suspend", "op_disable_charge", "store_mode", "batt_slate_mode":
		return base
	default:
		return "charging_enabled"
	}
}

func writeSysfs(path string, val string) error {
	err := os.WriteFile(path, []byte(val+"\n"), 0644)
	if err == nil {
		return nil
	}
	cmd := exec.Command(config.SUBin, "-c", fmt.Sprintf("echo %s > %s", val, path))
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
	case "force_main_fcc", "force_main_icl":
		enableVal = "1500000"
		disableVal = "0"
	case "charge_limit":
		enableVal = "-1"
		disableVal = strconv.Itoa(m.config.StopPercent)
	case "charge_control_limit", "charge_control_limit_max":
		enableVal = "100"
		disableVal = strconv.Itoa(m.config.StopPercent)
	case "input_suspend", "op_disable_charge", "store_mode", "batt_slate_mode":
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
			m.log(fmt.Sprintf("Charging disabled/limited (wrote %s to %s)", targetVal, m.detectedPath))
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

	// For percentage threshold kernels (Pixel 5 charge_limit, ROG charge_control_limit): write target percentage directly when enabled
	if m.detectedType == "charge_limit" || m.detectedType == "charge_control_limit" || m.detectedType == "charge_control_limit_max" {
		expectedVal := strconv.Itoa(m.config.StopPercent)
		if !m.chargingDisabled {
			m.setChargingStateLocked(true)
		} else {
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

// M-6: GetStatus reads current state only — does NOT call evaluateLocked() to
// avoid sysfs writes/side-effects from a read-only GET request. Actual
// charging evaluation continues on the background ticker (start()).
func (m *Manager) GetStatus() StatusResponse {
	m.mu.RLock()
	resp := StatusResponse{
		Config:           m.config,
		DetectedPath:     m.detectedPath,
		DetectedType:     m.detectedType,
		ChargingDisabled: m.chargingDisabled,
		CurrentLevel:     getBatteryCapacity(),
		Logs:             make([]string, len(m.logs)),
	}
	copy(resp.Logs, m.logs)
	m.mu.RUnlock()
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
