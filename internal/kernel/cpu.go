package kernel

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"bfr-webui-go/internal/config"
)

// ReadCPUClusters detects and reads all CPU policies and cluster properties.
func ReadCPUClusters() ([]CPUCluster, error) {
	policyDirs, _ := filepath.Glob("/sys/devices/system/cpu/cpufreq/policy*")
	if len(policyDirs) == 0 {
		return readLegacyClusters()
	}

	// Sort policy directories by policy ID
	sort.Slice(policyDirs, func(i, j int) bool {
		idA, _ := strconv.Atoi(strings.TrimPrefix(filepath.Base(policyDirs[i]), "policy"))
		idB, _ := strconv.Atoi(strings.TrimPrefix(filepath.Base(policyDirs[j]), "policy"))
		return idA < idB
	})

	totalPolicies := len(policyDirs)
	clusters := make([]CPUCluster, 0, totalPolicies)

	for idx, pDir := range policyDirs {
		pID, _ := strconv.Atoi(strings.TrimPrefix(filepath.Base(pDir), "policy"))

		cores := readSysfsTrim(filepath.Join(pDir, "affected_cpus"))
		if cores == "" {
			cores = fmt.Sprintf("cpu%d", pID)
		}

		gov := readSysfsTrim(filepath.Join(pDir, "scaling_governor"))
		availGovsStr := readSysfsTrim(filepath.Join(pDir, "scaling_available_governors"))
		availGovs := strings.Fields(availGovsStr)

		minFreq, _ := strconv.ParseInt(readSysfsTrim(filepath.Join(pDir, "scaling_min_freq")), 10, 64)
		maxFreq, _ := strconv.ParseInt(readSysfsTrim(filepath.Join(pDir, "scaling_max_freq")), 10, 64)
		curFreq, _ := strconv.ParseInt(readSysfsTrim(filepath.Join(pDir, "scaling_cur_freq")), 10, 64)

		availFreqsStr := readSysfsTrim(filepath.Join(pDir, "scaling_available_frequencies"))
		availFreqs := parseFrequencies(availFreqsStr)

		label := formatClusterLabel(idx, totalPolicies)

		clusters = append(clusters, CPUCluster{
			PolicyID:   pID,
			Cores:      cores,
			Label:      label,
			Governor:   gov,
			AvailGovs:  availGovs,
			MinFreq:    minFreq,
			MaxFreq:    maxFreq,
			CurFreq:    curFreq,
			AvailFreqs: availFreqs,
		})
	}

	return clusters, nil
}

func readLegacyClusters() ([]CPUCluster, error) {
	// Fallback when /cpufreq/policy* is not present: check cpu0
	pDir := "/sys/devices/system/cpu/cpu0/cpufreq"
	if _, err := os.Stat(pDir); os.IsNotExist(err) {
		return []CPUCluster{}, nil
	}

	gov := readSysfsTrim(filepath.Join(pDir, "scaling_governor"))
	availGovsStr := readSysfsTrim(filepath.Join(pDir, "scaling_available_governors"))
	minFreq, _ := strconv.ParseInt(readSysfsTrim(filepath.Join(pDir, "scaling_min_freq")), 10, 64)
	maxFreq, _ := strconv.ParseInt(readSysfsTrim(filepath.Join(pDir, "scaling_max_freq")), 10, 64)
	curFreq, _ := strconv.ParseInt(readSysfsTrim(filepath.Join(pDir, "scaling_cur_freq")), 10, 64)
	availFreqs := parseFrequencies(readSysfsTrim(filepath.Join(pDir, "scaling_available_frequencies")))

	return []CPUCluster{
		{
			PolicyID:   0,
			Cores:      "all",
			Label:      "Standard CPU Cluster",
			Governor:   gov,
			AvailGovs:  strings.Fields(availGovsStr),
			MinFreq:    minFreq,
			MaxFreq:    maxFreq,
			CurFreq:    curFreq,
			AvailFreqs: availFreqs,
		},
	}, nil
}

func formatClusterLabel(idx, total int) string {
	switch total {
	case 1:
		return "All Cores"
	case 2:
		if idx == 0 {
			return "Little (Efficiency)"
		}
		return "Big (Performance)"
	case 3:
		if idx == 0 {
			return "Little (Efficiency)"
		} else if idx == 1 {
			return "Big (Performance)"
		}
		return "Prime (Gold/Max)"
	default:
		return fmt.Sprintf("Cluster %d", idx+1)
	}
}

func parseFrequencies(str string) []int64 {
	if str == "" {
		return []int64{}
	}
	fields := strings.Fields(str)
	freqs := make([]int64, 0, len(fields))
	for _, f := range fields {
		if val, err := strconv.ParseInt(f, 10, 64); err == nil {
			freqs = append(freqs, val)
		}
	}
	sort.Slice(freqs, func(i, j int) bool { return freqs[i] < freqs[j] })
	return freqs
}

func readSysfsTrim(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// ApplyClusterSettings applies governor and frequency bounds for a policy.
func ApplyClusterSettings(policyID int, governor string, minFreq, maxFreq int64) error {
	pDir := fmt.Sprintf("/sys/devices/system/cpu/cpufreq/policy%d", policyID)

	var cmds []string

	// Check if directory exists
	if _, err := os.Stat(pDir); err == nil {
		if governor != "" {
			cmds = append(cmds, fmt.Sprintf("echo %s > %s/scaling_governor", governor, pDir))
		}
		if minFreq > 0 {
			cmds = append(cmds, fmt.Sprintf("echo %d > %s/scaling_min_freq", minFreq, pDir))
		}
		if maxFreq > 0 {
			cmds = append(cmds, fmt.Sprintf("echo %d > %s/scaling_max_freq", maxFreq, pDir))
		}
	} else {
		// Fallback to all cpus
		if governor != "" {
			cmds = append(cmds, fmt.Sprintf("for f in /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor; do echo %s > $f; done", governor))
		}
		if minFreq > 0 {
			cmds = append(cmds, fmt.Sprintf("for f in /sys/devices/system/cpu/cpu*/cpufreq/scaling_min_freq; do echo %d > $f; done", minFreq))
		}
		if maxFreq > 0 {
			cmds = append(cmds, fmt.Sprintf("for f in /sys/devices/system/cpu/cpu*/cpufreq/scaling_max_freq; do echo %d > $f; done", maxFreq))
		}
	}

	if len(cmds) == 0 {
		return nil
	}

	fullCmd := strings.Join(cmds, " ; ")
	_, err := config.ExecSuTimeout(5*time.Second, fullCmd)
	return err
}

// ApplyGlobalGovernor applies a governor across all CPU clusters and cores.
func ApplyGlobalGovernor(governor string) error {
	if governor == "" {
		return nil
	}
	cmd := fmt.Sprintf("for f in /sys/devices/system/cpu/cpufreq/policy*/scaling_governor /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor; do [ -f $f ] && echo %s > $f; done", governor)
	_, err := config.ExecSuTimeout(5*time.Second, cmd)
	return err
}
