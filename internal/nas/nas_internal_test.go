package nas

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInternalLoadConfigWithBOMAndCorrupted(t *testing.T) {
	tempDir := t.TempDir()
	dataPath := filepath.Join(tempDir, "nas.json")
	mgr := &Manager{
		dataPath: dataPath,
	}

	// 1. File with BOM
	dataWithBOM := append([]byte("\xef\xbb\xbf"), []byte(`{"port":99,"share_path":"/custom","protocol":"ftp"}`)...)
	_ = os.WriteFile(dataPath, dataWithBOM, 0644)

	cfg, err := mgr.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig with BOM failed: %v", err)
	}
	if cfg.Port != 99 || cfg.SharePath != "/custom" || cfg.Protocol != "ftp" {
		t.Errorf("mismatch loading BOM config: %+v", cfg)
	}

	// 2. Corrupted file
	_ = os.WriteFile(dataPath, []byte("invalid json"), 0644)
	cfg2, err := mgr.LoadConfig()
	if err != nil || cfg2 == nil {
		t.Errorf("expected fallback on corrupted json: %v", err)
	}

	// 3. SaveConfig with empty values
	err = mgr.SaveConfig(&NASConfig{
		Port:      0,
		SharePath: "",
		Protocol:  "",
	})
	if err != nil {
		t.Fatalf("SaveConfig with zero values failed: %v", err)
	}
	if mgr.config.Port != 85 || mgr.config.SharePath != "/sdcard" || mgr.config.Protocol != "webdav" {
		t.Errorf("expected default values filled, got %+v", mgr.config)
	}
}
