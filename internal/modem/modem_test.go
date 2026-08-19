package modem_test

import (
	"reflect"
	"testing"

	"bfr-webui-go/internal/modem"
)

func TestBandsToHexBitmask(t *testing.T) {
	// Test empty slice
	if mask := modem.BandsToHexBitmask([]int{}); mask != "0x0" {
		t.Errorf("expected 0x0 for empty bands, got %s", mask)
	}

	// Test single band (Band 1 -> 2^0 = 1 -> 0x1)
	if mask := modem.BandsToHexBitmask([]int{1}); mask != "0x1" {
		t.Errorf("expected 0x1 for band [1], got %s", mask)
	}

	// Test multiple bands [1, 3, 8]
	// Band 1: 0x1 (bit 0)
	// Band 3: 0x4 (bit 2)
	// Band 8: 0x80 (bit 7)
	// Total: 0x1 | 0x4 | 0x80 = 0x85
	if mask := modem.BandsToHexBitmask([]int{1, 3, 8}); mask != "0x85" {
		t.Errorf("expected 0x85 for bands [1, 3, 8], got %s", mask)
	}

	// Test Band 40 (bit 39 -> 0x8000000000)
	// Band 1 + Band 3 + Band 40 = 0x1 | 0x4 | 0x8000000000 = 0x8000000005
	if mask := modem.BandsToHexBitmask([]int{1, 3, 40}); mask != "0x8000000005" {
		t.Errorf("expected 0x8000000005 for bands [1, 3, 40], got %s", mask)
	}
}

func TestHexBitmaskToBands(t *testing.T) {
	// Test empty or 0x0
	if bands := modem.HexBitmaskToBands("0x0"); len(bands) != 0 {
		t.Errorf("expected empty bands for 0x0, got %v", bands)
	}

	// Test 0x85 -> [1, 3, 8]
	expected85 := []int{1, 3, 8}
	bands85 := modem.HexBitmaskToBands("0x85")
	if !reflect.DeepEqual(bands85, expected85) {
		t.Errorf("expected %v for 0x85, got %v", expected85, bands85)
	}

	// Test 0x8000000005 -> [1, 3, 40]
	expected40 := []int{1, 3, 40}
	bands40 := modem.HexBitmaskToBands("0x8000000005")
	if !reflect.DeepEqual(bands40, expected40) {
		t.Errorf("expected %v for 0x8000000005, got %v", expected40, bands40)
	}
}

func TestFindModemPort(t *testing.T) {
	// Should run without panic, returning empty string or a valid path
	port := modem.FindModemPort()
	_ = port
}

func TestExecuteATCommand_Validation(t *testing.T) {
	// Invalid AT command containing illegal shell characters
	resp := modem.ExecuteATCommand("AT; rm -rf /")
	if resp.Success {
		t.Errorf("expected failure for invalid AT command format")
	}
}

func TestGetSignalInfo(t *testing.T) {
	sig, err := modem.GetSignalInfo()
	if err != nil {
		t.Fatalf("unexpected error getting signal info: %v", err)
	}
	_ = sig
}

func TestLoadAndSaveConfig(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	mgr := modem.GetManager()

	cfg := &modem.BandConfig{
		Engine:       "qualcomm_at",
		PreferredRAT: "4g_only",
		LTEBands:     []int{1, 3, 8, 40},
		NRBands:      []int{78},
		HexBitmask:   "0x8000000085",
	}

	err := mgr.SaveConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error saving modem config: %v", err)
	}

	loaded, err := mgr.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error loading modem config: %v", err)
	}

	if loaded.Engine != "qualcomm_at" || loaded.PreferredRAT != "4g_only" {
		t.Errorf("unexpected loaded config values: %+v", loaded)
	}
	if len(loaded.LTEBands) != 4 || loaded.LTEBands[3] != 40 {
		t.Errorf("unexpected LTEBands in loaded config: %v", loaded.LTEBands)
	}
}

func TestApplyAndResetBandLock(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("BFR_DATA_DIR", tempDir)

	mgr := modem.GetManager()

	cfg := modem.BandConfig{
		Engine:       "universal",
		PreferredRAT: "4g_only",
		LTEBands:     []int{1, 3},
		HexBitmask:   "0x5",
	}

	_ = mgr.ApplyBandLock(cfg)
	errReset := mgr.ResetBandLock()
	if errReset != nil {
		t.Errorf("unexpected error resetting band lock: %v", errReset)
	}
}
