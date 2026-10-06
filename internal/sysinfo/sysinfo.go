package sysinfo

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/logger"
)

type DiskPartition struct {
	Path    string  `json:"path"`
	Total   uint64  `json:"total"`
	Free    uint64  `json:"free"`
	Used    uint64  `json:"used"`
	UsedPct float64 `json:"used_pct"`
}


type LoadAverage struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

type ServiceStatus struct {
	Name    string  `json:"name"`
	Key     string  `json:"key"`
	Running bool    `json:"running"`
	Detail  string  `json:"detail"`
	PID     int     `json:"pid,omitempty"`
	CPU     float64 `json:"cpu,omitempty"`
	RAM     float64 `json:"ram,omitempty"` // dalam MB (RSS)
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
	DiskTotal      uint64          `json:"disk_total"`
	DiskFree       uint64          `json:"disk_free"`
	DiskUsed       uint64          `json:"disk_used"`
	DiskUsedPct    float64         `json:"disk_used_pct"`
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
	Governor       string          `json:"governor"`
	ROM            string          `json:"rom"`
	RootType       string          `json:"root_type"`
	Chipset        string          `json:"chipset"`
	DeviceName     string          `json:"device_name"`
	WebUIVersion   string          `json:"webui_version"`
	ServerTime     string          `json:"server_time"`
	LocalTime      string          `json:"local_time"`
	Hostname       string          `json:"hostname"`
	NetRx          uint64          `json:"net_rx"`
	NetTx          uint64          `json:"net_tx"`
	NetworkDetail  NetworkDetail   `json:"network_detail"`
}


type staticHardwareInfo struct {
	model         string
	androidVer    string
	sdkVer        string
	securityPatch string
	resolution    string
	density       string
	kernel        string
	rom           string
	rootType      string
	chipset       string
	deviceName    string
	webuiVersion  string
}


var (
	statsCacheMu   sync.RWMutex
	cachedStats    Stats
	lastStatsCache time.Time

	staticInfo     staticHardwareInfo
	staticInfoOnce sync.Once
	monitorOnce    sync.Once
)
func getStaticHardwareInfo() staticHardwareInfo {
	staticInfoOnce.Do(func() {
		model := getProp("ro.product.model")
		if model == "" {
			model = getProp("ro.product.brand") + " " + getProp("ro.product.device")
		}
		staticInfo = staticHardwareInfo{
			model:         model,
			androidVer:    getProp("ro.build.version.release"),
			sdkVer:        getProp("ro.build.version.sdk"),
			securityPatch: getProp("ro.build.version.security_patch"),
			resolution:    getScreenResolution(),
			density:       getScreenDensity(),
			kernel:        getKernelVersion(),
			rom:           getROMBuild(),
			rootType:      getRootProvider(),
			chipset:       getChipset(),
			deviceName:    getDeviceCodename(),
			webuiVersion:  "v1.2.7 (127)",
		}
	})
	return staticInfo
}

func StartBackgroundMonitor() {
	monitorOnce.Do(func() {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Get().Errorf("sysinfo", "Background monitor recovered from panic: %v", r)
				}
			}()
			if s, err := buildStats(); err == nil {
				statsCacheMu.Lock()
				cachedStats = s
				lastStatsCache = time.Now()
				statsCacheMu.Unlock()
			}
			ticker := time.NewTicker(1500 * time.Millisecond)
			defer ticker.Stop()
			for range ticker.C {
				func() {
					defer func() {
						if r := recover(); r != nil {
							logger.Get().Errorf("sysinfo", "Background monitor ticker loop recovered from panic: %v", r)
						}
					}()
					if s, err := buildStats(); err == nil {
						statsCacheMu.Lock()
						cachedStats = s
						lastStatsCache = time.Now()
						statsCacheMu.Unlock()
					}
				}()
			}
		}()
	})
}

func init() {
	lastCoreCPUs = make(map[int]cpuStat)
	getCPUStats()
	StartBackgroundMonitor()
}

func GetStats() (Stats, error) {
	statsCacheMu.RLock()
	if !lastStatsCache.IsZero() {
		s := cachedStats
		statsCacheMu.RUnlock()
		return s, nil
	}
	statsCacheMu.RUnlock()

	return buildStats()
}

func buildStats() (Stats, error) {
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
	for _, d := range s.Disks {
		if d.Path == "/data" || s.DiskTotal == 0 {
			s.DiskTotal = d.Total
			s.DiskFree = d.Free
			s.DiskUsed = d.Used
			s.DiskUsedPct = d.UsedPct
		}
	}

	s.LoadAvg = getLoadAvg()
	s.ActiveServices = getActiveServices()

	info := getStaticHardwareInfo()
	s.Model = info.model
	s.AndroidVer = info.androidVer
	s.SDKVer = info.sdkVer
	s.SecurityPatch = info.securityPatch
	s.Resolution = info.resolution
	s.Density = info.density
	s.Kernel = info.kernel
	s.Governor = getGovernor()
	s.ROM = info.rom
	s.RootType = info.rootType
	s.Chipset = info.chipset
	s.DeviceName = info.deviceName
	s.WebUIVersion = info.webuiVersion

	s.SELinux = getSELinux()
	s.MTU = getMTU()
	s.DefaultTTL = getDefaultTTL()
	s.ServerTime = getServerTime()
	s.LocalTime = getAndroidLocalTime()
	s.Hostname = getHostname()
	s.NetRx, s.NetTx = getNetDevBytes()
	s.NetworkDetail = getNetworkDetail()

	return s, nil
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
	pid := 0

	if key == "webui" {
		pid = os.Getpid()
		running = true
	} else {
		if port > 0 {
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 50*time.Millisecond)
			if err == nil {
				conn.Close()
				running = true
			}
		}
		if len(processNames) > 0 {
			for _, proc := range processNames {
				if foundPID, ok := findPID(proc); ok {
					pid = foundPID
					running = true
					break
				}
				out, err := exec.Command("pidof", proc).Output()
				if err == nil {
					fields := strings.Fields(strings.TrimSpace(string(out)))
					if len(fields) > 0 {
						if p, errConv := strconv.Atoi(fields[0]); errConv == nil && p > 0 {
							pid = p
							running = true
							break
						}
					}
				}
			}
		}
		if running && pid == 0 && len(processNames) > 0 {
			for _, proc := range processNames {
				if foundPID, ok := findPID(proc); ok {
					pid = foundPID
					break
				}
			}
		}
	}

	detail := "Off"
	var cpu, ram float64
	if running {
		if port > 0 {
			detail = fmt.Sprintf("Running (Port %d)", port)
		} else {
			detail = "Running"
		}
		if pid > 0 {
			ram = getProcessMemoryMB(pid)
			cpu = getProcessCPU(pid)
		}
	}

	return ServiceStatus{
		Name:    name,
		Key:     key,
		Running: running,
		Detail:  detail,
		PID:     pid,
		CPU:     cpu,
		RAM:     ram,
	}
}

func getActiveServices() []ServiceStatus {
	return []ServiceStatus{
		checkService("Clash Core", "clash", 9090, "mihomo", "clash"),
		checkService("SSH Daemon", "ssh", 22, "sshd", "dropbear"),
		checkService("ADB Wireless", "adb", 5555, "adbd"),
		checkService("Web UI Server", "webui", 80),
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

func getROMBuild() string {
	if val := getProp("ro.build.display.id"); val != "" {
		return val
	}
	if val := getProp("ro.build.version.incremental"); val != "" {
		return val
	}
	if val := getProp("ro.system.build.version.incremental"); val != "" {
		return val
	}
	return "Stock Android"
}

func getRootProvider() string {
	if _, err := os.Stat("/data/adb/ksu/bin/ksud"); err == nil {
		out := runCmdTimeout(1*time.Second, "/data/adb/ksu/bin/ksud", "-V")
		if out != "" {
			return "KernelSU " + strings.TrimSpace(out)
		}
		return "KernelSU"
	}
	if _, err := os.Stat("/data/adb/ap/bin/apd"); err == nil {
		out := runCmdTimeout(1*time.Second, "/data/adb/ap/bin/apd", "-V")
		if out != "" {
			return "APatch " + strings.TrimSpace(out)
		}
		return "APatch"
	}
	if out := runCmdTimeout(1*time.Second, "magisk", "-v"); out != "" {
		return "Magisk " + strings.TrimSpace(out)
	}
	if _, err := os.Stat("/data/adb/magisk"); err == nil {
		return "Magisk"
	}
	return "Rooted (su)"
}

func getChipset() string {
	soc := getProp("ro.soc.model")
	if soc == "" {
		soc = getProp("ro.board.platform")
	}
	if soc == "" {
		soc = getProp("ro.hardware")
	}
	if soc == "" {
		soc = "Generic ARM64"
	}
	return soc
}

func getDeviceCodename() string {
	dev := getProp("ro.product.device")
	if dev == "" {
		dev = getProp("ro.product.name")
	}
	return dev
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

