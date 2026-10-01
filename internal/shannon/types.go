package shannon

// DeviceStatus represents hardware and driver capability for Shannon modem on Google Tensor.
type DeviceStatus struct {
	IsShannon       bool   `json:"is_shannon"`
	DeviceModel     string `json:"device_model"`
	SoC             string `json:"soc"`
	ModemType       string `json:"modem_type"`
	RouterAvailable bool   `json:"router_available"`
	DMAvailable     bool   `json:"dm_available"`
	ActiveSlot      int    `json:"active_slot"`

	// NR Mode: "both" (SA+NSA), "nsa" (NSA only), "sa" (SA only), "lte_only" (Disabled)
	Mode string `json:"mode"`

	// Active & Manual Bands
	LTEManualEnabled   bool  `json:"lte_manual_enabled"`
	ActiveLTEBands     []int `json:"active_lte_bands"`
	SAManualEnabled    bool  `json:"sa_manual_enabled"`
	ActiveSABands      []int `json:"active_sa_bands"`
	NSAManualEnabled   bool  `json:"nsa_manual_enabled"`
	ActiveNSABands     []int `json:"active_nsa_bands"`
	WCDMAManualEnabled bool  `json:"wcdma_manual_enabled"`
	ActiveWCDMABands   []int `json:"active_wcdma_bands"`
	GSMManualEnabled   bool  `json:"gsm_manual_enabled"`
	ActiveGSMBands     []int `json:"active_gsm_bands"`

	// RAT Active status: ["GSM", "WCDMA", "LTE", "NR"]
	ActiveRATs []string `json:"active_rats"`

	// VoNR: 1 (Forced ON), 0 (Forced OFF), -1 (Default / Carrier Policy), -2 (Unknown)
	VoNRState int `json:"vonr_state"`

	// Hardware Supported Bands
	SupportedLTE   []int `json:"supported_lte"`
	SupportedSA    []int `json:"supported_sa"`
	SupportedWCDMA []int `json:"supported_wcdma"`
	SupportedGSM   []int `json:"supported_gsm"`

	// Diagnostics
	NSGRunning    bool   `json:"nsg_running"`
	StatusMessage string `json:"status_message,omitempty"`
}

// LockRequest represents a comprehensive payload to lock Shannon frequency bands, RATs, and VoNR.
type LockRequest struct {
	Slot int `json:"slot"`

	// NR Mode: "both", "nsa", "sa", "lte_only"
	Mode      string `json:"mode,omitempty"`
	ApplyMode bool   `json:"apply_mode"`

	// LTE Bands
	LTEBands []int `json:"lte_bands,omitempty"`
	ApplyLTE bool  `json:"apply_lte"`
	LTEAll   bool  `json:"lte_all"`

	// 5G NR SA Bands
	SABands []int `json:"sa_bands,omitempty"`
	ApplySA bool  `json:"apply_sa"`
	SAAll   bool  `json:"sa_all"`

	// 5G NR NSA Bands
	NSABands []int `json:"nsa_bands,omitempty"`
	ApplyNSA bool  `json:"apply_nsa"`
	NSAAll   bool  `json:"nsa_all"`

	// WCDMA 3G Bands
	WCDMABands []int `json:"wcdma_bands,omitempty"`
	ApplyWCDMA bool  `json:"apply_wcdma"`
	WCDMAAll   bool  `json:"wcdma_all"`

	// GSM 2G Bands
	GSMBands []int `json:"gsm_bands,omitempty"`
	ApplyGSM bool  `json:"apply_gsm"`
	GSMAll   bool  `json:"gsm_all"`

	// RAT Lock (e.g. ["LTE", "NR"])
	RATs     []string `json:"rats,omitempty"`
	ApplyRAT bool     `json:"apply_rat"`

	// VoNR (1 = ON, 0 = OFF, -1 = Default)
	VoNRState int  `json:"vonr_state,omitempty"`
	ApplyVoNR bool `json:"apply_vonr"`
}
