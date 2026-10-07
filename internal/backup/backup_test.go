package backup

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExportAndImportBackup(t *testing.T) {
	// Create a dummy config in current working directory with a recognized backup name
	testCfgName := "network_config.json"
	origContent, hadOrig := os.ReadFile(testCfgName)
	defer func() {
		if hadOrig == nil {
			_ = os.WriteFile(testCfgName, origContent, 0644)
		} else {
			_ = os.Remove(testCfgName)
		}
	}()

	testData := `{"dns_enabled":true}`
	if err := os.WriteFile(testCfgName, []byte(testData), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Export
	bundleBytes, err := ExportBackup()
	if err != nil {
		t.Fatalf("ExportBackup failed: %v", err)
	}

	var bundle BackupBundle
	if err := json.Unmarshal(bundleBytes, &bundle); err != nil {
		t.Fatalf("failed to unmarshal exported bundle: %v", err)
	}
	if bundle.Configs[testCfgName] != testData {
		t.Errorf("expected config %s to be present in bundle", testCfgName)
	}

	// Import valid bundle
	if err := ImportBackup(bundleBytes); err != nil {
		t.Fatalf("ImportBackup failed: %v", err)
	}

	// Import invalid format should fail
	if err := ImportBackup([]byte("invalid json")); err == nil {
		t.Error("expected error importing invalid json")
	}

	// Import empty configs bundle should fail
	emptyBundle, _ := json.Marshal(BackupBundle{
		Timestamp: time.Now(),
		Configs:   make(map[string]string),
	})
	if err := ImportBackup(emptyBundle); err == nil {
		t.Error("expected error importing empty bundle")
	}
}

func TestCreateTarGzBackup(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	testFile := filepath.Join(tempDir, "settings.json")
	_ = os.WriteFile(testFile, []byte(`{"key":"value"}`), 0644)

	tarData, err := CreateTarGzBackup()
	if err != nil {
		t.Fatalf("CreateTarGzBackup failed: %v", err)
	}
	if len(tarData) == 0 {
		t.Fatal("expected non-empty tar.gz data")
	}
}

func TestCloudManagerConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)
	mgr := GetCloudManager()
	if mgr == nil {
		t.Fatal("expected non-nil CloudManager")
	}

	cfg := mgr.GetConfig()
	cfg.Provider = "webdav"
	cfg.URL = "https://example.com/dav"
	cfg.Username = "user"
	cfg.Password = "pass"

	err := mgr.SaveConfig(cfg)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	updated := mgr.GetConfig()
	if updated.Provider != "webdav" || updated.URL != "https://example.com/dav" {
		t.Errorf("config mismatch: %+v", updated)
	}
}

func TestSyncBackupWithMockServer(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	mgr := GetCloudManager()

	// 1. Empty URL error
	_ = mgr.SaveConfig(CloudConfig{URL: ""})
	if err := mgr.SyncBackup(); err == nil {
		t.Error("expected error with empty URL")
	}

	// 2. Success with mock WebDAV server
	var receivedMethod string
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	testFile := filepath.Join(tempDir, "settings.json")
	_ = os.WriteFile(testFile, []byte(`{"hello":"world"}`), 0644)

	err := mgr.SaveConfig(CloudConfig{
		Enabled:  true,
		Provider: "webdav",
		URL:      server.URL + "/",
		Username: "admin",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	err = mgr.SyncBackup()
	if err != nil {
		t.Fatalf("SyncBackup failed: %v", err)
	}
	if receivedMethod != http.MethodPut {
		t.Errorf("expected PUT method, got %s", receivedMethod)
	}
	if receivedAuth == "" {
		t.Error("expected basic auth header to be present")
	}

	// 3. Error response from server (HTTP 500)
	errServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer errServer.Close()

	_ = mgr.SaveConfig(CloudConfig{URL: errServer.URL + "/"})
	if err := mgr.SyncBackup(); err == nil {
		t.Error("expected error when server returns 500")
	}
}

func TestCloudManagerCorruptedConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)
	cfgPath := getCloudStoragePath()

	// Write corrupted JSON
	_ = os.WriteFile(cfgPath, []byte("invalid json {{"), 0644)

	mgr := &CloudManager{dataPath: cfgPath, config: CloudConfig{IntervalHours: 24}}
	mgr.loadConfig()
	cfg := mgr.GetConfig()
	if cfg.IntervalHours != 24 {
		t.Errorf("expected default interval hours 24 on corrupted file, got %d", cfg.IntervalHours)
	}
}
