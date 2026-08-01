package hotspot

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"bfr-webui-go/internal/config"
)

var (
	// H-6: SSID and Passphrase validation regexes
	reHotspotSSID       = regexp.MustCompile(`^[a-zA-Z0-9 _\-]{1,32}$`)
	reHotspotPassphrase = regexp.MustCompile(`^[a-zA-Z0-9!@#$%^&*()_+\-=\[\]{}|:<>,.?/~]{8,63}$`)
)

type ConnectedClient struct {
	IP     string `json:"ip"`
	MAC    string `json:"mac"`
	Device string `json:"device"`
	State  string `json:"state"`
}

type HotspotStatus struct {
	Enabled bool   `json:"enabled"`
	SSID    string `json:"ssid"`
}

func GetHotspotStatus() HotspotStatus {
	var status HotspotStatus

	out, err := exec.Command(config.SUBin, "-c", "cmd wifi status 2>/dev/null | grep 'Wifi AP'").Output()
	if err == nil && strings.Contains(string(out), "enabled") {
		status.Enabled = true
	} else {
		// Fallback check
		out2, err2 := exec.Command(config.SUBin, "-c", "ifconfig wlan1 2>/dev/null || ifconfig ap0 2>/dev/null").Output()
		if err2 == nil && strings.Contains(string(out2), "inet addr") {
			status.Enabled = true
		}
	}

	ssidOut, err := exec.Command(config.SUBin, "-c", "settings get global softap_ssid 2>/dev/null").Output()
	if err == nil {
		status.SSID = strings.TrimSpace(string(ssidOut))
	}
	if status.SSID == "" || status.SSID == "null" {
		status.SSID = "AndroidAP"
	}

	return status
}

func ToggleHotspot(enable bool, ssid string, pass string) error {
	// H-6: Validate SSID and passphrase against strict regex patterns
	if ssid != "" {
		if !reHotspotSSID.MatchString(ssid) {
			return fmt.Errorf("invalid SSID: must be 1-32 alphanumeric characters, spaces, underscores, or hyphens")
		}
		_ = exec.Command(config.SUBin, "-c", fmt.Sprintf("settings put global softap_ssid \"%s\"", ssid)).Run()
	}
	if pass != "" {
		if !reHotspotPassphrase.MatchString(pass) {
			return fmt.Errorf("invalid passphrase: must be 8-63 valid characters")
		}
		_ = exec.Command(config.SUBin, "-c", fmt.Sprintf("settings put global softap_passphrase \"%s\"", pass)).Run()
	}

	var cmdStr string
	if enable {
		cmdStr = "cmd wifi start-softap 2>/dev/null || service call wifi 24 i32 1"
	} else {
		cmdStr = "cmd wifi stop-softap 2>/dev/null || service call wifi 24 i32 0"
	}

	out, err := exec.Command(config.SUBin, "-c", cmdStr).CombinedOutput()
	if err != nil {
		return fmt.Errorf("hotspot error: %v, out: %s", err, string(out))
	}
	return nil
}

func GetConnectedClients() ([]ConnectedClient, error) {
	var clients []ConnectedClient

	// Try ip neigh show
	out, err := exec.Command(config.SUBin, "-c", "ip neigh show 2>/dev/null").Output()
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
	// B-2: Validate IP before putting it into a shell grep command
	if net.ParseIP(ip) == nil {
		return ""
	}
	out, err := exec.Command(config.SUBin, "-c", fmt.Sprintf("grep '%s' %s 2>/dev/null", ip, config.LeasesFile)).Output()
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
