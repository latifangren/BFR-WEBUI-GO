package sysinfo

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

type GovernorInfo struct {
	Available []string          `json:"available"`
	Current   string            `json:"current"`
	Cores     []CPUGovernorCore `json:"cores"`
}

type CPUGovernorCore struct {
	Core     int    `json:"core"`
	Governor string `json:"governor"`
	CurFreq  string `json:"cur_freq"`
}

var reGovernor = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func GetGovernorInfo() (GovernorInfo, error) {
	var info GovernorInfo
	info.Available = []string{}
	info.Cores = []CPUGovernorCore{}

	availData, err := os.ReadFile("/sys/devices/system/cpu/cpu0/cpufreq/scaling_available_governors")
	if err == nil {
		fields := strings.Fields(strings.TrimSpace(string(availData)))
		info.Available = fields
	}

	currData, err := os.ReadFile("/sys/devices/system/cpu/cpu0/cpufreq/scaling_governor")
	if err == nil {
		info.Current = strings.TrimSpace(string(currData))
	} else {
		info.Current = "Unknown"
	}

	coreDirs, _ := filepath.Glob("/sys/devices/system/cpu/cpu[0-9]*")
	for _, cDir := range coreDirs {
		base := filepath.Base(cDir)
		coreIdx, err := strconv.Atoi(strings.TrimPrefix(base, "cpu"))
		if err != nil {
			continue
		}

		gov := "N/A"
		if gData, err := os.ReadFile(filepath.Join(cDir, "cpufreq", "scaling_governor")); err == nil {
			gov = strings.TrimSpace(string(gData))
		}

		freq := "N/A"
		if fData, err := os.ReadFile(filepath.Join(cDir, "cpufreq", "scaling_cur_freq")); err == nil {
			if khz, err := strconv.ParseFloat(strings.TrimSpace(string(fData)), 64); err == nil {
				freq = fmt.Sprintf("%.0f MHz", khz/1000.0)
			}
		}

		info.Cores = append(info.Cores, CPUGovernorCore{
			Core:     coreIdx,
			Governor: gov,
			CurFreq:  freq,
		})
	}

	return info, nil
}

func SetGovernor(gov string) error {
	if !reGovernor.MatchString(gov) {
		return fmt.Errorf("invalid governor name: %s", gov)
	}

	cmdStr := fmt.Sprintf("for f in /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor; do echo %s > $f; done", gov)
	out, err := config.ExecSuTimeout(5*time.Second, cmdStr)
	if err != nil {
		return fmt.Errorf("set governor failed: %v, output: %s", err, string(out))
	}

	return nil
}
