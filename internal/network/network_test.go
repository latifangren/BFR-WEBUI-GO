package network_test

import (
	"testing"

	"bfr-webui-go/internal/network"
)

func TestLoadAndSaveTweaks(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// Load tweaks when file doesn't exist returns default config
	cfg, err := network.LoadTweaks()
	if err != nil {
		t.Fatalf("unexpected error loading non-existent tweaks: %v", err)
	}

	// Modify config and save
	cfg.LTECarrierAggregation = true
	cfg.BBR2CongestionControl = true
	cfg.TCPBufferOptimization = true

	if err := network.SaveTweaks(cfg); err != nil {
		t.Fatalf("failed to save tweaks: %v", err)
	}

	// Reload and verify persistence
	reloaded, err := network.LoadTweaks()
	if err != nil {
		t.Fatalf("failed to reload tweaks: %v", err)
	}

	if !reloaded.LTECarrierAggregation {
		t.Errorf("expected LTECarrierAggregation to be true")
	}
	if !reloaded.BBR2CongestionControl {
		t.Errorf("expected BBR2CongestionControl to be true")
	}
}

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

func TestApplyAllTweaks(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// Save a test configuration
	_ = network.SaveTweaks(network.TweaksConfig{
		LTECarrierAggregation: true,
		BBR2CongestionControl: true,
	})

	// Run ApplyAllTweaks (which executes su commands; should not crash)
	_ = network.ApplyAllTweaks()
}
