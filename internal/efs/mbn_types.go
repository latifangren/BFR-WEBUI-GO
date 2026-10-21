package efs

const (
	EfsMcfgAutoselectSim1 = "/nv/item_files/mcfg/mcfg_autoselect_by_uim"
	EfsMcfgAutoselectSim2 = "/nv/item_files/mcfg/mcfg_autoselect_by_uim_Subscription01"

	EfsDirMcfgSwGeneric = "/mcfg_sw/generic"
	EfsDirMcfgHwGeneric = "/mcfg_hw/generic"
)

// MbnConfigItem represents an MBN configuration profile file on modem EFS.
type MbnConfigItem struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Type     string `json:"type"`              // "sw" or "hw"
	Category string `json:"category,omitempty"` // e.g. "commercial", "apac", "eu", "na"
	Size     int64  `json:"size"`
	Active   bool   `json:"active"`
}

// MbnAutoselectStatus holds autoselect status for SIM1 and SIM2.
type MbnAutoselectStatus struct {
	Sim1Mode int  `json:"sim1_mode"` // 0: disabled, 1: enabled, 2: with MBN update only
	Sim2Mode int  `json:"sim2_mode"`
	Sim1Auto bool `json:"sim1_auto"`
	Sim2Auto bool `json:"sim2_auto"`
}

// MbnOverview represents all available MBN configs and autoselect state.
type MbnOverview struct {
	ActiveProfile   string              `json:"active_profile"`
	PreviousProfile string              `json:"previous_profile,omitempty"`
	Autoselect      MbnAutoselectStatus `json:"autoselect"`
	SwConfigs       []MbnConfigItem     `json:"sw_configs"`
	HwConfigs       []MbnConfigItem     `json:"hw_configs"`
}
