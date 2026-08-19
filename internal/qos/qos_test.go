package qos_test

import (
	"testing"

	"bfr-webui-go/internal/qos"
)

func TestGetManager(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	mgr := qos.GetManager()
	if mgr == nil {
		t.Fatalf("expected non-nil QoS manager")
	}
}

func TestLoadAndSaveConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	mgr := qos.GetManager()

	cfg := &qos.QoSConfig{
		Enabled:          true,
		Engine:           "auto",
		GlobalDownload:   100,
		GlobalUpload:     50,
		PrioritizeGaming: true,
		PrioritizeVoip:   true,
		ClientLimits: []qos.ClientLimit{
			{
				IP:            "192.168.43.100",
				MAC:           "00:11:22:33:44:55",
				Hostname:      "TestDevice",
				DownloadLimit: 10,
				UploadLimit:   5,
				Priority:      "gaming",
				Enabled:       true,
			},
		},
	}

	err := mgr.SaveConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error saving QoS config: %v", err)
	}

	loaded, err := mgr.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error loading QoS config: %v", err)
	}

	if !loaded.Enabled {
		t.Errorf("expected loaded config Enabled to be true")
	}
	if loaded.GlobalDownload != 100 || loaded.GlobalUpload != 50 {
		t.Errorf("expected 100/50 Mbps limits, got %d/%d", loaded.GlobalDownload, loaded.GlobalUpload)
	}
	if len(loaded.ClientLimits) != 1 || loaded.ClientLimits[0].IP != "192.168.43.100" {
		t.Errorf("unexpected client limits slice in loaded config: %+v", loaded.ClientLimits)
	}
}

func TestApplyAndClearQoS(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	mgr := qos.GetManager()

	// Apply enabled QoS config
	cfg := &qos.QoSConfig{
		Enabled:        true,
		Engine:         "iptables",
		GlobalDownload: 50,
		GlobalUpload:   20,
		ClientLimits: []qos.ClientLimit{
			{
				IP:            "192.168.43.50",
				DownloadLimit: 5,
				UploadLimit:   2,
				Enabled:       true,
			},
		},
	}

	_ = mgr.ApplyQoS(cfg)
	status := mgr.GetStatus()
	// Verification status struct response contains expected field structure
	_ = status.Active
	_ = status.EngineUsed

	// Clear QoS
	err := mgr.ClearQoS()
	if err != nil {
		t.Fatalf("unexpected error clearing QoS: %v", err)
	}

	clearedStatus := mgr.GetStatus()
	if clearedStatus.Active {
		t.Errorf("expected QoS Active status to be false after clear")
	}
}
