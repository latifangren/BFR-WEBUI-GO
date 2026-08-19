package qos

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

type ClientLimit struct {
	IP            string `json:"ip"`
	MAC           string `json:"mac"`
	Hostname      string `json:"hostname"`
	DownloadLimit int    `json:"download_limit"` // in Mbps
	UploadLimit   int    `json:"upload_limit"`   // in Mbps
	Priority      string `json:"priority"`       // "gaming", "stream", "browsing", "strict", "custom"
	Enabled       bool   `json:"enabled"`
}

type QoSConfig struct {
	Enabled               bool          `json:"enabled"`
	Engine                string        `json:"engine"` // "auto", "tc", "iptables"
	GlobalDownload        int           `json:"global_download"`
	GlobalUpload          int           `json:"global_upload"`
	PrioritizeGaming      bool          `json:"prioritize_gaming"`
	PrioritizeVoip        bool          `json:"prioritize_voip"`
	ScheduledLimitEnabled bool          `json:"scheduled_limit_enabled"`
	ScheduleStart         string        `json:"schedule_start"`
	ScheduleEnd           string        `json:"schedule_end"`
	ClientLimits          []ClientLimit `json:"client_limits"`
}

type QoSStatus struct {
	Active           bool   `json:"active"`
	EngineUsed       string `json:"engine_used"` // "tc_htb", "iptables_mangle", "none"
	ActiveRulesCount int    `json:"active_rules_count"`
	DetectedIface    string `json:"detected_iface"`
}

type Manager struct {
	mu               sync.RWMutex
	dataPath         string
	config           QoSConfig
	active           bool
	engineUsed       string
	activeRulesCount int
	detectedIface    string
}

var (
	globalManager *Manager
	once          sync.Once
)

func getStoragePath() string {
	return config.GetPersistentFilePath("qos.json")
}

func newManager() *Manager {
	m := &Manager{
		dataPath: getStoragePath(),
		config: QoSConfig{
			Enabled:        false,
			Engine:         "auto",
			GlobalDownload: 0,
			GlobalUpload:   0,
			ClientLimits:   make([]ClientLimit, 0),
		},
		engineUsed: "none",
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

func (m *Manager) LoadConfig() (*QoSConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	buf, err := os.ReadFile(m.dataPath)
	if err != nil && m.dataPath != "qos.json" {
		buf, err = os.ReadFile("qos.json")
		if err == nil {
			m.dataPath = "qos.json"
		}
	}

	if err == nil {
		buf = bytes.TrimPrefix(buf, []byte("\xef\xbb\xbf"))
		var cfg QoSConfig
		if err := json.Unmarshal(buf, &cfg); err == nil {
			if cfg.ClientLimits == nil {
				cfg.ClientLimits = make([]ClientLimit, 0)
			}
			if cfg.Engine == "" {
				cfg.Engine = "auto"
			}
			m.config = cfg
			return &m.config, nil
		}
	}

	return &m.config, nil
}

func (m *Manager) SaveConfig(cfg *QoSConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if cfg.ClientLimits == nil {
		cfg.ClientLimits = make([]ClientLimit, 0)
	}

	m.config = *cfg

	buf, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(m.dataPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		m.dataPath = "qos.json"
	}

	if err := os.WriteFile(m.dataPath, buf, 0644); err != nil {
		if m.dataPath != "qos.json" {
			m.dataPath = "qos.json"
			_ = os.WriteFile(m.dataPath, buf, 0644)
		}
		return err
	}

	return nil
}

func detectHotspotInterface() string {
	candidates := []string{"wlan1", "ap0", "wlan0", "rndis0"}
	for _, iface := range candidates {
		if _, err := net.InterfaceByName(iface); err == nil {
			return iface
		}
	}
	return "wlan1"
}

func hasTCCmd() bool {
	if _, err := exec.LookPath("tc"); err == nil {
		return true
	}
	out, err := exec.Command(config.SUBin, "-c", "which tc || type tc").Output()
	return err == nil && len(bytes.TrimSpace(out)) > 0
}

func (m *Manager) ClearQoS() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	iface := m.detectedIface
	if iface == "" {
		iface = detectHotspotInterface()
	}

	// 1. Clear TC qdiscs on iface and ifb0
	cmdTC := fmt.Sprintf(
		"tc qdisc del dev %s root 2>/dev/null; "+
			"tc qdisc del dev %s ingress 2>/dev/null; "+
			"tc qdisc del dev ifb0 root 2>/dev/null; "+
			"ip link set dev ifb0 down 2>/dev/null",
		iface, iface,
	)
	_, _ = config.ExecSuTimeout(5, cmdTC)

	// 2. Clear iptables BFR_QOS mangle chains
	cmdIPTables := "iptables -t mangle -D FORWARD -j BFR_QOS 2>/dev/null; " +
		"iptables -t mangle -F BFR_QOS 2>/dev/null; " +
		"iptables -t mangle -X BFR_QOS 2>/dev/null"
	_, _ = config.ExecSuTimeout(5, cmdIPTables)

	m.active = false
	m.engineUsed = "none"
	m.activeRulesCount = 0
	logger.Get().Infof("QoS", "Cleared QoS rules on %s", iface)

	return nil
}

func (m *Manager) ApplyQoS(cfg *QoSConfig) error {
	if cfg == nil {
		return fmt.Errorf("invalid config")
	}

	if err := m.SaveConfig(cfg); err != nil {
		return err
	}

	if !cfg.Enabled {
		return m.ClearQoS()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	iface := detectHotspotInterface()
	m.detectedIface = iface
	rulesCount := 0

	useTC := false
	if cfg.Engine == "tc" || (cfg.Engine == "auto" && hasTCCmd()) {
		useTC = true
	}

	if useTC {
		err := m.applyTCHTB(iface, cfg, &rulesCount)
		if err == nil {
			m.active = true
			m.engineUsed = "tc_htb"
			m.activeRulesCount = rulesCount
			logger.Get().Infof("QoS", "Applied TC HTB QoS engine on %s (%d rules)", iface, rulesCount)
			return nil
		}
		logger.Get().Warnf("QoS", "TC HTB engine failed (%v), falling back to iptables mangle engine", err)
	}

	// Fallback or explicit iptables mangle engine
	err := m.applyIPTablesMangle(iface, cfg, &rulesCount)
	if err != nil {
		m.active = false
		m.engineUsed = "none"
		m.activeRulesCount = 0
		return fmt.Errorf("failed to apply QoS engine: %w", err)
	}

	m.active = true
	m.engineUsed = "iptables_mangle"
	m.activeRulesCount = rulesCount
	logger.Get().Infof("QoS", "Applied iptables mangle QoS engine on %s (%d rules)", iface, rulesCount)
	return nil
}

func (m *Manager) applyTCHTB(iface string, cfg *QoSConfig, rulesCount *int) error {
	// Clean previous rules
	cmdClean := fmt.Sprintf(
		"tc qdisc del dev %s root 2>/dev/null; "+
			"tc qdisc del dev %s ingress 2>/dev/null; "+
			"tc qdisc del dev ifb0 root 2>/dev/null",
		iface, iface,
	)
	_, _ = config.ExecSuTimeout(5, cmdClean)

	var cmds []string

	// Root HTB Qdisc for egress (Upload)
	globalUpMbps := cfg.GlobalUpload
	if globalUpMbps <= 0 {
		globalUpMbps = 1000 // Default 1Gbps unthrottled root
	}
	cmds = append(cmds, fmt.Sprintf("tc qdisc add dev %s root handle 1: htb default 30", iface))
	cmds = append(cmds, fmt.Sprintf("tc class add dev %s parent 1: classid 1:1 htb rate %dmbit ceil %dmbit", iface, globalUpMbps, globalUpMbps))
	cmds = append(cmds, fmt.Sprintf("tc class add dev %s parent 1:1 classid 1:30 htb rate %dmbit ceil %dmbit prio 3", iface, globalUpMbps/2+1, globalUpMbps))
	*rulesCount++

	// Setup IFB redirection for ingress (Download)
	cmds = append(cmds,
		"ip link add name ifb0 type ifb 2>/dev/null || true",
		"ip link set dev ifb0 up",
		fmt.Sprintf("tc qdisc add dev %s handle ffff: ingress 2>/dev/null || true", iface),
		fmt.Sprintf("tc filter add dev %s parent ffff: protocol ip u32 match u32 0 0 action mirred egress redirect dev ifb0 2>/dev/null || true", iface),
	)

	globalDownMbps := cfg.GlobalDownload
	if globalDownMbps <= 0 {
		globalDownMbps = 1000
	}
	cmds = append(cmds,
		"tc qdisc add dev ifb0 root handle 1: htb default 30",
		fmt.Sprintf("tc class add dev ifb0 parent 1: classid 1:1 htb rate %dmbit ceil %dmbit", globalDownMbps, globalDownMbps),
		fmt.Sprintf("tc class add dev ifb0 parent 1:1 classid 1:30 htb rate %dmbit ceil %dmbit prio 3", globalDownMbps/2+1, globalDownMbps),
	)

	// Apply Per-Client HTB Rules
	classID := 100
	for _, client := range cfg.ClientLimits {
		if !client.Enabled || client.IP == "" {
			continue
		}

		if net.ParseIP(client.IP) == nil {
			continue
		}

		upLimit := client.UploadLimit
		if upLimit <= 0 {
			upLimit = globalUpMbps
		}

		downLimit := client.DownloadLimit
		if downLimit <= 0 {
			downLimit = globalDownMbps
		}

		prio := "3"
		switch client.Priority {
		case "gaming":
			prio = "1"
		case "voip", "stream":
			prio = "2"
		case "strict":
			prio = "5"
		}

		// Upload class + filter
		cmds = append(cmds, fmt.Sprintf("tc class add dev %s parent 1:1 classid 1:%d htb rate %dmbit ceil %dmbit prio %s", iface, classID, upLimit, upLimit, prio))
		cmds = append(cmds, fmt.Sprintf("tc filter add dev %s protocol ip parent 1:0 prio 1 u32 match ip src %s/32 flowid 1:%d", iface, client.IP, classID))

		// Download class + filter on ifb0
		cmds = append(cmds, fmt.Sprintf("tc class add dev ifb0 parent 1:1 classid 1:%d htb rate %dmbit ceil %dmbit prio %s", classID, downLimit, downLimit, prio))
		cmds = append(cmds, fmt.Sprintf("tc filter add dev ifb0 protocol ip parent 1:0 prio 1 u32 match ip dst %s/32 flowid 1:%d", client.IP, classID))

		classID++
		*rulesCount += 2
	}

	// Priority TOS/DSCP tagging for Gaming & VoIP
	if cfg.PrioritizeGaming {
		cmds = append(cmds, fmt.Sprintf("tc filter add dev %s protocol ip parent 1:0 prio 1 u32 match ip protocol 17 0xff match ip dport 27015 0xffff flowid 1:1", iface))
		*rulesCount++
	}

	execCmdStr := strings.Join(cmds, " && ")
	out, err := config.ExecSuTimeout(10, execCmdStr)
	if err != nil {
		return fmt.Errorf("tc execution failed: %v, output: %s", err, string(out))
	}

	return nil
}

func (m *Manager) applyIPTablesMangle(iface string, cfg *QoSConfig, rulesCount *int) error {
	// Setup BFR_QOS chain in mangle table
	cmds := []string{
		"iptables -t mangle -D FORWARD -j BFR_QOS 2>/dev/null || true",
		"iptables -t mangle -F BFR_QOS 2>/dev/null || true",
		"iptables -t mangle -X BFR_QOS 2>/dev/null || true",
		"iptables -t mangle -N BFR_QOS",
		"iptables -t mangle -A FORWARD -j BFR_QOS",
	}

	// TOS / DSCP priority marking for Gaming & VoIP
	if cfg.PrioritizeGaming {
		cmds = append(cmds,
			"iptables -t mangle -A BFR_QOS -p udp --dport 27005:27015 -j TOS --set-tos 0x10",
			"iptables -t mangle -A BFR_QOS -p udp --dport 3478:3479 -j TOS --set-tos 0x10",
		)
		*rulesCount += 2
	}
	if cfg.PrioritizeVoip {
		cmds = append(cmds,
			"iptables -t mangle -A BFR_QOS -p udp --dport 5060:5061 -j DSCP --set-dscp-class EF",
		)
		*rulesCount++
	}

	// Per-client rate limit rules using hashlimit / limit match
	for _, client := range cfg.ClientLimits {
		if !client.Enabled || client.IP == "" {
			continue
		}
		if net.ParseIP(client.IP) == nil {
			continue
		}

		if client.DownloadLimit > 0 {
			cmds = append(cmds, fmt.Sprintf(
				"iptables -t mangle -A BFR_QOS -d %s -m hashlimit --hashlimit-name qos_down_%s --hashlimit-above %dmbit/sec -j DROP",
				client.IP, strings.ReplaceAll(client.IP, ".", "_"), client.DownloadLimit,
			))
			*rulesCount++
		}

		if client.UploadLimit > 0 {
			cmds = append(cmds, fmt.Sprintf(
				"iptables -t mangle -A BFR_QOS -s %s -m hashlimit --hashlimit-name qos_up_%s --hashlimit-above %dmbit/sec -j DROP",
				client.IP, strings.ReplaceAll(client.IP, ".", "_"), client.UploadLimit,
			))
			*rulesCount++
		}
	}

	execCmdStr := strings.Join(cmds, " && ")
	out, err := config.ExecSuTimeout(10, execCmdStr)
	if err != nil {
		return fmt.Errorf("iptables execution failed: %v, output: %s", err, string(out))
	}

	return nil
}

func (m *Manager) GetStatus() QoSStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return QoSStatus{
		Active:           m.active,
		EngineUsed:       m.engineUsed,
		ActiveRulesCount: m.activeRulesCount,
		DetectedIface:    m.detectedIface,
	}
}
