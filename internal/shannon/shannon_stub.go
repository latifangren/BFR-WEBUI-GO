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
		RouterAvailable: false,
		DMAvailable:     false,
		ActiveSlot:      0,
		Mode:            "both",
		ActiveLTEBands:  []int{},
		ActiveNRBands:   []int{},
	}
}

func LockBands(slot int, mode string, lteBands []int, nrBands []int) error {
	return fmt.Errorf("shannon modem is only supported on Linux/Android ARM64")
}

func ResetBands(slot int) error {
	return fmt.Errorf("shannon modem is only supported on Linux/Android ARM64")
}
