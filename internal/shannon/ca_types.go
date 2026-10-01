package shannon

// Shannon LTE CA protobuf wire types and data structures
// Compatible with ShannonLteUECap proto on Pixel 9 (Tensor G4) and Pixel 10 (Tensor G5).

// Component represents a single LTE carrier component in a CA combination.
type Component struct {
	Band          int32  `json:"band"`
	BwClassMimoDl int32  `json:"bwClassMimoDl"`
	BwClassMimoUl int32  `json:"bwClassMimoUl"`
	DlClass       string `json:"dl_class"`
	DlMimo        int    `json:"dl_mimo"`
	UlClass       string `json:"ul_class"`
}

// Combo represents a carrier aggregation combination.
type Combo struct {
	Components    []Component `json:"components"`
	BCS           uint32      `json:"bcs"`
	ConfigMaskLow uint64      `json:"configMaskLow"`
	ConfigMaskHigh uint32     `json:"configMaskHigh"`
	ConfIDs       []int       `json:"conf_ids"`
	ComboString   string      `json:"combo_string"`
	TotalStreams  int         `json:"total_streams"`
}

// ComboDocument represents the root ShannonLteUECap document.
type ComboDocument struct {
	Version int32   `json:"version"`
	Bitmask uint32  `json:"bitmask"`
	Combos  []Combo `json:"combos"`
}

// CAEditorStatus represents device capability and file discovery for LTE CA combos.
type CAEditorStatus struct {
	SupportedDevice bool     `json:"supported_device"` // True for Pixel 9 & Pixel 10
	DeviceModel     string   `json:"device_model"`
	TensorGen       string   `json:"tensor_gen"`
	DetectedPaths   []string `json:"detected_paths"`
	ActiveFilePath  string   `json:"active_file_path,omitempty"`
	ComboCount      int      `json:"combo_count"`
	Version         int32    `json:"version"`
	Bitmask         uint32   `json:"bitmask"`
	WarningMessage  string   `json:"warning_message,omitempty"`
}

// ValidationResult represents rule validation checks on CA combos.
type ValidationResult struct {
	Valid          bool     `json:"valid"`
	ErrorCount     int      `json:"error_count"`
	WarningCount   int      `json:"warning_count"`
	Issues         []string `json:"issues"`
	RepairedCombos []Combo  `json:"repaired_combos,omitempty"`
}

// AutoGenRequest specifies parameters to generate combinations.
type AutoGenRequest struct {
	InputCombos []string `json:"input_combos"`
	MaxCC       int      `json:"max_cc"`
	ULOption    string   `json:"ul_option"` // "none", "fdd_aa", "tdd_aa", "tdd_fdd"
	BCSOption   uint32   `json:"bcs_option"`
	ConfIDs     []int    `json:"conf_ids"`
}
