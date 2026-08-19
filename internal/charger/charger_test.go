package charger_test

import (
	"testing"

	"bfr-webui-go/internal/charger"
)

func TestGetManager(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	mgr := charger.GetManager()
	if mgr == nil {
		t.Fatalf("expected non-nil charger manager")
	}
}

func TestGetStatus(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	mgr := charger.GetManager()
	status := mgr.GetStatus()

	if status.Config.StartPercent <= 0 {
		t.Errorf("expected positive start percent")
	}
	if status.Config.StopPercent <= 0 {
		t.Errorf("expected positive stop percent")
	}
}

func TestUpdateConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	mgr := charger.GetManager()

	// Update config with valid values
	status := mgr.UpdateConfig(charger.Config{
		Enabled:      true,
		StartPercent: 75,
		StopPercent:  85,
	})

	if status.Config.StartPercent != 75 || status.Config.StopPercent != 85 {
		t.Errorf("expected config updated to 75-85, got %d-%d", status.Config.StartPercent, status.Config.StopPercent)
	}

	// Update with non-positive fallback triggers defaults
	statusDefault := mgr.UpdateConfig(charger.Config{
		Enabled:      true,
		StartPercent: 0,
		StopPercent:  0,
	})
	if statusDefault.Config.StartPercent != 70 || statusDefault.Config.StopPercent != 80 {
		t.Errorf("expected fallback defaults 70-80, got %d-%d", statusDefault.Config.StartPercent, statusDefault.Config.StopPercent)
	}
}
