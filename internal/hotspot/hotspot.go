package hotspot

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type ConnectedClient struct {
	IP       string `json:"ip"`
	MAC      string `json:"mac"`
	Device   string `json:"device"`
	State    string `json:"state"`
}

type HotspotStatus struct {
	Enabled bool   `json:"enabled"`
	SSID    string `json:"ssid"`
}

func GetHotspotStatus() HotspotStatus {
	var status HotspotStatus

	out, err := exec.Command("su", "-c", "cmd wifi status 2>/dev/null | grep 'Wifi AP'").Output()
	if err == nil && strings.Contains(string(out), "enabled") {
		status.Enabled = true
	} else {
		// Fallback check
		out2, err2 := exec.Command("su", "-c", "ifconfig wlan1 2>/dev/null || ifconfig ap0 2>/dev/null").Output()
		if err2 == nil && strings.Contains(string(out2), "inet addr") {
			status.Enabled = true
		}
	}

	ssidOut, err := exec.Command("su", "-c", "settings get global softap_ssid 2>/dev/null").Output()
	if err == nil {
		status.SSID = strings.TrimSpace(string(ssidOut))
	}
	if status.SSID == "" || status.SSID == "null" {
		status.SSID = "AndroidAP"
	}

	return status
}

func ToggleHotspot(enable bool, ssid string, pass string) error {
	if ssid != "" {
		_ = exec.Command("su", "-c", fmt.Sprintf("settings put global softap_ssid \"%s\"", ssid)).Run()
	}
	if pass != "" {
		_ = exec.Command("su", "-c", fmt.Sprintf("settings put global softap_passphrase \"%s\"", pass)).Run()
	}

	var cmdStr string
	if enable {
		cmdStr = "cmd wifi start-softap 2>/dev/null || service call wifi 24 i32 1"
	} else {
		cmdStr = "cmd wifi stop-softap 2>/dev/null || service call wifi 24 i32 0"
	}

	out, err := exec.Command("su", "-c", cmdStr).CombinedOutput()
	if err != nil {
		return fmt.Errorf("hotspot error: %v, out: %s", err, string(out))
	}
	return nil
}

func GetConnectedClients() ([]ConnectedClient, error) {
	var clients []ConnectedClient

	// Try ip neigh show
	out, err := exec.Command("su", "-c", "ip neigh show 2>/dev/null").Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				ip := fields[0]
				var mac string
				state := fields[len(fields)-1]

				for i, f := range fields {
					if f == "lladdr" && i+1 < len(fields) {
						mac = fields[i+1]
						break
					}
				}

				if mac != "" && state != "FAILED" {
					clients = append(clients, ConnectedClient{
						IP:     ip,
						MAC:    mac,
						Device: resolveDeviceName(ip),
						State:  state,
					})
				}
			}
		}
		if len(clients) > 0 {
			return clients, nil
		}
	}

	// Fallback to /proc/net/arp
	file, err := os.Open("/proc/net/arp")
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		isFirst := true
		for scanner.Scan() {
			if isFirst {
				isFirst = false
				continue
			}
			fields := strings.Fields(scanner.Text())
			if len(fields) >= 6 {
				ip := fields[0]
				mac := fields[3]
				if mac != "00:00:00:00:00:00" {
					clients = append(clients, ConnectedClient{
						IP:     ip,
						MAC:    mac,
						Device: resolveDeviceName(ip),
						State:  "REACHABLE",
					})
				}
			}
		}
	}

	return clients, nil
}

func resolveDeviceName(ip string) string {
	// Check /data/misc/dhcp/dnsmasq.leases or gethostbyaddr
	out, err := exec.Command("su", "-c", fmt.Sprintf("grep '%s' /data/misc/dhcp/dnsmasq.leases 2>/dev/null", ip)).Output()
	if err == nil {
		fields := strings.Fields(string(out))
		if len(fields) >= 4 {
			name := fields[3]
			if name != "*" && name != "" {
				return name
			}
		}
	}
	return "Client-" + strings.ReplaceAll(ip, ".", "-")
}
