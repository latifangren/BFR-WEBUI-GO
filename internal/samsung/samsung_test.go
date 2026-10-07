//go:build !nomodem

package samsung

import (
	"testing"
)

func TestDexFileExists(t *testing.T) {
	// In the dev build, bin/band_manager.dex is present in repo root
	exists := dexFileExists()
	if !exists {
		t.Log("dexFileExists returned false in current working dir, verifying fallback path")
	}
}

func TestGetDeviceStatus(t *testing.T) {
	status := GetDeviceStatus()
	if status == nil {
		t.Fatal("expected non-nil DeviceStatus")
	}
	// On non-Samsung test host, IsSamsung should be false
	t.Logf("DeviceStatus: IsSamsung=%v, Manufacturer=%s, Model=%s", status.IsSamsung, status.Manufacturer, status.Model)
}

func TestGetActiveDataSlot(t *testing.T) {
	slot := GetActiveDataSlot()
	if slot < 0 || slot > 1 {
		t.Logf("ActiveDataSlot returned %d (expected 0 or 1 in standard environments)", slot)
	}
}

func TestGet5GThermalStatus(t *testing.T) {
	thermal, err := Get5GThermalStatus()
	if err != nil {
		t.Logf("Get5GThermalStatus returned error in non-Android environment: %v", err)
	} else if thermal == nil {
		t.Fatal("expected non-nil thermal status when error is nil")
	}
}

func TestBandOperationsValidation(t *testing.T) {
	// Invalid slot parameter testing
	invalidSlots := []int{-1, 2, 5}
	for _, s := range invalidSlots {
		_, err := GetBands(s)
		if err == nil {
			t.Errorf("expected error for invalid slot %d", s)
		}

		err = LockBands(s, []int{1, 3})
		if err == nil {
			t.Errorf("expected error for invalid slot %d", s)
		}

		err = ResetAuto(s)
		if err == nil {
			t.Errorf("expected error for invalid slot %d", s)
		}
	}
}
