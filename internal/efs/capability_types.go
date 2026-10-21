package efs

// CapabilityFileStatus represents the presence, readability, and write-only nature of an EFS capability file.
type CapabilityFileStatus struct {
	Path        string `json:"path"`
	Present     bool   `json:"present"`
	Readable    bool   `json:"readable"`
	WriteOnly   bool   `json:"write_only"`
	Size        int64  `json:"size,omitempty"`
	Description string `json:"description"`
}

// LTESectionState holds toggle and parameter state for LTE RRC capabilities.
type LTESectionState struct {
	LTECA        bool   `json:"lte_ca"`         // [LTE_CA]
	CAFilter     string `json:"ca_filter"`      // [LTE_CA_FILTER]
	ULCA         bool   `json:"ul_ca"`          // [LTE_UL_CA]
	DL256Q       bool   `json:"dl_256q"`        // [LTE_DL_256Q]
	UL256Q       bool   `json:"ul_256q"`        // [LTE_UL_256Q]
	DLMIMO       bool   `json:"dl_mimo"`        // [LTE_DL_MIMO]
	MaxMIMOLayer uint8  `json:"max_mimo_layer"` // 2 or 4 layers
	HORXD        bool   `json:"horxd"`          // [LTE_HORXD] Adaptive Rx Diversity
	Rx0          bool   `json:"rx0"`            // [LTE_RX0]
	Rx1          bool   `json:"rx1"`            // [LTE_RX1]
	Rx2          bool   `json:"rx2"`            // [LTE_RX2]
	Rx3          bool   `json:"rx3"`            // [LTE_RX3]
	VoLTE        bool   `json:"volte"`          // [LTE_VOICE]
}

// NSASectionState holds toggle state for NSA capabilities.
type NSASectionState struct {
	NSALTE     bool `json:"nsa_lte"`      // [NSA_LTE]
	NSALTEDSDA bool `json:"nsa_lte_dsda"` // [NSA_LTE_DSDA]
	NSANDDS    bool `json:"nsa_ndds"`     // [NSA_NDDS] NSA during data-call on other SIM
	NSALTEDSS  bool `json:"nsa_lte_dss"`  // [NSA_LTE_DSS] Dynamic Spectrum Sharing
}

// NRSectionState holds toggle and parameter state for NR (5G SA/NSA) capabilities.
type NRSectionState struct {
	NRCA        bool `json:"nr_ca"`         // [NR_CA]
	DL256Q      bool `json:"dl_256q"`       // [NR_DL_256Q]
	UL256Q      bool `json:"ul_256q"`       // [NR_UL_256Q]
	DLMIMO      bool `json:"dl_mimo"`       // [NR_DL_MIMO]
	ULMIMO      bool `json:"ul_mimo"`       // [NR_UL_MIMO]
	VoNR        bool `json:"vonr"`          // [NR_VOICE]
	RedCap      bool `json:"redcap"`        // [NR_REDCAP] Reduced Capability
	NRNRDSDA    bool `json:"nr_nr_dsda"`    // [NR_NR_DSDA]
	LimitRFMIMO bool `json:"limit_rf_mimo"` // [NR_LIMIT_RF_MIMO]
}

// CapabilityState represents the full 38-token RRC capability state.
type CapabilityState struct {
	TokenString string                 `json:"token_string"` // Serialized token string
	LTE         LTESectionState        `json:"lte"`
	NSA         NSASectionState        `json:"nsa"`
	NR          NRSectionState         `json:"nr"`
	Files       []CapabilityFileStatus `json:"files,omitempty"`
}

// Known EFS capability paths relative to /nv/item_files/modem/
const (
	EfsCapLteFeatureEnable     = "/nv/item_files/modem/lte/rrc/efs/lte_feature_enable"
	EfsCapLteFeatureDisable    = "/nv/item_files/modem/lte/rrc/efs/lte_feature_disable"
	EfsCapLteCADisable         = "/nv/item_files/modem/lte/common/ca_disable"
	EfsCapLteML1ARD            = "/nv/item_files/modem/lte/ML1/ard_config"
	EfsCapLteML1RxSelect       = "/nv/item_files/modem/lte/ML1/rx_select"
	EfsCapMcsAntDivMaster      = "/nv/item_files/mcs/trm/ant_div_switch_master"
	EfsCapLteMaxMimoLayer      = "/nv/item_files/modem/lte/rrc/efs/max_num_mimo_lyr"
	EfsCapNrCapControlDL256Q   = "/nv/item_files/modem/nr5g/RRC/cap_control_dl_256qam"
	EfsCapNrCapDisableUL256Q   = "/nv/item_files/modem/nr5g/RRC/cap_disable_ul_256qam"
	EfsCapNrCapLimitRFMIMO     = "/nv/item_files/modem/nr5g/RRC/cap_limit_rf_mimo"
	EfsCapNrCapNrcaDowngrade1CC = "/nv/item_files/modem/nr5g/RRC/cap_nrca_downgrade_1cc"
)
