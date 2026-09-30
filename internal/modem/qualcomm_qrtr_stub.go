//go:build !android && !linux

package modem

import (
	"fmt"
)

type QRTRBandStatus struct {
	RATMode  uint16 `json:"rat_mode"`
	LTEBands []int  `json:"lte_bands"`
	NRBands  []int  `json:"nr_bands"`
	NRMode   string `json:"nr_mode"`
}

type HardwareBandsInfo struct {
	LTEBands []int `json:"lte_bands"`
	NRBands  []int `json:"nr_bands"`
}

type BandLockVerificationResult struct {
	Success        bool     `json:"success"`
	Verified       bool     `json:"verified"`
	TargetSlot     int      `json:"target_slot"`
	TargetSlotName string   `json:"target_slot_name"`
	RequestedLTE   []int    `json:"requested_lte"`
	ActiveLTEBands []int    `json:"active_lte_bands"`
	ActiveNRBands  []int    `json:"active_nr_bands"`
	Message        string   `json:"message"`
	StepLogs       []string `json:"step_logs"`
}

func IsQRTRSupported() bool {
	return false
}

func GetQRTRActiveBands(simSlot int) (*QRTRBandStatus, error) {
	return nil, fmt.Errorf("QRTR not supported on this platform")
}

func ApplyQRTRBandLockDetailed(simSlot int, lteBands []int, nrBands []int, rat string, nrMode string) (*BandLockVerificationResult, error) {
	return nil, fmt.Errorf("QRTR not supported on this platform")
}

func ApplyQRTRBandLock(simSlot int, lteBands []int, nrBands []int, rat string, nrMode string) error {
	return fmt.Errorf("QRTR not supported on this platform")
}

func ResetQRTRBandLock(simSlot int) error {
	return fmt.Errorf("QRTR not supported on this platform")
}

func ApplyLTECellLock(simSlot int, earfcn uint32, pci uint16) error {
	return fmt.Errorf("QRTR not supported on this platform")
}

func ClearLTECellLock(simSlot int) error {
	return fmt.Errorf("QRTR not supported on this platform")
}

func GetLTECellLockStatus(simSlot int) (bool, uint32, uint16, error) {
	return false, 0, 0, fmt.Errorf("QRTR not supported on this platform")
}

func ApplyPLMNLock(simSlot int, mode string, mcc uint16, mnc uint16) error {
	return fmt.Errorf("QRTR not supported on this platform")
}

func GetHardwareBandsInfo(simSlot int) (*HardwareBandsInfo, error) {
	return &HardwareBandsInfo{
		LTEBands: []int{1, 2, 3, 4, 5, 7, 8, 12, 13, 14, 17, 18, 19, 20, 25, 26, 28, 29, 30, 32, 38, 39, 40, 41, 42, 46, 48, 66, 71},
		NRBands:  []int{1, 2, 3, 5, 7, 8, 12, 20, 25, 28, 38, 40, 41, 66, 71, 77, 78},
	}, nil
}

func ScanNearbyCells(simSlot int) (*CellScanResult, error) {
	return &CellScanResult{
		Success:     true,
		TotalCells:  1,
		ScanTime:    "00:00:00",
		ServingCell: &NearbyCellInfo{
			Type:        "lte",
			Role:        "serving",
			CellID:      1234567,
			PCI:         142,
			TAC:         45201,
			EARFCN:      1325,
			Band:        3,
			BandName:    "B3 (1800 MHz)",
			RSRP:        -95.0,
			RSRQ:        -10.5,
			SINR:        14.2,
			DistanceStr: "Serving Cell",
		},
		CarrierAggr: make([]NearbyCellInfo, 0),
		Neighbors:   make([]NearbyCellInfo, 0),
	}, nil
}

func GetCachedScanResult() *CellScanResult {
	return nil
}

