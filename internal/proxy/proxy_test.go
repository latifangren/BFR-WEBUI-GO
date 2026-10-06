package proxy_test

import (
	"testing"

	"bfr-webui-go/internal/proxy"
)

func TestBoxSettings(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_BOX_BASE", tempDir)

	settings, err := proxy.LoadBoxSettings()
	if err != nil {
		t.Fatalf("unexpected error loading default settings: %v", err)
	}

	settings.BinName = "sing-box"
	settings.NetworkMode = "tproxy"
	settings.IPv6 = true
	settings.MemcgLimit = "200M"

	if err := proxy.SaveBoxSettings(settings); err != nil {
		t.Fatalf("failed to save box settings: %v", err)
	}

	reloaded, err := proxy.LoadBoxSettings()
	if err != nil {
		t.Fatalf("failed to reload box settings: %v", err)
	}

	if reloaded.BinName != "sing-box" {
		t.Errorf("expected bin_name sing-box, got %s", reloaded.BinName)
	}
	if reloaded.NetworkMode != "tproxy" {
		t.Errorf("expected network_mode tproxy, got %s", reloaded.NetworkMode)
	}
	if !reloaded.IPv6 {
		t.Errorf("expected ipv6 true")
	}
}

func TestAppProxyConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_BOX_BASE", tempDir)

	cfg, err := proxy.LoadAppProxyConfig()
	if err != nil {
		t.Fatalf("unexpected error loading app proxy config: %v", err)
	}

	cfg.Mode = "whitelist"
	cfg.Packages = []string{"com.termux", "com.google.android.youtube"}

	if err := proxy.SaveAppProxyConfig(cfg); err != nil {
		t.Fatalf("failed to save app proxy config: %v", err)
	}

	reloaded, err := proxy.LoadAppProxyConfig()
	if err != nil {
		t.Fatalf("failed to reload app proxy config: %v", err)
	}

	if reloaded.Mode != "whitelist" {
		t.Errorf("expected mode whitelist, got %s", reloaded.Mode)
	}
	if len(reloaded.Packages) != 2 {
		t.Errorf("expected 2 packages, got %d", len(reloaded.Packages))
	}
}

func TestHotspotProxyConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_BOX_BASE", tempDir)

	cfg, err := proxy.LoadHotspotProxyConfig()
	if err != nil {
		t.Fatalf("unexpected error loading hotspot config: %v", err)
	}

	cfg.ProxyHotspot = false
	cfg.ApList = []string{"ap0", "wlan1"}

	if err := proxy.SaveHotspotProxyConfig(cfg); err != nil {
		t.Fatalf("failed to save hotspot config: %v", err)
	}

	reloaded, err := proxy.LoadHotspotProxyConfig()
	if err != nil {
		t.Fatalf("failed to reload hotspot config: %v", err)
	}

	if reloaded.ProxyHotspot {
		t.Errorf("expected proxy_hotspot false")
	}
}
