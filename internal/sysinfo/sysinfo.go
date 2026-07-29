package sysinfo

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type CPUCoreStat struct {
	Core    int     `json:"core"`
	FreqMHz float64 `json:"freq_mhz"`
	Usage   float64 `json:"usage"`
}

type ThermalZone struct {
	Name string  `json:"name"`
	Temp float64 `json:"temp"`
}

type DiskPartition struct {
	Path    string  `json:"path"`
	Total   uint64  `json:"total"`
	Free    uint64  `json:"free"`
	Used    uint64  `json:"used"`
	UsedPct float64 `json:"used_pct"`
}

type DetailedBattery struct {
	Capacity   int     `json:"capacity"`
	Status     string  `json:"status"`
	Temp       float64 `json:"temp"`
	VoltagemV  float64 `json:"voltage_mv"`
	Health     string  `json:"health"`
	Technology string  `json:"technology"`
}

type LoadAverage struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

type ServiceStatus struct {
	Name    string `json:"name"`
	Key     string `json:"key"`
	Running bool   `json:"running"`
	Detail  string `json:"detail"`
}

type Stats struct {
	CPUUsage       float64         `json:"cpu_usage"`
	CPUCores       []CPUCoreStat   `json:"cpu_cores"`
	MemTotal       uint64          `json:"mem_total"`
	MemFree        uint64          `json:"mem_free"`
	MemAvailable   uint64          `json:"mem_available"`
	MemUsed        uint64          `json:"mem_used"`
	MemUsedPct     float64         `json:"mem_used_pct"`
	SwapTotal      uint64          `json:"swap_total"`
	SwapFree       uint64          `json:"swap_free"`
	SwapUsed       uint64          `json:"swap_used"`
	SwapUsedPct    float64         `json:"swap_used_pct"`
	LoadAvg        LoadAverage     `json:"load_avg"`
	ActiveServices []ServiceStatus `json:"active_services"`
	Uptime         float64         `json:"uptime"`
	BatteryLevel   int             `json:"battery_level"`
	BatteryStatus  string          `json:"battery_status"`
	BatteryTemp    float64         `json:"battery_temp"`
	BatteryDetail  DetailedBattery `json:"battery_detail"`
	Thermals       []ThermalZone   `json:"thermals"`
	Disks          []DiskPartition `json:"disks"`
	Model          string          `json:"model"`
	AndroidVer     string          `json:"android_ver"`
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

func init() {
	lastCoreCPUs = make(map[int]cpuStat)
	getCPUStats()
	time.Sleep(100 * time.Millisecond)
}

func GetStats() (Stats, error) {
	var s Stats

	overallUsage, cores := getCPUStats()
	s.CPUUsage = overallUsage
	s.CPUCores = cores

	s.MemTotal, s.MemFree, s.MemAvailable, s.MemUsed, s.MemUsedPct, s.SwapTotal, s.SwapFree, s.SwapUsed, s.SwapUsedPct = getMemInfo()
	s.Uptime = getUptime()

	s.BatteryDetail = getDetailedBattery()
	s.BatteryLevel = s.BatteryDetail.Capacity
	s.BatteryStatus = s.BatteryDetail.Status
	s.BatteryTemp = s.BatteryDetail.Temp

	s.Thermals = getThermalZones()
	s.Disks = getDiskPartitions()

	s.LoadAvg = getLoadAvg()
	s.ActiveServices = getActiveServices()

	s.Model = getProp("ro.product.model")
	if s.Model == "" {
		s.Model = getProp("ro.product.brand") + " " + getProp("ro.product.device")
	}
	s.AndroidVer = getProp("ro.build.version.release")

	return s, nil
}

func getCPUStats() (float64, []CPUCoreStat) {
	cpuMux.Lock()
	defer cpuMux.Unlock()

	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, nil
	}
	defer file.Close()

	var overallUsage float64
	var cores []CPUCoreStat

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

				freq := getCoreFreq(coreIdx)
				cores = append(cores, CPUCoreStat{
					Core:    coreIdx,
					FreqMHz: freq,
					Usage:   usage,
				})
			}
		}
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

	if prev.total == 0 {
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

func getUptime() float64 {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) > 0 {
		up, _ := strconv.ParseFloat(fields[0], 64)
		return up
	}
	return 0
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

	for _, z := range files {
		typeData, err := os.ReadFile(filepath.Join(z, "type"))
		if err != nil {
			continue
		}
		name := strings.TrimSpace(string(typeData))

		tempData, err := os.ReadFile(filepath.Join(z, "temp"))
		if err != nil {
			continue
		}
		t, _ := strconv.ParseFloat(strings.TrimSpace(string(tempData)), 64)
		if t > 1000 {
			t = t / 1000.0
		}

		if t > -20 && t < 150 { // Reasonable sanity range
			zones = append(zones, ThermalZone{
				Name: name,
				Temp: t,
			})
		}
	}
	return zones
}

func getDiskPartitions() []DiskPartition {
	var disks []DiskPartition
	targets := []string{"/data", "/system", "/sdcard"}

	for _, target := range targets {
		out, err := exec.Command("df", "-k", target).Output()
		if err != nil {
			continue
		}
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) >= 2 {
			fields := strings.Fields(lines[len(lines)-1])
			if len(fields) >= 5 {
				totalKb, _ := strconv.ParseUint(fields[1], 10, 64)
				usedKb, _ := strconv.ParseUint(fields[2], 10, 64)
				freeKb, _ := strconv.ParseUint(fields[3], 10, 64)

				total := totalKb * 1024
				used := usedKb * 1024
				free := freeKb * 1024

				var pct float64
				if total > 0 {
					pct = (float64(used) / float64(total)) * 100
				}

				disks = append(disks, DiskPartition{
					Path:    target,
					Total:   total,
					Free:    free,
					Used:    used,
					UsedPct: pct,
				})
			}
		}
	}
	return disks
}

func RunPing(host string, count int) (string, error) {
	if count <= 0 || count > 10 {
		count = 4
	}
	cmd := exec.Command("ping", "-c", strconv.Itoa(count), host)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func RunSpeedtest() (map[string]interface{}, error) {
	// Simple HTTP benchmark download check
	start := time.Now()
	resp, err := exec.Command("curl", "-s", "-w", "%{speed_download}", "-o", "/dev/null", "https://speed.cloudflare.com/__down?bytes=25000000").Output()
	duration := time.Since(start).Seconds()

	result := make(map[string]interface{})
	if err == nil {
		bytesPerSec, _ := strconv.ParseFloat(strings.TrimSpace(string(resp)), 64)
		mbps := (bytesPerSec * 8) / 1000000.0
		result["download_mbps"] = mbps
		result["duration_sec"] = duration
		return result, nil
	}
	return nil, err
}

func getProp(prop string) string {
	out, err := exec.Command("getprop", prop).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func getLoadAvg() LoadAverage {
	var l LoadAverage
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return l
	}
	fields := strings.Fields(string(data))
	if len(fields) >= 3 {
		l.Load1, _ = strconv.ParseFloat(fields[0], 64)
		l.Load5, _ = strconv.ParseFloat(fields[1], 64)
		l.Load15, _ = strconv.ParseFloat(fields[2], 64)
	}
	return l
}

func checkService(name, key string, port int, processNames ...string) ServiceStatus {
	running := false
	if port > 0 {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 50*time.Millisecond)
		if err == nil {
			conn.Close()
			running = true
		}
	}
	if !running && len(processNames) > 0 {
		for _, proc := range processNames {
			out, err := exec.Command("pidof", proc).Output()
			if err == nil && len(strings.TrimSpace(string(out))) > 0 {
				running = true
				break
			}
		}
	}
	detail := "Off"
	if running {
		if port > 0 {
			detail = fmt.Sprintf("Running (Port %d)", port)
		} else {
			detail = "Running"
		}
	}
	return ServiceStatus{
		Name:    name,
		Key:     key,
		Running: running,
		Detail:  detail,
	}
}

func getActiveServices() []ServiceStatus {
	return []ServiceStatus{
		checkService("Clash Core", "clash", 9090, "mihomo", "clash"),
		checkService("SSH Daemon", "ssh", 22, "sshd", "dropbear"),
		checkService("ADB Wireless", "adb", 5555, "adbd"),
		checkService("Web UI Server", "webui", 8080),
	}
}
