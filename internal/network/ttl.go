package network

import (
	"fmt"
	"os/exec"
	"strconv"

	"bfr-webui-go/internal/config"
)

// GetDefaultTTL returns the recommended TTL value based on Android SDK level (65 for Android 11+, 64 otherwise).
func GetDefaultTTL() int {
	sdkStr := GetProp("ro.build.version.sdk")
	sdk, _ := strconv.Atoi(sdkStr)

	if sdk >= 30 { // Android 11+
		return 65
	}
	return 64
}

// SetTTLSpoofSDK configures dynamic TTL/HL iptables rules based on target TTL.
// SetTTLSpoofSDK configures dynamic TTL/HL iptables rules based on target TTL and saves configuration persistently.
func SetTTLSpoofSDK(enable bool, targetTTL int) error {
	// Remove existing rules
	_ = exec.Command(config.SUBin, "-c", "iptables -t mangle -D POSTROUTING -j TTL --ttl-set 64 2>/dev/null").Run()
	_ = exec.Command(config.SUBin, "-c", "ip6tables -t mangle -D POSTROUTING -j HL --hl-set 64 2>/dev/null").Run()

	if targetTTL <= 0 {
		targetTTL = GetDefaultTTL()
	}

	// Persist TTL state
	settings, _ := LoadNetworkSettings()
	settings.TTLEnabled = enable
	settings.TargetTTL = targetTTL
	_ = SaveNetworkSettings(settings)

	if !enable {
		return nil
	}

	cmdV4 := fmt.Sprintf("iptables -t mangle -A POSTROUTING -j TTL --ttl-set %d", targetTTL)
	cmdV6 := fmt.Sprintf("ip6tables -t mangle -A POSTROUTING -j HL --hl-set %d", targetTTL)

	if out, err := exec.Command(config.SUBin, "-c", cmdV4).CombinedOutput(); err != nil {
		return fmt.Errorf("iptables error: %v, output: %s", err, string(out))
	}
	_ = exec.Command(config.SUBin, "-c", cmdV6).Run()
	return nil
}
