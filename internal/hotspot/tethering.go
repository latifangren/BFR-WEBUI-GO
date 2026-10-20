package hotspot

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

var (
	tetherMu               sync.Mutex
	onTetheringStateChange func(activeIface string)
)

// SetTetheringStateChangeHook registers a callback triggered when tethering state changes.
func SetTetheringStateChangeHook(fn func(activeIface string)) {
	tetherMu.Lock()
	defer tetherMu.Unlock()
	onTetheringStateChange = fn
}

func notifyTetheringChange() {
	tetherMu.Lock()
	fn := onTetheringStateChange
	tetherMu.Unlock()

	names := GetActiveInterfaceNames()
	primary := ""
	if len(names) > 0 {
		primary = names[0]
	} else {
		// No tether interface is up: remove any BFR-owned gateway state.
		RemoveStaticGateway("")
	}

	if fn != nil {
		fn(primary)
	}
}

// ToggleTethering toggles specific tethering channel ("wifi", "usb", "ethernet").
func ToggleTethering(tetherType string, enable bool) error {
	tetherType = strings.ToLower(strings.TrimSpace(tetherType))

	switch tetherType {
	case "wifi", "softap":
		return ToggleHotspot(enable, "", "")
	case "usb", "rndis":
		return toggleUSBTethering(enable)
	case "ethernet", "eth":
		return toggleEthernetTethering(enable)
	default:
		return fmt.Errorf("unsupported tethering type: %s", tetherType)
	}
}

// toggleUSBTethering toggles USB RNDIS/NCM tethering mode.
func toggleUSBTethering(enable bool) error {
	var cmdStr string
	if enable {
		// Tier 1: cmd tethering start usb
		// Tier 2: svc usb setFunctions rndis
		// Tier 3: setprop sys.usb.config rndis,adb
		cmdStr = "cmd tethering start usb 2>/dev/null || svc usb setFunctions rndis 2>/dev/null || setprop sys.usb.config rndis,adb"
	} else {
		cmdStr = "cmd tethering stop usb 2>/dev/null || svc usb setFunctions mtp 2>/dev/null || setprop sys.usb.config mtp,adb"
	}

	out, err := config.ExecSuTimeout(5*time.Second, cmdStr)
	if err != nil {
		logger.Get().Warnf("hotspot", "toggle USB tethering error: %v, out: %s", err, string(out))
		return fmt.Errorf("failed to toggle USB tethering: %w", err)
	}

	time.Sleep(1 * time.Second)
	notifyTetheringChange()
	return nil
}

// toggleEthernetTethering toggles Ethernet OTG tethering.
func toggleEthernetTethering(enable bool) error {
	var cmdStr string
	if enable {
		cmdStr = "cmd tethering start ethernet 2>/dev/null"
	} else {
		cmdStr = "cmd tethering stop ethernet 2>/dev/null"
	}

	out, err := config.ExecSuTimeout(5*time.Second, cmdStr)
	if err != nil {
		return fmt.Errorf("failed to toggle Ethernet tethering: %w, out: %s", err, string(out))
	}

	time.Sleep(1 * time.Second)
	notifyTetheringChange()
	return nil
}

// UpdateSavedSoftApCredentials persists new SSID, Passphrase and SecurityType to AOSP XML stores.
func UpdateSavedSoftApCredentials(newSSID, newPass, securityType string) error {
	candidatePaths := []string{
		"/data/misc_ce/0/apexdata/com.android.wifi/WifiConfigStoreSoftAp.xml",
		"/data/misc/apexdata/com.android.wifi/WifiConfigStoreSoftAp.xml",
		"/data/misc/wifi/WifiConfigStoreSoftAp.xml",
	}

	for _, p := range candidatePaths {
		if newSSID != "" {
			cleanSSID := strings.ReplaceAll(newSSID, "\"", "")
			cmdSSID := fmt.Sprintf("sed -i 's|<string name=\"WifiSsid\">.*</string>|<string name=\"WifiSsid\">&quot;%s&quot;</string>|' %s 2>/dev/null", cleanSSID, p)
			_, _ = config.ExecSuTimeout(2*time.Second, cmdSSID)
		}

		if securityType == "open" || newPass == "none" {
			cmdSec := fmt.Sprintf("sed -i 's|<int name=\"SecurityType\" value=\".*\" />|<int name=\"SecurityType\" value=\"0\" />|' %s 2>/dev/null", p)
			cmdPass := fmt.Sprintf("sed -i 's|<string name=\"Passphrase\">.*</string>|<string name=\"Passphrase\"></string>|' %s 2>/dev/null", p)
			_, _ = config.ExecSuTimeout(2*time.Second, cmdSec)
			_, _ = config.ExecSuTimeout(2*time.Second, cmdPass)
		} else if newPass != "" {
			cmdSec := fmt.Sprintf("sed -i 's|<int name=\"SecurityType\" value=\".*\" />|<int name=\"SecurityType\" value=\"1\" />|' %s 2>/dev/null", p)
			cmdPass := fmt.Sprintf("sed -i 's|<string name=\"Passphrase\">.*</string>|<string name=\"Passphrase\">%s</string>|' %s 2>/dev/null", newPass, p)
			_, _ = config.ExecSuTimeout(2*time.Second, cmdSec)
			_, _ = config.ExecSuTimeout(2*time.Second, cmdPass)
		}
	}

	if newSSID != "" {
		_, _ = config.ExecSuTimeout(2*time.Second, fmt.Sprintf("settings put global softap_ssid \"%s\"", newSSID))
	}
	if securityType == "open" || newPass == "none" {
		_, _ = config.ExecSuTimeout(2*time.Second, "settings put global softap_passphrase \"\"")
	} else if newPass != "" {
		_, _ = config.ExecSuTimeout(2*time.Second, fmt.Sprintf("settings put global softap_passphrase \"%s\"", newPass))
	}

	return nil
}

// ToggleHotspot enables or disables Wi-Fi Hotspot with persistent credential updates.
func ToggleHotspot(enable bool, ssid string, pass string, security ...string) error {
	// 1. If disabling, stop the radio and clean up BFR-owned gateway state
	if !enable {
		// Remove the BFR-owned address and rules first: once the radio stops,
		// the interface disappears and the objects can no longer be addressed.
		RemoveStaticGateway("")
		_, _ = config.ExecSuTimeout(5*time.Second, "cmd wifi stop-softap 2>/dev/null || cmd tethering stop wifi 2>/dev/null")
		go func() {
			time.Sleep(1 * time.Second)
			notifyTetheringChange()
		}()
		return nil
	}
	// 2. If enabling, resolve credentials & router config
	savedSSID, savedPass, savedSec := ReadSavedSoftApCredentials()
	routerCfg, _ := LoadRouterConfig()

	if ssid != "" {
		if !reHotspotSSID.MatchString(ssid) {
			return fmt.Errorf("invalid SSID: must be 1-32 alphanumeric characters, spaces, underscores, or hyphens")
		}
	} else {
		ssid = savedSSID
	}

	secType := savedSec
	if len(security) > 0 && security[0] != "" {
		secType = strings.ToLower(security[0])
	}

	if secType == "open" || pass == "none" {
		secType = "open"
		pass = "none"
	} else {
		secType = "wpa2"
		if pass != "" {
			if !reHotspotPassphrase.MatchString(pass) {
				return fmt.Errorf("invalid passphrase: must be 8-63 valid characters")
			}
		} else {
			pass = savedPass
			if pass == "" {
				pass = "12345678"
			}
		}
	}

	// Persist to XML store so reboot / UI stays in sync
	_ = UpdateSavedSoftApCredentials(ssid, pass, secType)

	// 3. Resolve Band flag (-b 2, -b 5, or omitted for any)
	bandFlag := ""
	if routerCfg != nil && routerCfg.Band != "" && routerCfg.Band != "any" {
		bandFlag = fmt.Sprintf("-b %s", routerCfg.Band)
	}

	// 4. Start SoftAP with explicit parameters
	var cmdStr string
	if secType == "open" {
		cmdStr = strings.TrimSpace(fmt.Sprintf("cmd wifi start-softap \"%s\" open none %s", ssid, bandFlag))
	} else {
		cmdStr = strings.TrimSpace(fmt.Sprintf("cmd wifi start-softap \"%s\" wpa2 \"%s\" %s", ssid, pass, bandFlag))
	}

	out, err := config.ExecSuTimeout(7*time.Second, cmdStr)
	outStr := string(out)

	if err != nil && !strings.Contains(outStr, "enabled successfully") && !strings.Contains(outStr, "SAP is enabled") {
		var cmdStrFallback string
		if secType == "open" {
			cmdStrFallback = strings.TrimSpace(fmt.Sprintf("cmd wifi start-softap %s open none %s", ssid, bandFlag))
		} else {
			cmdStrFallback = strings.TrimSpace(fmt.Sprintf("cmd wifi start-softap %s wpa2 %s %s", ssid, pass, bandFlag))
		}
		out2, err2 := config.ExecSuTimeout(7*time.Second, cmdStrFallback)
		if err2 != nil && !strings.Contains(string(out2), "enabled") {
			return fmt.Errorf("failed to start softap: %v (out: %s)", err2, string(out2))
		}
	}

	// 5. Post-startup: Apply static gateway only when in static mode.
	go func() {
		time.Sleep(2 * time.Second)
		if routerCfg != nil && routerCfg.GatewayMode == "static" {
			names := GetActiveInterfaceNames()
			if len(names) > 0 {
				_ = ApplyStaticGateway(names[0], routerCfg)
			}
		}
		notifyTetheringChange()
	}()

	return nil
}
