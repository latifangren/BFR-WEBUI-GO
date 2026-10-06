package network_test

import (
	"testing"

	"bfr-webui-go/internal/network"
)

func TestGetInterfaces(t *testing.T) {
	ifaces, err := network.GetInterfaces()
	if err != nil {
		t.Fatalf("unexpected error getting network interfaces: %v", err)
	}
	// In test environment, ifaces slice should be non-nil
	if ifaces == nil {
		t.Errorf("expected non-nil interfaces slice")
	}
}

func TestGetSysctlKeyValidation(t *testing.T) {
	// Key containing invalid shell characters should fail
	_, err := network.GetSysctl("net.ipv4.tcp_fastopen; rm -rf /")
	if err == nil {
		t.Errorf("expected error for invalid sysctl key with command injection attempts")
	}
}

func TestGetDefaultTTL(t *testing.T) {
	ttl := network.GetDefaultTTL()
	if ttl != 64 && ttl != 65 {
		t.Errorf("expected default TTL to be 64 or 65, got %d", ttl)
	}
}

func TestNetworkSettingsPersistence(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	settings, err := network.LoadNetworkSettings()
	if err != nil {
		t.Fatalf("unexpected error loading empty settings: %v", err)
	}

	settings.TTLEnabled = true
	settings.TargetTTL = 65
	settings.DNSEnabled = true
	settings.DNSPrimary = "1.1.1.1"
	settings.DNSSecondary = "1.0.0.1"

	if err := network.SaveNetworkSettings(settings); err != nil {
		t.Fatalf("failed to save network settings: %v", err)
	}

	reloaded, err := network.LoadNetworkSettings()
	if err != nil {
		t.Fatalf("failed to reload network settings: %v", err)
	}
	if !reloaded.TTLEnabled || reloaded.TargetTTL != 65 {
		t.Errorf("expected TTL enabled 65, got %+v", reloaded)
	}
	if !reloaded.DNSEnabled || reloaded.DNSPrimary != "1.1.1.1" {
		t.Errorf("expected DNS primary 1.1.1.1, got %+v", reloaded)
	}
}

func TestApplyAllTweaks(t *testing.T) {
	if err := network.ApplyAllTweaks(); err != nil {
		t.Errorf("expected ApplyAllTweaks to return nil, got %v", err)
	}
}
