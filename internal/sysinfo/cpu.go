package sysinfo

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

type CPUCoreStat struct {
	Core    int     `json:"core"`
	FreqMHz float64 `json:"freq_mhz"`
	Usage   float64 `json:"usage"`
}


type cpuStat struct {
	idle  uint64
	total uint64
}


var (
	lastCPU      cpuStat
	lastCoreCPUs map[int]cpuStat
	cpuMux       sync.Mutex
)
func readCPUCoreFreqs(coreIndices []int) map[int]float64 {
	freqs := make(map[int]float64, len(coreIndices))
	for _, idx := range coreIndices {
		freqs[idx] = getCoreFreq(idx)
	}
	return freqs
}

type coreUsageTemp struct {
	core  int
	usage float64
}

func getCPUStats() (float64, []CPUCoreStat) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, nil
	}
	defer file.Close()

	var overallUsage float64
	var coreUsages []coreUsageTemp
	var coreIndices []int

	cpuMux.Lock()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		name := fields[0]
		if name == "cpu" {
			overallUsage = calcCPUUsage(&lastCPU, fields[1:])
		} else if strings.HasPrefix(name, "cpu") {
			coreIdx, err := strconv.Atoi(strings.TrimPrefix(name, "cpu"))
			if err == nil {
				lastStat, ok := lastCoreCPUs[coreIdx]
				if !ok {
					lastStat = cpuStat{}
				}
				usage := calcCPUUsage(&lastStat, fields[1:])
				lastCoreCPUs[coreIdx] = lastStat

				coreIndices = append(coreIndices, coreIdx)
				coreUsages = append(coreUsages, coreUsageTemp{core: coreIdx, usage: usage})
			}
		}
	}
	cpuMux.Unlock()

	// Read sysfs frequencies into a local map outside cpuMux lock to avoid I/O contention
	freqs := readCPUCoreFreqs(coreIndices)

	var cores []CPUCoreStat
	for _, cu := range coreUsages {
		cores = append(cores, CPUCoreStat{
			Core:    cu.core,
			FreqMHz: freqs[cu.core],
			Usage:   cu.usage,
		})
	}

	return overallUsage, cores
}

func calcCPUUsage(prev *cpuStat, fields []string) float64 {
	var numFields []uint64
	for _, f := range fields {
		val, _ := strconv.ParseUint(f, 10, 64)
		numFields = append(numFields, val)
	}
	if len(numFields) < 4 {
		return 0
	}

	var total uint64
	for _, val := range numFields {
		total += val
	}
	idle := numFields[3]
	if len(numFields) >= 5 {
		idle += numFields[4] // iowait
	}

	if prev.total == 0 || total < prev.total || idle < prev.idle {
		*prev = cpuStat{idle: idle, total: total}
		return 0
	}

	totalDelta := float64(total - prev.total)
	idleDelta := float64(idle - prev.idle)
	*prev = cpuStat{idle: idle, total: total}

	if totalDelta == 0 {
		return 0
	}

	usage := ((totalDelta - idleDelta) / totalDelta) * 100
	if usage < 0 {
		return 0
	}
	if usage > 100 {
		return 100
	}
	return usage
}

func getCoreFreq(core int) float64 {
	path := fmt.Sprintf("/sys/devices/system/cpu/cpu%d/cpufreq/scaling_cur_freq", core)
	data, err := os.ReadFile(path)
	if err != nil {
		path = fmt.Sprintf("/sys/devices/system/cpu/cpu%d/cpufreq/cpuinfo_cur_freq", core)
		data, err = os.ReadFile(path)
	}
	if err == nil {
		khz, _ := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
		return khz / 1000.0 // MHz
	}
	return 0
}


func getGovernor() string {
	paths := []string{
		"/sys/devices/system/cpu/cpufreq/policy0/scaling_governor",
		"/sys/devices/system/cpu/cpu0/cpufreq/scaling_governor",
	}
	for _, p := range paths {
		if data, err := os.ReadFile(p); err == nil {
			val := strings.TrimSpace(string(data))
			if val != "" {
				return val
			}
		}
	}
	return "Unknown"
}


