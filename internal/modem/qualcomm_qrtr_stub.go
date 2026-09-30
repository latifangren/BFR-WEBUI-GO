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
