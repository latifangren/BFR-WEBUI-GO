package sysinfo

import (
	"bufio"
	"context"
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
	CPUTemp        float64         `json:"cpu_temp"`
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
	SELinux        string          `json:"selinux"`
	SecurityPatch  string          `json:"security_patch"`
	SDKVer         string          `json:"sdk_ver"`
	Resolution     string          `json:"resolution"`
	Density        string          `json:"density"`
	MTU            string          `json:"mtu"`
	DefaultTTL     string          `json:"default_ttl"`
	Kernel         string          `json:"kernel"`
	ServerTime     string          `json:"server_time"`
	LocalTime      string          `json:"local_time"`
	Hostname       string          `json:"hostname"`
	NetRx          uint64          `json:"net_rx"`
	NetTx          uint64          `json:"net_tx"`
	NetworkDetail  NetworkDetail   `json:"network_detail"`
}

type NetworkDetail struct {
	IPAddresses    []string  `json:"ip_addresses"`
	Gateway        string    `json:"gateway"`
	DNS            []string  `json:"dns"`
	DNS1           string    `json:"dns1"`
	DNS2           string    `json:"dns2"`
	WiFiSSID       string    `json:"wifi_ssid"`
	WiFiSignal     string    `json:"wifi_signal"`
	WiFiSignalDBM  string    `json:"wifi_signal_dbm"`
	WiFiRSSI       int       `json:"wifi_rssi"`
	WiFiFullInfo   string    `json:"wifi_full_info"`
	MCCMNC         []string  `json:"mcc_mnc"`
	Roaming        string    `json:"roaming"`
	HotspotClients int       `json:"hotspot_clients"`
	SIMSlots       []SIMSlot `json:"sim_slots"`
}

type SIMSlot struct {
	Slot          int     `json:"slot"`
	Operator      string  `json:"operator"`
	NetworkType   string  `json:"network_type"`
	RSRP          string  `json:"rsrp"`
	RSRQ          string  `json:"rsrq"`
	SINR          string  `json:"sinr"`
	RSRPInt       int     `json:"rsrp_int"`
	RSRQInt       int     `json:"rsrq_int"`
	SINRInt       int     `json:"sinr_int"`
	SignalStatus  string  `json:"signal_status"`
	SignalQuality string  `json:"signal_quality"`
	SignalScore   float64 `json:"signal_score"`
}

type cpuStat struct {
	idle  uint64
	total uint64
}

var (
	lastCPU      cpuStat
	lastCoreCPUs map[int]cpuStat
	cpuMux       sync.Mutex

	networkDetailCache   NetworkDetail
	networkDetailCacheAt time.Time
	networkDetailCacheMu sync.Mutex
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
	// Calculate average CPU thermal temperature
	var cpuThermalSum float64
	var cpuThermalCount int
	for _, tz := range s.Thermals {
		// Only average genuine CPU/SoC zones
		lowerName := strings.ToLower(tz.Name)
		if strings.Contains(lowerName, "cpu") || strings.Contains(lowerName, "soc") || strings.Contains(lowerName, "ap-therm") || strings.Contains(lowerName, "ap_therm") {
			cpuThermalSum += tz.Temp
			cpuThermalCount++
		}
	}
	if cpuThermalCount > 0 {
		s.CPUTemp = cpuThermalSum / float64(cpuThermalCount)
	} else if len(s.Thermals) > 0 {
		// Fallback to first zone
		s.CPUTemp = s.Thermals[0].Temp
	} else {
		// Fallback to battery temperature if no CPU thermals found
		s.CPUTemp = s.BatteryTemp
	}

	s.Disks = getDiskPartitions()

	s.LoadAvg = getLoadAvg()
	s.ActiveServices = getActiveServices()

	s.Model = getProp("ro.product.model")
	if s.Model == "" {
		s.Model = getProp("ro.product.brand") + " " + getProp("ro.product.device")
	}
	s.AndroidVer = getProp("ro.build.version.release")
	s.SELinux = getSELinux()
	s.SecurityPatch = getProp("ro.build.version.security_patch")
	s.SDKVer = getProp("ro.build.version.sdk")
	s.Resolution = getScreenResolution()
	s.Density = getScreenDensity()
	s.MTU = getMTU()
	s.DefaultTTL = getDefaultTTL()
	s.Kernel = getKernelVersion()
	s.ServerTime = getServerTime()
	s.LocalTime = getAndroidLocalTime()
	s.Hostname = getHostname()
	s.NetRx, s.NetTx = getNetDevBytes()
	s.NetworkDetail = getNetworkDetail()

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

		// Calculate valid scaled temperatures within 15°C to 90°C
		if t >= 15 && t <= 90 {
			zones = append(zones, ThermalZone{
				Name: typeName,
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
		out := runCmdTimeout(2*time.Second, "df", "-k", target)
		if out == "" {
			continue
		}
		lines := strings.Split(strings.TrimSpace(out), "\n")
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
	return runCmdTimeout(3*time.Second, "getprop", prop)
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

func getSELinux() string {
	data, err := os.ReadFile("/sys/fs/selinux/enforce")
	if err == nil {
		if strings.TrimSpace(string(data)) == "1" {
			return "Enforcing"
		}
		return "Permissive"
	}
	out := runCmdTimeout(2*time.Second, "getenforce")
	if out != "" {
		return out
	}
	return "Disabled / Unknown"
}

func getHostname() string {
	h, err := os.Hostname()
	if err == nil {
		return h
	}
	out := runCmdTimeout(2*time.Second, "hostname")
	if out != "" {
		return out
	}
	return "Android"
}

func getServerTime() string {
	return time.Now().Format("2006-01-02 15:04:05 MST")
}

func getKernelVersion() string {
	data, err := os.ReadFile("/proc/version")
	if err == nil {
		parts := strings.Split(string(data), " ")
		if len(parts) > 2 {
			return strings.Join(parts[:3], " ")
		}
		return strings.TrimSpace(string(data))
	}
	out := runCmdTimeout(2*time.Second, "uname", "-r")
	if out != "" {
		return out
	}
	return "Unknown Kernel"
}

func getScreenResolution() string {
	out := runCmdTimeout(2*time.Second, "wm", "size")
	if out != "" {
		if strings.Contains(out, "Physical size:") {
			return strings.TrimSpace(strings.Replace(out, "Physical size:", "", 1))
		}
		return out
	}
	return "Unknown"
}

func getScreenDensity() string {
	out := runCmdTimeout(2*time.Second, "wm", "density")
	if out != "" {
		if strings.Contains(out, "Physical density:") {
			return strings.TrimSpace(strings.Replace(out, "Physical density:", "", 1))
		}
		return out
	}
	d := getProp("ro.sf.lcd_density")
	if d != "" {
		return d + " DPI"
	}
	return "Unknown"
}

func getMTU() string {
	data, err := os.ReadFile("/sys/class/net/wlan0/mtu")
	if err == nil {
		return strings.TrimSpace(string(data))
	}
	return "1500 (Default)"
}

func getDefaultTTL() string {
	data, err := os.ReadFile("/proc/sys/net/ipv4/ip_default_ttl")
	if err == nil {
		return strings.TrimSpace(string(data))
	}
	return "64"
}

func getNetDevBytes() (uint64, uint64) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return 0, 0
	}
	defer file.Close()

	var totalRx, totalTx uint64
	scanner := bufio.NewScanner(file)
	// Skip first 2 header lines
	if scanner.Scan() {
		_ = scanner.Text()
	}
	if scanner.Scan() {
		_ = scanner.Text()
	}

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}
		ifname := strings.TrimSuffix(fields[0], ":")
		if ifname == "lo" || strings.HasPrefix(ifname, "dummy") || strings.HasPrefix(ifname, "tun") {
			continue
		}
		rx, _ := strconv.ParseUint(fields[1], 10, 64)
		tx, _ := strconv.ParseUint(fields[9], 10, 64)
		totalRx += rx
		totalTx += tx
	}
	return totalRx, totalTx
}

// runCmdTimeout runs an external command with a hard timeout, returns trimmed stdout.
func runCmdTimeout(timeout time.Duration, name string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, _ := exec.CommandContext(ctx, name, args...).Output()
	return strings.TrimSpace(string(out))
}

func getNetworkDetail() NetworkDetail {
	networkDetailCacheMu.Lock()
	defer networkDetailCacheMu.Unlock()

	if !networkDetailCacheAt.IsZero() && time.Since(networkDetailCacheAt) < 15*time.Second {
		return networkDetailCache
	}

	nd := buildNetworkDetail()
	networkDetailCache = nd
	networkDetailCacheAt = time.Now()
	return nd
}

func parseSignalQuality(rsrp int) float64 {
	// Scale: -140 dBm → 0.0, -44 dBm → 10.0 (3GPP typical range)
	if rsrp >= 0 {
		return 0.0
	}
	score := (float64(rsrp) + 140.0) / 96.0 * 10.0
	if score < 0 {
		score = 0
	}
	if score > 10 {
		score = 10
	}
	// Round to 1 decimal
	score = float64(int(score*10+0.5)) / 10.0
	return score
}

func buildNetworkDetail() NetworkDetail {
	var nd NetworkDetail
	nd.WiFiSSID = "Unknown"
	nd.WiFiSignalDBM = "—"
	nd.WiFiSignal = "—"
	nd.Roaming = "Unknown"

	const cmdTimeout = 2 * time.Second

	// 1. IP Addresses (cheap — no subprocess)
	ifaces, err := net.Interfaces()
	if err == nil {
		for _, i := range ifaces {
			if i.Flags&net.FlagLoopback != 0 {
				continue
			}
			addrs, err := i.Addrs()
			if err == nil {
				for _, addr := range addrs {
					ip := addr.String()
					if !strings.Contains(ip, ":") { // simple IPv4 filter
						nd.IPAddresses = append(nd.IPAddresses, strings.Split(ip, "/")[0])
					}
				}
			}
		}
	}

	// 2. Gateway
	if out := runCmdTimeout(cmdTimeout, "ip", "route", "show", "default"); out != "" {
		parts := strings.Fields(out)
		for i, p := range parts {
			if p == "via" && i+1 < len(parts) {
				nd.Gateway = parts[i+1]
				break
			}
		}
	}

	// 3. DNS
	dns1 := getProp("net.dns1")
	if dns1 != "" {
		nd.DNS = append(nd.DNS, dns1)
		nd.DNS1 = dns1
	}
	dns2 := getProp("net.dns2")
	if dns2 != "" && dns2 != dns1 {
		nd.DNS = append(nd.DNS, dns2)
		nd.DNS2 = dns2
	}
	if len(nd.DNS) == 0 {
		out := runCmdTimeout(cmdTimeout, "getprop")
		for _, ln := range strings.Split(out, "\n") {
			if strings.Contains(ln, "dns") && strings.Contains(ln, "]: [") {
				parts := strings.Split(ln, "]: [")
				if len(parts) == 2 {
					val := strings.TrimRight(parts[1], "]")
					if net.ParseIP(val) != nil {
						found := false
						for _, d := range nd.DNS {
							if d == val {
								found = true
								break
							}
						}
						if !found {
							nd.DNS = append(nd.DNS, val)
						}
					}
				}
			}
		}
	}

	// 4. WiFi SSID / RSSI
	wifiStr := runCmdTimeout(cmdTimeout, "cmd", "wifi", "status")
	nd.WiFiFullInfo = wifiStr
	if strings.Contains(wifiStr, "SSID: ") {
		ssidParts := strings.Split(wifiStr, "SSID: ")
		if len(ssidParts) > 1 {
			rawSSID := strings.Trim(strings.TrimSpace(strings.Split(ssidParts[1], "\n")[0]), "\" ")
			nd.WiFiSSID = rawSSID
		}
	}
	var rssiVal int
	if strings.Contains(wifiStr, "RSSI: ") {
		rssiParts := strings.Split(wifiStr, "RSSI: ")
		if len(rssiParts) > 1 {
			rssiEnd := strings.Split(rssiParts[1], "\n")[0]
			// e.g. -34
			valStr := strings.TrimSpace(rssiEnd)
			if r, err := strconv.Atoi(valStr); err == nil {
				rssiVal = r
			}
			nd.WiFiRSSI = rssiVal
			nd.WiFiSignalDBM = valStr
			nd.WiFiSignal = nd.WiFiSignalDBM + " dBm"
		}
	}
	// Try to get link speed
	if strings.Contains(wifiStr, "Link speed: ") {
		speedParts := strings.Split(wifiStr, "Link speed: ")
		if len(speedParts) > 1 {
			speedEnd := strings.Split(speedParts[1], "\n")[0] // e.g. 866Mbps
			if nd.WiFiSignal != "—" {
				nd.WiFiSignal += " / " + strings.TrimSpace(speedEnd)
			}
		}
	}

	// 5. MCC/MNC and Roaming
	numString := getProp("gsm.operator.numeric")
	if numString == "" {
		numString = getProp("gsm.sim.operator.numeric")
	}
	for _, num := range strings.Split(numString, ",") {
		n := strings.TrimSpace(num)
		if n != "" {
			nd.MCCMNC = append(nd.MCCMNC, n)
		}
	}
	roam := getProp("gsm.operator.isroaming")
	if strings.Contains(roam, "true") {
		nd.Roaming = "Yes"
	} else if strings.Contains(roam, "false") {
		nd.Roaming = "No"
	}

	// 6. Hotspot clients via /proc/net/arp (no subprocess)
	var arpCount int
	if f, err := os.Open("/proc/net/arp"); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			txt := scanner.Text()
			if strings.Contains(txt, "0x2") && !strings.Contains(txt, "00:00:00:00:00:00") {
				arpCount++
			}
		}
		f.Close()
	}
	nd.HotspotClients = arpCount

	// 7. SIM Slots
	opers := strings.Split(getProp("gsm.operator.alpha"), ",")
	types := strings.Split(getProp("gsm.network.type"), ",")
	nums := strings.Split(getProp("gsm.operator.numeric"), ",")

	numSlots := len(nums)
	if len(opers) > numSlots {
		numSlots = len(opers)
	}
	if len(types) > numSlots {
		numSlots = len(types)
	}

	// Telephony Registry — best-effort, timeout-bounded
	// we will run once and match by subscription/slot index if possible, otherwise apply to slot 1
	telCtx, cancelTel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancelTel()
	telOutBytes, _ := exec.CommandContext(telCtx, "su", "-c", "dumpsys telephony.registry").Output()
	telOut := string(telOutBytes)

	// Naive per-slot extraction from dumpsys
	signalBlocks := strings.Split(telOut, "mSignalStrength=SignalStrength:")
	var rsrps []int
	var rsrqs []int
	var sinrs []int

	for i := 1; i < len(signalBlocks); i++ {
		block := signalBlocks[i]

		rsrp, rsrq, sinr := 0, 0, 0
		if idx := strings.Index(block, "rsrp="); idx != -1 {
			valStart := idx + 5
			valEnd := valStart
			for valEnd < len(block) && (block[valEnd] == '-' || (block[valEnd] >= '0' && block[valEnd] <= '9')) {
				valEnd++
			}
			if valStart < valEnd {
				if r, err := strconv.Atoi(block[valStart:valEnd]); err == nil {
					rsrp = r
				}
			}
		}
		if idx := strings.Index(block, "rsrq="); idx != -1 {
			valStart := idx + 5
			valEnd := valStart
			for valEnd < len(block) && (block[valEnd] == '-' || (block[valEnd] >= '0' && block[valEnd] <= '9')) {
				valEnd++
			}
			if valStart < valEnd {
				if r, err := strconv.Atoi(block[valStart:valEnd]); err == nil {
					rsrq = r
				}
			}
		}
		if idx := strings.Index(block, "rssnr="); idx != -1 {
			valStart := idx + 6
			valEnd := valStart
			for valEnd < len(block) && (block[valEnd] == '-' || (block[valEnd] >= '0' && block[valEnd] <= '9')) {
				valEnd++
			}
			if valStart < valEnd {
				if r, err := strconv.Atoi(block[valStart:valEnd]); err == nil {
					sinr = r
				}
			}
		}

		// some devices return MAX_INT for unavailable
		if rsrp == 2147483647 {
			rsrp = 0
		}
		if rsrq == 2147483647 {
			rsrq = 0
		}
		if sinr == 2147483647 {
			sinr = 0
		}

		rsrps = append(rsrps, rsrp)
		rsrqs = append(rsrqs, rsrq)
		sinrs = append(sinrs, sinr)
	}

	for i := 0; i < numSlots; i++ {
		slot := SIMSlot{Slot: i + 1, Operator: "Unknown", NetworkType: "Unknown"}
		if i < len(opers) && strings.TrimSpace(opers[i]) != "" {
			slot.Operator = strings.TrimSpace(opers[i])
		}
		if i < len(types) && strings.TrimSpace(types[i]) != "" {
			slot.NetworkType = strings.TrimSpace(types[i])
		}
		if i < len(nums) && strings.TrimSpace(nums[i]) != "" && slot.Operator == "Unknown" {
			slot.Operator = strings.TrimSpace(nums[i])
		}

		rsrp, rsrq, sinr := 0, 0, 0
		if i < len(rsrps) {
			rsrp = rsrps[i]
			rsrq = rsrqs[i]
			sinr = sinrs[i]
		}

		slot.RSRPInt = rsrp
		slot.RSRQInt = rsrq
		slot.SINRInt = sinr

		if rsrp == 0 || rsrp == -1 {
			slot.RSRP = "—"
			slot.RSRQ = "—"
			slot.SINR = "—"
			slot.SignalStatus = "—"
		} else {
			slot.RSRP = strconv.Itoa(rsrp)
			slot.SignalStatus = slot.RSRP + " dBm"
			slot.RSRQ = strconv.Itoa(rsrq)
			slot.SINR = strconv.Itoa(sinr)
		}

		score := parseSignalQuality(rsrp)
		slot.SignalScore = score

		if rsrp < 0 {
			if rsrp >= -90 {
				slot.SignalQuality = "Excellent"
			} else if rsrp >= -105 {
				slot.SignalQuality = "Good"
			} else if rsrp >= -115 {
				slot.SignalQuality = "Fair"
			} else {
				slot.SignalQuality = "Poor"
			}
		} else {
			slot.SignalQuality = "Unavailable"
			slot.SignalScore = 0
		}
		nd.SIMSlots = append(nd.SIMSlots, slot)
	}

	return nd
}

func getAndroidLocalTime() string {
	tz := getProp("persist.sys.timezone")
	if tz == "" {
		return time.Now().Format("2006-01-02 15:04:05 MST")
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Now().Format("2006-01-02 15:04:05 MST")
	}
	return time.Now().In(loc).Format("2006-01-02 15:04:05 MST")
}
