package hotspot_test

import (
	"testing"

	"bfr-webui-go/internal/hotspot"
)

func TestLoadAndSaveMACFilterConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	cfg, err := hotspot.LoadMACFilterConfig()
	if err != nil {
		t.Fatalf("unexpected error loading MAC filter config: %v", err)
	}

	if cfg.Mode != "disabled" {
		t.Errorf("expected default mode 'disabled', got %s", cfg.Mode)
	}

	// Update and Save config
	newCfg := &hotspot.MACFilterConfig{
		Mode:        "blacklist",
		BlockedMACs: []string{"00:11:22:33:44:55", "AA:BB:CC:DD:EE:FF"},
		AllowedMACs: []string{"11:22:33:44:55:66"},
	}

	if err := hotspot.SaveMACFilterConfig(newCfg); err != nil {
		t.Fatalf("failed to save MAC filter config: %v", err)
	}

	reloaded, err := hotspot.LoadMACFilterConfig()
	if err != nil {
		t.Fatalf("failed to reload MAC filter config: %v", err)
	}

	if reloaded.Mode != "blacklist" {
		t.Errorf("expected mode 'blacklist', got %s", reloaded.Mode)
	}
	if len(reloaded.BlockedMACs) != 2 {
		t.Errorf("expected 2 blocked MACs, got %d", len(reloaded.BlockedMACs))
	}
}

func TestApplyAndClearMACFilter(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// Apply blacklist filter
	cfgBlacklist := &hotspot.MACFilterConfig{
		Mode:        "blacklist",
		BlockedMACs: []string{"00:11:22:33:44:55"},
		AllowedMACs: []string{},
	}

	_ = hotspot.ApplyMACFilter(cfgBlacklist)
	status := hotspot.GetMACFilterStatus()
	if status.ActiveMode != "blacklist" {
		t.Errorf("expected ActiveMode 'blacklist', got %s", status.ActiveMode)
	}
	if status.BlockedCount != 1 {
		t.Errorf("expected BlockedCount 1, got %d", status.BlockedCount)
	}

	// Apply whitelist filter
	cfgWhitelist := &hotspot.MACFilterConfig{
		Mode:        "whitelist",
		BlockedMACs: []string{},
		AllowedMACs: []string{"11:22:33:44:55:66"},
	}

	_ = hotspot.ApplyMACFilter(cfgWhitelist)
	statusWhitelist := hotspot.GetMACFilterStatus()
	if statusWhitelist.ActiveMode != "whitelist" {
		t.Errorf("expected ActiveMode 'whitelist', got %s", statusWhitelist.ActiveMode)
	}

	// Clear filter
	errClear := hotspot.ClearMACFilter()
	if errClear != nil {
		t.Fatalf("unexpected error clearing MAC filter: %v", errClear)
	}

	statusCleared := hotspot.GetMACFilterStatus()
	if statusCleared.RulesCount != 0 {
		t.Errorf("expected RulesCount 0 after clearing, got %d", statusCleared.RulesCount)
	}
}
