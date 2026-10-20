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
	Preset                string        `json:"preset"` // "custom", "gaming", "fair", "saver"
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

	if err := config.WriteFileAtomic(m.dataPath, buf, 0644); err != nil {
		if m.dataPath != "qos.json" {
			m.dataPath = "qos.json"
			_ = config.WriteFileAtomic(m.dataPath, buf, 0644)
		}
		return err
	}

	return nil
}
// detectHotspotInterface returns a tether interface candidate. It deliberately
// excludes the station/WAN interfaces (wlan0, rmnet) so QoS never shapes the
// phone's own uplink instead of the clients' downlink.
func detectHotspotInterface() string {
	candidates := []string{"wlan1", "ap0", "softap0", "rndis0", "usb0", "eth0"}
	for _, iface := range candidates {
		if ifc, err := net.InterfaceByName(iface); err == nil {
			if (ifc.Flags & net.FlagUp) != 0 {
				return iface
			}
		}
	}
	return ""
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

	// 1. Clear TC qdiscs on iface
	cmdTC := fmt.Sprintf(
		"tc qdisc del dev %s root 2>/dev/null; "+
			"tc qdisc del dev %s ingress 2>/dev/null",
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
	return m.ApplyQoSOnInterface(cfg, "")
}

// ApplyQoSOnInterface applies the QoS rules to an explicit interface. When
// iface is empty the manager falls back to auto-detection.
func (m *Manager) ApplyQoSOnInterface(cfg *QoSConfig, iface string) error {
	if cfg == nil {
		return fmt.Errorf("invalid config")
	}

	if err := m.SaveConfig(cfg); err != nil {
		return err
	}

	if !cfg.Enabled {
		return m.ClearQoS()
	}

	if iface == "" {
		iface = detectHotspotInterface()
	}
	if iface == "" {
		// No tether interface is up: clear BFR-owned state and stay inactive.
		_ = m.ClearQoS()
		return fmt.Errorf("no active tether interface available for QoS")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

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
	// Clean previous rules on hotspot interface
	cmdClean := fmt.Sprintf(
		"tc qdisc del dev %s root 2>/dev/null; "+
			"tc qdisc del dev %s ingress 2>/dev/null",
		iface, iface,
	)
	_, _ = config.ExecSuTimeout(5, cmdClean)

	var cmds []string

	// 1. Egress HTB on hotspot iface: regulates DOWNLOAD to connected clients
	globalDownMbps := cfg.GlobalDownload
	if globalDownMbps <= 0 {
		globalDownMbps = 1000 // Default 1Gbps unthrottled root
	}
	cmds = append(cmds, fmt.Sprintf("tc qdisc add dev %s root handle 1: htb default 30", iface))
	cmds = append(cmds, fmt.Sprintf("tc class add dev %s parent 1: classid 1:1 htb rate %dmbit ceil %dmbit", iface, globalDownMbps, globalDownMbps))

	// Class 1:10 (High Priority: Gaming & Real-time ICMP/DNS)
	prioRate := globalDownMbps / 3
	if prioRate < 5 {
		prioRate = 5
	}
	cmds = append(cmds, fmt.Sprintf("tc class add dev %s parent 1:1 classid 1:10 htb rate %dmbit ceil %dmbit prio 1", iface, prioRate, globalDownMbps))

	// Class 1:30 (Default traffic)
	defaultRate := globalDownMbps / 2
	if defaultRate < 2 {
		defaultRate = 2
	}
	cmds = append(cmds, fmt.Sprintf("tc class add dev %s parent 1:1 classid 1:30 htb rate %dmbit ceil %dmbit prio 3", iface, defaultRate, globalDownMbps))
	*rulesCount += 3

	// Priority packet filters for Gaming & Low-Latency
	if cfg.PrioritizeGaming {
		cmds = append(cmds,
			// ICMP (Ping)
			fmt.Sprintf("tc filter add dev %s protocol ip parent 1:0 prio 1 u32 match ip protocol 1 0xff flowid 1:10", iface),
			// DNS (UDP port 53)
			fmt.Sprintf("tc filter add dev %s protocol ip parent 1:0 prio 1 u32 match ip protocol 17 0xff match ip sport 53 0xffff flowid 1:10", iface),
			// Mobile & PC Gaming ports
			fmt.Sprintf("tc filter add dev %s protocol ip parent 1:0 prio 1 u32 match ip protocol 17 0xff match ip dport 27015 0xffff flowid 1:10", iface),
			fmt.Sprintf("tc filter add dev %s protocol ip parent 1:0 prio 1 u32 match ip protocol 17 0xff match ip dport 3478 0xffff flowid 1:10", iface),
		)
		*rulesCount += 4
	}

	if cfg.PrioritizeVoip {
		cmds = append(cmds,
			// VoIP & Discord / WhatsApp SIP
			fmt.Sprintf("tc filter add dev %s protocol ip parent 1:0 prio 1 u32 match ip protocol 17 0xff match ip dport 5060 0xffff flowid 1:10", iface),
		)
		*rulesCount++
	}

	// 2. Ingress Qdisc on hotspot iface: regulates UPLOAD from connected clients via police
	cmds = append(cmds, fmt.Sprintf("tc qdisc add dev %s handle ffff: ingress", iface))
	*rulesCount++

	globalUpMbps := cfg.GlobalUpload
	if globalUpMbps > 0 {
		cmds = append(cmds, fmt.Sprintf(
			"tc filter add dev %s parent ffff: protocol ip prio 100 u32 match u32 0 0 action police rate %dmbit burst 200k drop",
			iface, globalUpMbps,
		))
		*rulesCount++
	}

	// 3. Per-Client Rate Limits
	classID := 100
	for _, client := range cfg.ClientLimits {
		if !client.Enabled || client.IP == "" || net.ParseIP(client.IP) == nil {
			continue
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

		// Download limit for client (Egress)
		cmds = append(cmds, fmt.Sprintf("tc class add dev %s parent 1:1 classid 1:%d htb rate %dmbit ceil %dmbit prio %s", iface, classID, downLimit, downLimit, prio))
		cmds = append(cmds, fmt.Sprintf("tc filter add dev %s protocol ip parent 1:0 prio 2 u32 match ip dst %s/32 flowid 1:%d", iface, client.IP, classID))
		*rulesCount += 2

		// Upload limit for client (Ingress Police)
		if client.UploadLimit > 0 {
			cmds = append(cmds, fmt.Sprintf(
				"tc filter add dev %s parent ffff: protocol ip prio 10 u32 match ip src %s/32 action police rate %dmbit burst 50k drop",
				iface, client.IP, client.UploadLimit,
			))
			*rulesCount++
		}

		classID++
	}

	execCmdStr := strings.Join(cmds, " && ")
	out, err := config.ExecSuTimeout(10, execCmdStr)
	if err != nil {
		return fmt.Errorf("tc execution failed: %v, output: %s", err, string(out))
	}

	return nil
}

func (m *Manager) applyIPTablesMangle(iface string, cfg *QoSConfig, rulesCount *int) error {
	cmds := []string{
		"iptables -t mangle -D FORWARD -j BFR_QOS 2>/dev/null || true",
		"iptables -t mangle -F BFR_QOS 2>/dev/null || true",
		"iptables -t mangle -X BFR_QOS 2>/dev/null || true",
		"iptables -t mangle -N BFR_QOS",
		"iptables -t mangle -A FORWARD -j BFR_QOS",
	}

	// MARK packet classification for low latency (Gaming & VoIP)
	if cfg.PrioritizeGaming {
		cmds = append(cmds,
			"iptables -t mangle -A BFR_QOS -p udp --dport 53 -j MARK --set-mark 1",
			"iptables -t mangle -A BFR_QOS -p icmp -j MARK --set-mark 1",
			"iptables -t mangle -A BFR_QOS -p udp --dport 27005:27015 -j MARK --set-mark 1",
			"iptables -t mangle -A BFR_QOS -p udp --dport 3478:3479 -j MARK --set-mark 1",
		)
		*rulesCount += 4
	}
	if cfg.PrioritizeVoip {
		cmds = append(cmds,
			"iptables -t mangle -A BFR_QOS -p udp --dport 5060:5061 -j MARK --set-mark 2",
			"iptables -t mangle -A BFR_QOS -p udp --dport 5000:5500 -j MARK --set-mark 2",
		)
		*rulesCount += 2
	}

	// Global bandwidth shaping using hashlimit
	if cfg.GlobalDownload > 0 {
		cmds = append(cmds, fmt.Sprintf(
			"iptables -t mangle -A BFR_QOS -o %s -m hashlimit --hashlimit-name qos_g_down --hashlimit-above %dmbit/sec -j DROP",
			iface, cfg.GlobalDownload,
		))
		*rulesCount++
	}
	if cfg.GlobalUpload > 0 {
		cmds = append(cmds, fmt.Sprintf(
			"iptables -t mangle -A BFR_QOS -i %s -m hashlimit --hashlimit-name qos_g_up --hashlimit-above %dmbit/sec -j DROP",
			iface, cfg.GlobalUpload,
		))
		*rulesCount++
	}

	// Per-client rate limit rules using hashlimit match
	for _, client := range cfg.ClientLimits {
		if !client.Enabled || client.IP == "" || net.ParseIP(client.IP) == nil {
			continue
		}

		ipClean := strings.ReplaceAll(client.IP, ".", "_")
		if client.DownloadLimit > 0 {
			cmds = append(cmds, fmt.Sprintf(
				"iptables -t mangle -A BFR_QOS -d %s -m hashlimit --hashlimit-name qos_d_%s --hashlimit-above %dmbit/sec -j DROP",
				client.IP, ipClean, client.DownloadLimit,
			))
			*rulesCount++
		}

		if client.UploadLimit > 0 {
			cmds = append(cmds, fmt.Sprintf(
				"iptables -t mangle -A BFR_QOS -s %s -m hashlimit --hashlimit-name qos_u_%s --hashlimit-above %dmbit/sec -j DROP",
				client.IP, ipClean, client.UploadLimit,
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

// OnTetheringChanged handles interface state changes from the hotspot/tethering
// engine. The interface reported by the tethering layer is authoritative.
func (m *Manager) OnTetheringChanged(activeIface string) {
	m.mu.RLock()
	cfg := m.config
	previousIface := m.detectedIface
	m.mu.RUnlock()

	if !cfg.Enabled {
		return
	}

	// No tether interface is up: remove BFR-owned rules and mark inactive.
	if activeIface == "" {
		if previousIface != "" {
			logger.Get().Infof("QoS", "No tether interface active, clearing QoS rules from %s", previousIface)
		}
		_ = m.ClearQoS()
		return
	}

	// Interface changed: clear the old interface before installing on the new one.
	if previousIface != "" && previousIface != activeIface {
		logger.Get().Infof("QoS", "Tethering interface changed %s -> %s, re-applying QoS rules", previousIface, activeIface)
		_ = m.ClearQoS()
	} else {
		logger.Get().Infof("QoS", "Tethering interface %s active, re-applying QoS rules", activeIface)
	}

	_ = m.ApplyQoSOnInterface(&cfg, activeIface)
}
