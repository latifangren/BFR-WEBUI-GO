package charger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AllSwitchProfiles defines the ACC database of hardware charging control switches.
var AllSwitchProfiles = []SwitchProfile{
	// Qualcomm SMB5 (Top Priority for Snapdragon devices, Pixel 5, Xiaomi, POCO)
	{
		ID:          "qc_smb5_force_main_fcc",
		Name:        "Qualcomm SMB5 (force_main_fcc)",
		Description: "Proven hardware fast charge current cutoff for Snapdragon devices (Pixel 5, Xiaomi, etc.)",
		Nodes: []SwitchNode{
			{Path: "/sys/class/power_supply/main/force_main_fcc", OnValue: "550000", OffValue: "0"},
		},
		IsBypass: false,
	},
	{
		ID:          "qc_smb5_force_main_icl",
		Name:        "Qualcomm SMB5 (force_main_icl)",
		Description: "Qualcomm SMB5 force input current limit cutoff (0 mA)",
		Nodes: []SwitchNode{
			{Path: "/sys/class/power_supply/main/force_main_icl", OnValue: "550000", OffValue: "0"},
		},
		IsBypass: false,
	},

	// Google Pixel
	{
		ID:          "pixel_charge_stop_level",
		Name:        "Google Pixel (charge_stop_level)",
		Description: "Charge stop percentage threshold for Google Pixel devices",
		Nodes: []SwitchNode{
			{Path: "/sys/class/power_supply/battery/charge_stop_level", OnValue: "100", OffValue: "80"},
		},
		IsBypass: false,
	},
	{
		ID:          "pixel_charge_limit",
		Name:        "Google Pixel (charge_limit)",
		Description: "Kernel charging limit node for Pixel devices (Tensor GS101+ / Newer)",
		Nodes: []SwitchNode{
			{Path: "/sys/class/power_supply/battery/charge_limit", OnValue: "-1", OffValue: "80"},
		},
		IsBypass: false,
	},
	{
		ID:          "pixel_google_charger_suspend",
		Name:        "Google Pixel (google_charger/input_suspend)",
		Description: "Google charger input suspend switch for true idle power bypass",
		Nodes: []SwitchNode{
			{Path: "/sys/class/power_supply/google_charger/input_suspend", OnValue: "0", OffValue: "1"},
		},
		IsBypass: true,
	},
	{
		ID:          "pixel_battery_input_suspend",
		Name:        "Google Pixel (battery/input_suspend)",
		Description: "Pixel battery input suspend node for idle power bypass",
		Nodes: []SwitchNode{
			{Path: "/sys/class/power_supply/battery/input_suspend", OnValue: "0", OffValue: "1"},
		},
		IsBypass: true,
	},

	// Qualcomm PMIC
	{
		ID:          "qc_battery_charging_enabled",
		Name:        "Qualcomm PMIC (battery/charging_enabled)",
		Description: "Standard Qualcomm battery charging enable switch",
		Nodes: []SwitchNode{
			{Path: "/sys/class/power_supply/battery/charging_enabled", OnValue: "1", OffValue: "0"},
		},
		IsBypass: false,
	},
	{
		ID:          "qc_battery_input_suspend",
		Name:        "Qualcomm PMIC (battery/input_suspend)",
		Description: "Qualcomm battery input suspend for idle power bypass",
		Nodes: []SwitchNode{
			{Path: "/sys/class/power_supply/battery/input_suspend", OnValue: "0", OffValue: "1"},
		},
		IsBypass: true,
	},
	{
		ID:          "qc_main_charging_enabled",
		Name:        "Qualcomm PMIC (main/charging_enabled)",
		Description: "Qualcomm SMB charger main charging enabled switch",
		Nodes: []SwitchNode{
			{Path: "/sys/class/power_supply/main/charging_enabled", OnValue: "1", OffValue: "0"},
		},
		IsBypass: false,
	},
	{
		ID:          "qc_main_input_suspend",
		Name:        "Qualcomm PMIC (main/input_suspend)",
		Description: "Qualcomm SMB charger main input suspend switch",
		Nodes: []SwitchNode{
			{Path: "/sys/class/power_supply/main/input_suspend", OnValue: "0", OffValue: "1"},
		},
		IsBypass: true,
	},
	{
		ID:          "qc_bms_charging_enabled",
		Name:        "Qualcomm BMS (bms/charging_enabled)",
		Description: "Qualcomm Battery Management System charging enabled switch",
		Nodes: []SwitchNode{
			{Path: "/sys/class/power_supply/bms/charging_enabled", OnValue: "1", OffValue: "0"},
		},
		IsBypass: false,
	},

	// OnePlus / OPPO / Realme
	{
		ID:          "oplus_op_disable_charge",
		Name:        "OnePlus/OPPO/Realme (op_disable_charge)",
		Description: "OPlus proprietary charge cutoff switch",
		Nodes: []SwitchNode{
			{Path: "/sys/class/power_supply/battery/op_disable_charge", OnValue: "0", OffValue: "1"},
		},
		IsBypass: false,
	},
	{
		ID:          "oplus_multi_node",
		Name:        "OnePlus/OPPO/Realme (multi-node)",
		Description: "OPlus multi-node charge cutoff (charging_enabled 0 0 + op_disable_charge 0 1)",
		Nodes: []SwitchNode{
			{Path: "/sys/class/power_supply/battery/charging_enabled", OnValue: "0", OffValue: "0"},
			{Path: "/sys/class/power_supply/battery/op_disable_charge", OnValue: "0", OffValue: "1"},
		},
		IsBypass: false,
	},

	// Samsung
	{
		ID:          "samsung_batt_slate_mode",
		Name:        "Samsung (batt_slate_mode)",
		Description: "Samsung kiosk slate mode battery bypass switch",
		Nodes: []SwitchNode{
			{Path: "/sys/class/power_supply/battery/batt_slate_mode", OnValue: "0", OffValue: "1"},
		},
		IsBypass: true,
	},
	{
		ID:          "samsung_store_mode",
		Name:        "Samsung (store_mode)",
		Description: "Samsung store demo battery freeze switch",
		Nodes: []SwitchNode{
			{Path: "/sys/class/power_supply/battery/store_mode", OnValue: "0", OffValue: "1"},
		},
		IsBypass: true,
	},
	{
		ID:          "samsung_siop_level",
		Name:        "Samsung (siop_level)",
		Description: "Samsung SIOP battery charge throttling level",
		Nodes: []SwitchNode{
			{Path: "/sys/class/power_supply/battery/siop_level", OnValue: "100", OffValue: "0"},
		},
		IsBypass: false,
	},

	// MediaTek
	{
		ID:          "mtk_sub_charging_enabled",
		Name:        "MediaTek (sub_charging_enabled)",
		Description: "MediaTek PumpExpress sub-charger enable switch",
		Nodes: []SwitchNode{
			{Path: "/sys/class/power_supply/battery/sub_charging_enabled", OnValue: "1", OffValue: "0"},
		},
		IsBypass: false,
	},
	{
		ID:          "mtk_battery_cmd",
		Name:        "MediaTek (/proc/mtk_battery_cmd/current_cmd)",
		Description: "MediaTek kernel battery command interface",
		Nodes: []SwitchNode{
			{Path: "/proc/mtk_battery_cmd/current_cmd", OnValue: "0::0", OffValue: "0::1"},
		},
		IsBypass: false,
	},

	// Asus ROG
	{
		ID:          "asus_rog_charging_suspend_en",
		Name:        "Asus ROG (charging_suspend_en)",
		Description: "Asus ROG Phone direct charging suspend switch",
		Nodes: []SwitchNode{
			{Path: "/sys/class/power_supply/battery/charging_suspend_en", OnValue: "0", OffValue: "1"},
		},
		IsBypass: true,
	},
	{
		ID:          "asus_rog_charger_limit_en",
		Name:        "Asus ROG (charger_limit_en)",
		Description: "Asus ROG Phone charger limit toggle",
		Nodes: []SwitchNode{
			{Path: "/sys/class/power_supply/battery/charger_limit_en", OnValue: "0", OffValue: "1"},
		},
		IsBypass: false,
	},

	// Universal fallback
	{
		ID:          "universal_charge_control_limit",
		Name:        "Universal (charge_control_limit)",
		Description: "Linux kernel charge control limit switch",
		Nodes: []SwitchNode{
			{Path: "/sys/class/power_supply/battery/charge_control_limit", OnValue: "1", OffValue: "0"},
		},
		IsBypass: false,
	},
	{
		ID:          "universal_mmi_charging_enable",
		Name:        "Universal/Motorola (mmi_charging_enable)",
		Description: "Motorola and generic MMI charging enable switch",
		Nodes: []SwitchNode{
			{Path: "/sys/class/power_supply/battery/mmi_charging_enable", OnValue: "1", OffValue: "0"},
		},
		IsBypass: false,
	},
	{
		ID:          "universal_charging_switch",
		Name:        "Universal (charging_switch)",
		Description: "Generic battery charging switch node",
		Nodes: []SwitchNode{
			{Path: "/sys/class/power_supply/battery/charging_switch", OnValue: "1", OffValue: "0"},
		},
		IsBypass: false,
	},
	{
		ID:          "universal_usb_charging_enabled",
		Name:        "Universal USB (usb/charging_enabled)",
		Description: "USB power supply charging enable switch",
		Nodes: []SwitchNode{
			{Path: "/sys/class/power_supply/usb/charging_enabled", OnValue: "1", OffValue: "0"},
		},
		IsBypass: false,
	},
	{
		ID:          "universal_usb_input_suspend",
		Name:        "Universal USB (usb/input_suspend)",
		Description: "USB power supply input suspend for idle bypass",
		Nodes: []SwitchNode{
			{Path: "/sys/class/power_supply/usb/input_suspend", OnValue: "0", OffValue: "1"},
		},
		IsBypass: true,
	},
}

// inferOnOffValues determines on/off values and bypass mode for dynamically detected switches.
func inferOnOffValues(path string) (onVal, offVal string, isBypass bool) {
	base := filepath.Base(path)
	switch base {
	case "force_main_fcc", "force_main_icl":
		return "550000", "0", false
	case "charge_limit":
		return "-1", "80", false
	case "charge_control_limit", "charge_control_limit_max":
		return "100", "80", false
	case "input_suspend", "op_disable_charge", "store_mode", "batt_slate_mode", "charging_suspend_en", "charger_limit_en":
		return "0", "1", true
	default:
		return "1", "0", false
	}
}

// GetExistingSwitchProfiles inspects sysfs and returns all available switch profiles on the current device.
func GetExistingSwitchProfiles() []SwitchProfile {
	var available []SwitchProfile
	seenPaths := make(map[string]bool)

	for _, p := range AllSwitchProfiles {
		allWritable := true
		for _, n := range p.Nodes {
			if !fileExistsAndWritable(n.Path) {
				allWritable = false
				break
			}
		}
		if allWritable {
			available = append(available, p)
			for _, n := range p.Nodes {
				seenPaths[n.Path] = true
			}
		}
	}

	// Dynamic sysfs auto-scanner fallback for custom/vendor devices
	psDir := "/sys/class/power_supply"
	if entries, err := os.ReadDir(psDir); err == nil {
		keywords := []string{"charging", "suspend", "limit", "fcc", "store_mode", "switch"}
		for _, entry := range entries {
			subPath := filepath.Join(psDir, entry.Name())
			if files, err := os.ReadDir(subPath); err == nil {
				for _, f := range files {
					name := strings.ToLower(f.Name())
					fullPath := filepath.Join(subPath, f.Name())
					if seenPaths[fullPath] {
						continue
					}
					for _, kw := range keywords {
						if strings.Contains(name, kw) && fileExistsAndWritable(fullPath) {
							onVal, offVal, isBypass := inferOnOffValues(fullPath)
							p := SwitchProfile{
								ID:          "auto_" + entry.Name() + "_" + f.Name(),
								Name:        fmt.Sprintf("Auto-detected (%s/%s)", entry.Name(), f.Name()),
								Description: fmt.Sprintf("Dynamically discovered charging switch: %s", fullPath),
								Nodes: []SwitchNode{
									{Path: fullPath, OnValue: onVal, OffValue: offVal},
								},
								IsBypass: isBypass,
							}
							available = append(available, p)
							seenPaths[fullPath] = true
							break
						}
					}
				}
			}
		}
	}

	return available
}
