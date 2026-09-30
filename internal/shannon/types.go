package shannon

// DeviceStatus represents hardware and driver capability for Shannon modem on Google Tensor.
type DeviceStatus struct {
	IsShannon       bool   `json:"is_shannon"`
	DeviceModel     string `json:"device_model"`
	SoC             string `json:"soc"`
	RouterAvailable bool   `json:"router_available"`
	DMAvailable     bool   `json:"dm_available"`
	ActiveSlot      int    `json:"active_slot"`
	Mode            string `json:"mode"` // "both" | "sa" | "nsa" | "lte_only"
	ActiveLTEBands  []int  `json:"active_lte_bands"`
	ActiveNRBands   []int  `json:"active_nr_bands"`
}

// LockRequest represents a payload to lock Shannon frequency bands.
type LockRequest struct {
	Slot     int    `json:"slot"`
	Mode     string `json:"mode"` // "both", "sa", "nsa", "lte_only"
	LTEBands []int  `json:"lte_bands"`
	NRBands  []int  `json:"nr_bands"`
}
