package shannon

// Shannon 5G NR and EN-DC (Dual Connectivity) protobuf wire types and data structures.
// Based on 3GPP TS 38.306 / TS 38.331 and Samsung Shannon Modem ShannonNrUECap schema,
// compatible with Google Tensor G1-G5 (Google Pixel 6, 7, 8, 9, and 10 series).

// NRComponent represents a single carrier component (either 4G LTE or 5G NR) in an EN-DC or NR combo.
type NRComponent struct {
	Band             int    `json:"band"`
	IsNR             bool   `json:"is_nr"`
	IsFR2            bool   `json:"is_fr2"`
	BwClassDl        string `json:"bw_class_dl"`
	BwClassUl        string `json:"bw_class_ul"`
	MimoDl           int    `json:"mimo_dl"`
	MimoUl           int    `json:"mimo_ul"`
	ModulationDl     string `json:"modulation_dl"`
	ModulationUl     string `json:"modulation_ul"`
	MaxScs           int    `json:"max_scs"`            // in kHz: 15, 30, 60, 120
	MaxBwDl          int    `json:"max_bw_dl"`          // in MHz: 5, 10, 20, 50, 100, etc.
	MaxBwUl          int    `json:"max_bw_ul"`          // in MHz
	Bw90MHzSupported bool   `json:"bw_90mhz_supported"`
	SrsTxSwitch      string `json:"srs_tx_switch,omitempty"` // t1r1, t1r2, t1r4, t2r2, t2r4
}

// NRCombo represents an EN-DC, NR-CA, or NR-DC combination.
type NRCombo struct {
	Type           string        `json:"type"`             // "ENDC", "NRCA", "NRDC"
	ComponentsLte  []NRComponent `json:"components_lte"`   // E-UTRA anchor carriers
	ComponentsNr   []NRComponent `json:"components_nr"`    // NR carriers
	BCSNr          uint32        `json:"bcs_nr"`
	BCSEutra       uint32        `json:"bcs_eutra"`
	BCSIntraEndc   uint32        `json:"bcs_intra_endc"`
	SkuBitmask     uint32        `json:"sku_bitmask"`
	PowerClass     string        `json:"power_class"`      // "Default", "PC2", "PC1.5"
	ComboString    string        `json:"combo_string"`     // Human readable string: DC_1A[2]-n78A[4]
	TotalDlStreams int           `json:"total_dl_streams"` // Total spatial streams across all carriers
}

// NRComboDocument represents a parsed ShannonNrUECap document.
type NRComboDocument struct {
	Version      uint32    `json:"version"`
	ID           int32     `json:"id"`
	Combos       []NRCombo `json:"combos"`
	TotalCombos  int       `json:"total_combos"`
	EndcCount    int       `json:"endc_count"`
	NrcaCount    int       `json:"nrca_count"`
	NrdcCount    int       `json:"nrdc_count"`
	SourceFile   string    `json:"source_file,omitempty"`
}

// Raw Shannon Protobuf Wire Types (for intermediate decoding)

type rawFeatureDlPerCCNr struct {
	MaxScs           int32
	MaxMimo          int32
	MaxBw            uint32
	MaxModOrder      int32
	Bw90MHzSupported bool
}

type rawFeatureUlPerCCNr struct {
	MaxScs           int32
	MaxMimo          int32
	MaxBw            int32
	MaxModOrder      int32
	Bw90MHzSupported bool
	MaxNumSRSRes     uint32
}

type rawComponentNr struct {
	RawBand            uint32
	RawBwClassDl       uint32
	RawBwClassUl       uint32
	DlFeatureIndex     uint32
	UlFeatureIndex     uint32
	DlFeaturePerCCIds  []uint32
	UlFeaturePerCCIds  []uint32
	SrsTxSwitch        int32
}

type rawComboNr struct {
	Components []rawComponentNr
	SkuBitmask uint32
}

type rawComboGroupNr struct {
	BcsNr                 uint32
	BcsIntraEndc          uint32
	BcsEutra              uint32
	PowerClass            int32
	IntraBandEndcSupport  int32
	SupportedNumTag       uint32
	Combos                []rawComboNr
}
