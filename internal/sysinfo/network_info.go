package sysinfo

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/config"
)

type NetworkDetail struct {
	IPAddresses    []string  `json:"ip_addresses"`
	Gateway        string    `json:"gateway"`
	DNS            []string  `json:"dns"`
	DNS1           string    `json:"dns1"`
	DNS2           string    `json:"dns2"`
	WiFiSSID       string    `json:"wifi_ssid"`
	WiFiSignal     string    `json:"wifi_signal"`
	WiFiSignalDBM  string    `json:"wifi_signal_dbm"`
	WiFiRSSI       int       `json:"wifi_rssi"`
	WiFiFullInfo   string    `json:"wifi_full_info"`
	MCCMNC         []string  `json:"mcc_mnc"`
	Roaming        string    `json:"roaming"`
	HotspotClients int       `json:"hotspot_clients"`
	SIMSlots       []SIMSlot `json:"sim_slots"`
}

type SIMSlot struct {
	Slot          int     `json:"slot"`
	Operator      string  `json:"operator"`
	NetworkType   string  `json:"network_type"`
	RSRP          string  `json:"rsrp"`
	RSRQ          string  `json:"rsrq"`
	SINR          string  `json:"sinr"`
	RSRPInt       int     `json:"rsrp_int"`
	RSRQInt       int     `json:"rsrq_int"`
	SINRInt       int     `json:"sinr_int"`
	SignalStatus  string  `json:"signal_status"`
	SignalQuality string  `json:"signal_quality"`
	SignalScore   float64 `json:"signal_score"`
}


var (
	networkDetailCache   NetworkDetail
	networkDetailCacheAt time.Time
	networkDetailCacheMu sync.Mutex
)
var rePingHost = regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)

// RunPing executes ICMP ping with a strict timeout, platform awareness, and packet loss resilience.
func RunPing(host string, count int) (string, error) {
	if host == "" || strings.HasPrefix(host, "-") {
		return "", fmt.Errorf("invalid ping host")
	}
	if net.ParseIP(host) == nil && !rePingHost.MatchString(host) {
		return "", fmt.Errorf("invalid ping host format")
	}
	if count <= 0 || count > 10 {
		count = 4
	}

	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()

	var out []byte
	var err error

	if runtime.GOOS == "windows" {
		cmd := exec.CommandContext(ctx, "ping", "-n", strconv.Itoa(count), "-w", "1500", host)
		out, err = cmd.CombinedOutput()
	} else {
		cmd := exec.CommandContext(ctx, "ping", "-c", strconv.Itoa(count), "-W", "2", host)
		out, err = cmd.CombinedOutput()
		if len(out) == 0 && config.SUBin != "" {
			// Fallback to su root execution if non-root ping lacks ICMP privileges
			cmdRoot := exec.CommandContext(ctx, config.SUBin, "-c", fmt.Sprintf("ping -c %d -W 2 %s", count, host))
			out, err = cmdRoot.CombinedOutput()
		}
	}

	if len(out) > 0 {
		return string(out), nil
	}
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "Ping request timed out (7s limit reached).", fmt.Errorf("ping timeout")
		}
		return "", err
	}
	return "No ping response received.", nil
}

func RunSpeedtest() (map[string]interface{}, error) {
	// Simple HTTP benchmark download check
	start := time.Now()
	resp, err := exec.Command("curl", "-s", "-w", "%{speed_download}", "-o", "/dev/null", "https://speed.cloudflare.com/__down?bytes=25000000").Output()
	duration := time.Since(start).Seconds()

	result := make(map[string]interface{})
	if err == nil {
		bytesPerSec, _ := strconv.ParseFloat(strings.TrimSpace(string(resp)), 64)
		mbps := (bytesPerSec * 8) / 1000000.0
		result["download_mbps"] = mbps
		result["duration_sec"] = duration
		return result, nil
	}
	return nil, err
}


func getMTU() string {
	data, err := os.ReadFile("/sys/class/net/wlan0/mtu")
	if err == nil {
		return strings.TrimSpace(string(data))
	}
	return "1500 (Default)"
}

func getDefaultTTL() string {
	data, err := os.ReadFile("/proc/sys/net/ipv4/ip_default_ttl")
	if err == nil {
		return strings.TrimSpace(string(data))
	}
	return "64"
}

func getNetDevBytes() (uint64, uint64) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return 0, 0
	}
	defer file.Close()

	var totalRx, totalTx uint64
	scanner := bufio.NewScanner(file)
	// Skip first 2 header lines
	if scanner.Scan() {
		_ = scanner.Text()
	}
	if scanner.Scan() {
		_ = scanner.Text()
	}

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}
		ifname := strings.TrimSuffix(fields[0], ":")
		if ifname == "lo" || strings.HasPrefix(ifname, "dummy") || strings.HasPrefix(ifname, "tun") {
			continue
		}
		rx, _ := strconv.ParseUint(fields[1], 10, 64)
		tx, _ := strconv.ParseUint(fields[9], 10, 64)
		totalRx += rx
		totalTx += tx
	}
	return totalRx, totalTx
}

// runCmdTimeout runs an external command with a hard timeout, returns trimmed stdout.
func runCmdTimeout(timeout time.Duration, name string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, _ := exec.CommandContext(ctx, name, args...).Output()
	return strings.TrimSpace(string(out))
}

func getNetworkDetail() NetworkDetail {
	networkDetailCacheMu.Lock()
	defer networkDetailCacheMu.Unlock()

	if !networkDetailCacheAt.IsZero() && time.Since(networkDetailCacheAt) < 15*time.Second {
		return networkDetailCache
	}

	nd := buildNetworkDetail()
	networkDetailCache = nd
	networkDetailCacheAt = time.Now()
	return nd
}

func parseSignalQuality(rsrp int) float64 {
	// Scale: -140 dBm → 0.0, -44 dBm → 10.0 (3GPP typical range)
	if rsrp >= 0 {
		return 0.0
	}
	score := (float64(rsrp) + 140.0) / 96.0 * 10.0
	if score < 0 {
		score = 0
	}
	if score > 10 {
		score = 10
	}
	// Round to 1 decimal
	score = float64(int(score*10+0.5)) / 10.0
	return score
}

func buildNetworkDetail() NetworkDetail {
	var nd NetworkDetail
	nd.WiFiSSID = "Unknown"
	nd.WiFiSignalDBM = "—"
	nd.WiFiSignal = "—"
	nd.Roaming = "Unknown"

	const cmdTimeout = 2 * time.Second

	// 1. IP Addresses (cheap — no subprocess)
	ifaces, err := net.Interfaces()
	if err == nil {
		var wlanIPs []string
		var otherIPs []string
		for _, i := range ifaces {
			if i.Flags&net.FlagLoopback != 0 {
				continue
			}
			addrs, err := i.Addrs()
			if err == nil {
				for _, addr := range addrs {
					ip := addr.String()
					if !strings.Contains(ip, ":") { // simple IPv4 filter
						cleanIP := strings.Split(ip, "/")[0]
						if strings.Contains(strings.ToLower(i.Name), "wlan") {
							wlanIPs = append(wlanIPs, cleanIP)
						} else {
							otherIPs = append(otherIPs, cleanIP)
						}
					}
				}
			}
		}
		nd.IPAddresses = append(nd.IPAddresses, wlanIPs...)
		nd.IPAddresses = append(nd.IPAddresses, otherIPs...)
	}

	// 2. Gateway
	if out := runCmdTimeout(cmdTimeout, "ip", "route", "get", "8.8.8.8"); out != "" {
		parts := strings.Fields(out)
		for i, p := range parts {
			if p == "via" && i+1 < len(parts) {
				gw := parts[i+1]
				if net.ParseIP(gw) != nil {
					nd.Gateway = gw
					break
				}
			}
		}
	}
	if nd.Gateway == "" {
		if out := runCmdTimeout(cmdTimeout, "ip", "route", "show", "table", "all"); out != "" {
			for _, line := range strings.Split(out, "\n") {
				line = strings.TrimSpace(line)
				if strings.Contains(line, "default via ") {
					parts := strings.Fields(line)
					for i, p := range parts {
						if p == "via" && i+1 < len(parts) {
							gw := parts[i+1]
							if net.ParseIP(gw) != nil {
								nd.Gateway = gw
								break
							}
						}
					}
					if nd.Gateway != "" {
						break
					}
				}
			}
		}
	}
	if nd.Gateway == "" {
		if out := runCmdTimeout(cmdTimeout, "ip", "route"); out != "" {
			for _, line := range strings.Split(out, "\n") {
				line = strings.TrimSpace(line)
				if strings.Contains(line, "default via ") {
					parts := strings.Fields(line)
					for i, p := range parts {
						if p == "via" && i+1 < len(parts) {
							gw := parts[i+1]
							if net.ParseIP(gw) != nil {
								nd.Gateway = gw
								break
							}
						}
					}
					if nd.Gateway != "" {
						break
					}
				}
			}
		}
	}

	// 3. DNS
	seenIPs := make(map[string]bool)

	addDNS := func(rawIP, label string) {
		rawIP = strings.TrimPrefix(strings.TrimSpace(rawIP), "/")
		if net.ParseIP(rawIP) != nil {
			if !seenIPs[rawIP] {
				seenIPs[rawIP] = true
				nd.DNS = append(nd.DNS, fmt.Sprintf("%s (%s)", rawIP, label))
			}
		}
	}

	// 3a. Active/Running DNS (dumpsys dnsresolver)
	if resolverOut := runCmdTimeout(cmdTimeout, config.SUBin, "-c", "dumpsys dnsresolver"); resolverOut != "" {
		blocks := strings.Split(resolverOut, "netId")
		reDNSList := regexp.MustCompile(`(?i)(?:DNS servers|DnsAddresses|Server[s]?):\s*\[?\s*([^\]\n]+)\]?`)
		reIfaceList := regexp.MustCompile(`(?i)(?:Interface names|Interface|iface):\s*\[?\s*([^\]\n]+)\]?`)

		for _, block := range blocks {
			if strings.TrimSpace(block) == "" {
				continue
			}
			iface := ""
			if ifaceMatch := reIfaceList.FindStringSubmatch(block); len(ifaceMatch) > 1 {
				iface = strings.ToLower(strings.TrimSpace(ifaceMatch[1]))
			}

			label := "Active"
			if strings.Contains(iface, "wlan") {
				label = "Active / Wi-Fi"
			} else if strings.Contains(iface, "rmnet") || strings.Contains(iface, "ccmni") || strings.Contains(iface, "p2p") {
				label = "Active / Cellular"
			}

			dnsMatches := reDNSList.FindAllStringSubmatch(block, -1)
			for _, m := range dnsMatches {
				if len(m) > 1 {
					tokens := strings.Fields(strings.ReplaceAll(m[1], ",", " "))
					for _, tok := range tokens {
						addDNS(tok, label)
					}
				}
			}
		}

		if len(seenIPs) == 0 {
			lines := strings.Split(resolverOut, "\n")
			curIface := ""
			reIP := regexp.MustCompile(`([0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3})`)
			for _, line := range lines {
				lower := strings.ToLower(line)
				if strings.Contains(lower, "wlan") {
					curIface = "wlan"
				} else if strings.Contains(lower, "rmnet") || strings.Contains(lower, "ccmni") || strings.Contains(lower, "p2p") {
					curIface = "rmnet"
				}
				for _, ipMatch := range reIP.FindAllString(line, -1) {
					label := "Active"
					if curIface == "wlan" {
						label = "Active / Wi-Fi"
					} else if curIface == "rmnet" {
						label = "Active / Cellular"
					}
					addDNS(ipMatch, label)
				}
			}
		}
	}

	// 3b. Network Profile Config DNS (dumpsys connectivity)
	if connOut := runCmdTimeout(cmdTimeout, config.SUBin, "-c", "dumpsys connectivity"); connOut != "" {
		blocks := strings.Split(connOut, "NetworkAgentInfo")
		reDnsAddr := regexp.MustCompile(`(?i)DnsAddresses:\s*\[\s*([^\]]+)\]`)

		for _, block := range blocks {
			if strings.TrimSpace(block) == "" {
				continue
			}
			blockLower := strings.ToLower(block)
			isWifi := strings.Contains(blockLower, "wifi") || strings.Contains(blockLower, "wlan")
			isCell := strings.Contains(blockLower, "cellular") || strings.Contains(blockLower, "rmnet") || strings.Contains(blockLower, "ccmni")

			label := "Config"
			if isWifi {
				label = "Wi-Fi Config"
			} else if isCell {
				label = "Cellular Config"
			}

			if dnsMatch := reDnsAddr.FindStringSubmatch(block); len(dnsMatch) > 1 {
				tokens := strings.Fields(strings.ReplaceAll(dnsMatch[1], ",", " "))
				for _, tok := range tokens {
					addDNS(tok, label)
				}
			}
		}

		reDnsAddrAll := regexp.MustCompile(`(?i)DnsAddresses:\s*\[\s*([^\]]+)\]`)
		for _, m := range reDnsAddrAll.FindAllStringSubmatch(connOut, -1) {
			if len(m) > 1 {
				tokens := strings.Fields(strings.ReplaceAll(m[1], ",", " "))
				for _, tok := range tokens {
					addDNS(tok, "Config")
				}
			}
		}
	}

	// 3c. Fallback properties
	if dns1 := getProp("net.dns1"); dns1 != "" {
		addDNS(dns1, "Config")
	}
	if dns2 := getProp("net.dns2"); dns2 != "" {
		addDNS(dns2, "Config")
	}

	if out := runCmdTimeout(cmdTimeout, "getprop"); out != "" {
		reDNS := regexp.MustCompile(`(?i)\[.*dns.*\]:\s*\[([0-9\.]+)\]`)
		matches := reDNS.FindAllStringSubmatch(out, -1)
		for _, match := range matches {
			if len(match) > 1 {
				addDNS(match[1], "Config")
			}
		}
	}

	if len(nd.DNS) > 0 {
		nd.DNS1 = nd.DNS[0]
	}
	if len(nd.DNS) > 1 {
		nd.DNS2 = nd.DNS[1]
	}

	// 4. WiFi SSID / RSSI
	wifiStr := runCmdTimeout(cmdTimeout, "cmd", "wifi", "status")
	nd.WiFiFullInfo = wifiStr

	reSSID := regexp.MustCompile(`(?i)SSID:\s*"([^"]+)"|SSID:\s*([^\s,]+)`)
	ssidMatch := reSSID.FindStringSubmatch(wifiStr)
	if len(ssidMatch) > 1 {
		if ssidMatch[1] != "" {
			nd.WiFiSSID = ssidMatch[1]
		} else if len(ssidMatch) > 2 && ssidMatch[2] != "" {
			nd.WiFiSSID = ssidMatch[2]
		}
	}

	var rssiVal int
	if strings.Contains(wifiStr, "RSSI: ") {
		rssiParts := strings.Split(wifiStr, "RSSI: ")
		if len(rssiParts) > 1 {
			// Split by newline first, then by comma to handle single-line output
			rssiEnd := strings.Trim(strings.Split(strings.Split(rssiParts[1], "\n")[0], ",")[0], " ")
			if r, err := strconv.Atoi(rssiEnd); err == nil {
				rssiVal = r
			}
			nd.WiFiRSSI = rssiVal
			nd.WiFiSignalDBM = rssiEnd
			nd.WiFiSignal = nd.WiFiSignalDBM + " dBm"
		}
	}
	// Try to get link speed
	if strings.Contains(wifiStr, "Link speed: ") {
		speedParts := strings.Split(wifiStr, "Link speed: ")
		if len(speedParts) > 1 {
			// Split by newline first, then by comma to handle single-line output
			speedEnd := strings.Trim(strings.Split(strings.Split(speedParts[1], "\n")[0], ",")[0], " ")
			if nd.WiFiSignal != "—" {
				nd.WiFiSignal += " / " + speedEnd
			}
		}
	}

	// 5. MCC/MNC and Roaming
	numString := getProp("gsm.operator.numeric")
	if numString == "" {
		numString = getProp("gsm.sim.operator.numeric")
	}
	for _, num := range strings.Split(numString, ",") {
		n := strings.TrimSpace(num)
		if n != "" {
			nd.MCCMNC = append(nd.MCCMNC, n)
		}
	}
	roam := getProp("gsm.operator.isroaming")
	if strings.Contains(roam, "true") {
		nd.Roaming = "Yes"
	} else if strings.Contains(roam, "false") {
		nd.Roaming = "No"
	}

	// 6. Hotspot clients via /proc/net/arp (filtered by tether interfaces)
	var arpCount int
	if f, err := os.Open("/proc/net/arp"); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) >= 6 && fields[2] == "0x2" && fields[3] != "00:00:00:00:00:00" {
				dev := strings.ToLower(fields[5])
				if strings.HasPrefix(dev, "ap") || strings.HasPrefix(dev, "softap") || strings.HasPrefix(dev, "wlan1") || strings.HasPrefix(dev, "rndis") || strings.HasPrefix(dev, "usb") || strings.HasPrefix(dev, "eth") {
					arpCount++
				}
			}
		}
		f.Close()
	}
	nd.HotspotClients = arpCount

	// 7. SIM Slots
	opers := strings.Split(getProp("gsm.operator.alpha"), ",")
	types := strings.Split(getProp("gsm.network.type"), ",")
	nums := strings.Split(getProp("gsm.operator.numeric"), ",")

	numSlots := len(nums)
	if len(opers) > numSlots {
		numSlots = len(opers)
	}
	if len(types) > numSlots {
		numSlots = len(types)
	}

	// Telephony Registry — best-effort, timeout-bounded
	// we will run once and match by subscription/slot index if possible, otherwise apply to slot 1
	telCtx, cancelTel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancelTel()
	telOutBytes, _ := exec.CommandContext(telCtx, "su", "-c", "dumpsys telephony.registry").Output()
	telOut := string(telOutBytes)

	// Naive per-slot extraction from dumpsys
	signalBlocks := strings.Split(telOut, "mSignalStrength=SignalStrength:")
	var rsrps []int
	var rsrqs []int
	var sinrs []int

	for i := 1; i < len(signalBlocks); i++ {
		block := signalBlocks[i]

		rsrp, rsrq, sinr := 0, 0, 0
		if idx := strings.Index(block, "rsrp="); idx != -1 {
			valStart := idx + 5
			valEnd := valStart
			for valEnd < len(block) && (block[valEnd] == '-' || (block[valEnd] >= '0' && block[valEnd] <= '9')) {
				valEnd++
			}
			if valStart < valEnd {
				if r, err := strconv.Atoi(block[valStart:valEnd]); err == nil {
					rsrp = r
				}
			}
		}
		if idx := strings.Index(block, "rsrq="); idx != -1 {
			valStart := idx + 5
			valEnd := valStart
			for valEnd < len(block) && (block[valEnd] == '-' || (block[valEnd] >= '0' && block[valEnd] <= '9')) {
				valEnd++
			}
			if valStart < valEnd {
				if r, err := strconv.Atoi(block[valStart:valEnd]); err == nil {
					rsrq = r
				}
			}
		}
		if idx := strings.Index(block, "rssnr="); idx != -1 {
			valStart := idx + 6
			valEnd := valStart
			for valEnd < len(block) && (block[valEnd] == '-' || (block[valEnd] >= '0' && block[valEnd] <= '9')) {
				valEnd++
			}
			if valStart < valEnd {
				if r, err := strconv.Atoi(block[valStart:valEnd]); err == nil {
					sinr = r
				}
			}
		}

		// some devices return MAX_INT for unavailable
		if rsrp == 2147483647 {
			rsrp = 0
		}
		if rsrq == 2147483647 {
			rsrq = 0
		}
		if sinr == 2147483647 {
			sinr = 0
		}

		rsrps = append(rsrps, rsrp)
		rsrqs = append(rsrqs, rsrq)
		sinrs = append(sinrs, sinr)
	}

	for i := 0; i < numSlots; i++ {
		slot := SIMSlot{Slot: i + 1, Operator: "Unknown", NetworkType: "Unknown"}
		if i < len(opers) && strings.TrimSpace(opers[i]) != "" {
			slot.Operator = strings.TrimSpace(opers[i])
		}
		if i < len(types) && strings.TrimSpace(types[i]) != "" {
			slot.NetworkType = strings.TrimSpace(types[i])
		}
		if i < len(nums) && strings.TrimSpace(nums[i]) != "" && slot.Operator == "Unknown" {
			slot.Operator = strings.TrimSpace(nums[i])
		}

		rsrp, rsrq, sinr := 0, 0, 0
		if i < len(rsrps) {
			rsrp = rsrps[i]
			rsrq = rsrqs[i]
			sinr = sinrs[i]
		}

		slot.RSRPInt = rsrp
		slot.RSRQInt = rsrq
		slot.SINRInt = sinr

		if rsrp == 0 || rsrp == -1 {
			slot.RSRP = "—"
			slot.RSRQ = "—"
			slot.SINR = "—"
			slot.SignalStatus = "—"
		} else {
			slot.RSRP = strconv.Itoa(rsrp)
			slot.SignalStatus = slot.RSRP + " dBm"
			slot.RSRQ = strconv.Itoa(rsrq)
			slot.SINR = strconv.Itoa(sinr)
		}

		score := parseSignalQuality(rsrp)
		slot.SignalScore = score

		if rsrp < 0 {
			if rsrp >= -90 {
				slot.SignalQuality = "Excellent"
			} else if rsrp >= -105 {
				slot.SignalQuality = "Good"
			} else if rsrp >= -115 {
				slot.SignalQuality = "Fair"
			} else {
				slot.SignalQuality = "Poor"
			}
		} else {
			slot.SignalQuality = "Unavailable"
			slot.SignalScore = 0
		}
		nd.SIMSlots = append(nd.SIMSlots, slot)
	}

	return nd
}


