package hotspot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMACFilterEdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// Save nil
	if err := SaveMACFilterConfig(nil); err == nil {
		t.Error("expected error saving nil MACFilterConfig")
	}

	// Apply nil
	if err := ApplyMACFilter(nil); err == nil {
		t.Error("expected error applying nil config")
	}

	// Apply disabled
	if err := ApplyMACFilter(&MACFilterConfig{Mode: "disabled"}); err != nil {
		t.Logf("Apply disabled returned: %v", err)
	}

	// Apply whitelist
	_ = ApplyMACFilter(&MACFilterConfig{
		Mode:        "whitelist",
		AllowedMACs: []string{"aa:bb:cc:dd:ee:ff"},
	})

	// Apply blacklist
	_ = ApplyMACFilter(&MACFilterConfig{
		Mode:        "blacklist",
		BlockedMACs: []string{"11:22:33:44:55:66"},
	})

	// GetStatus
	status := GetMACFilterStatus()
	if status.ActiveMode == "" {
		t.Error("expected non-empty status mode")
	}
}

func TestResolveDeviceNameAndStatus(t *testing.T) {
	// Invalid IP
	if name := resolveDeviceName("not-an-ip"); name != "" {
		t.Errorf("expected empty string for invalid IP, got %s", name)
	}

	// Valid IP
	_ = resolveDeviceName("192.168.43.10")

	// GetHotspotStatus
	st := GetHotspotStatus()
	t.Logf("Hotspot status: %+v", st)
}

func TestParseIPNeighAndArp(t *testing.T) {
	sampleNeigh := "192.168.43.15 dev wlan1 lladdr 11:22:33:44:55:66 REACHABLE\n192.168.43.16 dev wlan1 lladdr 22:33:44:55:66:77 FAILED\n"
	clients := parseIPNeigh(sampleNeigh)
	if len(clients) != 1 {
		t.Fatalf("expected 1 valid client, got %d", len(clients))
	}
	if clients[0].IP != "192.168.43.15" || clients[0].MAC != "11:22:33:44:55:66" {
		t.Errorf("client mismatch: %+v", clients[0])
	}

	sampleArp := strings.NewReader("IP address HW type Flags HW address Mask Device\n192.168.43.10 0x1 0x2 aa:bb:cc:dd:ee:ff * wlan1\n192.168.43.11 0x1 0x0 00:00:00:00:00:00 * wlan1\n")
	arpClients := parseArp(sampleArp)
	if len(arpClients) != 1 {
		t.Fatalf("expected 1 valid arp client, got %d", len(arpClients))
	}
	if arpClients[0].IP != "192.168.43.10" {
		t.Errorf("arp client mismatch: %+v", arpClients[0])
	}

	// GetConnectedClients call
	allClients, err := GetConnectedClients()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Logf("GetConnectedClients returned %d clients", len(allClients))
}

func TestToggleHotspotValidation(t *testing.T) {
	t.Setenv("BFR_DATA_DIR", t.TempDir())

	// Invalid SSID
	if err := ToggleHotspot(true, "", "password123"); err == nil {
		t.Error("expected error for empty SSID")
	}
	if err := ToggleHotspot(true, "this_ssid_is_way_too_long_and_exceeds_thirty_two_chars", "password123"); err == nil {
		t.Error("expected error for long SSID")
	}
	if err := ToggleHotspot(true, "bad;ssid", "password123"); err == nil {
		t.Error("expected error for SSID with forbidden characters")
	}

	// Invalid password
	if err := ToggleHotspot(true, "ValidSSID", "1234"); err == nil {
		t.Error("expected error for short passphrase (<8 chars)")
	}
	if err := ToggleHotspot(true, "ValidSSID", "invalid;pass"); err == nil {
		t.Error("expected error for passphrase with forbidden characters")
	}

	// Valid toggle disabled
	_ = ToggleHotspot(false, "ValidSSID", "")
}

func TestRouterConfigStore_ColdStartAOSPFallbackAndPointerIsolation(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	xmlPath := filepath.Join(tempDir, "WifiConfigStoreSoftAp.xml")
	mockXML := `<?xml version='1.0' encoding='utf-8' standalone='yes' ?>
<WifiConfigStoreData>
<SoftAp>
<string name="WifiSsid">&quot;BFR_Router_5G&quot;</string>
<int name="SecurityType" value="1" />
<string name="Passphrase">MyStrongPass99</string>
</SoftAp>
</WifiConfigStoreData>`
	if err := os.WriteFile(xmlPath, []byte(mockXML), 0644); err != nil {
		t.Fatalf("failed to write mock AOSP XML: %v", err)
	}

	origPaths := softApXMLCandidatePaths
	softApXMLCandidatePaths = []string{xmlPath}
	defer func() { softApXMLCandidatePaths = origPaths }()

	// 1. Cold start without hotspot_settings.json must read genuine AOSP XML credentials
	cfg, err := LoadRouterConfig()
	if err != nil {
		t.Fatalf("LoadRouterConfig failed: %v", err)
	}
	if cfg.SSID != "BFR_Router_5G" || cfg.Password != "MyStrongPass99" || cfg.SecurityType != "wpa2" {
		t.Fatalf("expected AOSP credentials (BFR_Router_5G / MyStrongPass99 / wpa2), got (%s / %s / %s)",
			cfg.SSID, cfg.Password, cfg.SecurityType)
	}

	// 2. Mutating caller struct must not mutate cachedCfg
	cfg.SSID = "MutatedInCaller"
	reloaded, _ := LoadRouterConfig()
	if reloaded.SSID != "BFR_Router_5G" {
		t.Fatalf("cachedCfg was mutated through caller pointer: got %s", reloaded.SSID)
	}

	// 3. Saving gateway settings with omitted SSID/Password must preserve existing credentials
	gatewayOnly := DefaultRouterConfig()
	gatewayOnly.SSID = ""
	gatewayOnly.Password = ""
	gatewayOnly.SecurityType = ""
	gatewayOnly.AutoStartWifi = true
	if err := SaveRouterConfig(&gatewayOnly); err != nil {
		t.Fatalf("SaveRouterConfig failed: %v", err)
	}

	afterSave, _ := LoadRouterConfig()
	if afterSave.SSID != "BFR_Router_5G" || afterSave.Password != "MyStrongPass99" || !afterSave.AutoStartWifi {
		t.Fatalf("expected preserved credentials and AutoStartWifi=true, got %+v", afterSave)
	}
}

func TestIsProcNetRouteWAN(t *testing.T) {
	// Case 1: eth0 connected to a router (default route 00000000 via gateway 0101A8C0)
	wanRouteTable := strings.Join([]string{
		"Iface\tDestination\tGateway \tFlags\tRefCnt\tUse\tMetric\tMask\t\tMTU\tWindow\tIRTT",
		"eth0\t00000000\t0101A8C0\t0003\t0\t0\t100\t00000000\t0\t0\t0",
		"eth0\t0001A8C0\t00000000\t0001\t0\t0\t100\t00FFFFFF\t0\t0\t0",
	}, "\n")
	if !isProcNetRouteWAN("eth0", wanRouteTable) {
		t.Error("expected eth0 with default gateway route to be detected as WAN upstream")
	}

	// Case 2: eth0 acting as downstream tethering server (only local subnet route, gateway 00000000)
	tetherRouteTable := strings.Join([]string{
		"Iface\tDestination\tGateway \tFlags\tRefCnt\tUse\tMetric\tMask\t\tMTU\tWindow\tIRTT",
		"eth0\t002AA8C0\t00000000\t0001\t0\t0\t0\t00FFFFFF\t0\t0\t0",
	}, "\n")
	if isProcNetRouteWAN("eth0", tetherRouteTable) {
		t.Error("expected eth0 without default gateway route to be treated as downstream tether")
	}
}
