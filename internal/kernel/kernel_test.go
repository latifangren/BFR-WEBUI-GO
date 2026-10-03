package kernel

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFrequencies(t *testing.T) {
	str := "300000 576000 1200000 1800000"
	freqs := parseFrequencies(str)
	if len(freqs) != 4 {
		t.Fatalf("expected 4 frequencies, got %d", len(freqs))
	}
	if freqs[0] != 300000 || freqs[3] != 1800000 {
		t.Errorf("unexpected frequency values: %v", freqs)
	}

	empty := parseFrequencies("")
	if len(empty) != 0 {
		t.Errorf("expected 0 frequencies for empty string, got %d", len(empty))
	}
}

func TestFormatClusterLabel(t *testing.T) {
	if formatClusterLabel(0, 1) != "All Cores" {
		t.Errorf("expected 'All Cores', got '%s'", formatClusterLabel(0, 1))
	}
	if formatClusterLabel(0, 2) != "Little (Efficiency)" {
		t.Errorf("expected 'Little (Efficiency)', got '%s'", formatClusterLabel(0, 2))
	}
	if formatClusterLabel(1, 2) != "Big (Performance)" {
		t.Errorf("expected 'Big (Performance)', got '%s'", formatClusterLabel(1, 2))
	}
	if formatClusterLabel(2, 3) != "Prime (Gold/Max)" {
		t.Errorf("expected 'Prime (Gold/Max)', got '%s'", formatClusterLabel(2, 3))
	}
}

func TestAvailableProfiles(t *testing.T) {
	profiles := []string{"powersave", "balanced", "performance"}
	for _, p := range profiles {
		prof, ok := AvailableProfiles[p]
		if !ok {
			t.Fatalf("expected profile %s to exist", p)
		}
		if prof.Governor == "" {
			t.Errorf("profile %s missing governor", p)
		}
		if prof.Swappiness < 0 || prof.Swappiness > 200 {
			t.Errorf("invalid swappiness for %s: %d", p, prof.Swappiness)
		}
	}
}

func TestManagerLoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tmpDir)

	mgr := &Manager{
		configPath: filepath.Join(tmpDir, "kernel_config.json"),
	}

	status, err := mgr.GetStatus()
	if err != nil {
		t.Fatalf("unexpected error from GetStatus: %v", err)
	}
	if status.KernelRelease == "" {
		t.Errorf("expected non-empty KernelRelease")
	}

	// Test saving a profile
	err = mgr.ApplyProfile("balanced")
	if err != nil {
		t.Fatalf("unexpected error from ApplyProfile: %v", err)
	}

	// Verify file was written to disk
	if _, err := os.Stat(mgr.configPath); os.IsNotExist(err) {
		t.Fatalf("expected kernel_config.json to exist at %s", mgr.configPath)
	}

	// Re-load
	err = mgr.loadConfig()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	if mgr.cachedCfg.ActiveProfile != "balanced" {
		t.Errorf("expected active profile 'balanced', got '%s'", mgr.cachedCfg.ActiveProfile)
	}
	if mgr.cachedCfg.GlobalGovernor != "schedutil" {
		t.Errorf("expected global governor 'schedutil', got '%s'", mgr.cachedCfg.GlobalGovernor)
	}
}
