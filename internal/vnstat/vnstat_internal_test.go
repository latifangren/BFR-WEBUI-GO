package vnstat

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInternalTrackerCorruptedAndSave(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)
	dataPath := filepath.Join(tempDir, "vnstat_data.json")

	// Corrupted file
	_ = os.WriteFile(dataPath, []byte("invalid json"), 0644)
	tr := newTracker()
	snap := tr.GetSnapshot()
	if snap.Daily.Interfaces == nil {
		t.Error("expected initialized snapshot")
	}

	// Save
	tr.mu.Lock()
	tr.saveLocked()
	tr.mu.Unlock()

	data, err := os.ReadFile(dataPath)
	if err != nil || len(data) == 0 {
		t.Fatalf("expected saved file: %v", err)
	}

	// parseProcNetDev
	raw := parseProcNetDev()
	t.Logf("Parsed %d raw network devices", len(raw))
}

func TestParseProcNetDevMock(t *testing.T) {
	tempDir := t.TempDir()
	mockDev := filepath.Join(tempDir, "dev")

	content := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 12345       10    0    0    0     0          0         0    12345       10    0    0    0     0       0          0
 wlan0: 1000000    100    0    0    0     0          0         0   500000       50    0    0    0     0       0          0
rmnet0: 2000000    200    0    0    0     0          0         0  1000000      100    0    0    0     0       0          0
 tun0: 9000000    200    0    0    0     0          0         0  8000000      100    0    0    0     0       0          0
 ap0: 7000000    200    0    0    0     0          0         0  6000000      100    0    0    0     0       0          0
`
	_ = os.WriteFile(mockDev, []byte(content), 0644)
	t.Setenv("BFR_MOCK_PROC_NET_DEV", mockDev)

	raw := parseProcNetDev()
	if len(raw) != 2 {
		t.Fatalf("expected only wlan0 and rmnet0, got %d interfaces: %+v", len(raw), raw)
	}
	if _, ok := raw["tun0"]; ok {
		t.Fatal("VPN interface should not be included in accounting")
	}
	if _, ok := raw["ap0"]; ok {
		t.Fatal("hotspot interface should not be included in accounting")
	}
	if raw["wlan0"].RxBytes != 1000000 || raw["wlan0"].TxBytes != 500000 {
		t.Errorf("wlan0 mismatch: %+v", raw["wlan0"])
	}

	// Test updateLocked and getStatsLocked with mock interfaces
	tr := newTracker()
	tr.mu.Lock()
	tr.updateLocked()

	// Update again with higher numbers to test delta calculations
	content2 := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
 wlan0: 1500000    150    0    0    0     0          0         0   750000       75    0    0    0     0       0          0
rmnet0: 2500000    250    0    0    0     0          0         0  1250000      125    0    0    0     0       0          0
`
	_ = os.WriteFile(mockDev, []byte(content2), 0644)
	tr.updateLocked()

	stats := tr.getStatsLocked()
	tr.mu.Unlock()

	if stats.Daily.Total.RxBytes != 1000000 {
		t.Errorf("expected RX total 1000000 bytes, got %d", stats.Daily.Total.RxBytes)
	}
	if stats.Daily.Total.TxBytes != 500000 {
		t.Errorf("expected TX total 500000 bytes, got %d", stats.Daily.Total.TxBytes)
	}
	if _, ok := stats.Daily.Interfaces["tun0"]; ok {
		t.Fatal("VPN interface should not be included in accounting")
	}
	if _, ok := stats.Daily.Interfaces["ap0"]; ok {
		t.Fatal("hotspot interface should not be included in accounting")
	}
}
