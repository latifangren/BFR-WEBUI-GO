package charger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInternalChargerOperations(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	dataPath := filepath.Join(tempDir, "charger_config.json")
	mgr := &Manager{
		dataPath: dataPath,
	}

	// 1. GetStatus
	status := mgr.GetStatus()
	t.Logf("Initial status: %+v", status)

	// 2. UpdateConfig
	updated := mgr.UpdateConfig(Config{
		Enabled:        true,
		StartPercent:   60,
		StopPercent:    80,
		SelectedSwitch: "mock_profile",
	})
	if updated.Config.StartPercent != 60 || updated.Config.StopPercent != 80 {
		t.Errorf("expected 60-80 config, got %+v", updated.Config)
	}

	// 3. ToggleCharging
	toggled := mgr.ToggleCharging()
	t.Logf("Toggled: ChargingDisabled=%v", toggled.ChargingDisabled)

	// 4. ProbeSwitches
	probeResults := mgr.ProbeSwitches()
	t.Logf("ProbeSwitches returned %d results", len(probeResults))

	// 5. Corrupted config file fallback in loadConfig
	_ = os.WriteFile(dataPath, []byte("invalid json {{"), 0644)
	mgr.loadConfig()
	if mgr.config.StartPercent <= 0 {
		t.Error("expected non-zero fallback StartPercent")
	}

	// 6. Config with BOM prefix in loadConfig
	bomData := append([]byte("\xef\xbb\xbf"), []byte(`{"enabled":true,"start_percent":65,"stop_percent":85}`)...)
	_ = os.WriteFile(dataPath, bomData, 0644)
	mgr.loadConfig()
	if mgr.config.StartPercent != 65 {
		t.Errorf("expected 65 from BOM config, got %d", mgr.config.StartPercent)
	}

	// 7. evaluateLocked
	mgr.mu.Lock()
	mgr.evaluateLocked()
	mgr.mu.Unlock()

	// 8. applySwitchLocked with mock switch profile
	mockNode := filepath.Join(tempDir, "charging_enabled")
	_ = os.WriteFile(mockNode, []byte("1\n"), 0644)
	mgr.activeProfile = &SwitchProfile{
		ID:   "mock_profile",
		Name: "Mock",
		Nodes: []SwitchNode{
			{Path: mockNode, OnValue: "1", OffValue: "0"},
		},
	}
	mgr.mu.Lock()
	mgr.applySwitchLocked(false)
	mgr.applySwitchLocked(true)
	mgr.mu.Unlock()

	// 9. Log
	mgr.Log("Test log entry")
}

func TestPackageLevelChargerFunctions(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	st := GetStatus()
	t.Logf("GetStatus: %+v", st)

	up := UpdateConfig(Config{StartPercent: 60, StopPercent: 80, Enabled: true})
	t.Logf("UpdateConfig: %+v", up)

	tog := ToggleCharging()
	t.Logf("ToggleCharging: %+v", tog)

	sws := ProbeSwitches()
	t.Logf("ProbeSwitches: %d", len(sws))

	m := GetManager()
	all := ProbeAll(m)
	t.Logf("ProbeAll: %d", len(all))

	res := ProbeSingle(m, "unknown_id")
	t.Logf("ProbeSingle: %+v", res)
}

func TestGetBatteryInfoWithMockSysfs(t *testing.T) {
	tempDir := t.TempDir()
	battDir := filepath.Join(tempDir, "battery")
	_ = os.MkdirAll(battDir, 0755)

	_ = os.WriteFile(filepath.Join(battDir, "capacity"), []byte("85\n"), 0644)
	_ = os.WriteFile(filepath.Join(battDir, "status"), []byte("Charging\n"), 0644)
	_ = os.WriteFile(filepath.Join(battDir, "current_now"), []byte("-1500000\n"), 0644)
	_ = os.WriteFile(filepath.Join(battDir, "voltage_now"), []byte("4100000\n"), 0644)
	_ = os.WriteFile(filepath.Join(battDir, "temp"), []byte("325\n"), 0644)
	_ = os.WriteFile(filepath.Join(battDir, "online"), []byte("1\n"), 0644)

	t.Setenv("BFR_MOCK_SYSFS_POWER_SUPPLY", battDir)

	info := GetBatteryInfo()
	if info.Capacity != 85 {
		t.Errorf("expected capacity 85, got %d", info.Capacity)
	}
	if info.Status != "Charging" {
		t.Errorf("expected status Charging, got %s", info.Status)
	}
	if info.VoltageNow <= 0 {
		t.Errorf("expected positive voltage, got %f", info.VoltageNow)
	}
	if info.Temperature <= 0 {
		t.Errorf("expected positive temperature, got %f", info.Temperature)
	}

	// Test evaluateLocked with mock battery
	mgr := &Manager{
		config: Config{
			Enabled:      true,
			StartPercent: 60,
			StopPercent:  80,
		},
		activeProfile: &SwitchProfile{
			ID: "mock_sw",
			Nodes: []SwitchNode{
				{Path: filepath.Join(tempDir, "node"), OnValue: "1", OffValue: "0"},
			},
		},
	}

	mgr.mu.Lock()
	// 85% >= 80% -> should stop charging
	mgr.evaluateLocked()
	if !mgr.chargingDisabled {
		t.Error("expected chargingDisabled to be true when capacity 85 >= 80")
	}

	// Change capacity to 50% <= 60% -> should resume charging
	_ = os.WriteFile(filepath.Join(battDir, "capacity"), []byte("50\n"), 0644)
	mgr.evaluateLocked()
	if mgr.chargingDisabled {
		t.Error("expected chargingDisabled to be false when capacity 50 <= 60")
	}

	// Test percentage-threshold kernel (detectedType == "charge_control_limit")
	mgr.detectedType = "charge_control_limit"
	mgr.evaluateLocked()
	mgr.mu.Unlock()
}

func TestAutoScanCustomAndSelectedSwitch(t *testing.T) {
	tempDir := t.TempDir()
	customFile := filepath.Join(tempDir, "custom_node")
	_ = os.WriteFile(customFile, []byte("1\n"), 0644)

	mgr := &Manager{
		config: Config{
			CustomPath: customFile,
			CustomOn:   "1",
			CustomOff:  "0",
		},
	}

	mgr.mu.Lock()
	mgr.autoScanLocked()
	if mgr.detectedType != "custom" || mgr.activeProfile == nil {
		t.Errorf("expected custom detected type, got %s", mgr.detectedType)
	}

	// Test selected switch from existing
	mgr.config.CustomPath = ""
	mgr.config.SelectedSwitch = "qc_smb5_force_main_fcc"
	mgr.autoScanLocked()
	mgr.mu.Unlock()

	// Test ProbeSingleSwitch
	res := mgr.ProbeSingleSwitch("qc_smb5_force_main_fcc")
	t.Logf("ProbeSingleSwitch result: %+v", res)

	pkgRes := ProbeSingleSwitch("qc_smb5_force_main_fcc")
	t.Logf("ProbeSingleSwitch package level: %+v", pkgRes)
}

func TestDynamicSysfsAutoScanner(t *testing.T) {
	tempDir := t.TempDir()
	battDir := filepath.Join(tempDir, "battery")
	_ = os.MkdirAll(battDir, 0755)

	_ = os.WriteFile(filepath.Join(battDir, "charging_enabled"), []byte("1\n"), 0644)
	_ = os.WriteFile(filepath.Join(battDir, "force_main_fcc"), []byte("550000\n"), 0644)
	_ = os.WriteFile(filepath.Join(battDir, "charge_limit"), []byte("80\n"), 0644)
	_ = os.WriteFile(filepath.Join(battDir, "store_mode"), []byte("0\n"), 0644)
	_ = os.WriteFile(filepath.Join(battDir, "charge_control_limit"), []byte("80\n"), 0644)
	_ = os.WriteFile(filepath.Join(battDir, "input_suspend"), []byte("0\n"), 0644)

	t.Setenv("BFR_MOCK_SYSFS_POWER_SUPPLY_ROOT", tempDir)

	profiles := GetExistingSwitchProfiles()
	if len(profiles) == 0 {
		t.Error("expected auto-detected profiles from mock sysfs")
	}

	// Test autoScanLocked with dynamic profiles
	mgr := &Manager{}
	mgr.mu.Lock()
	mgr.autoScanLocked()
	if mgr.activeProfile == nil {
		t.Error("expected autoScanLocked to pick an auto-detected profile")
	}
	mgr.mu.Unlock()
}
