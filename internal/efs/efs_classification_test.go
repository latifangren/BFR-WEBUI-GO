package efs

import (
	"testing"
)

func TestIsItemPath(t *testing.T) {
	testCases := []struct {
		path     string
		expected bool
	}{
		// Standard NV item files
		{"/nv/item_files/modem/lte/rrc/efs/max_num_mimo_lyr", true},
		{"/nv/item_files/mcfg/mcfg_autoselect_by_uim", true},
		{"/nv/item_files/ims/IMS_enable", true},
		{"/nv/item_files/mcs/trm/ant_div_switch_master", true},
		{"/nv/item_files/modem/nr5g/RRC/cap_control_dl_256qam", true},

		// Registry files (also item files in Qualcomm architecture)
		{"/nv/reg_files/modem/lte/rrc/csp/cell_lock_list", true},
		{"/nv/reg_files/modem/geran/grr/feature_lock_arfcn_enabled", true},

		// CGPS NV item files
		{"/cgps/nv/item_files/cgps_sm_nv_gnss_dyn_feature", true},

		// SD card / service discovery
		{"/sd/carrier_policy.xml", true},

		// Regular EFS files (should NOT be classified as item files)
		{"/policyman/carrier_policy.xml", false},
		{"/rfc/rffe_scan/rffe_scan.dat", false},
		{"/mcfg_sw/generic/common/default.mbn", false},
		{"/mcfg_hw/generic/common/hw.mbn", false},
		{"/etc/modem_prl", false},
		{"/data/test.bin", false},
		{"", false},
	}

	for _, tc := range testCases {
		got := IsItemPath(tc.path)
		if got != tc.expected {
			t.Errorf("IsItemPath(%q) = %v; want %v", tc.path, got, tc.expected)
		}
	}
}

func TestResolveItemClassification(t *testing.T) {
	resolveItem := func(path string, forceItem int) bool {
		isItem := IsItemPath(path)
		if forceItem > 0 {
			isItem = true
		} else if forceItem == 0 {
			isItem = false
		}
		return isItem
	}

	// 1. Auto mode (-1) respects IsItemPath
	if !resolveItem("/nv/item_files/test", -1) {
		t.Error("expected auto mode to recognize /nv/item_files as item")
	}
	if !resolveItem("/nv/reg_files/test", -1) {
		t.Error("expected auto mode to recognize /nv/reg_files as item")
	}
	if resolveItem("/policyman/carrier_policy.xml", -1) {
		t.Error("expected auto mode to recognize regular file as non-item")
	}

	// 2. Force item mode (1) forces item regardless of path
	if !resolveItem("/policyman/carrier_policy.xml", 1) {
		t.Error("expected force 1 to force item mode")
	}

	// 3. Force regular mode (0) forces regular file even for item paths
	if resolveItem("/nv/item_files/test", 0) {
		t.Error("expected force 0 to force regular mode even on /nv/item_files")
	}
}
