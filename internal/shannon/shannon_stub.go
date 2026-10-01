//go:build !linux && !android

package shannon

import "fmt"

func IsSupported() bool {
	return false
}

func GetStatus() DeviceStatus {
	return DeviceStatus{
		IsShannon:       false,
		DeviceModel:     "Non-Android/Desktop Environment",
		SoC:             "Host OS",
		ModemType:       "Stub / Emulated",
		RouterAvailable: false,
		DMAvailable:     false,
		ActiveSlot:      0,
		Mode:            "both",
		ActiveLTEBands:  []int{},
		ActiveSABands:   []int{},
		ActiveNSABands:  []int{},
		ActiveWCDMABands: []int{},
		ActiveGSMBands:  []int{},
		ActiveRATs:      []string{},
		SupportedLTE:    []int{1, 2, 3, 4, 5, 7, 8, 20, 28, 38, 40, 41, 66, 71},
		SupportedSA:     []int{1, 2, 3, 5, 7, 8, 20, 28, 38, 40, 41, 66, 71, 77, 78},
		SupportedWCDMA:  []int{1, 2, 4, 5, 8},
		SupportedGSM:    []int{850, 900, 1800, 1900},
	}
}

func ApplyShannonSettings(req LockRequest) error {
	return fmt.Errorf("shannon modem is only supported on Linux/Android ARM64")
}

func LockBands(slot int, mode string, lteBands []int, nrBands []int) error {
	return fmt.Errorf("shannon modem is only supported on Linux/Android ARM64")
}

func ResetBands(slot int) error {
	return fmt.Errorf("shannon modem is only supported on Linux/Android ARM64")
}
