package sysinfo

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

func getMemInfo() (total, free, avail, used uint64, usedPct float64, swapTotal, swapFree, swapUsed uint64, swapUsedPct float64) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	memMap := make(map[string]uint64)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			valStr := strings.TrimSpace(parts[1])
			valFields := strings.Fields(valStr)
			if len(valFields) > 0 {
				val, _ := strconv.ParseUint(valFields[0], 10, 64)
				memMap[key] = val * 1024
			}
		}
	}

	total = memMap["MemTotal"]
	free = memMap["MemFree"]
	avail = memMap["MemAvailable"]
	if avail == 0 {
		avail = free + memMap["Buffers"] + memMap["Cached"]
	}
	if total > avail {
		used = total - avail
	}
	if total > 0 {
		usedPct = (float64(used) / float64(total)) * 100
	}

	swapTotal = memMap["SwapTotal"]
	swapFree = memMap["SwapFree"]
	if swapTotal > swapFree {
		swapUsed = swapTotal - swapFree
	}
	if swapTotal > 0 {
		swapUsedPct = (float64(swapUsed) / float64(swapTotal)) * 100
	}
	return
}


func findPID(name string) (int, bool) {
	if out, err := exec.Command("/system/bin/pidof", "-s", name).Output(); err == nil {
		pStr := strings.TrimSpace(string(out))
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			return p, true
		}
	}
	if out, err := exec.Command("/system/bin/pgrep", "-f", "-o", name).Output(); err == nil {
		pStr := strings.TrimSpace(string(out))
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			return p, true
		}
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0, false
	}

	nameLower := strings.ToLower(name)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 {
			continue
		}

		commPath := fmt.Sprintf("/proc/%d/comm", pid)
		if commBytes, err := os.ReadFile(commPath); err == nil {
			commStr := strings.TrimSpace(string(commBytes))
			if strings.EqualFold(commStr, name) || strings.Contains(strings.ToLower(commStr), nameLower) {
				return pid, true
			}
		}
	}

	return 0, false
}

func getProcessMemoryMB(pid int) float64 {
	if pid <= 0 {
		return 0
	}
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/statm", pid))
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 0
	}
	residentPages, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	pageSize := uint64(os.Getpagesize())
	if pageSize == 0 {
		pageSize = 4096
	}
	ramMB := float64(residentPages*pageSize) / (1024 * 1024)
	if ramMB > 0 && ramMB < 0.1 {
		return math.Round(ramMB*100) / 100
	}
	return math.Round(ramMB*10) / 10
}

type pidCPUSample struct {
	totalTicks uint64
	timestamp  time.Time
}

var (
	pidCPUMu    sync.RWMutex
	pidCPUCache = make(map[int]pidCPUSample)
)

func getProcessCPU(pid int) float64 {
	if pid <= 0 {
		return 0
	}
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0
	}
	content := string(data)
	lastParen := strings.LastIndex(content, ")")
	if lastParen == -1 || lastParen+2 >= len(content) {
		return 0
	}
	rest := strings.Fields(content[lastParen+2:])
	if len(rest) <= 19 {
		return 0
	}

	utime, err1 := strconv.ParseUint(rest[11], 10, 64)
	stime, err2 := strconv.ParseUint(rest[12], 10, 64)
	if err1 != nil || err2 != nil {
		return 0
	}

	totalTicks := utime + stime
	hertz := 100.0
	now := time.Now()

	numCores := float64(runtime.NumCPU())
	if numCores <= 0 {
		numCores = 1
	}

	pidCPUMu.Lock()
	prev, hasPrev := pidCPUCache[pid]
	pidCPUCache[pid] = pidCPUSample{totalTicks: totalTicks, timestamp: now}
	pidCPUMu.Unlock()

	var cpu float64
	if hasPrev && now.Sub(prev.timestamp).Seconds() >= 0.4 && totalTicks >= prev.totalTicks {
		deltaTicks := totalTicks - prev.totalTicks
		deltaSec := now.Sub(prev.timestamp).Seconds()
		if deltaSec > 0 {
			rawCPU := (float64(deltaTicks) / hertz) / deltaSec * 100.0
			cpu = rawCPU / numCores
		}
	} else if !hasPrev {
		cpu = 0.0
	}

	if cpu < 0 {
		cpu = 0
	} else if cpu > 100 {
		cpu = 100
	}
	return math.Round(cpu*10) / 10
}


