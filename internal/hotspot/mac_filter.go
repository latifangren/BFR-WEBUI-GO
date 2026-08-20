package hotspot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

type MACFilterConfig struct {
	Mode        string   `json:"mode"` // "disabled", "blacklist", "whitelist"
	BlockedMACs []string `json:"blocked_macs"`
	AllowedMACs []string `json:"allowed_macs"`
}

type MACFilterStatus struct {
	ActiveMode   string `json:"active_mode"`
	BlockedCount int    `json:"blocked_count"`
	AllowedCount int    `json:"allowed_count"`
	RulesCount   int    `json:"rules_count"`
}

var (
	macFilterMu   sync.RWMutex
	currentFilter MACFilterConfig
	activeRules   int
)

func getMACFilterStoragePath() string {
	return config.GetPersistentFilePath("hotspot_mac_filter.json")
}

// LoadMACFilterConfig reads persisted MAC filter configuration from JSON file.
func LoadMACFilterConfig() (*MACFilterConfig, error) {
	macFilterMu.Lock()
	defer macFilterMu.Unlock()

	dataPath := getMACFilterStoragePath()
	buf, err := os.ReadFile(dataPath)
	if err != nil && dataPath != "hotspot_mac_filter.json" {
		buf, err = os.ReadFile("hotspot_mac_filter.json")
	}

	if err == nil {
		buf = bytes.TrimPrefix(buf, []byte("\xef\xbb\xbf"))
		var cfg MACFilterConfig
		if err := json.Unmarshal(buf, &cfg); err == nil {
			if cfg.BlockedMACs == nil {
				cfg.BlockedMACs = make([]string, 0)
			}
			if cfg.AllowedMACs == nil {
				cfg.AllowedMACs = make([]string, 0)
			}
			if cfg.Mode == "" {
				cfg.Mode = "disabled"
			}
			currentFilter = cfg
			return &currentFilter, nil
		}
	}

	currentFilter = MACFilterConfig{
		Mode:        "disabled",
		BlockedMACs: make([]string, 0),
		AllowedMACs: make([]string, 0),
	}
	return &currentFilter, nil
}

// SaveMACFilterConfig saves MAC filter configuration to JSON file.
func SaveMACFilterConfig(cfg *MACFilterConfig) error {
	macFilterMu.Lock()
	defer macFilterMu.Unlock()

	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if cfg.BlockedMACs == nil {
		cfg.BlockedMACs = make([]string, 0)
	}
	if cfg.AllowedMACs == nil {
		cfg.AllowedMACs = make([]string, 0)
	}

	currentFilter = *cfg

	buf, err := json.MarshalIndent(currentFilter, "", "  ")
	if err != nil {
		return err
	}

	dataPath := getMACFilterStoragePath()
	dir := filepath.Dir(dataPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		dataPath = "hotspot_mac_filter.json"
	}

	if err := os.WriteFile(dataPath, buf, 0644); err != nil {
		if dataPath != "hotspot_mac_filter.json" {
			_ = os.WriteFile("hotspot_mac_filter.json", buf, 0644)
		}
		return err
	}

	return nil
}

func isValidMAC(mac string) bool {
	_, err := net.ParseMAC(strings.TrimSpace(mac))
	return err == nil
}

// ClearMACFilter removes custom iptables chain rules for MAC filtering.
func ClearMACFilter() error {
	macFilterMu.Lock()
	defer macFilterMu.Unlock()

	cmdClean := "iptables -D FORWARD -j BFR_HOTSPOT_MAC 2>/dev/null; " +
		"iptables -F BFR_HOTSPOT_MAC 2>/dev/null; " +
		"iptables -X BFR_HOTSPOT_MAC 2>/dev/null; " +
		"ip6tables -D FORWARD -j BFR_HOTSPOT_MAC 2>/dev/null; " +
		"ip6tables -F BFR_HOTSPOT_MAC 2>/dev/null; " +
		"ip6tables -X BFR_HOTSPOT_MAC 2>/dev/null"

	_, _ = config.ExecSuTimeout(5*time.Second, cmdClean)

	activeRules = 0
	logger.Get().Infof("hotspot", "Cleared MAC filter iptables rules")
	return nil
}

// ApplyMACFilter configures iptables rules according to the MACFilterConfig mode.
func ApplyMACFilter(cfg *MACFilterConfig) error {
	if cfg == nil {
		return fmt.Errorf("invalid config")
	}

	if err := SaveMACFilterConfig(cfg); err != nil {
		return err
	}

	if cfg.Mode == "disabled" || cfg.Mode == "" {
		return ClearMACFilter()
	}

	macFilterMu.Lock()
	defer macFilterMu.Unlock()

	// Clear existing BFR_HOTSPOT_MAC chain hooks
	cmdClean := "iptables -D FORWARD -j BFR_HOTSPOT_MAC 2>/dev/null || true; " +
		"iptables -F BFR_HOTSPOT_MAC 2>/dev/null || true; " +
		"iptables -X BFR_HOTSPOT_MAC 2>/dev/null || true; " +
		"iptables -N BFR_HOTSPOT_MAC; " +
		"iptables -I FORWARD 1 -j BFR_HOTSPOT_MAC"

	rulesCount := 0
	var cmds []string
	cmds = append(cmds, cmdClean)

	switch cfg.Mode {
	case "blacklist":
		for _, mac := range cfg.BlockedMACs {
			cleanMAC := strings.TrimSpace(mac)
			if isValidMAC(cleanMAC) {
				cmds = append(cmds, fmt.Sprintf("iptables -A BFR_HOTSPOT_MAC -m mac --mac-source %s -j DROP", cleanMAC))
				rulesCount++
			}
		}

	case "whitelist":
		for _, mac := range cfg.AllowedMACs {
			cleanMAC := strings.TrimSpace(mac)
			if isValidMAC(cleanMAC) {
				cmds = append(cmds, fmt.Sprintf("iptables -A BFR_HOTSPOT_MAC -m mac --mac-source %s -j ACCEPT", cleanMAC))
				rulesCount++
			}
		}
		// Default rule for whitelist: drop all other clients
		cmds = append(cmds, "iptables -A BFR_HOTSPOT_MAC -j DROP")
		rulesCount++
	}

	execCmdStr := strings.Join(cmds, " && ")
	out, err := config.ExecSuTimeout(5*time.Second, execCmdStr)
	if err != nil {
		activeRules = 0
		return fmt.Errorf("failed to apply iptables MAC filter rules: %v, out: %s", err, string(out))
	}

	activeRules = rulesCount
	logger.Get().Infof("hotspot", "Applied MAC filter rules (mode: %s, rules: %d)", cfg.Mode, rulesCount)
	return nil
}

// GetMACFilterStatus returns the current status of MAC filtering.
func GetMACFilterStatus() MACFilterStatus {
	macFilterMu.RLock()
	defer macFilterMu.RUnlock()

	mode := currentFilter.Mode
	if mode == "" {
		mode = "disabled"
	}

	return MACFilterStatus{
		ActiveMode:   mode,
		BlockedCount: len(currentFilter.BlockedMACs),
		AllowedCount: len(currentFilter.AllowedMACs),
		RulesCount:   activeRules,
	}
}
