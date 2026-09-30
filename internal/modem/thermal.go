package modem

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bfr-webui-go/internal/config"
)

// ThermalSensors represents temperature sensor readings in degrees Celsius.
type ThermalSensors struct {
	PA1Sub6C       float64 `json:"pa1_sub6_c"`
	PA2Sub6C       float64 `json:"pa2_sub6_c"`
	CellFrontCFC   float64 `json:"cell_front_cf_c"`
	ModemSkinC     float64 `json:"modem_skin_c"`
	RfPmicC        float64 `json:"rf_pmic_c"`
	ModemBasebandC float64 `json:"modem_baseband_c"`
	ApSocC         float64 `json:"ap_soc_c"`
	BatteryC       float64 `json:"battery_c"`
}

// ThermalThrottlers represents cooling device throttling state counters.
type ThermalThrottlers struct {
	ModemPaFr1Dsc  int `json:"modem_pa_fr1_dsc"`
	ModemSkinNrDsc int `json:"modem_skin_nr_dsc"`
	ModemPaFr1     int `json:"modem_pa_fr1"`
}

// CriticalThresholds represents hardware safety temperature limits.
type CriticalThresholds struct {
	ModemSkinLimitC float64 `json:"modem_skin_limit_c"`
	PA1CutoffLimitC float64 `json:"pa1_cutoff_limit_c"`
}

// Thermal5GStatus holds the complete thermal diagnostic report and root cause.
type Thermal5GStatus struct {
	Available          bool               `json:"available"`
	NetworkType        string             `json:"network_type"`
	Is5GDisconnect     bool               `json:"is_5g_disconnect"`
	NRBand             string             `json:"nr_band"`
	NRRSRP             int                `json:"nr_rsrp"`
	TrafficMbps        float64            `json:"traffic_mbps"`
	ThermalHALStatus   int                `json:"thermal_hal_status"`
	ThermalHALDesc     string             `json:"thermal_hal_desc"`
	Sensors            ThermalSensors     `json:"sensors"`
	Throttlers         ThermalThrottlers  `json:"throttlers"`
	CriticalThresholds CriticalThresholds `json:"critical_thresholds"`
	IsThrottled        bool               `json:"is_throttled"`
	Culprit            string             `json:"culprit"`
	CulpritDetail      string             `json:"culprit_detail"`
	Error              string             `json:"error,omitempty"`
}

func readThermalZoneTemp(typeMatch string) float64 {
	zones, err := filepath.Glob("/sys/class/thermal/thermal_zone*")
	if err != nil {
		return 0.0
	}

	for _, z := range zones {
		typeData, err := os.ReadFile(filepath.Join(z, "type"))
		if err != nil {
			continue
		}
		zType := strings.TrimSpace(string(typeData))
		if strings.EqualFold(zType, typeMatch) {
			tempData, err := os.ReadFile(filepath.Join(z, "temp"))
			if err != nil {
				continue
			}
			rawStr := strings.TrimSpace(string(tempData))
			if rawVal, err := strconv.ParseFloat(rawStr, 64); err == nil {
				if rawVal > 1000.0 {
					return rawVal / 1000.0
				}
				return rawVal
			}
		}
	}
	return 0.0
}

func readFirstMatchingThermalZoneTemp(typeMatches []string) float64 {
	for _, m := range typeMatches {
		if val := readThermalZoneTemp(m); val > 0.0 {
			return val
		}
	}
	return 0.0
}

func readCoolingDeviceState(typeMatch string) (cur int, max int) {
	devices, err := filepath.Glob("/sys/class/thermal/cooling_device*")
	if err != nil {
		return 0, 0
	}

	for _, d := range devices {
		typeData, err := os.ReadFile(filepath.Join(d, "type"))
		if err != nil {
			continue
		}
		dType := strings.TrimSpace(string(typeData))
		if strings.EqualFold(dType, typeMatch) {
			curData, _ := os.ReadFile(filepath.Join(d, "cur_state"))
			maxData, _ := os.ReadFile(filepath.Join(d, "max_state"))
			cVal, _ := strconv.Atoi(strings.TrimSpace(string(curData)))
			mVal, _ := strconv.Atoi(strings.TrimSpace(string(maxData)))
			return cVal, mVal
		}
	}
	return 0, 0
}

func readSecThermistor(path string) float64 {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0.0
	}
	raw, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
	if err != nil || raw <= 0 {
		return 0.0
	}
	return raw / 10.0
}

func readBatteryTemp() float64 {
	// 1. Try standard sysfs
	if data, err := os.ReadFile("/sys/class/power_supply/battery/temp"); err == nil {
		if raw, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64); err == nil && raw > 0 {
			if raw > 100.0 {
				return raw / 10.0
			}
			return raw
		}
	}

	// 2. Fallback via getprop
	out, err := config.ExecSuTimeout(1*time.Second, "getprop sys.battery.temp")
	if err == nil {
		if raw, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64); err == nil && raw > 0 {
			if raw > 100.0 {
				return raw / 10.0
			}
			return raw
		}
	}
	return 0.0
}

func getThermalHALStatus() (int, string) {
	out, err := config.ExecSuTimeout(2*time.Second, "dumpsys thermalservice")
	if err != nil {
		return 0, "NONE (Normal)"
	}

	re := regexp.MustCompile(`Thermal Status:\s*(\d+)`)
	matches := re.FindStringSubmatch(string(out))
	if len(matches) > 1 {
		status, err := strconv.Atoi(matches[1])
		if err == nil {
			return status, getThermalHALDesc(status)
		}
	}

	return 0, "NONE (Normal)"
}

func getThermalHALDesc(status int) string {
	switch status {
	case 0:
		return "NONE (Normal)"
	case 1:
		return "LIGHT (Hangat Ringan)"
	case 2:
		return "MODERATE (Siaga Panas)"
	case 3:
		return "SEVERE (Throttling 5G Aktif!)"
	case 4:
		return "CRITICAL (Bahaya)"
	default:
		if status > 4 {
			return "EMERGENCY"
		}
		return fmt.Sprintf("Level %d", status)
	}
}

func getCellularDCNrStatus() (netType string, isDisconnect bool) {
	out, err := config.ExecSuTimeout(2*time.Second, "dumpsys telephony.registry")
	if err != nil {
		return "LTE", false
	}

	outStr := string(out)
	if strings.Contains(strings.ToLower(outStr), "isdcnrrestricted = true") {
		return "5G Disconnect", true
	}
	if strings.Contains(outStr, "nrState=CONNECTED") {
		return "5G-NSA", false
	}
	return "LTE", false
}

func evaluateCulprit(status *Thermal5GStatus) {
	status.Culprit = "TIDAK ADA (SUHU NORMAL)"
	status.CulpritDetail = "Semua sensor dalam batas aman. 5G tidak mengalami pembatasan suhu."
	status.IsThrottled = false

	if status.ThermalHALStatus >= 3 {
		status.IsThrottled = true
		status.Culprit = "Modem Skin Board (SKIN) / sec-cf"
		status.CulpritDetail = fmt.Sprintf("Suhu permukaan board/skin mencapai %.1f°C (limit: 42.0°C). Android Thermal HAL masuk status SEVERE (Level %d). Sistem memerintahkan pelepasan 5G NR Dual Connectivity!", status.Sensors.ModemSkinC, status.ThermalHALStatus)
	} else if status.Throttlers.ModemSkinNrDsc > 0 {
		status.IsThrottled = true
		status.Culprit = fmt.Sprintf("Modem Skin NR Controller (modem_skin_nr_dsc = %d)", status.Throttlers.ModemSkinNrDsc)
		status.CulpritDetail = fmt.Sprintf("Qualcomm Thermal Engine membatasi 5G NR karena suhu skin modem (%.1f°C) melebihi ambang batas.", status.Sensors.ModemSkinC)
	} else if status.Throttlers.ModemPaFr1Dsc > 0 {
		status.IsThrottled = true
		status.Culprit = fmt.Sprintf("5G Sub-6 Power Amplifier (modem_pa_fr1_dsc = %d)", status.Throttlers.ModemPaFr1Dsc)
		status.CulpritDetail = fmt.Sprintf("Chip Power Amplifier 5G (%.1f°C) melewati limit proteksi RF. Modem memotong daya transmisi 5G NR!", status.Sensors.PA1Sub6C)
	} else if status.Sensors.PA1Sub6C >= 48.0 {
		status.IsThrottled = true
		status.Culprit = fmt.Sprintf("5G Sub-6 Power Amplifier 1 (PA1 = %.1f°C)", status.Sensors.PA1Sub6C)
		status.CulpritDetail = fmt.Sprintf("Suhu PA1 sangat panas (%.1f°C). Transceiver 5G bersiap memutus SCG 5G untuk perlindungan hardware.", status.Sensors.PA1Sub6C)
	} else if status.Sensors.ModemSkinC >= 43.0 {
		status.IsThrottled = true
		status.Culprit = fmt.Sprintf("Modem Skin Board (SKIN = %.1f°C)", status.Sensors.ModemSkinC)
		status.CulpritDetail = fmt.Sprintf("Sensor bodi modem (%.1f°C) melewati batas 42.0°C. Ini adalah sensor utama pemicu 5G Disconnect di Samsung.", status.Sensors.ModemSkinC)
	} else if status.Sensors.CellFrontCFC >= 45.0 {
		status.IsThrottled = true
		status.Culprit = fmt.Sprintf("Cellular Front-End Thermistor (sec-cf = %.1f°C)", status.Sensors.CellFrontCFC)
		status.CulpritDetail = fmt.Sprintf("Sensor RF depan (%.1f°C) terlalu panas akibat traffic bandwidth besar.", status.Sensors.CellFrontCFC)
	}
}

// Get5GThermalStatus reads all thermal zones, cooling devices, and HAL states natively in Go.
func Get5GThermalStatus() (*Thermal5GStatus, error) {
	sensors := ThermalSensors{
		PA1Sub6C: readFirstMatchingThermalZoneTemp([]string{
			"modem-lte-sub6-pa1", "pa-therm-0", "pa1-therm", "pa_therm0", "pa0-therm",
		}),
		PA2Sub6C: readFirstMatchingThermalZoneTemp([]string{
			"modem-lte-sub6-pa2", "pa-therm-1", "pa-therm-2", "pa2-therm", "pa_therm1",
		}),
		CellFrontCFC: readSecThermistor("/sys/class/sec/sec-cf-thermistor/temperature"),
		ModemSkinC: readFirstMatchingThermalZoneTemp([]string{
			"modem-skin-usr", "skin-therm", "modem_skin", "skin_therm", "skin-msm-therm",
		}),
		RfPmicC: readFirstMatchingThermalZoneTemp([]string{
			"pmr735a_tz", "pmr735a-tz", "pmic_tz", "pm8350b_tz",
		}),
		ModemBasebandC: readFirstMatchingThermalZoneTemp([]string{
			"mdmss-0-usr", "mdmss-1-usr", "mdm-core-usr", "mdm-dsp-usr", "modem-usr",
		}),
		ApSocC:   readSecThermistor("/sys/class/sec/sec-ap-thermistor/temperature"),
		BatteryC: readBatteryTemp(),
	}

	paFr1Dsc, _ := readCoolingDeviceState("modem_pa_fr1_dsc")
	skinNrDsc, _ := readCoolingDeviceState("modem_skin_nr_dsc")
	paFr1, _ := readCoolingDeviceState("modem_pa_fr1")

	throttlers := ThermalThrottlers{
		ModemPaFr1Dsc:  paFr1Dsc,
		ModemSkinNrDsc: skinNrDsc,
		ModemPaFr1:     paFr1,
	}

	halStatus, halDesc := getThermalHALStatus()
	netType, isDisconnect := getCellularDCNrStatus()

	status := &Thermal5GStatus{
		Available:        true,
		NetworkType:      netType,
		Is5GDisconnect:   isDisconnect,
		NRBand:           "",
		NRRSRP:           0,
		TrafficMbps:      0.0,
		ThermalHALStatus: halStatus,
		ThermalHALDesc:   halDesc,
		Sensors:          sensors,
		Throttlers:       throttlers,
		CriticalThresholds: CriticalThresholds{
			ModemSkinLimitC: 42.0,
			PA1CutoffLimitC: 48.0,
		},
	}

	evaluateCulprit(status)
	return status, nil
}
