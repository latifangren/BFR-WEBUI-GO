package efs

// NVDataType represents the semantic encoding of a stored NV item value.
type NVDataType string

const (
	NVDataHex    NVDataType = "hex"
	NVDataBCD    NVDataType = "bcd"
	NVDataASCII  NVDataType = "ascii"
	NVDataBinary NVDataType = "binary"
)

// NVItem holds the metadata for a single Qualcomm NV item from the built-in catalog.
type NVItem struct {
	ID          uint16     `json:"id"`
	Name        string     `json:"name"`
	Category    string     `json:"category"`
	Description string     `json:"description"`
	Size        uint16     `json:"size"`
	IsExtended  bool       `json:"is_extended"`
	MaxIndex    uint16     `json:"max_index,omitempty"`
	DataType    NVDataType `json:"data_type"`
	ReadOnly    bool       `json:"read_only"`
}

// nvCatalog is the built-in repository of known Qualcomm NV item definitions.
// Ported from NVDatabase.h (QC-Native-Diag, MIT License).
var nvCatalog = map[uint16]NVItem{
	// === Identity ===
	0:    {0, "ESN", "Identity", "Electronic Serial Number", 8, false, 0, NVDataBinary, true},
	1:    {1, "MEID", "Identity", "Mobile Equipment Identifier", 14, false, 0, NVDataHex, true},
	550:  {550, "IMEI 1", "Identity", "Primary IMEI (BCD packed, 9 bytes)", 9, false, 0, NVDataBCD, false},
	2497: {2497, "IMEI 2 (Alt1)", "Identity", "Secondary IMEI — Alternative location 1", 9, false, 0, NVDataBCD, false},
	5014: {5014, "IMEI 2 (Alt2)", "Identity", "Secondary IMEI — Alternative location 2", 9, false, 0, NVDataBCD, false},
	1192: {1192, "IMSI", "Identity", "International Mobile Subscriber Identity", 9, false, 0, NVDataBCD, false},
	2824: {2824, "IMSI (M)", "Identity", "IMSI_M for multiple SIM configurations", 9, false, 0, NVDataBCD, false},
	71:   {71, "SIM_NAME", "Identity", "SIM Card Operator Name", 16, false, 0, NVDataASCII, false},

	// === Network ===
	176:  {176, "MCC", "Network", "Mobile Country Code", 2, false, 0, NVDataBinary, false},
	177:  {177, "MNC", "Network", "Mobile Network Code", 2, false, 0, NVDataBinary, false},
	85:   {85, "Home SID", "Network", "Home System ID", 2, false, 0, NVDataBinary, false},
	86:   {86, "Home NID", "Network", "Home Network ID", 2, false, 0, NVDataBinary, false},
	161:  {161, "MIN1", "Network", "Mobile Identification Number 1", 4, false, 0, NVDataBCD, false},
	162:  {162, "MIN2", "Network", "Mobile Identification Number 2", 2, false, 0, NVDataBCD, false},
	10:   {10, "MDN", "Network", "Mobile Directory Number (Phone Number)", 10, false, 0, NVDataASCII, false},
	178:  {178, "MDN_BCD", "Network", "MDN in BCD format", 10, false, 0, NVDataBCD, false},
	259:  {259, "PRL Version", "Network", "Preferred Roaming List Version", 2, false, 0, NVDataBinary, false},
	442:  {442, "SID/NID Pairs", "Network", "System ID / Network ID Pairs", 40, false, 0, NVDataBinary, false},
	4105: {4105, "ICCID", "Network", "SIM Card Serial Number (ICCID)", 10, true, 0, NVDataBCD, false},
	4106: {4106, "SIM State", "Network", "SIM Card State", 1, true, 0, NVDataBinary, true},
	4107: {4107, "SIM Type", "Network", "SIM Card Type", 1, true, 0, NVDataBinary, true},

	// === Security ===
	6:   {6, "Security Code", "Security", "Phone Lock Code", 2, false, 0, NVDataBinary, false},
	16:  {16, "SPC", "Security", "Service Programming Code (6 ASCII digits)", 6, false, 0, NVDataASCII, false},
	466: {466, "SP Lock", "Security", "Service Programming Lock Status", 1, false, 0, NVDataBinary, false},
	82:  {82, "Lock Code", "Security", "Device Lock Code", 4, false, 0, NVDataBinary, false},

	// === Radio ===
	441:  {441, "Band Preference", "Radio", "Preferred Band Configuration", 8, false, 0, NVDataBinary, false},
	1877: {1877, "RF Cal Date", "Radio", "RF Calibration Date", 6, false, 0, NVDataBCD, false},
	6828: {6828, "LTE Band Pref", "Radio", "LTE Band Preference (64-bit bitmask)", 8, false, 0, NVDataBinary, false},
	6829: {6829, "LTE Band Pref Ext", "Radio", "LTE Band Preference Extended (128-bit)", 16, false, 0, NVDataBinary, false},
	2954: {2954, "RF_BC_CONFIG", "Radio", "RF Band Class Configuration", 256, false, 0, NVDataBinary, false},
	453:  {453, "FTM Mode", "Radio", "Factory Test Mode Status", 1, false, 0, NVDataBinary, false},

	// === System ===
	447:  {447, "Firmware Rev", "System", "Mobile Firmware Revision", 2, false, 0, NVDataBinary, true},
	2825: {2825, "SW Version", "System", "Software Version String", 20, false, 0, NVDataASCII, true},
	5:    {5, "MOB_CAI_REV", "System", "Mobile CAI Revision", 1, false, 0, NVDataBinary, true},
	4:    {4, "MOB_MODEL", "System", "Mobile Model Number", 1, false, 0, NVDataBinary, true},
	3:    {3, "MOB_TERM_HOME", "System", "Home Terminal Setting", 5, false, 0, NVDataBinary, false},
	2:    {2, "MOB_TERM_FOR_SID", "System", "Foreign SID Termination", 5, false, 0, NVDataBinary, false},
	1943: {1943, "MEID_NV", "System", "MEID NV Storage", 7, false, 0, NVDataHex, true},

	// === User ===
	14: {14, "Air Timer", "User", "Total Air Time Counter", 4, false, 0, NVDataBinary, false},
	15: {15, "Roam Timer", "User", "Total Roaming Time Counter", 4, false, 0, NVDataBinary, false},
	67: {67, "Life Timer", "User", "Device Life Timer (read-only)", 4, false, 0, NVDataBinary, true},
	13: {13, "Call Count", "User", "Total Call Count", 2, false, 0, NVDataBinary, false},
	11: {11, "Auto Lock", "User", "Auto Lock Timer", 2, false, 0, NVDataBinary, false},

	// === Advanced ===
	429:  {429, "Data Scrubbing", "Advanced", "Data Scrubbing Status", 1, false, 0, NVDataBinary, false},
	21:   {21, "CDMA SID", "Advanced", "CDMA System ID", 2, false, 0, NVDataBinary, false},
	22:   {22, "CDMA NID", "Advanced", "CDMA Network ID", 2, false, 0, NVDataBinary, false},
	8:    {8, "CDMA RX State", "Advanced", "CDMA Receive State", 1, false, 0, NVDataBinary, false},
	9:    {9, "CDMA Good Frames", "Advanced", "Good Frames Counter", 2, false, 0, NVDataBinary, false},
	553:  {553, "RF Config", "Advanced", "RF Configuration", 4, false, 0, NVDataBinary, false},
	4964: {4964, "WiFi MAC", "Advanced", "WiFi MAC Address (6 bytes)", 6, true, 0, NVDataHex, false},
	4965: {4965, "BT Address", "Advanced", "Bluetooth MAC Address (6 bytes)", 6, true, 0, NVDataHex, false},
}

// NVCategories lists all known NV item categories in display order.
var NVCategories = []string{
	"Identity",
	"Network",
	"Security",
	"Radio",
	"System",
	"User",
	"Advanced",
}

// GetNVItem returns the NVItem metadata for a given NV ID, or nil if unknown.
func GetNVItem(id uint16) *NVItem {
	if item, ok := nvCatalog[id]; ok {
		return &item
	}
	return nil
}

// GetAllNVItems returns all known NV items from the catalog.
func GetAllNVItems() []NVItem {
	items := make([]NVItem, 0, len(nvCatalog))
	for _, item := range nvCatalog {
		items = append(items, item)
	}
	return items
}

// GetNVItemsByCategory returns all NV items belonging to the specified category.
func GetNVItemsByCategory(category string) []NVItem {
	var items []NVItem
	for _, item := range nvCatalog {
		if item.Category == category {
			items = append(items, item)
		}
	}
	return items
}
