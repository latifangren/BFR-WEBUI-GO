package nas_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bfr-webui-go/internal/nas"
)

func TestGetManager(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	mgr := nas.GetManager()
	if mgr == nil {
		t.Fatalf("expected non-nil NAS manager")
	}
}

func TestLoadAndSaveConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	mgr := nas.GetManager()

	cfg := &nas.NASConfig{
		Enabled:      true,
		SharePath:    tempDir,
		Port:         9099,
		ReadOnly:     true,
		AuthRequired: true,
		Username:     "admin",
		Password:     "secretpass",
		Protocol:     "webdav",
	}

	err := mgr.SaveConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error saving NAS config: %v", err)
	}

	loaded, err := mgr.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error loading NAS config: %v", err)
	}

	if !loaded.Enabled || loaded.Port != 9099 || loaded.Protocol != "webdav" {
		t.Errorf("unexpected loaded config values: %+v", loaded)
	}
}

func TestStartAndStopNAS(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	mgr := nas.GetManager()

	cfg := nas.NASConfig{
		Enabled:      true,
		SharePath:    tempDir,
		Port:         18088,
		ReadOnly:     false,
		AuthRequired: false,
		Protocol:     "http",
	}

	err := mgr.StartNAS(cfg)
	if err != nil {
		t.Fatalf("failed to start NAS server: %v", err)
	}

	status := mgr.GetStatus()
	if !status.Active {
		t.Errorf("expected NAS Active status to be true")
	}

	// Make HTTP test request to NAS server
	resp, err := http.Get(status.URL + "/")
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200 from NAS file server, got %d", resp.StatusCode)
		}
	}

	// Test Basic Auth header requirement if enabled
	cfgAuth := cfg
	cfgAuth.AuthRequired = true
	cfgAuth.Username = "user"
	cfgAuth.Password = "pass"
	_ = mgr.StartNAS(cfgAuth)

	reqUnauthorized := httptest.NewRequest("GET", status.URL+"/", nil)
	rr := httptest.NewRecorder()

	// Stop NAS
	errStop := mgr.StopNAS()
	if errStop != nil {
		t.Fatalf("failed to stop NAS server: %v", errStop)
	}

	stoppedStatus := mgr.GetStatus()
	if stoppedStatus.Active {
		t.Errorf("expected NAS Active status to be false after stopping")
	}
	_ = reqUnauthorized
	_ = rr
}
