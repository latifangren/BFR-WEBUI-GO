package sysinfo

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ThermalZone struct {
	Name string  `json:"name"`
	Temp float64 `json:"temp"`
}


type DetailedBattery struct {
	Capacity   int     `json:"capacity"`
	Status     string  `json:"status"`
	Temp       float64 `json:"temp"`
	VoltagemV  float64 `json:"voltage_mv"`
	Health     string  `json:"health"`
	Technology string  `json:"technology"`
}


func getDetailedBattery() DetailedBattery {
	var b DetailedBattery
	batDir := "/sys/class/power_supply/battery"

	if capData, err := os.ReadFile(filepath.Join(batDir, "capacity")); err == nil {
		b.Capacity, _ = strconv.Atoi(strings.TrimSpace(string(capData)))
	} else {
		b.Capacity = -1
	}

	if statData, err := os.ReadFile(filepath.Join(batDir, "status")); err == nil {
		b.Status = strings.TrimSpace(string(statData))
	} else {
		b.Status = "Unknown"
	}

	if tempData, err := os.ReadFile(filepath.Join(batDir, "temp")); err == nil {
		t, _ := strconv.ParseFloat(strings.TrimSpace(string(tempData)), 64)
		if t > 100 {
			b.Temp = t / 10.0
		} else {
			b.Temp = t
		}
	}

	if voltData, err := os.ReadFile(filepath.Join(batDir, "voltage_now")); err == nil {
		v, _ := strconv.ParseFloat(strings.TrimSpace(string(voltData)), 64)
		if v > 10000 {
			b.VoltagemV = v / 1000.0 // microvolts to mV
		} else {
			b.VoltagemV = v
		}
	}

	if healthData, err := os.ReadFile(filepath.Join(batDir, "health")); err == nil {
		b.Health = strings.TrimSpace(string(healthData))
	}

	if techData, err := os.ReadFile(filepath.Join(batDir, "technology")); err == nil {
		b.Technology = strings.TrimSpace(string(techData))
	}

	return b
}

func getThermalZones() []ThermalZone {
	var zones []ThermalZone
	files, err := filepath.Glob("/sys/class/thermal/thermal_zone*")
	if err != nil {
		return zones
	}

	prioritizedTypes := []string{"cpu-0", "cpu-1", "cpuss-0", "cpu-top", "soc-thermal", "mtktscpu"}
	excludedKeywords := []string{"modem", "pa_", "pa-", "gpu", "npu", "pmic", "charger", "camera", "xo_", "battery", "bms", "wlan", "soc", "soc_"}

	for _, z := range files {
		typeData, err := os.ReadFile(filepath.Join(z, "type"))
		if err != nil {
			continue
		}
		typeName := strings.TrimSpace(string(typeData))
		lowerType := strings.ToLower(typeName)

		// Strictly exclude non-CPU thermistors
		exclude := false
		for _, kw := range excludedKeywords {
			if strings.Contains(lowerType, kw) {
				exclude = true
				break
			}
		}
		if exclude {
			continue
		}

		// Match only CPU SoC core sensor types
		matched := false
		for _, pType := range prioritizedTypes {
			if strings.Contains(lowerType, pType) {
				matched = true
				break
			}
		}

		// If prioritize match doesn't hit, generic checks
		if !matched {
			if strings.Contains(lowerType, "cpu") || strings.Contains(lowerType, "cpuss") || strings.Contains(lowerType, "mtktscpu") {
				matched = true
			} else if lowerType == "ap-therm" || lowerType == "ap_therm" {
				matched = true
			} else if strings.Contains(lowerType, "tsens_tz_sensor") {
				matched = true
			}
		}

		if !matched {
			continue
		}

		tempData, err := os.ReadFile(filepath.Join(z, "temp"))
		if err != nil {
			continue
		}
		t, _ := strconv.ParseFloat(strings.TrimSpace(string(tempData)), 64)

		if t > 100 && t < 1000 {
			t = t / 10.0
		} else if t >= 1000 && t < 150000 {
			t = t / 1000.0
		}

		// Calculate valid scaled temperatures within 15°C to 115°C
		if t >= 15 && t <= 115 {
			zones = append(zones, ThermalZone{
				Name: typeName,
				Temp: t,
			})
		}
	}
	return zones
}

