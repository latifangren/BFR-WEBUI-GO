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
)

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

// ReadSavedSoftApCredentials reads the genuine SoftAP SSID, Passphrase, and SecurityType from AOSP XML stores.
func ReadSavedSoftApCredentials() (ssid string, pass string, secType string) {
	candidatePaths := []string{
		"/data/misc_ce/0/apexdata/com.android.wifi/WifiConfigStoreSoftAp.xml",
		"/data/misc/apexdata/com.android.wifi/WifiConfigStoreSoftAp.xml",
		"/data/misc/wifi/WifiConfigStoreSoftAp.xml",
	}

	secType = "wpa2"

	for _, p := range candidatePaths {
		var content []byte
		var err error
		if content, err = os.ReadFile(p); err != nil {
			out, suErr := config.ExecSuTimeout(2*time.Second, "cat "+p+" 2>/dev/null")
			if suErr == nil && len(out) > 0 {
				content = out
			}
		}

		if len(content) > 0 {
			str := string(content)
			// Match WifiSsid or SSID
			reSSID := regexp.MustCompile(`<string name="(?:WifiSsid|SSID)">(?:&quot;)?([^<"&]+)(?:&quot;)?</string>`)
			if m := reSSID.FindStringSubmatch(str); len(m) > 1 && len(m[1]) > 0 {
				ssid = strings.TrimSpace(m[1])
			}
			// Match SecurityType (0 = open, 1 = wpa2)
			reSec := regexp.MustCompile(`<int name="SecurityType" value="([0-9]+)" />`)
			if m := reSec.FindStringSubmatch(str); len(m) > 1 {
				if m[1] == "0" {
					secType = "open"
				} else {
					secType = "wpa2"
				}
			}
			// Match Passphrase
			rePass := regexp.MustCompile(`<string name="Passphrase">([^<]*)</string>`)
			if m := rePass.FindStringSubmatch(str); len(m) > 1 {
				pass = strings.TrimSpace(m[1])
			}
			if ssid != "" {
				return ssid, pass, secType
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
