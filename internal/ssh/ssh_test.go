package ssh

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSSHConfigValidationAndSave(t *testing.T) {
	tempDir := t.TempDir()
	dataPath := filepath.Join(tempDir, "ssh_config.json")

	mgr := &Manager{
		dataPath: dataPath,
		config: Config{
			Port: 22,
			Bind: "0.0.0.0",
		},
	}

	// Invalid ports
	_, err := mgr.SaveConfig(Config{Port: 0})
	if err == nil {
		t.Error("expected error for port 0")
	}

	_, err = mgr.SaveConfig(Config{Port: 70000})
	if err == nil {
		t.Error("expected error for port 70000")
	}

	// Valid port
	validCfg := Config{
		Enabled:     true,
		Port:        2222,
		Bind:        "127.0.0.1",
		KeyAuthOnly: true,
	}

	status, err := mgr.SaveConfig(validCfg)
	if err != nil {
		t.Fatalf("unexpected error saving valid config: %v", err)
	}

	if status.Config.Port != 2222 || !status.Config.Enabled || !status.Config.KeyAuthOnly {
		t.Errorf("saved config mismatch: %+v", status.Config)
	}

	// Load into a new manager instance
	mgr2 := &Manager{
		dataPath: dataPath,
	}
	mgr2.loadConfig()
	st2 := mgr2.GetStatus()
	if st2.Config.Port != 2222 || st2.Config.Bind != "127.0.0.1" || !st2.Config.KeyAuthOnly {
		t.Errorf("reloaded config mismatch: %+v", st2.Config)
	}
}

func TestSSHLegacyConfigDefaultsToKeyOnly(t *testing.T) {
	tempDir := t.TempDir()
	dataPath := filepath.Join(tempDir, "ssh_config.json")
	if err := os.WriteFile(dataPath, []byte(`{"enabled":true,"port":2222,"bind":"0.0.0.0"}`), 0600); err != nil {
		t.Fatal(err)
	}

	mgr := &Manager{dataPath: dataPath}
	mgr.loadConfig()
	if !mgr.config.KeyAuthOnly {
		t.Fatal("legacy SSH config must default to key-only authentication")
	}
}

func TestSSHDefaultManager(t *testing.T) {
	mgr := GetManager()
	if mgr == nil {
		t.Fatal("expected non-nil default SSH manager")
	}

	st := mgr.GetStatus()
	if st.Config.Port <= 0 {
		t.Errorf("expected positive default port, got %d", st.Config.Port)
	}
}
