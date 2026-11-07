package hotspot

import (
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	"bfr-webui-go/internal/config"
)

var (
	wifiIfaceCandidates     = []string{"wlan1", "ap0", "wlan2", "softap0", "swlan0"}
	usbIfaceCandidates      = []string{"rndis0", "usb0", "ncm0"}
	ethernetIfaceCandidates = []string{"eth0", "eth1"}

	softApXMLCandidatePaths = []string{
		"/data/misc_ce/0/apexdata/com.android.wifi/WifiConfigStoreSoftAp.xml",
		"/data/misc/apexdata/com.android.wifi/WifiConfigStoreSoftAp.xml",
		"/data/misc/wifi/WifiConfigStoreSoftAp.xml",
	}
)

// isProcNetRouteWAN checks /proc/net/route text to determine whether ifaceName
// has an IPv4 default gateway route (Destination 00000000 with non-zero Gateway),
// which indicates the interface is connected to an upstream router as a WAN client.
func isProcNetRouteWAN(ifaceName, procRouteContent string) bool {
	if ifaceName == "" || procRouteContent == "" {
		return false
	}
	for line := range strings.SplitSeq(procRouteContent, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		if strings.EqualFold(fields[0], ifaceName) && fields[1] == "00000000" && fields[2] != "00000000" {
			return true
		}
	}
	return false
}

// isUpstreamWANInterface returns true if the interface is operating as an
// upstream WAN client (e.g., connected to an external router) rather than a
// downstream tethering server.
func isUpstreamWANInterface(ifaceName string) bool {
	cfgStoreMu.RLock()
	if cachedCfg != nil && strings.EqualFold(cachedCfg.UpstreamInterface, ifaceName) {
		cfgStoreMu.RUnlock()
		return true
	}
	cfgStoreMu.RUnlock()

	if data, err := os.ReadFile("/proc/net/route"); err == nil {
		if isProcNetRouteWAN(ifaceName, string(data)) {
			return true
		}
	}

	// Check Android per-interface policy routing tables: if Android installed an
	// incoming tether rule ("iif <ifaceName> lookup"), it is a tether downstream.
	if ruleOut, err := config.ExecSuTimeout(1*time.Second, "ip rule show 2>/dev/null"); err == nil {
		if strings.Contains(string(ruleOut), "iif "+ifaceName+" ") {
			return false
		}
	}
	if routeOut, err := config.ExecSuTimeout(1*time.Second, "ip route show table all 2>/dev/null"); err == nil {
		for line := range strings.SplitSeq(string(routeOut), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "default ") && strings.Contains(line, "via ") && strings.Contains(line, " dev "+ifaceName) {
				return true
			}
		}
	}

	return false
}

// ActiveTetherInfo holds detected interface details.
type ActiveTetherInfo struct {
	Name      string
	Type      string // "wifi", "usb", "ethernet"
	IP        string
	CIDR      string
	IsUp      bool
	HasInetIP bool
}

// GetActiveTetherInterfaces scans network interfaces and returns list of active tethering devices.
func GetActiveTetherInterfaces() []ActiveTetherInfo {
	var results []ActiveTetherInfo

	ifaces, err := net.Interfaces()
	if err != nil {
		return results
	}

	for _, ifc := range ifaces {
		name := strings.ToLower(ifc.Name)
		var ifaceType string

		for _, cand := range wifiIfaceCandidates {
			if strings.HasPrefix(name, cand) {
				ifaceType = "wifi"
				break
			}
		}
		if ifaceType == "" {
			for _, cand := range usbIfaceCandidates {
				if strings.HasPrefix(name, cand) {
					ifaceType = "usb"
					break
				}
			}
		}
		if ifaceType == "" {
			for _, cand := range ethernetIfaceCandidates {
				if strings.HasPrefix(name, cand) {
					ifaceType = "ethernet"
					break
				}
			}
		}

		if ifaceType == "" {
			continue
		}

		isUp := (ifc.Flags & net.FlagUp) != 0
		addrs, err := ifc.Addrs()
		var ipStr, cidrStr string
		hasInet := false

		if err == nil {
			for _, addr := range addrs {
				if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
					if ipNet.IP.To4() != nil {
						ipStr = ipNet.IP.String()
						cidrStr = ipNet.String()
						hasInet = true
						break
					}
				}
			}
		}

		if isUp && hasInet {
			if ifaceType == "ethernet" && isUpstreamWANInterface(ifc.Name) {
				continue
			}
			results = append(results, ActiveTetherInfo{
				Name:      ifc.Name,
				Type:      ifaceType,
				IP:        ipStr,
				CIDR:      cidrStr,
				IsUp:      isUp,
				HasInetIP: hasInet,
			})
		}
	}

	return results
}

// GetActiveInterfaceNames returns string slice of active tethering interfaces.
func GetActiveInterfaceNames() []string {
	active := GetActiveTetherInterfaces()
	names := make([]string, 0, len(active))
	for _, a := range active {
		names = append(names, a.Name)
	}
	return names
}

func parseSoftApXMLCredentials(str string) (ssid string, pass string, secType string) {
	secType = "wpa2"
	reSSID := regexp.MustCompile(`<string name="(?:WifiSsid|SSID)">(?:&quot;)?([^<"&]+)(?:&quot;)?</string>`)
	if m := reSSID.FindStringSubmatch(str); len(m) > 1 && len(m[1]) > 0 {
		ssid = strings.TrimSpace(m[1])
	}
	reSec := regexp.MustCompile(`<int name="SecurityType" value="([0-9]+)" />`)
	if m := reSec.FindStringSubmatch(str); len(m) > 1 {
		if m[1] == "0" {
			secType = "open"
		} else {
			secType = "wpa2"
		}
	}
	rePass := regexp.MustCompile(`<string name="Passphrase">([^<]*)</string>`)
	if m := rePass.FindStringSubmatch(str); len(m) > 1 {
		pass = strings.TrimSpace(m[1])
	}
	return ssid, pass, secType
}

// readAOSPSoftApCredentials reads SoftAP credentials directly from AOSP XML stores
// or global settings without consulting BFR's cached config.
func readAOSPSoftApCredentials() (ssid string, pass string, secType string) {
	secType = "wpa2"

	for _, p := range softApXMLCandidatePaths {
		var content []byte
		var err error
		if content, err = os.ReadFile(p); err != nil {
			out, suErr := config.ExecSuTimeout(2*time.Second, "cat "+p+" 2>/dev/null")
			if suErr == nil && len(out) > 0 {
				content = out
			}
		}

		if len(content) > 0 {
			s, pw, sec := parseSoftApXMLCredentials(string(content))
			if s != "" {
				return s, pw, sec
			}
		}
	}

	// Fallback to legacy global settings
	out, err := config.ExecSuTimeout(2*time.Second, "settings get global softap_ssid 2>/dev/null")
	if err == nil {
		trimmed := strings.TrimSpace(string(out))
		if trimmed != "" && trimmed != "null" {
			ssid = trimmed
		}
	}

	return ssid, pass, secType
}

// ReadSavedSoftApCredentials reads the genuine SoftAP SSID, Passphrase, and SecurityType.
// Prioritizes BFR-owned persisted config, then falls back to AOSP XML stores.
func ReadSavedSoftApCredentials() (ssid string, pass string, secType string) {
	p := getRouterConfigPath()
	cfgStoreMu.RLock()
	if cachedCfg != nil && cachedCfgPath == p && cachedCfg.SSID != "" {
		s, pw, sec := cachedCfg.SSID, cachedCfg.Password, cachedCfg.SecurityType
		cfgStoreMu.RUnlock()
		return s, pw, sec
	}
	cfgStoreMu.RUnlock()

	if cfg, err := LoadRouterConfig(); err == nil && cfg != nil && cfg.SSID != "" {
		return cfg.SSID, cfg.Password, cfg.SecurityType
	}

	ssid, pass, secType = readAOSPSoftApCredentials()
	if ssid == "" {
		ssid = "AndroidAP"
	}
	return ssid, pass, secType
}

// DetectCurrentSSID returns the configured SoftAP broadcast SSID without client mode dump leaks.
func DetectCurrentSSID() string {
	ssid, _, _ := ReadSavedSoftApCredentials()
	return ssid
}
