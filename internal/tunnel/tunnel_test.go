package tunnel_test

import (
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
	if loaded.CloudflareToken != "test_token_123" {
		t.Errorf("expected token 'test_token_123', got '%s'", loaded.CloudflareToken)
	}
}

func TestFindBinary(t *testing.T) {
	// Search for non-existent engine returns empty string
	if bin := tunnel.FindBinary("unknown_engine"); bin != "" {
		t.Errorf("expected empty string for unknown engine, got '%s'", bin)
	}

	// Discovery call for supported engines should not panic
	_ = tunnel.FindBinary("cloudflare")
	_ = tunnel.FindBinary("tailscale")
	_ = tunnel.FindBinary("zerotier")
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
