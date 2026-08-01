package network

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

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
	out, err := exec.Command("su", "-c", "sysctl -n "+key).Output()
	if err != nil {
		out, err = exec.Command("sysctl", "-n", key).Output()
		if err != nil {
			return "", err
		}
	}
	return strings.TrimSpace(string(out)), nil
}

func SetSysctl(key, value string) error {
	cmdStr := fmt.Sprintf("sysctl -w %s=\"%s\"", key, value)
	out, err := exec.Command("su", "-c", cmdStr).CombinedOutput()
	if err != nil {
		return fmt.Errorf("sysctl error: %v, output: %s", err, string(out))
	}
	return nil
}

func GetTTLSpoofStatus() bool {
	out, err := exec.Command("su", "-c", "iptables -t mangle -C POSTROUTING -j TTL --ttl-set 64").CombinedOutput()
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
	exec.Command("su", "-c", "iptables -t mangle -D POSTROUTING -j TTL --ttl-set 64 2>/dev/null").Run()
	exec.Command("su", "-c", "ip6tables -t mangle -D POSTROUTING -j HL --hl-set 64 2>/dev/null").Run()

	if enable {
		cmdV4 := fmt.Sprintf("iptables -t mangle -A POSTROUTING -j TTL --ttl-set %d", ttl)
		cmdV6 := fmt.Sprintf("ip6tables -t mangle -A POSTROUTING -j HL --hl-set %d", ttl)
		if out, err := exec.Command("su", "-c", cmdV4).CombinedOutput(); err != nil {
			return fmt.Errorf("iptables error: %v, output: %s", err, string(out))
		}
		exec.Command("su", "-c", cmdV6).Run()
	}
	return nil
}

func SetInterfaceConfig(iface string, mtu int, txqueuelen int) error {
	if mtu > 0 {
		cmd := fmt.Sprintf("ip link set %s mtu %d", iface, mtu)
		if out, err := exec.Command("su", "-c", cmd).CombinedOutput(); err != nil {
			return fmt.Errorf("mtu error: %v, out: %s", err, string(out))
		}
	}
	if txqueuelen > 0 {
		cmd := fmt.Sprintf("ip link set %s txqueuelen %d", iface, txqueuelen)
		if out, err := exec.Command("su", "-c", cmd).CombinedOutput(); err != nil {
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
	cmds := []string{
		fmt.Sprintf("setprop net.dns1 %s", primary),
		fmt.Sprintf("setprop net.dns2 %s", secondary),
		"iptables -t nat -D OUTPUT -p udp --dport 53 -j REDIRECT --to-ports 53 2>/dev/null || true",
	}
	if primary != "" {
		cmds = append(cmds, fmt.Sprintf("iptables -t nat -A OUTPUT -p udp --dport 53 -j DNAT --to-destination %s:53 2>/dev/null || true", primary))
	}
	cmdStr := strings.Join(cmds, " && ")
	out, err := exec.Command("su", "-c", cmdStr).CombinedOutput()
	if err != nil {
		return fmt.Errorf("dns error: %v, out: %s", err, string(out))
	}
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
