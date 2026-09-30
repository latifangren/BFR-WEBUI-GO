package modem

import (
	"fmt"
)

// NearbyCellInfo represents detailed RF measurements for a detected cell tower.
type NearbyCellInfo struct {
	Type          string  `json:"type"`            // "lte", "nr", "gsm", "wcdma"
	Role          string  `json:"role"`            // "serving", "scell", "neighbor"
	CellID        uint32  `json:"cell_id"`         // eNodeB Global Cell ID
	PCI           uint16  `json:"pci"`             // Physical Cell ID (0-503)
	TAC           uint16  `json:"tac"`             // Tracking Area Code
	EARFCN        uint32  `json:"earfcn"`          // Radio frequency channel number
	Band          int     `json:"band"`            // Band number (e.g. 3, 5, 40)
	BandName      string  `json:"band_name"`       // "B3 (1800 MHz)"
	RSRP          float64 `json:"rsrp"`            // Signal power in dBm (e.g. -95.0)
	RSRQ          float64 `json:"rsrq"`            // Signal quality in dB (e.g. -11.0)
	RSSI          float64 `json:"rssi"`            // Received signal strength
	SINR          float64 `json:"sinr"`            // Signal to interference ratio
	TimingAdvance int     `json:"timing_advance"`  // TA steps (0-1282, -1 if unavailable)
	DistanceMeter int     `json:"distance_meters"` // Distance in meters (~78m * TA, 0 if unavailable)
	DistanceStr   string  `json:"distance_str"`    // "~390 m" or "Unknown"
	IsCurrentLock bool    `json:"is_current_lock"` // true if matching active cell lock
}

// CellScanResult holds all serving, carrier aggregation, and neighbor cell towers detected.
type CellScanResult struct {
	Success      bool             `json:"success"`
	ServingCell  *NearbyCellInfo  `json:"serving_cell,omitempty"`
	CarrierAggr  []NearbyCellInfo `json:"carrier_aggr"`
	Neighbors    []NearbyCellInfo `json:"neighbors"`
	TotalCells   int              `json:"total_cells"`
	ScanTime     string           `json:"scan_time"`
	ErrorMessage string           `json:"error_message,omitempty"`
}

// SavedCellTower represents a user-bookmarked tower profile.
type SavedCellTower struct {
	ID        string  `json:"id"`
	Label     string  `json:"label"`
	EARFCN    uint32  `json:"earfcn"`
	PCI       uint16  `json:"pci"`
	BandName  string  `json:"band_name"`
	CellID    uint32  `json:"cell_id,omitempty"`
	LastRSRP  float64 `json:"last_rsrp,omitempty"`
	CreatedAt string  `json:"created_at"`
}

// EarfcnToBand maps 3GPP LTE EARFCN frequency numbers to their corresponding LTE band.
func EarfcnToBand(earfcn uint32) (int, string) {
	switch {
	case earfcn <= 599:
		return 1, "B1 (2100 MHz)"
	case earfcn >= 600 && earfcn <= 1199:
		return 2, "B2 (1900 MHz)"
	case earfcn >= 1200 && earfcn <= 1949:
		return 3, "B3 (1800 MHz)"
	case earfcn >= 1950 && earfcn <= 2399:
		return 4, "B4 (AWS 1700/2100)"
	case earfcn >= 2400 && earfcn <= 2649:
		return 5, "B5 (850 MHz)"
	case earfcn >= 2750 && earfcn <= 3449:
		return 7, "B7 (2600 MHz)"
	case earfcn >= 3450 && earfcn <= 3799:
		return 8, "B8 (900 MHz)"
	case earfcn >= 5000 && earfcn <= 5179:
		return 12, "B12 (700 MHz)"
	case earfcn >= 5180 && earfcn <= 5279:
		return 13, "B13 (700 MHz)"
	case earfcn >= 5730 && earfcn <= 5849:
		return 17, "B17 (700 MHz)"
	case earfcn >= 5850 && earfcn <= 5999:
		return 18, "B18 (850 MHz)"
	case earfcn >= 6000 && earfcn <= 6149:
		return 19, "B19 (850 MHz)"
	case earfcn >= 6150 && earfcn <= 6449:
		return 20, "B20 (800 MHz)"
	case earfcn >= 6600 && earfcn <= 7399:
		return 26, "B26 (850 MHz)"
	case earfcn >= 9210 && earfcn <= 9659:
		return 28, "B28 (700 MHz)"
	case earfcn >= 37750 && earfcn <= 38249:
		return 38, "B38 (TDD 2600)"
	case earfcn >= 38250 && earfcn <= 38649:
		return 39, "B39 (TDD 1900)"
	case earfcn >= 38650 && earfcn <= 39649:
		return 40, "B40 (TDD 2300)"
	case earfcn >= 39650 && earfcn <= 41589:
		return 41, "B41 (TDD 2500)"
	case earfcn >= 65536 && earfcn <= 66435:
		return 66, "B66 (AWS-3)"
	case earfcn >= 68586 && earfcn <= 68935:
		return 71, "B71 (600 MHz)"
	default:
		return 0, fmt.Sprintf("EARFCN %d", earfcn)
	}
}

// EstimateDistanceMeters computes physical tower distance from LTE Timing Advance (TA).
// In LTE, 1 TA step is approximately 78.12 meters (Ts = 1 / (15000 * 2048), 16 Ts * c / 2).
func EstimateDistanceMeters(ta int) (int, string) {
	if ta < 0 || ta > 1282 {
		return -1, "N/A (No TA report)"
	}
	meters := int(float64(ta) * 78.12)
	if meters >= 1000 {
		return meters, fmt.Sprintf("~%.2f km (TA: %d)", float64(meters)/1000.0, ta)
	}
	return meters, fmt.Sprintf("~%d m (TA: %d)", meters, ta)
}
