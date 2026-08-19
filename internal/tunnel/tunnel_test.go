package tunnel_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"bfr-webui-go/internal/tunnel"
)

func TestGetManager(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	mgr := tunnel.GetManager()
	if mgr == nil {
		t.Fatalf("expected non-nil tunnel manager")
	}
}

func TestLoadAndSaveConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	mgr := tunnel.GetManager()

	cfg := &tunnel.TunnelConfig{
		Engine:            "cloudflare",
		Enabled:           true,
		CloudflareToken:   "test_token_123",
		CloudflareQuick:   false,
		TailscaleAuthKey:  "ts_key_xyz",
		ZeroTierNetworkID: "n12345",
		NgrokAuthToken:    "ngrok_tok_999",
		PinggyToken:       "pinggy_tok_888",
	}

	err := mgr.SaveConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error saving tunnel config: %v", err)
	}

	loaded, err := mgr.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error loading tunnel config: %v", err)
	}

	if loaded.Engine != "cloudflare" || !loaded.Enabled {
		t.Errorf("unexpected loaded config values: %+v", loaded)
	}
	if loaded.NgrokAuthToken != "ngrok_tok_999" || loaded.PinggyToken != "pinggy_tok_888" {
		t.Errorf("unexpected tokens: ngrok=%s, pinggy=%s", loaded.NgrokAuthToken, loaded.PinggyToken)
	}
}

func TestDownloadBinaryAndFindBinary(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	// Stream mock cloudflared binary
	r := bytes.NewReader([]byte("#!/bin/sh\necho mock cloudflared"))
	path, err := tunnel.DownloadBinary("cloudflare", r, "cloudflared")
	if err != nil {
		t.Fatalf("failed downloading binary: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected binary to exist at %s", path)
	}

	// Verify FindBinary picks up installed binary from persistent bin dir
	found := tunnel.FindBinary("cloudflare")
	if found != path && filepath.Base(found) != "cloudflared" {
		t.Errorf("expected FindBinary to locate %s, got %s", path, found)
	}
}

func TestGetStatusAndStopTunnel(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	mgr := tunnel.GetManager()
	status := mgr.GetStatus()

	if status.Engine == "" {
		t.Errorf("expected default engine in status")
	}

	err := mgr.StopTunnel()
	if err != nil {
		t.Fatalf("unexpected error stopping tunnel: %v", err)
	}

	stoppedStatus := mgr.GetStatus()
	if stoppedStatus.Active {
		t.Errorf("expected Active to be false after stopping tunnel")
	}
}
