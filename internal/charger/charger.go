package charger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"bfr-webui-go/internal/config"
)

var (
	globalManager *Manager
	once          sync.Once
)

// Manager coordinates smart battery charging, threshold evaluation, and ACC switch profiles.
type Manager struct {
	mu               sync.RWMutex
	config           Config
	dataPath         string
	activeProfile    *SwitchProfile
	detectedPath     string
	detectedType     string
	chargingDisabled bool
	probeResults     []ProbeResult
	logs             []string
	stopCh           chan struct{}
	stopOnce         sync.Once
}

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
		logs:         make([]string, 0),
		probeResults: make([]ProbeResult, 0),
		stopCh:       make(chan struct{}),
	}
	m.loadConfig()
	m.mu.Lock()
	m.autoScanLocked()
	m.mu.Unlock()
	return m
}

// GetManager returns the singleton Manager instance.
func GetManager() *Manager {
	once.Do(func() {
		globalManager = newManager()
		globalManager.start()
	})
	return globalManager
}

// Stop halts background monitoring goroutines cleanly.
func (m *Manager) Stop() {
	if m == nil {
		return
	}
	m.stopOnce.Do(func() {
		if m.stopCh != nil {
			close(m.stopCh)
		}
	})
}

// Stop halts the singleton Manager's background monitoring goroutines.
func Stop() {
	if globalManager != nil {
		globalManager.Stop()
	}
}

func (m *Manager) logLocked(msg string) {
	entry := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg)
	m.logs = append(m.logs, entry)
	if len(m.logs) > 50 {
		m.logs = m.logs[len(m.logs)-50:]
	}
}

// Log records a message to the internal charger log history in a thread-safe manner.
func (m *Manager) Log(msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logLocked(msg)
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
			if cfg.StartPercent >= cfg.StopPercent {
				cfg.StartPercent = cfg.StopPercent - 5
				if cfg.StartPercent < 0 {
					cfg.StartPercent = 0
				}
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

// autoScanLocked resolves the active switch profile based on configuration or auto-detection.
func (m *Manager) autoScanLocked() {
	// 1. Custom Path
	if m.config.CustomPath != "" && fileExistsAndWritable(m.config.CustomPath) {
		onVal := m.config.CustomOn
		offVal := m.config.CustomOff
		if onVal == "" {
			onVal = "1"
		}
		if offVal == "" {
			offVal = "0"
		}
		profile := SwitchProfile{
			ID:          "custom",
			Name:        "Custom Switch Node",
			Description: fmt.Sprintf("User custom path: %s", m.config.CustomPath),
			Nodes: []SwitchNode{
				{Path: m.config.CustomPath, OnValue: onVal, OffValue: offVal},
			},
			IsBypass: false,
		}
		m.activeProfile = &profile
		m.detectedPath = m.config.CustomPath
		m.detectedType = "custom"
		m.logLocked(fmt.Sprintf("Active profile set to custom path: %s (on=%s, off=%s)", m.detectedPath, onVal, offVal))
		return
	}

	available := GetExistingSwitchProfiles()

	// 2. Selected Switch Profile ID
	if m.config.SelectedSwitch != "" {
		for _, p := range available {
			if p.ID == m.config.SelectedSwitch {
				cp := p
				m.activeProfile = &cp
				if len(cp.Nodes) > 0 {
					m.detectedPath = cp.Nodes[0].Path
				}
				m.detectedType = cp.ID
				m.logLocked(fmt.Sprintf("Active profile set to selected switch: %s (%s)", cp.Name, cp.ID))
				return
			}
		}
		for _, p := range AllSwitchProfiles {
			if p.ID == m.config.SelectedSwitch {
				cp := p
				m.activeProfile = &cp
				if len(cp.Nodes) > 0 {
					m.detectedPath = cp.Nodes[0].Path
				}
				m.detectedType = cp.ID
				m.logLocked(fmt.Sprintf("Active profile fallback to configured switch: %s (%s)", cp.Name, cp.ID))
				return
			}
		}
	}

	// 3. First available detected profile
	if len(available) > 0 {
		cp := available[0]
		m.activeProfile = &cp
		if len(cp.Nodes) > 0 {
			m.detectedPath = cp.Nodes[0].Path
		}
		m.detectedType = cp.ID
		m.logLocked(fmt.Sprintf("Auto-detected charging control profile: %s (%s)", cp.Name, cp.ID))
		return
	}

	m.logLocked("No writable charging control switch profile found on device")
}

// applySwitchLocked applies either the ON or OFF values to all nodes in the active profile.
func (m *Manager) applySwitchLocked(disable bool) {
	if m.activeProfile == nil {
		return
	}
	for _, node := range m.activeProfile.Nodes {
		val := node.OnValue
		if disable {
			val = node.OffValue
		}
		_ = writeSysfs(node.Path, val)
	}
}

// setChargingStateLocked sets the internal charging disabled flag and updates hardware nodes.
func (m *Manager) setChargingStateLocked(disable bool) {
	m.chargingDisabled = disable
	actionStr := "enabled"
	if disable {
		actionStr = "disabled/limited"
	}

	if m.activeProfile == nil {
		m.logLocked(fmt.Sprintf("Charging %s (no active profile on device)", actionStr))
		return
	}

	m.applySwitchLocked(disable)
	m.logLocked(fmt.Sprintf("Charging %s via profile: %s", actionStr, m.activeProfile.Name))
}

// evaluateLocked checks battery capacity against StartPercent/StopPercent limits.
func (m *Manager) evaluateLocked() {
	// When limiter is disabled, ensure hardware charging is normal
	if !m.config.Enabled {
		if m.chargingDisabled {
			m.setChargingStateLocked(false)
			m.logLocked("Limiter disabled: Charge state restored to normal (enabled)")
		}
		return
	}

	if m.activeProfile == nil {
		m.autoScanLocked()
		if m.activeProfile == nil {
			return
		}
	}

	batt := GetBatteryInfo()
	level := batt.Capacity

	if level < 0 {
		return
	}

	// Percentage-threshold kernels: write target percentage directly
	if m.detectedType == "charge_limit" ||
		m.detectedType == "universal_charge_control_limit" || m.detectedType == "charge_control_limit" {
		expectedVal := strconv.Itoa(m.config.StopPercent)
		if !m.chargingDisabled {
			m.setChargingStateLocked(true)
		} else if len(m.activeProfile.Nodes) > 0 {
			_ = writeSysfs(m.activeProfile.Nodes[0].Path, expectedVal)
		}
		return
	}

	// Standard threshold cycle
	if level >= m.config.StopPercent {
		if !m.chargingDisabled {
			m.logLocked(fmt.Sprintf("Battery level (%d%%) >= stop limit (%d%%). Stopping charge.", level, m.config.StopPercent))
			m.setChargingStateLocked(true)
		}
	} else if level <= m.config.StartPercent {
		if m.chargingDisabled {
			m.logLocked(fmt.Sprintf("Battery level (%d%%) <= start limit (%d%%). Resuming charge.", level, m.config.StartPercent))
			m.setChargingStateLocked(false)
		}
	}
}

func (m *Manager) start() {
	m.mu.Lock()
	if m.config.Enabled || m.chargingDisabled {
		m.evaluateLocked()
	}
	m.mu.Unlock()

	// 1. Anti-reset loop (force_off): re-assert OFF value every 2 seconds when charging is disabled
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-m.stopCh:
				return
			case <-ticker.C:
				m.mu.Lock()
				if m.config.Enabled && m.chargingDisabled && m.activeProfile != nil {
					m.applySwitchLocked(true)
				}
				m.mu.Unlock()
			}
		}
	}()

	// 2. Battery monitor loop: check capacity against thresholds every 10 seconds
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-m.stopCh:
				return
			case <-ticker.C:
				m.mu.Lock()
				if m.config.Enabled || m.chargingDisabled {
					m.evaluateLocked()
				}
				m.mu.Unlock()
			}
		}
	}()
}

// GetStatus returns the current status and state of the charging manager.
func (m *Manager) GetStatus() StatusResponse {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.getStatusLocked()
}

func (m *Manager) getStatusLocked() StatusResponse {
	batt := GetBatteryInfo()
	resp := StatusResponse{
		Config:            m.config,
		ActiveProfile:     m.activeProfile,
		DetectedPath:      m.detectedPath,
		DetectedType:      m.detectedType,
		ChargingDisabled:  m.chargingDisabled,
		CurrentLevel:      batt.Capacity,
		Battery:           batt,
		AvailableSwitches: GetExistingSwitchProfiles(),
		ProbeResults:      make([]ProbeResult, len(m.probeResults)),
		Logs:              make([]string, len(m.logs)),
	}
	copy(resp.ProbeResults, m.probeResults)
	copy(resp.Logs, m.logs)
	return resp
}

// UpdateConfig updates the manager configuration, saves to persistent storage, and evaluates state.
func (m *Manager) UpdateConfig(cfg Config) StatusResponse {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cfg.StartPercent <= 0 {
		cfg.StartPercent = 70
	}
	if cfg.StopPercent <= 0 {
		cfg.StopPercent = 80
	}
	if cfg.StartPercent >= cfg.StopPercent {
		cfg.StartPercent = cfg.StopPercent - 5
		if cfg.StartPercent < 0 {
			cfg.StartPercent = 0
		}
	}

	m.config = cfg
	m.saveConfigLocked()
	m.autoScanLocked()
	m.evaluateLocked()

	return m.getStatusLocked()
}

// ToggleCharging manually toggles the chargingDisabled state.
func (m *Manager) ToggleCharging() StatusResponse {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.setChargingStateLocked(!m.chargingDisabled)
	return m.getStatusLocked()
}

// ProbeSwitches tests all available hardware charging switches and updates test records.
func (m *Manager) ProbeSwitches() []ProbeResult {
	return ProbeAll(m)
}

// ProbeSingleSwitch tests a specific profile by its ID.
func (m *Manager) ProbeSingleSwitch(profileID string) ProbeResult {
	return ProbeSingle(m, profileID)
}

// Exported package-level helper functions for global access and compatibility.

func GetStatus() StatusResponse {
	return GetManager().GetStatus()
}

func UpdateConfig(cfg Config) StatusResponse {
	return GetManager().UpdateConfig(cfg)
}

func ToggleCharging() StatusResponse {
	return GetManager().ToggleCharging()
}

func ProbeSwitches() []ProbeResult {
	return GetManager().ProbeSwitches()
}

func ProbeSingleSwitch(profileID string) ProbeResult {
	return GetManager().ProbeSingleSwitch(profileID)
}
