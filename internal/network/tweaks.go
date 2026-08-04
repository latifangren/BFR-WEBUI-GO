package network

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

var (
	// H-1 & N-3 validation regexes
	reSysctlKey = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)
	reSysctlVal = regexp.MustCompile(`^[a-zA-Z0-9_. -]+$`)
	reIfaceName = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

	sysctlDefaultsMu sync.Mutex
	sysctlDefaults   map[string]string
)

var defaultTunedKeys = []string{
	"net.core.rmem_max",
	"net.core.wmem_max",
	"net.core.rmem_default",
	"net.core.wmem_default",
	"net.core.somaxconn",
	"net.ipv4.tcp_rmem",
	"net.ipv4.tcp_wmem",
	"net.ipv4.tcp_tw_reuse",
	"net.ipv4.tcp_fin_timeout",
	"net.ipv4.tcp_max_syn_backlog",
	"net.ipv4.tcp_keepalive_time",
	"net.ipv4.tcp_keepalive_intvl",
	"net.ipv4.tcp_keepalive_probes",
	"net.ipv4.tcp_timestamps",
	"net.ipv4.tcp_sack",
	"net.ipv4.tcp_window_scaling",
	"net.core.default_qdisc",
	"net.ipv4.tcp_congestion_control",
}

func BackupSysctlDefaults() {
	sysctlDefaultsMu.Lock()
	defer sysctlDefaultsMu.Unlock()
	if sysctlDefaults != nil {
		return
	}
	sysctlDefaults = make(map[string]string)
	for _, key := range defaultTunedKeys {
		if val, err := GetSysctl(key); err == nil && val != "" {
			sysctlDefaults[key] = val
		}
	}
}

func RestoreSysctlDefaults() error {
	sysctlDefaultsMu.Lock()
	defer sysctlDefaultsMu.Unlock()
	if len(sysctlDefaults) == 0 {
		return fmt.Errorf("no sysctl defaults snapshot available")
	}
	var errs []string
	for key, val := range sysctlDefaults {
		if err := SetSysctl(key, val); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", key, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to restore sysctl defaults: %s", strings.Join(errs, "; "))
	}

	// Persistently save disabled/default state to tweaks.json
	_ = SaveTweaks(TweaksConfig{})

	logger.Get().Infof("network", "Restored initial sysctl defaults successfully")
	return nil
}

type SysctlTweak struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type InterfaceInfo struct {
	Name       string   `json:"name"`
	IPs        []string `json:"ips"`
	MTU        int      `json:"mtu"`
	TxQueueLen int      `json:"txqueuelen"`
	RxBytes    uint64   `json:"rx_bytes"`
	TxBytes    uint64   `json:"tx_bytes"`
	RxPackets  uint64   `json:"rx_packets"`
	TxPackets  uint64   `json:"tx_packets"`
}

type DNSServer struct {
	Name      string `json:"name"`
	Primary   string `json:"primary"`
	Secondary string `json:"secondary"`
}

var PresetDNS = []DNSServer{
	{Name: "Cloudflare", Primary: "1.1.1.1", Secondary: "1.0.0.1"},
	{Name: "Google", Primary: "8.8.8.8", Secondary: "8.8.4.4"},
	{Name: "AdGuard", Primary: "94.140.14.14", Secondary: "94.140.15.15"},
	{Name: "Quad9", Primary: "9.9.9.9", Secondary: "149.112.112.112"},
}

func GetSysctl(key string) (string, error) {
	// H-1: validate key before shell command
	if !reSysctlKey.MatchString(key) {
		return "", fmt.Errorf("invalid sysctl key")
	}
	out, err := exec.Command(config.SUBin, "-c", "sysctl -n "+key).Output()
	if err != nil {
		out, err = exec.Command("sysctl", "-n", key).Output()
		if err != nil {
			return "", err
		}
	}
	return strings.TrimSpace(string(out)), nil
}

func SetSysctl(key, value string) error {
	// H-1: validate key and value before shell execution
	if !reSysctlKey.MatchString(key) {
		return fmt.Errorf("invalid sysctl key: %s", key)
	}
	if !reSysctlVal.MatchString(value) {
		return fmt.Errorf("invalid sysctl value: %s", value)
	}
	cmdStr := fmt.Sprintf("sysctl -w %s=\"%s\"", key, value)
	out, err := exec.Command(config.SUBin, "-c", cmdStr).CombinedOutput()
	if err != nil {
		return fmt.Errorf("sysctl error: %v, output: %s", err, string(out))
	}
	return nil
}

func GetTTLSpoofStatus() bool {
	out, err := exec.Command(config.SUBin, "-c", "iptables -t mangle -C POSTROUTING -j TTL --ttl-set 64").CombinedOutput()
	if err == nil {
		return true
	}
	return strings.Contains(string(out), "target") // Rule exists or iptables matched
}

func SetTTLSpoof(enable bool, ttl int) error {
	if ttl <= 0 {
		ttl = 64
	}
	// Clear existing
	_ = exec.Command(config.SUBin, "-c", "iptables -t mangle -D POSTROUTING -j TTL --ttl-set 64 2>/dev/null").Run()
	_ = exec.Command(config.SUBin, "-c", "ip6tables -t mangle -D POSTROUTING -j HL --hl-set 64 2>/dev/null").Run()

	if enable {
		cmdV4 := fmt.Sprintf("iptables -t mangle -A POSTROUTING -j TTL --ttl-set %d", ttl)
		cmdV6 := fmt.Sprintf("ip6tables -t mangle -A POSTROUTING -j HL --hl-set %d", ttl)
		if out, err := exec.Command(config.SUBin, "-c", cmdV4).CombinedOutput(); err != nil {
			return fmt.Errorf("iptables error: %v, output: %s", err, string(out))
		}
		_ = exec.Command(config.SUBin, "-c", cmdV6).Run()
	}
	return nil
}

func SetInterfaceConfig(iface string, mtu int, txqueuelen int) error {
	// N-3: validate iface, MTU, and TxQueueLen
	if !reIfaceName.MatchString(iface) {
		return fmt.Errorf("invalid interface name: %s", iface)
	}
	if mtu > 0 {
		if mtu < 68 || mtu > 9000 {
			return fmt.Errorf("invalid MTU %d: must be between 68 and 9000", mtu)
		}
		cmd := fmt.Sprintf("ip link set %s mtu %d", iface, mtu)
		if out, err := exec.Command(config.SUBin, "-c", cmd).CombinedOutput(); err != nil {
			return fmt.Errorf("mtu error: %v, out: %s", err, string(out))
		}
	}
	if txqueuelen > 0 {
		if txqueuelen > 100000 {
			return fmt.Errorf("invalid txqueuelen %d: must be between 0 and 100000", txqueuelen)
		}
		cmd := fmt.Sprintf("ip link set %s txqueuelen %d", iface, txqueuelen)
		if out, err := exec.Command(config.SUBin, "-c", cmd).CombinedOutput(); err != nil {
			return fmt.Errorf("txqueuelen error: %v, out: %s", err, string(out))
		}
	}
	return nil
}

func GetActiveDNS() (string, string) {
	d1, _ := exec.Command("getprop", "net.dns1").Output()
	d2, _ := exec.Command("getprop", "net.dns2").Output()
	dns1 := strings.TrimSpace(string(d1))
	dns2 := strings.TrimSpace(string(d2))
	if dns1 == "" {
		dns1 = "Default/DHCP"
	}
	if dns2 == "" {
		dns2 = "None"
	}
	return dns1, dns2
}

func SetDNS(primary, secondary string) error {
	// N-2: validate DNS IP addresses
	parsedPrimary := net.ParseIP(primary)
	if primary == "" || parsedPrimary == nil {
		return fmt.Errorf("invalid primary DNS IP address: %s", primary)
	}
	if secondary != "" && net.ParseIP(secondary) == nil {
		return fmt.Errorf("invalid secondary DNS IP address: %s", secondary)
	}

	isIPv6 := parsedPrimary.To4() == nil

	cmds := []string{
		fmt.Sprintf("setprop net.dns1 %s", primary),
		fmt.Sprintf("setprop net.dns2 %s", secondary),
		"iptables -t nat -D OUTPUT -p udp --dport 53 -j REDIRECT --to-ports 53 2>/dev/null || true",
		"iptables -t nat -D OUTPUT -p tcp --dport 53 -j REDIRECT --to-ports 53 2>/dev/null || true",
	}

	if isIPv6 {
		cmds = append(cmds,
			fmt.Sprintf("ip6tables -t nat -A OUTPUT -p udp --dport 53 -j DNAT --to-destination [%s]:53 2>/dev/null || true", primary),
			fmt.Sprintf("ip6tables -t nat -A OUTPUT -p tcp --dport 53 -j DNAT --to-destination [%s]:53 2>/dev/null || true", primary),
		)
	} else {
		cmds = append(cmds,
			fmt.Sprintf("iptables -t nat -A OUTPUT -p udp --dport 53 -j DNAT --to-destination %s:53 2>/dev/null || true", primary),
			fmt.Sprintf("iptables -t nat -A OUTPUT -p tcp --dport 53 -j DNAT --to-destination %s:53 2>/dev/null || true", primary),
		)
	}

	cmdStr := strings.Join(cmds, " && ")
	out, err := exec.Command(config.SUBin, "-c", cmdStr).CombinedOutput()
	if err != nil {
		logger.Get().Errorf("network", "Failed setting DNS: %v, out: %s", err, string(out))
		return fmt.Errorf("dns error: %v, out: %s", err, string(out))
	}

	if ifaces, err := net.Interfaces(); err == nil {
		for _, ifc := range ifaces {
			name := strings.ToLower(ifc.Name)
			if strings.Contains(name, "wlan") || strings.Contains(name, "rmnet") {
				ndcCmd := fmt.Sprintf("ndc resolver setnetdns %s \"\" %s %s", ifc.Name, primary, secondary)
				_ = exec.Command(config.SUBin, "-c", ndcCmd).Run()
			}
		}
	}

	logger.Get().Infof("network", "DNS updated successfully: primary=%s, secondary=%s", primary, secondary)
	return nil
}

func GetInterfaces() ([]InterfaceInfo, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	netDevStats := parseProcNetDev()

	var result []InterfaceInfo
	for _, ifc := range ifaces {
		info := InterfaceInfo{
			Name: ifc.Name,
			MTU:  ifc.MTU,
		}

		addrs, err := ifc.Addrs()
		if err == nil {
			for _, addr := range addrs {
				info.IPs = append(info.IPs, addr.String())
			}
		}

		// Read txqueuelen
		qlenData, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/tx_queue_len", ifc.Name))
		if err == nil {
			// L-1: handle strconv.Atoi error instead of silently ignoring it.
			if qlen, err := strconv.Atoi(strings.TrimSpace(string(qlenData))); err == nil {
				info.TxQueueLen = qlen
			}
		}

		if stats, ok := netDevStats[ifc.Name]; ok {
			info.RxBytes = stats.RxBytes
			info.TxBytes = stats.TxBytes
			info.RxPackets = stats.RxPackets
			info.TxPackets = stats.TxPackets
		}

		result = append(result, info)
	}

	return result, nil
}

type devStat struct {
	RxBytes   uint64
	RxPackets uint64
	TxBytes   uint64
	TxPackets uint64
}

func GetProp(prop string) string {
	out, err := exec.Command("getprop", prop).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func parseProcNetDev() map[string]devStat {
	res := make(map[string]devStat)
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return res
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) == 2 {
			iface := strings.TrimSpace(parts[0])
			fields := strings.Fields(parts[1])
			if len(fields) >= 16 {
				rxBytes, _ := strconv.ParseUint(fields[0], 10, 64)
				rxPackets, _ := strconv.ParseUint(fields[1], 10, 64)
				txBytes, _ := strconv.ParseUint(fields[8], 10, 64)
				txPackets, _ := strconv.ParseUint(fields[9], 10, 64)
				res[iface] = devStat{
					RxBytes:   rxBytes,
					RxPackets: rxPackets,
					TxBytes:   txBytes,
					TxPackets: txPackets,
				}
			}
		}
	}
	return res
}
