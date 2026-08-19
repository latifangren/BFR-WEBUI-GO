package modem

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

type SignalInfo struct {
	RSRP        int    `json:"rsrp"`
	RSRQ        int    `json:"rsrq"`
	SINR        int    `json:"sinr"`
	RSSI        int    `json:"rssi"`
	Operator    string `json:"operator"`
	NetworkType string `json:"network_type"` // "LTE", "NR_5G", "HSPA", "GSM"
	Band        string `json:"band"`
	CellID      string `json:"cell_id"`
	TAC         string `json:"tac"`
	PCI         string `json:"pci"`
}

type BandConfig struct {
	Engine       string `json:"engine"`        // "universal", "qualcomm_at", "intent"
	PreferredRAT string `json:"preferred_rat"` // "5g_only", "4g_only", "hybrid", "3g_only"
	LTEBands     []int  `json:"lte_bands"`
	NRBands      []int  `json:"nr_bands"`
	HexBitmask   string `json:"hex_bitmask"`
}

type ATResponse struct {
	Command   string `json:"command"`
	Response  string `json:"response"`
	Success   bool   `json:"success"`
	Timestamp string `json:"timestamp"`
	PortUsed  string `json:"port_used"`
}

type Manager struct {
	mu       sync.RWMutex
	dataPath string
	config   BandConfig
}

var (
	globalManager *Manager
	once          sync.Once

	serialPorts = []string{
		"/dev/smd11",
		"/dev/smd7",
		"/dev/ttyUSB2",
		"/dev/ttyUSB1",
		"/dev/ttyUSB0",
		"/dev/atcmd1",
		"/dev/radio/at",
	}

	reATCommand = regexp.MustCompile(`^[a-zA-Z0-9+=?,_\-\s"#$*]+$`)
)

func getStoragePath() string {
	return config.GetPersistentFilePath("modem_config.json")
}

func newManager() *Manager {
	m := &Manager{
		dataPath: getStoragePath(),
		config: BandConfig{
			Engine:       "universal",
			PreferredRAT: "hybrid",
			LTEBands:     make([]int, 0),
			NRBands:      make([]int, 0),
			HexBitmask:   "0x0",
		},
	}
	_, _ = m.LoadConfig()
	return m
}

func GetManager() *Manager {
	once.Do(func() {
		globalManager = newManager()
	})
	return globalManager
}

// BandsToHexBitmask converts a slice of band numbers into a hex bitmask string (e.g. [1, 3, 8] -> "0x85").
func BandsToHexBitmask(bands []int) string {
	if len(bands) == 0 {
		return "0x0"
	}

	mask := new(big.Int)
	for _, b := range bands {
		if b > 0 {
			// Band N sets the (N-1)-th bit
			bit := new(big.Int).Lsh(big.NewInt(1), uint(b-1))
			mask.Or(mask, bit)
		}
	}

	return fmt.Sprintf("0x%X", mask)
}

// HexBitmaskToBands converts a hex bitmask string (e.g. "0x85" or "85") into a slice of band numbers.
func HexBitmaskToBands(hexStr string) []int {
	clean := strings.TrimPrefix(strings.TrimSpace(hexStr), "0x")
	clean = strings.TrimPrefix(clean, "0X")
	if clean == "" || clean == "0" {
		return []int{}
	}

	mask := new(big.Int)
	_, ok := mask.SetString(clean, 16)
	if !ok {
		return []int{}
	}

	bands := make([]int, 0)
	bitLen := mask.BitLen()
	for i := 0; i < bitLen; i++ {
		if mask.Bit(i) == 1 {
			bands = append(bands, i+1)
		}
	}

	return bands
}

// FindModemPort searches for an available serial port for Qualcomm/AT modem communication.
func FindModemPort() string {
	for _, port := range serialPorts {
		if _, err := os.Stat(port); err == nil {
			return port
		}
	}
	return ""
}

// ExecuteATCommand sends an AT command to an available serial port or fallback shell tool and returns an ATResponse.
func ExecuteATCommand(cmd string) ATResponse {
	resp := ATResponse{
		Command:   cmd,
		Timestamp: time.Now().Format(time.RFC3339),
		Success:   false,
	}

	// Validate AT command to prevent shell injection
	trimmedCmd := strings.TrimSpace(cmd)
	if trimmedCmd == "" || !reATCommand.MatchString(trimmedCmd) {
		resp.Response = "ERROR: Invalid or empty AT command format"
		return resp
	}

	port := FindModemPort()
	resp.PortUsed = port

	if port != "" {
		// Use root shell with stty/echo to interact with serial port safely
		execCmdStr := fmt.Sprintf(
			"stty -F %s 115200 raw -echo 2>/dev/null; "+
				"echo -e '%s\\r' > %s; "+
				"timeout 2 cat %s 2>/dev/null | head -n 20",
			port, trimmedCmd, port, port,
		)
		out, err := config.ExecSuTimeout(3*time.Second, execCmdStr)
		if err == nil && len(out) > 0 {
			resp.Response = strings.TrimSpace(string(out))
			resp.Success = strings.Contains(resp.Response, "OK")
			return resp
		}
	}

	// Fallback check: use gsmctl or qmicli if serial port unavailable
	fallbackCmdStr := fmt.Sprintf("gsmctl -A '%s' 2>/dev/null || qmicli -d /dev/cdc-wdm0 --dms-at-command='%s' 2>/dev/null", trimmedCmd, trimmedCmd)
	out, err := config.ExecSuTimeout(3*time.Second, fallbackCmdStr)
	if err == nil && len(out) > 0 {
		resp.Response = strings.TrimSpace(string(out))
		resp.Success = strings.Contains(resp.Response, "OK")
		return resp
	}

	resp.Response = "ERROR: No accessible modem port or command failed"
	return resp
}

// GetSignalInfo parses telephony registry dumpsys or AT commands to construct a SignalInfo struct.
func GetSignalInfo() (SignalInfo, error) {
	sig := SignalInfo{
		RSRP:        -999,
		RSRQ:        -999,
		SINR:        -999,
		RSSI:        -999,
		Operator:    "Unknown",
		NetworkType: "LTE",
		Band:        "Unknown",
		CellID:      "Unknown",
		TAC:         "Unknown",
		PCI:         "Unknown",
	}

	out, err := exec.Command(config.SUBin, "-c", "dumpsys telephony.registry 2>/dev/null | grep -E 'mSignalStrength|mServiceState|mCellIdentity' | head -n 30").Output()
	if err == nil && len(out) > 0 {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "rsrp=") || strings.Contains(line, "rsrp:") {
				sig.RSRP = extractIntFromLine(line, "rsrp")
			}
			if strings.Contains(line, "rsrq=") || strings.Contains(line, "rsrq:") {
				sig.RSRQ = extractIntFromLine(line, "rsrq")
			}
			if strings.Contains(line, "rssi=") || strings.Contains(line, "rssi:") {
				sig.RSSI = extractIntFromLine(line, "rssi")
			}
			if strings.Contains(line, "mOperatorAlphaLong=") {
				parts := strings.Split(line, "mOperatorAlphaLong=")
				if len(parts) > 1 {
					op := strings.Trim(strings.Fields(parts[1])[0], "\",")
					if op != "" && op != "null" {
						sig.Operator = op
					}
				}
			}
			if strings.Contains(line, "mBand=") || strings.Contains(line, "mBandwidth=") {
				parts := strings.Split(line, "mBand=")
				if len(parts) > 1 {
					sig.Band = strings.Trim(strings.Fields(parts[1])[0], "\",")
				}
			}
		}
		return sig, nil
	}

	// AT command fallback for Qualcomm signal information
	atResp := ExecuteATCommand("AT+CSQ")
	if atResp.Success {
		fields := strings.Split(atResp.Response, ":")
		if len(fields) > 1 {
			vals := strings.Split(strings.TrimSpace(fields[1]), ",")
			if len(vals) > 0 {
				if csq, err := strconv.Atoi(strings.TrimSpace(vals[0])); err == nil && csq != 99 {
					sig.RSSI = -113 + (csq * 2)
				}
			}
		}
	}

	return sig, nil
}

func extractIntFromLine(line, key string) int {
	re := regexp.MustCompile(fmt.Sprintf(`%s[=:]\s*(-?\d+)`, key))
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		val, _ := strconv.Atoi(matches[1])
		return val
	}
	return -999
}

func (m *Manager) LoadConfig() (*BandConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	buf, err := os.ReadFile(m.dataPath)
	if err != nil && m.dataPath != "modem_config.json" {
		buf, err = os.ReadFile("modem_config.json")
		if err == nil {
			m.dataPath = "modem_config.json"
		}
	}

	if err == nil {
		buf = bytes.TrimPrefix(buf, []byte("\xef\xbb\xbf"))
		var cfg BandConfig
		if err := json.Unmarshal(buf, &cfg); err == nil {
			if cfg.LTEBands == nil {
				cfg.LTEBands = make([]int, 0)
			}
			if cfg.NRBands == nil {
				cfg.NRBands = make([]int, 0)
			}
			m.config = cfg
			return &m.config, nil
		}
	}

	return &m.config, nil
}

func (m *Manager) SaveConfig(cfg *BandConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if cfg.LTEBands == nil {
		cfg.LTEBands = make([]int, 0)
	}
	if cfg.NRBands == nil {
		cfg.NRBands = make([]int, 0)
	}

	m.config = *cfg

	buf, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(m.dataPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		m.dataPath = "modem_config.json"
	}

	if err := os.WriteFile(m.dataPath, buf, 0644); err != nil {
		if m.dataPath != "modem_config.json" {
			m.dataPath = "modem_config.json"
			_ = os.WriteFile(m.dataPath, buf, 0644)
		}
		return err
	}

	return nil
}

func (m *Manager) ApplyBandLock(cfg BandConfig) error {
	if err := m.SaveConfig(&cfg); err != nil {
		return err
	}

	hexMask := cfg.HexBitmask
	if hexMask == "" || hexMask == "0x0" {
		hexMask = BandsToHexBitmask(cfg.LTEBands)
	}

	switch cfg.Engine {
	case "qualcomm_at":
		// Format 1: Qualcomm AT+QNWPREFCFG band lock
		var lteJoin []string
		for _, b := range cfg.LTEBands {
			lteJoin = append(lteJoin, fmt.Sprintf("%d", b))
		}
		bandStr := strings.Join(lteJoin, ":")

		atCmd := fmt.Sprintf("AT+QNWPREFCFG=\"lte_band\",%s", bandStr)
		resp := ExecuteATCommand(atCmd)
		if !resp.Success {
			// Fallback Format 2: NV write / SYSCONFIG
			resp = ExecuteATCommand(fmt.Sprintf("AT^SYSCONFIG=14,2,2,4,%s", hexMask))
			if !resp.Success {
				logger.Get().Warnf("Modem", "Qualcomm AT band lock fallback attempted: %s", resp.Response)
			}
		}
		logger.Get().Infof("Modem", "Applied Qualcomm AT Band Lock (mask: %s)", hexMask)

	case "universal", "intent":
		// Android Radio CMD framework locking
		var cmds []string

		// Set preferred network mode
		ratVal := "9" // LTE / GSM / WCDMA auto
		switch cfg.PreferredRAT {
		case "5g_only":
			ratVal = "26" // NR 5G Only
		case "4g_only":
			ratVal = "11" // LTE Only
		case "3g_only":
			ratVal = "2" // WCDMA Only
		}

		cmds = append(cmds, fmt.Sprintf("cmd phone set-preferred-network-type %s 2>/dev/null || true", ratVal))

		// Apply LTE band mode bitmask if provided
		if len(cfg.LTEBands) > 0 {
			cmds = append(cmds, fmt.Sprintf("cmd phone lte-set-band-mode %s 2>/dev/null || true", hexMask))
		}

		execCmdStr := strings.Join(cmds, " && ")
		out, err := config.ExecSuTimeout(5*time.Second, execCmdStr)
		if err != nil {
			logger.Get().Warnf("Modem", "Universal band lock executed with out: %s", string(out))
		}
		logger.Get().Infof("Modem", "Applied Universal Band Lock (RAT: %s, mask: %s)", cfg.PreferredRAT, hexMask)
	}

	return nil
}

func (m *Manager) ResetBandLock() error {
	defaultCfg := BandConfig{
		Engine:       "universal",
		PreferredRAT: "hybrid",
		LTEBands:     make([]int, 0),
		NRBands:      make([]int, 0),
		HexBitmask:   "0x0",
	}

	if err := m.SaveConfig(&defaultCfg); err != nil {
		return err
	}

	// Reset AT / Qualcomm band mask
	_ = ExecuteATCommand("AT+QNWPREFCFG=\"lte_band\",1:3:5:8:40")

	// Reset Universal network type to Auto (mode 9/10)
	cmdReset := "cmd phone set-preferred-network-type 9 2>/dev/null || true"
	_, _ = config.ExecSuTimeout(5*time.Second, cmdReset)

	logger.Get().Infof("Modem", "Reset modem band locking to default settings")
	return nil
}

// Unused imports check dummy (bytes, hex, net)
var _ = bytes.TrimSpace
var _ = hex.EncodeToString
var _ = net.ParseIP
