//go:build (!linux && !android) || nomodem

package modem

import "fmt"

var errPdcNotSupported = fmt.Errorf("qualcomm PDC service is only supported on Linux/Android ARM64 platform without nomodem tag")

func QueryActiveMbnProfile(simSlot int) (string, error) {
	return "", errPdcNotSupported
}

func ApplyMbnProfile(profileID string, simSlot int) error {
	return errPdcNotSupported
}

func GetPreviousMbnProfile() string {
	return ""
}

func SavePreviousMbnProfile(profile string) {}
