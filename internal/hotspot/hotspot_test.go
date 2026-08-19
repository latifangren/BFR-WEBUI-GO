package hotspot_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/hotspot"
)

func TestToggleHotspot_Validation(t *testing.T) {
	// Test invalid SSID
	err := hotspot.ToggleHotspot(true, "invalid_ssid_longer_than_32_characters_long!!!!!", "validPass123")
	if err == nil {
		t.Errorf("expected error for SSID longer than 32 chars")
	}

	// Test invalid Passphrase
	err = hotspot.ToggleHotspot(true, "validSSID", "short")
	if err == nil {
		t.Errorf("expected error for passphrase shorter than 8 chars")
	}
}

func TestGetConnectedClients(t *testing.T) {
	clients, err := hotspot.GetConnectedClients()
	if err != nil {
		t.Fatalf("unexpected error fetching connected clients: %v", err)
	}
	// GetConnectedClients returns nil or empty slice when no clients connected
	_ = clients
}

func TestDnsmasqLeasesParsingMock(t *testing.T) {
	tempDir := t.TempDir()
	leasesFile := filepath.Join(tempDir, "dnsmasq.leases")

	// Sample dnsmasq.leases format:
	// timestamp mac ip hostname client-id
	leasesContent := `1700000000 00:11:22:33:44:55 192.168.43.100 android-device *
1700000000 66:77:88:99:aa:bb 192.168.43.101 * *
`
	if err := os.WriteFile(leasesFile, []byte(leasesContent), 0644); err != nil {
		t.Fatalf("failed to write mock leases file: %v", err)
	}

	originalLeasesFile := config.LeasesFile
	config.LeasesFile = leasesFile
	defer func() { config.LeasesFile = originalLeasesFile }()

	data, err := os.ReadFile(config.LeasesFile)
	if err != nil {
		t.Fatalf("failed to read configured leases file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines in leases file, got %d", len(lines))
	}

	// Line 1: Hostname present
	fields1 := strings.Fields(lines[0])
	if len(fields1) < 4 || fields1[3] != "android-device" {
		t.Errorf("expected hostname android-device, got %v", fields1)
	}

	// Line 2: Hostname is "*"
	fields2 := strings.Fields(lines[1])
	if len(fields2) < 4 || fields2[3] != "*" {
		t.Errorf("expected hostname *, got %v", fields2)
	}
}
