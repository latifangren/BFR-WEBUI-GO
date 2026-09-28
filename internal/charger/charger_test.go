package charger_test

import (
	"os"
	"path/filepath"
	"testing"

	"bfr-webui-go/internal/charger"
)

func TestGetManager(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	mgr := charger.GetManager()
	if mgr == nil {
		t.Fatalf("expected non-nil charger manager")
	}
}

func TestGetStatus(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	mgr := charger.GetManager()
	status := mgr.GetStatus()

	if status.Config.StartPercent <= 0 {
		t.Errorf("expected positive start percent")
	}
	if status.Config.StopPercent <= 0 {
		t.Errorf("expected positive stop percent")
	}
	if status.Battery.Capacity < -1 || status.Battery.Capacity > 100 {
		t.Errorf("unexpected battery capacity: %d", status.Battery.Capacity)
	}
}

func TestUpdateConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	mgr := charger.GetManager()

	// Update config with valid values
	status := mgr.UpdateConfig(charger.Config{
		Enabled:      true,
		StartPercent: 75,
		StopPercent:  85,
	})

	if status.Config.StartPercent != 75 || status.Config.StopPercent != 85 {
		t.Errorf("expected config updated to 75-85, got %d-%d", status.Config.StartPercent, status.Config.StopPercent)
	}

	// Update with non-positive fallback triggers defaults
	statusDefault := mgr.UpdateConfig(charger.Config{
		Enabled:      true,
		StartPercent: 0,
		StopPercent:  0,
	})
	if statusDefault.Config.StartPercent != 70 || statusDefault.Config.StopPercent != 80 {
		t.Errorf("expected fallback defaults 70-80, got %d-%d", statusDefault.Config.StartPercent, statusDefault.Config.StopPercent)
	}

	// Start >= Stop automatically adjusts Start
	statusAdjust := mgr.UpdateConfig(charger.Config{
		Enabled:      true,
		StartPercent: 90,
		StopPercent:  85,
	})
	if statusAdjust.Config.StartPercent >= statusAdjust.Config.StopPercent {
		t.Errorf("expected start < stop, got %d >= %d", statusAdjust.Config.StartPercent, statusAdjust.Config.StopPercent)
	}
}

func TestToggleCharging(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	mgr := charger.GetManager()
	initial := mgr.GetStatus().ChargingDisabled

	toggled := mgr.ToggleCharging()
	if toggled.ChargingDisabled == initial {
		t.Errorf("expected ChargingDisabled to flip from %v, got %v", initial, toggled.ChargingDisabled)
	}

	// Toggle back
	restored := mgr.ToggleCharging()
	if restored.ChargingDisabled != initial {
		t.Errorf("expected ChargingDisabled restored to %v, got %v", initial, restored.ChargingDisabled)
	}
}

func TestProbeProfileMock(t *testing.T) {
	tempDir := t.TempDir()
	mockNode := filepath.Join(tempDir, "charging_enabled")
	if err := os.WriteFile(mockNode, []byte("1\n"), 0644); err != nil {
		t.Fatalf("failed to create mock node: %v", err)
	}

	profile := charger.SwitchProfile{
		ID:          "mock_qc",
		Name:        "Mock Qualcomm",
		Description: "Mock node for testing",
		Nodes: []charger.SwitchNode{
			{Path: mockNode, OnValue: "1", OffValue: "0"},
		},
		IsBypass: false,
	}

	res := charger.ProbeProfile(profile)
	if res.ProfileID != "mock_qc" {
		t.Errorf("expected ProfileID mock_qc, got %s", res.ProfileID)
	}

	// Verify node value was restored after probe
	content, err := os.ReadFile(mockNode)
	if err != nil {
		t.Fatalf("failed to read mock node: %v", err)
	}
	if string(content) != "1\n" {
		t.Errorf("expected node restored to '1', got '%s'", string(content))
	}
}

func TestProbeSingleSwitchNotFound(t *testing.T) {
	res := charger.ProbeSingleSwitch("non_existent_profile_id")
	if res.Success {
		t.Errorf("expected failure for non-existent profile, got success")
	}
}

func TestAllSwitchProfilesDefinition(t *testing.T) {
	if len(charger.AllSwitchProfiles) == 0 {
		t.Fatalf("expected non-empty AllSwitchProfiles")
	}

	// Check required vendor profiles exist
	requiredIDs := []string{
		"qc_smb5_force_main_fcc",
		"qc_smb5_force_main_icl",
		"pixel_charge_stop_level",
		"pixel_charge_limit",
		"pixel_google_charger_suspend",
		"qc_battery_charging_enabled",
		"qc_battery_input_suspend",
		"oplus_op_disable_charge",
		"oplus_multi_node",
		"samsung_batt_slate_mode",
		"samsung_store_mode",
		"mtk_sub_charging_enabled",
		"mtk_battery_cmd",
		"asus_rog_charging_suspend_en",
		"asus_rog_charger_limit_en",
		"universal_charge_control_limit",
	}

	if charger.AllSwitchProfiles[0].ID != "qc_smb5_force_main_fcc" {
		t.Errorf("expected qc_smb5_force_main_fcc as top priority profile, got %s", charger.AllSwitchProfiles[0].ID)
	}
	if len(charger.AllSwitchProfiles[0].Nodes) == 0 ||
		charger.AllSwitchProfiles[0].Nodes[0].OnValue != "550000" ||
		charger.AllSwitchProfiles[0].Nodes[0].OffValue != "0" {
		t.Errorf("expected OnValue 550000 and OffValue 0 for qc_smb5_force_main_fcc")
	}

	profileMap := make(map[string]bool)
	for _, p := range charger.AllSwitchProfiles {
		profileMap[p.ID] = true
		if len(p.Nodes) == 0 {
			t.Errorf("profile %s has no nodes", p.ID)
		}
	}

	for _, req := range requiredIDs {
		if !profileMap[req] {
			t.Errorf("missing required ACC switch profile: %s", req)
		}
	}
}

func TestManagerLogAndStop(t *testing.T) {
	mgr := charger.GetManager()
	mgr.Log("Test log entry")

	status := mgr.GetStatus()
	found := false
	for _, l := range status.Logs {
		if len(l) > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected logs to contain entries")
	}

	mgr.Stop()
	// Idempotent Stop calls must not panic
	mgr.Stop()
	charger.Stop()
}

