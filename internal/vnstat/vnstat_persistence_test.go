package vnstat

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetStatsDoesNotPersistOnEveryRead(t *testing.T) {
	tempDir := t.TempDir()
	dataPath := filepath.Join(tempDir, "vnstat_data.json")
	tracker := &Tracker{
		dataPath:  dataPath,
		daily:     make(map[string]InterfaceStat),
		yesterday: make(map[string]InterfaceStat),
		weekly:    make(map[string]InterfaceStat),
		monthly:   make(map[string]InterfaceStat),
		lastRaw:   parseProcNetDev(),
	}
	if _, err := os.Stat(dataPath); !os.IsNotExist(err) {
		t.Fatalf("expected no persisted file before reading stats, stat error = %v", err)
	}

	tracker.GetStats()

	if _, err := os.Stat(dataPath); !os.IsNotExist(err) {
		t.Fatalf("GetStats unexpectedly persisted data on read, stat error = %v", err)
	}
}
