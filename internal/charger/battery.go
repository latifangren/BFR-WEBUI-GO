package charger

import (
	"math"
	"os"
	"strings"
)

// GetBatteryInfo queries kernel power supply drivers to obtain battery health and status metrics.
func GetBatteryInfo() BatteryInfo {
	info := BatteryInfo{
		Capacity:    -1,
		Status:      "Unknown",
		PowerSupply: "unknown",
	}

	baseCandidates := []string{
		"/sys/class/power_supply/battery",
		"/sys/class/power_supply/bms",
		"/sys/class/power_supply/main",
	}
	if mockBase := os.Getenv("BFR_MOCK_SYSFS_POWER_SUPPLY"); mockBase != "" {
		baseCandidates = []string{mockBase}
	}

	var activeBase string
	for _, cand := range baseCandidates {
		if fi, err := os.Stat(cand); err == nil && fi.IsDir() {
			activeBase = cand
			info.PowerSupply = fi.Name()
			break
		}
	}

	// 1. Capacity
	if activeBase != "" {
		if capVal, err := readSysfsInt(activeBase + "/capacity"); err == nil {
			info.Capacity = capVal
		}
	}
	if info.Capacity < 0 {
		for _, cand := range baseCandidates {
			if capVal, err := readSysfsInt(cand + "/capacity"); err == nil {
				info.Capacity = capVal
				if info.PowerSupply == "unknown" {
					info.PowerSupply = cand
				}
				break
			}
		}
	}

	// 2. Status
	if activeBase != "" {
		if st, err := readSysfs(activeBase + "/status"); err == nil && st != "" {
			info.Status = st
		}
	}
	if info.Status == "Unknown" {
		for _, cand := range baseCandidates {
			if st, err := readSysfs(cand + "/status"); err == nil && st != "" {
				info.Status = st
				break
			}
		}
	}

	// 3. CurrentNow (µA -> mA, normalize sign)
	var rawCurrent float64
	var foundCurrent bool
	if activeBase != "" {
		if c, err := readSysfsFloat(activeBase + "/current_now"); err == nil {
			rawCurrent = c
			foundCurrent = true
		}
	}
	if !foundCurrent {
		for _, cand := range baseCandidates {
			if c, err := readSysfsFloat(cand + "/current_now"); err == nil {
				rawCurrent = c
				foundCurrent = true
				break
			}
		}
	}

	if foundCurrent {
		curr := rawCurrent
		// Convert µA to mA if abs(val) > 10000
		if math.Abs(curr) > 10000 {
			curr = curr / 1000.0
		}
		// Normalize sign based on status
		if strings.EqualFold(info.Status, "Charging") {
			if curr < 0 {
				curr = -curr
			}
		} else if strings.EqualFold(info.Status, "Discharging") {
			if curr > 0 {
				curr = -curr
			}
		}
		info.CurrentNow = math.Round(curr*10) / 10
	}

	// 4. VoltageNow (µV -> V)
	var rawVoltage float64
	var foundVoltage bool
	if activeBase != "" {
		if v, err := readSysfsFloat(activeBase + "/voltage_now"); err == nil {
			rawVoltage = v
			foundVoltage = true
		}
	}
	if !foundVoltage {
		for _, cand := range baseCandidates {
			if v, err := readSysfsFloat(cand + "/voltage_now"); err == nil {
				rawVoltage = v
				foundVoltage = true
				break
			}
		}
	}

	if foundVoltage {
		volt := rawVoltage
		if volt > 100000 {
			volt = volt / 1000000.0
		} else if volt > 1000 {
			volt = volt / 1000.0
		}
		info.VoltageNow = math.Round(volt*100) / 100
	}

	// 5. Temperature (kernel 0.1°C -> °C)
	var rawTemp float64
	var foundTemp bool
	if activeBase != "" {
		if t, err := readSysfsFloat(activeBase + "/temp"); err == nil {
			rawTemp = t
			foundTemp = true
		}
	}
	if !foundTemp {
		for _, cand := range baseCandidates {
			if t, err := readSysfsFloat(cand + "/temp"); err == nil {
				rawTemp = t
				foundTemp = true
				break
			}
		}
	}

	if foundTemp {
		temp := rawTemp
		if temp > 100 {
			temp = temp / 10.0
		}
		info.Temperature = math.Round(temp*10) / 10
	}

	// 6. IsOnline (usb/online or ac/online or charger/online)
	onlinePaths := []string{
		"/sys/class/power_supply/usb/online",
		"/sys/class/power_supply/ac/online",
		"/sys/class/power_supply/charger/online",
		"/sys/class/power_supply/wireless/online",
	}
	if mockBase := os.Getenv("BFR_MOCK_SYSFS_POWER_SUPPLY"); mockBase != "" {
		onlinePaths = append(onlinePaths, mockBase+"/online")
	}
	for _, p := range onlinePaths {
		if val, err := readSysfs(p); err == nil && val == "1" {
			info.IsOnline = true
			break
		}
	}

	// 7. IsIdle (if IsOnline and abs(current_now) <= 30 mA or status == "Not charging" / "Idle")
	if info.IsOnline {
		isLowCurrent := math.Abs(info.CurrentNow) <= 30.0
		isNotCharging := strings.EqualFold(info.Status, "Not charging") || strings.EqualFold(info.Status, "Idle")
		if isLowCurrent || isNotCharging {
			info.IsIdle = true
		}
	}

	return info
}
