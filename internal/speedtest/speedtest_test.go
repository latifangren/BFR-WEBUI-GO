package speedtest

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseCloudflareTrace(t *testing.T) {
	sampleTrace := `fl=123f45
h=speed.cloudflare.com
ip=103.20.10.5
ts=1690000000
visit_scheme=https
uag=Mozilla/5.0
colo=CGK
sliver=none
http=http/2
loc=ID
tls=TLSv1.3
sni=plaintext
warp=off
gateway=off
rbi=off
kex=X25519`

	parsed := parseCloudflareTrace(sampleTrace)
	if parsed["ip"] != "103.20.10.5" {
		t.Errorf("expected ip 103.20.10.5, got %s", parsed["ip"])
	}
	if parsed["colo"] != "CGK" {
		t.Errorf("expected colo CGK, got %s", parsed["colo"])
	}
	if parsed["loc"] != "ID" {
		t.Errorf("expected loc ID, got %s", parsed["loc"])
	}

	// Empty trace should return empty map
	emptyMap := parseCloudflareTrace("")
	if len(emptyMap) != 0 {
		t.Errorf("expected empty map for empty trace, got %d entries", len(emptyMap))
	}
}

func TestGetColoCity(t *testing.T) {
	tests := []struct {
		colo     string
		expected string
	}{
		{"CGK", "Jakarta"},
		{"cgk", "Jakarta"},
		{"SIN", "Singapore"},
		{"NRT", "Tokyo"},
		{"UNKNOWN_CODE", "UNKNOWN_CODE"},
	}

	for _, tt := range tests {
		got := getColoCity(tt.colo)
		if got != tt.expected {
			t.Errorf("colo %s: expected %s, got %s", tt.colo, tt.expected, got)
		}
	}
}

func TestManagerHistory(t *testing.T) {
	tempDir := t.TempDir()
	dataPath := filepath.Join(tempDir, "speedtest_history.json")

	mgr := &Manager{
		dataPath: dataPath,
	}

	// Initial history should be empty
	hist := mgr.GetHistory()
	if len(hist) != 0 {
		t.Errorf("expected empty initial history, got %d", len(hist))
	}

	// Add an entry and save
	mgr.mu.Lock()
	mgr.history = []HistoryEntry{
		{
			Timestamp:    time.Now().Format(time.RFC3339),
			ServerName:   "Cloudflare Jakarta",
			ClientIP:     "103.20.10.5",
			ISP:          "Telkom",
			PingMs:       15.2,
			JitterMs:     2.1,
			DownloadMbps: 85.5,
			UploadMbps:   30.2,
		},
	}
	mgr.saveHistoryLocked()
	mgr.mu.Unlock()

	// Verify file was created on disk
	if _, err := os.Stat(dataPath); err != nil {
		t.Fatalf("expected dataPath file to exist, err: %v", err)
	}

	// Verify GetHistory returns copy
	savedHist := mgr.GetHistory()
	if len(savedHist) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(savedHist))
	}
	if savedHist[0].DownloadMbps != 85.5 {
		t.Errorf("expected DownloadMbps 85.5, got %f", savedHist[0].DownloadMbps)
	}

	// Create new manager instance with same dataPath and load
	mgr2 := &Manager{
		dataPath: dataPath,
	}
	mgr2.loadHistory()
	loadedHist := mgr2.GetHistory()
	if len(loadedHist) != 1 || loadedHist[0].ServerName != "Cloudflare Jakarta" {
		t.Errorf("failed to reload history into new manager instance, got %+v", loadedHist)
	}

	// Test GetStatus
	status := mgr2.GetStatus()
	if status.Running {
		t.Errorf("expected status.Running to be false by default")
	}
}
