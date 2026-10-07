package hotspot

import (
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
