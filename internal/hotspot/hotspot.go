package hotspot

import (
	"strings"
	"time"

	"bfr-webui-go/internal/config"
)

// GetHotspotStatus inspects network interfaces and system status to return tethering state.
func GetHotspotStatus() HotspotStatus {
	activeIfaces := GetActiveTetherInterfaces()

	var status HotspotStatus
	status.ActiveInterfaces = make([]string, 0, len(activeIfaces))
	ssid, _, secType := ReadSavedSoftApCredentials()
	status.SSID = ssid
	status.SecurityType = secType

	for _, ifc := range activeIfaces {
		status.ActiveInterfaces = append(status.ActiveInterfaces, ifc.Name)
		switch ifc.Type {
		case "wifi":
			status.WifiEnabled = true
			if status.GatewayIP == "" {
				status.GatewayIP = ifc.IP
				status.Subnet = ifc.CIDR
			}
		case "usb":
			status.UsbEnabled = true
			if status.GatewayIP == "" {
				status.GatewayIP = ifc.IP
				status.Subnet = ifc.CIDR
			}
		case "ethernet":
			status.EthernetEnabled = true
			if status.GatewayIP == "" {
				status.GatewayIP = ifc.IP
				status.Subnet = ifc.CIDR
			}
		}
	}

	// Hotspot is considered enabled if any Wi-Fi tethering interface is up
	// or if cmd wifi reports enabled
	if status.WifiEnabled {
		status.Enabled = true
	} else {
		out, err := config.ExecSuTimeout(2*time.Second, "cmd wifi status 2>/dev/null | grep -i 'Wifi AP'")
		if err == nil && strings.Contains(strings.ToLower(string(out)), "enabled") {
			status.Enabled = true
			status.WifiEnabled = true
		}
	}

	return status
}
