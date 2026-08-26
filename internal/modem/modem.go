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
	EARFCN      int    `json:"earfcn"`
	Bandwidth   string `json:"bandwidth"`
	QualityRSRP string `json:"quality_rsrp"`
	QualityRSRQ string `json:"quality_rsrq"`
	QualitySINR string `json:"quality_sinr"`
	RSRPPct     int    `json:"rsrp_pct"`
	RSRQPct     int    `json:"rsrq_pct"`
	SINRPct     int    `json:"sinr_pct"`
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

// CalculateSignalMetrics populates signal percentages and quality rating strings.
func CalculateSignalMetrics(sig *SignalInfo) {
	if sig.RSRP != -999 {
		if sig.RSRP >= -85 {
			sig.QualityRSRP = "Excellent"
			sig.RSRPPct = 100
		} else if sig.RSRP >= -95 {
			sig.QualityRSRP = "Good"
			sig.RSRPPct = 75
		} else if sig.RSRP >= -105 {
			sig.QualityRSRP = "Fair"
			sig.RSRPPct = 50
		} else {
			sig.QualityRSRP = "Poor"
			sig.RSRPPct = 25
		}
	} else {
		sig.QualityRSRP = "Unknown"
		sig.RSRPPct = 0
	}

	if sig.RSRQ != -999 {
		if sig.RSRQ >= -10 {
			sig.QualityRSRQ = "Excellent"
			sig.RSRQPct = 100
		} else if sig.RSRQ >= -15 {
			sig.QualityRSRQ = "Good"
			sig.RSRQPct = 75
		} else if sig.RSRQ >= -19 {
			sig.QualityRSRQ = "Fair"
			sig.RSRQPct = 50
		} else {
			sig.QualityRSRQ = "Poor"
			sig.RSRQPct = 25
		}
	} else {
		sig.QualityRSRQ = "Unknown"
		sig.RSRQPct = 0
	}

	if sig.SINR != -999 {
		if sig.SINR >= 20 {
			sig.QualitySINR = "High Speed"
			sig.SINRPct = 100
		} else if sig.SINR >= 13 {
			sig.QualitySINR = "Good"
			sig.SINRPct = 75
		} else if sig.SINR >= 0 {
			sig.QualitySINR = "Fair"
			sig.SINRPct = 50
		} else {
			sig.QualitySINR = "Poor"
			sig.SINRPct = 25
		}
	} else {
		sig.QualitySINR = "Unknown"
		sig.SINRPct = 0
	}
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
		Bandwidth:   "Unknown",
	}

	out, err := exec.Command(config.SUBin, "-c", "dumpsys telephony.registry 2>/dev/null | grep -E 'mSignalStrength|mServiceState|mCellIdentity|mOperatorAlpha|mAlphaLong|mAlphaShort|mPci|mCi|mTac|mEarfcn|mLteBandwidth' | head -n 60").Output()
	if err == nil && len(out) > 0 {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)

			// Signal metrics
			if strings.Contains(line, "rsrp=") || strings.Contains(line, "rsrp:") {
				if val := extractIntFromLine(line, "rsrp"); val != -999 && val != 2147483647 {
					sig.RSRP = val
				}
			}
			if strings.Contains(line, "rsrq=") || strings.Contains(line, "rsrq:") {
				if val := extractIntFromLine(line, "rsrq"); val != -999 && val != 2147483647 {
					sig.RSRQ = val
				}
			}
			if strings.Contains(line, "rssi=") || strings.Contains(line, "rssi:") {
				if val := extractIntFromLine(line, "rssi"); val != -999 && val != 2147483647 {
					sig.RSSI = val
				}
			}
			if strings.Contains(line, "rssnr=") || strings.Contains(line, "snr=") || strings.Contains(line, "sinr=") {
				if val := extractIntFromLine(line, "rssnr"); val != -999 && val != 2147483647 {
					sig.SINR = val
				} else if val := extractIntFromLine(line, "snr"); val != -999 && val != 2147483647 {
					sig.SINR = val
				} else if val := extractIntFromLine(line, "sinr"); val != -999 && val != 2147483647 {
					sig.SINR = val
				}
			}

			// Operator name parsing
			if strings.Contains(line, "mOperatorAlphaLong=") || strings.Contains(line, "mAlphaLong=") {
				op := extractStringValue(line)
				if op != "" && op != "null" && op != "Unknown" {
					sig.Operator = op
				}
			} else if sig.Operator == "Unknown" && (strings.Contains(line, "mOperatorAlphaShort=") || strings.Contains(line, "mAlphaShort=")) {
				op := extractStringValue(line)
				if op != "" && op != "null" && op != "Unknown" {
					sig.Operator = op
				}
			}

			// Cell details
			if strings.Contains(line, "mCi=") || strings.Contains(line, "cellId=") {
				if ci := extractIntFromLine(line, "mCi"); ci != -999 && ci != 2147483647 {
					sig.CellID = fmt.Sprintf("%d", ci)
				} else if ci := extractIntFromLine(line, "cellId"); ci != -999 && ci != 2147483647 {
					sig.CellID = fmt.Sprintf("%d", ci)
				}
			}
			if strings.Contains(line, "mPci=") || strings.Contains(line, "pci=") {
				if pci := extractIntFromLine(line, "mPci"); pci != -999 && pci != 2147483647 {
					sig.PCI = fmt.Sprintf("%d", pci)
				} else if pci := extractIntFromLine(line, "pci"); pci != -999 && pci != 2147483647 {
					sig.PCI = fmt.Sprintf("%d", pci)
				}
			}
			if strings.Contains(line, "mTac=") || strings.Contains(line, "tac=") {
				if tac := extractIntFromLine(line, "mTac"); tac != -999 && tac != 2147483647 {
					sig.TAC = fmt.Sprintf("%d", tac)
				} else if tac := extractIntFromLine(line, "tac"); tac != -999 && tac != 2147483647 {
					sig.TAC = fmt.Sprintf("%d", tac)
				}
			}
			if strings.Contains(line, "mEarfcn=") || strings.Contains(line, "earfcn=") {
				if earfcn := extractIntFromLine(line, "mEarfcn"); earfcn != -999 && earfcn != 2147483647 {
					sig.EARFCN = earfcn
				} else if earfcn := extractIntFromLine(line, "earfcn"); earfcn != -999 && earfcn != 2147483647 {
					sig.EARFCN = earfcn
				}
			}
			if strings.Contains(line, "mLteBandwidth=") || strings.Contains(line, "mBandwidth=") {
				if bw := extractIntFromLine(line, "mLteBandwidth"); bw != -999 && bw > 0 {
					sig.Bandwidth = fmt.Sprintf("%d MHz", bw/1000)
				} else if bw := extractIntFromLine(line, "mBandwidth"); bw != -999 && bw > 0 {
					sig.Bandwidth = fmt.Sprintf("%d MHz", bw)
				}
			}
			if strings.Contains(line, "mBand=") || strings.Contains(line, "mBands=") {
				parts := strings.Split(line, "=")
				if len(parts) > 1 {
					bStr := strings.Trim(strings.Fields(parts[1])[0], "\",[]")
					if bStr != "" && bStr != "null" && bStr != "0" {
						sig.Band = "B" + bStr
					}
				}
			}
		}
	}

	// AT command fallback for Operator & Signal Info if unknown
	if sig.Operator == "Unknown" || sig.RSRP == -999 {
		if cops := ExecuteATCommand("AT+COPS?"); cops.Success {
			re := regexp.MustCompile(`"([^"]+)"`)
			matches := re.FindStringSubmatch(cops.Response)
			if len(matches) > 1 && matches[1] != "" {
				sig.Operator = matches[1]
			}
		}

		if qeng := ExecuteATCommand("AT+QENG=\"servingcell\""); qeng.Success {
			// Parse Qualcomm QENG: +QENG: "servingcell","NOCONN","LTE","FDD",510,11,123456,142,1800,3,5,5,1A,-85,-10,-60,15,22
			parts := strings.Split(qeng.Response, ",")
			if len(parts) >= 15 {
				if pci, err := strconv.Atoi(strings.TrimSpace(parts[7])); err == nil {
					sig.PCI = fmt.Sprintf("%d", pci)
				}
				if earfcn, err := strconv.Atoi(strings.TrimSpace(parts[8])); err == nil {
					sig.EARFCN = earfcn
				}
				if rsrp, err := strconv.Atoi(strings.TrimSpace(parts[13])); err == nil {
					sig.RSRP = rsrp
				}
				if rsrq, err := strconv.Atoi(strings.TrimSpace(parts[14])); err == nil {
					sig.RSRQ = rsrq
				}
				if len(parts) >= 17 {
					if sinr, err := strconv.Atoi(strings.TrimSpace(parts[16])); err == nil {
						sig.SINR = sinr
					}
				}
			}
		}

		if sig.RSRP == -999 {
			if csq := ExecuteATCommand("AT+CSQ"); csq.Success {
				fields := strings.Split(csq.Response, ":")
				if len(fields) > 1 {
					vals := strings.Split(strings.TrimSpace(fields[1]), ",")
					if len(vals) > 0 {
						if val, err := strconv.Atoi(strings.TrimSpace(vals[0])); err == nil && val != 99 {
							sig.RSSI = -113 + (val * 2)
							sig.RSRP = sig.RSSI - 15
						}
					}
				}
			}
		}
	}

	CalculateSignalMetrics(&sig)
	return sig, nil
}

func extractStringValue(line string) string {
	parts := strings.Split(line, "=")
	if len(parts) > 1 {
		val := strings.Trim(strings.Fields(parts[1])[0], "\",;")
		val = strings.TrimSpace(val)
		if val != "" && val != "null" && val != "Unknown" {
			return val
		}
	}
	return ""
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

	if err := config.WriteFileAtomic(m.dataPath, buf, 0644); err != nil {
		if m.dataPath != "modem_config.json" {
			m.dataPath = "modem_config.json"
			_ = config.WriteFileAtomic(m.dataPath, buf, 0644)
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

	var lteJoin []string
	for _, b := range cfg.LTEBands {
		lteJoin = append(lteJoin, fmt.Sprintf("%d", b))
	}
	bandStr := strings.Join(lteJoin, ":")

	// Attempt Qualcomm AT Command direct band lock
	if bandStr != "" {
		_ = ExecuteATCommand(fmt.Sprintf("AT+QNWPREFCFG=\"lte_band\",%s", bandStr))
		_ = ExecuteATCommand(fmt.Sprintf("AT^SYSCONFIG=14,2,2,4,%s", hexMask))
	}

	// Determine Android radio network mode value
	ratVal := "9" // Hybrid/Auto default
	switch cfg.PreferredRAT {
	case "5g_only":
		ratVal = "26" // NR 5G Only
	case "4g_only", "lte_only":
		ratVal = "11" // LTE Only
	case "3g_only":
		ratVal = "2" // WCDMA Only
	}

	// Apply settings & network modes (supports Android 9 through Android 16)
	var cmds []string
	cmds = append(cmds, fmt.Sprintf("settings put global preferred_network_mode %s 2>/dev/null || true", ratVal))
	cmds = append(cmds, fmt.Sprintf("settings put global preferred_network_mode1 %s 2>/dev/null || true", ratVal))
	cmds = append(cmds, fmt.Sprintf("settings put global preferred_network_mode2 %s 2>/dev/null || true", ratVal))
	cmds = append(cmds, fmt.Sprintf("cmd phone set-preferred-network-type %s 2>/dev/null || true", ratVal))

	if len(cfg.LTEBands) > 0 {
		cmds = append(cmds, fmt.Sprintf("cmd phone lte-set-band-mode %s 2>/dev/null || true", hexMask))
	}

	// Airplane Mode Cycle to force radio stack to tear down and re-register on target band
	cmds = append(cmds, "settings put global airplane_mode_on 1 && am broadcast -a android.intent.action.AIRPLANE_MODE --ez state true >/dev/null 2>&1")
	cmds = append(cmds, "sleep 1")
	cmds = append(cmds, "settings put global airplane_mode_on 0 && am broadcast -a android.intent.action.AIRPLANE_MODE --ez state false >/dev/null 2>&1")

	execCmdStr := strings.Join(cmds, " && ")
	out, err := config.ExecSuTimeout(8*time.Second, execCmdStr)
	if err != nil {
		logger.Get().Warnf("Modem", "Band lock execution notice (out: %s)", string(out))
	}
	logger.Get().Infof("Modem", "Applied Band Lock (RAT: %s, Bands: %s, mask: %s)", cfg.PreferredRAT, bandStr, hexMask)

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
