package network

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"bfr-webui-go/internal/config"
)

var (
	// N-1: Regexes for RPS interface and bitmask validation
	reRPSIface  = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	reRPSBitmask = regexp.MustCompile(`^[0-9a-fA-F]+$`)
)

type RPSConfig struct {
	Interface string `json:"interface"`
	Bitmask   string `json:"bitmask"`
}

func SetTTLSpoofSDK(enable bool, targetTTL int) error {
	// Remove existing rules
	exec.Command(config.SUBin, "-c", "iptables -t mangle -D POSTROUTING -j TTL --ttl-set 64 2>/dev/null").Run()
	exec.Command(config.SUBin, "-c", "ip6tables -t mangle -D POSTROUTING -j HL --hl-set 64 2>/dev/null").Run()

	if !enable {
		return nil
	}

	if targetTTL <= 0 {
		sdkStr := GetProp("ro.build.version.sdk")
		sdk, _ := strconv.Atoi(sdkStr)

		if sdk >= 30 { // Android 11+
			targetTTL = 65
		} else {
			targetTTL = 64
		}
	}

	cmdV4 := fmt.Sprintf("iptables -t mangle -A POSTROUTING -j TTL --ttl-set %d", targetTTL)
	cmdV6 := fmt.Sprintf("ip6tables -t mangle -A POSTROUTING -j HL --hl-set %d", targetTTL)

	if out, err := exec.Command(config.SUBin, "-c", cmdV4).CombinedOutput(); err != nil {
		return fmt.Errorf("iptables error: %v, output: %s", err, string(out))
	}
	exec.Command(config.SUBin, "-c", cmdV6).Run()
	return nil
}

func ConfigureRPS(iface string, bitmask string) error {
	// N-1: validate iface and bitmask before shell execution
	if !reRPSIface.MatchString(iface) {
		return fmt.Errorf("invalid interface name: %s", iface)
	}
	if bitmask == "" {
		bitmask = "f" // Default all 4 cores or first quad
	}
	if !reRPSBitmask.MatchString(bitmask) {
		return fmt.Errorf("invalid bitmask: %s — must be a hexadecimal string", bitmask)
	}

	queues, err := filepath.Glob(fmt.Sprintf("/sys/class/net/%s/queues/rx-*/rps_cpus", iface))
	if err != nil || len(queues) == 0 {
		return fmt.Errorf("no RPS queue found for interface: %s", iface)
	}

	for _, q := range queues {
		cmdStr := fmt.Sprintf("echo %s > %s", bitmask, q)
		if out, err := exec.Command(config.SUBin, "-c", cmdStr).CombinedOutput(); err != nil {
			return fmt.Errorf("failed to set RPS for %s: %v, out: %s", q, err, string(out))
		}
	}
	return nil
}

func GetRPSConfigs() ([]RPSConfig, error) {
	ifaces, err := GetInterfaces()
	if err != nil {
		return nil, err
	}

	var configs []RPSConfig
	for _, ifc := range ifaces {
		queues, err := filepath.Glob(fmt.Sprintf("/sys/class/net/%s/queues/rx-0/rps_cpus", ifc.Name))
		if err == nil && len(queues) > 0 {
			data, err := os.ReadFile(queues[0])
			mask := ""
			if err == nil {
				mask = strings.TrimSpace(string(data))
			}
			configs = append(configs, RPSConfig{
				Interface: ifc.Name,
				Bitmask:   mask,
			})
		}
	}
	return configs, nil
}
