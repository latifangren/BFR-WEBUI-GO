package vnstat_test

import (
	"testing"

	"bfr-webui-go/internal/vnstat"
)

func TestGetTracker(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	tracker := vnstat.GetTracker()
	if tracker == nil {
		t.Fatalf("expected non-nil tracker instance")
	}
}

func TestGetStats(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	stats := vnstat.GetStats()
	if stats.Daily.Interfaces == nil {
		t.Errorf("expected Daily.Interfaces map to be initialized")
	}
	if stats.Monthly.Interfaces == nil {
		t.Errorf("expected Monthly.Interfaces map to be initialized")
	}
}

func TestResetStats(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	stats := vnstat.ResetStats()
	if stats.Daily.Interfaces == nil {
		t.Errorf("expected Daily.Interfaces map to be initialized after reset")
	}
	if stats.Monthly.Interfaces == nil {
		t.Errorf("expected Monthly.Interfaces map to be initialized after reset")
	}
}
