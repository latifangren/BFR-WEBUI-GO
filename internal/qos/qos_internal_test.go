package qos

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInternalHasTCCmd(t *testing.T) {
	hasTC := hasTCCmd()
	t.Logf("hasTCCmd returned: %v", hasTC)
}

func TestInternalApplyTCHTBProfiles(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	mgr := newManager()
	mgr.detectedIface = "wlan0"

	// 1. Gaming preset with tc engine
	gamingCfg := &QoSConfig{
		Enabled:          true,
		Preset:           "gaming",
		Engine:           "tc",
		GlobalDownload:   50,
		GlobalUpload:     20,
		PrioritizeGaming: true,
		PrioritizeVoip:   true,
		ClientLimits: []ClientLimit{
			{IP: "192.168.43.10", DownloadLimit: 10, UploadLimit: 5, Priority: "gaming", Enabled: true},
		},
	}
	_ = mgr.ApplyQoS(gamingCfg)

	// 2. Fair preset with tc engine
	fairCfg := &QoSConfig{
		Enabled:        true,
		Preset:         "fair",
		Engine:         "tc",
		GlobalDownload: 10,
		GlobalUpload:   5,
		ClientLimits: []ClientLimit{
			{IP: "192.168.43.20", DownloadLimit: 2, UploadLimit: 1, Priority: "browsing", Enabled: true},
		},
	}
	_ = mgr.ApplyQoS(fairCfg)

	// 3. Custom / saver preset with tc engine
	saverCfg := &QoSConfig{
		Enabled:        true,
		Preset:         "saver",
		Engine:         "tc",
		GlobalDownload: 5,
		GlobalUpload:   2,
	}
	_ = mgr.ApplyQoS(saverCfg)

	// 4. IPTables mangle engine
	mangleCfg := &QoSConfig{
		Enabled:          true,
		Preset:           "gaming",
		Engine:           "iptables",
		GlobalDownload:   50,
		GlobalUpload:     20,
		PrioritizeGaming: true,
		PrioritizeVoip:   true,
		ClientLimits: []ClientLimit{
			{IP: "192.168.43.10", DownloadLimit: 10, UploadLimit: 5, Priority: "strict", Enabled: true},
		},
	}
	_ = mgr.ApplyQoS(mangleCfg)

	// 5. Disabled config
	_ = mgr.ApplyQoS(&QoSConfig{
		Enabled: false,
	})
}

func TestInternalSaveConfigAndCorrupted(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)
	dataPath := filepath.Join(tempDir, "qos.json")

	mgr := newManager()
	mgr.dataPath = dataPath

	// Nil config error
	if err := mgr.SaveConfig(nil); err == nil {
		t.Error("expected error saving nil config")
	}

	// Corrupted file -> fallback
	_ = os.WriteFile(dataPath, []byte("invalid json {{"), 0644)
	cfg, err := mgr.LoadConfig()
	if err != nil || cfg == nil {
		t.Errorf("expected fallback config on corrupted file")
	}

	// Detect hotspot interface
	iface := detectHotspotInterface()
	t.Logf("detected hotspot interface: %s", iface)
}
