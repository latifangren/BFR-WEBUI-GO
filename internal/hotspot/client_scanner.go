package hotspot

import (
	"bufio"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"bfr-webui-go/internal/config"
)

// isAllowedTetherIface verifies if an interface belongs to tethering channels.
func isAllowedTetherIface(iface string) bool {
	iface = strings.ToLower(strings.TrimSpace(iface))
	if iface == "" {
		return true // Allow if unspecified for unit testing
	}
	// Explicitly reject non-tether interfaces: wlan0 (Wi-Fi STA client), ds-br0 (container bridge), rmnet (cellular WAN), lo
	if iface == "wlan0" || strings.HasPrefix(iface, "ds-") || strings.HasPrefix(iface, "rmnet") || strings.HasPrefix(iface, "dummy") || iface == "lo" {
		return false
	}
	for _, cand := range wifiIfaceCandidates {
		if strings.HasPrefix(iface, cand) {
			return true
		}
	}
	for _, cand := range usbIfaceCandidates {
		if strings.HasPrefix(iface, cand) {
			return true
		}
	}
	for _, cand := range ethernetIfaceCandidates {
		if strings.HasPrefix(iface, cand) {
			return true
		}
	}
	return false
}

// parseIPNeigh parses output of "ip neigh show" into ConnectedClient list.
func parseIPNeigh(out string) []ConnectedClient {
	var clients []ConnectedClient
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			ip := fields[0]
			var mac string
			var dev string
			state := fields[len(fields)-1]

			for i, f := range fields {
				if f == "lladdr" && i+1 < len(fields) {
					mac = fields[i+1]
				}
				if f == "dev" && i+1 < len(fields) {
					dev = fields[i+1]
				}
			}

			// Exclude FAILED, INCOMPLETE, blank MACs, or non-tether interfaces (wlan0, ds-br0)
			if mac != "" && state != "FAILED" && state != "INCOMPLETE" && isAllowedTetherIface(dev) {
				clients = append(clients, ConnectedClient{
					IP:        ip,
					MAC:       mac,
					Device:    resolveDeviceName(ip),
					State:     state,
					Interface: dev,
				})
			}
		}
	}
	return clients
}

// parseArp parses /proc/net/arp contents into ConnectedClient list.
func parseArp(r io.Reader) []ConnectedClient {
	var clients []ConnectedClient
	scanner := bufio.NewScanner(r)
	isFirst := true
	for scanner.Scan() {
		if isFirst {
			isFirst = false
			continue
		}
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 6 {
			ip := fields[0]
			flags := fields[2]
			mac := fields[3]
			dev := fields[5]

			if mac != "00:00:00:00:00:00" && flags == "0x2" && isAllowedTetherIface(dev) {
				clients = append(clients, ConnectedClient{
					IP:        ip,
					MAC:       mac,
					Device:    resolveDeviceName(ip),
					State:     "REACHABLE",
					Interface: dev,
				})
			}
		}
	}
	return clients
}

// resolveDeviceName looks up device hostname from leases cache or fallback format.
func resolveDeviceName(ip string) string {
	if net.ParseIP(ip) == nil {
		return ""
	}

	// Read lease file directly if accessible without spawning root shell per client
	if data, err := os.ReadFile(config.LeasesFile); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 4 && fields[2] == ip {
				name := fields[3]
				if name != "*" && name != "" {
					return name
				}
			}
		}
	} else {
		// Fallback root grep only when direct read fails
		out, err := config.ExecSuTimeout(1*time.Second, "grep ' "+ip+" ' "+config.LeasesFile+" 2>/dev/null")
		if err == nil {
			fields := strings.Fields(string(out))
			if len(fields) >= 4 {
				name := fields[3]
				if name != "*" && name != "" {
					return name
				}
			}
		}
	}

	return "Client-" + strings.ReplaceAll(ip, ".", "-")
}

// GetConnectedClients returns list of clients connected across active tethering interfaces.
func GetConnectedClients() ([]ConnectedClient, error) {
	activeIfaces := GetActiveTetherInterfaces()
	if len(activeIfaces) == 0 {
		// Strict isolation: When Hotspot / Tethering is OFF, return strictly 0 clients
		return []ConnectedClient{}, nil
	}

	var allClients []ConnectedClient
	seenIPs := make(map[string]bool)

	// Scoped scanning for each active tether interface only
	for _, ifc := range activeIfaces {
		if !isAllowedTetherIface(ifc.Name) {
			continue
		}
		out, err := config.ExecSuTimeout(2*time.Second, "ip neigh show dev "+ifc.Name+" 2>/dev/null")
		if err == nil && len(strings.TrimSpace(string(out))) > 0 {
			clients := parseIPNeigh(string(out))
			for _, c := range clients {
				if !seenIPs[c.IP] && isAllowedTetherIface(c.Interface) {
					seenIPs[c.IP] = true
					if c.Interface == "" {
						c.Interface = ifc.Name
					}
					allClients = append(allClients, c)
				}
			}
		}
	}

	return allClients, nil
}
