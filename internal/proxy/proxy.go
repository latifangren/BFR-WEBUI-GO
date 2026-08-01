package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/config"
	"github.com/gorilla/websocket"
)

type CoreInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Exists  bool   `json:"exists"`
	Running bool   `json:"running"`
	PID     int    `json:"pid"`
	Memory  string `json:"memory"`
}

type LogHub struct {
	mu        sync.RWMutex
	listeners map[chan string]bool
}

// M-7: dedicated HTTP client with timeout for Clash API requests.
var clashHTTPClient = &http.Client{
	Timeout: 5 * time.Second,
}

var (
	hub = &LogHub{
		listeners: make(map[chan string]bool),
	}
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	// L-4: allow overriding hardcoded base paths via environment variables.
	boxBasePath   = envOrDefault("BFR_BOX_BASE", "/data/adb/box")
	clashBasePath = envOrDefault("BFR_CLASH_BASE", "/data/adb/clash")

	possibleCores = []string{
		boxBasePath + "/bin/mihomo",
		clashBasePath + "/clash",
		"/data/adb/modules/box4magisk/bin/mihomo",
		"/data/adb/modules/clash_for_magisk/bin/clash",
		"/system/bin/mihomo",
		"/system/bin/clash",
	}

	watchdogEnabled = false
	watchdogMux     sync.Mutex
)

// envOrDefault returns the value of the environment variable if set, else the fallback.
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func init() {
	go runWatchdog()
}

func SetWatchdog(enable bool) {
	watchdogMux.Lock()
	defer watchdogMux.Unlock()
	watchdogEnabled = enable
}

func GetWatchdog() bool {
	watchdogMux.Lock()
	defer watchdogMux.Unlock()
	return watchdogEnabled
}

func runWatchdog() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if GetWatchdog() {
			cores := DetectCores()
			anyRunning := false
			for _, c := range cores {
				if c.Running {
					anyRunning = true
					break
				}
			}
			if !anyRunning {
				_ = ControlService("start")
			}
		}
	}
}

func DetectCores() []CoreInfo {
	var list []CoreInfo
	for _, p := range possibleCores {
		name := "mihomo"
		if strings.Contains(p, "clash") {
			name = "clash"
		}
		info := CoreInfo{
			Name: name,
			Path: p,
		}

		if _, err := os.Stat(p); err == nil {
			info.Exists = true
			pid, running := checkRunning(name)
			info.Running = running
			info.PID = pid
			if running && pid > 0 {
				info.Memory = getMemoryUsage(pid)
			}
		}
		list = append(list, info)
	}
	return list
}

func findPID(name string) (int, bool) {
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

		cmdlinePath := fmt.Sprintf("/proc/%d/cmdline", pid)
		if cmdlineBytes, err := os.ReadFile(cmdlinePath); err == nil && len(cmdlineBytes) > 0 {
			cmdStr := strings.ReplaceAll(string(cmdlineBytes), "\x00", " ")
			cmdStr = strings.TrimSpace(cmdStr)
			fields := strings.Fields(cmdStr)
			if len(fields) > 0 {
				base := filepath.Base(fields[0])
				if strings.EqualFold(base, name) || strings.Contains(strings.ToLower(cmdStr), nameLower) {
					return pid, true
				}
			}
		}
	}

	return 0, false
}

func checkRunning(name string) (int, bool) {
	if pid, ok := findPID(name); ok {
		return pid, true
	}
	out, err := exec.Command(config.SUBin, "-c", "pidof "+name).Output()
	if err == nil {
		fields := strings.Fields(string(out))
		if len(fields) > 0 {
			pid, _ := strconv.Atoi(fields[0])
			return pid, true
		}
	}
	return 0, false
}

func getMemoryUsage(pid int) string {
	statusPath := fmt.Sprintf("/proc/%d/status", pid)
	if data, err := os.ReadFile(statusPath); err == nil {
		scanner := bufio.NewScanner(bytes.NewReader(data))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "VmRSS:") {
				return strings.TrimSpace(line)
			}
		}
	}

	out, err := exec.Command(config.SUBin, "-c", fmt.Sprintf("cat /proc/%d/status | grep RSS", pid)).Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	return "N/A"
}

func ControlService(action string) error {
	var cmdStr string
	switch action {
	case "start":
		if _, ok := checkRunning("mihomo"); ok {
			return nil
		}
		if _, ok := checkRunning("clash"); ok {
			return nil
		}
		cmdStr = fmt.Sprintf(
			"if [ -f %s/scripts/box.service ]; then %s/scripts/box.service start; elif [ -f %s/scripts/clash.service ]; then %s/scripts/clash.service start; else %s -c mihomo -d %s/bin/ & fi",
			boxBasePath, boxBasePath, clashBasePath, clashBasePath, config.SUBin, boxBasePath,
		)
	case "stop":
		SetWatchdog(false)
		cmdStr = fmt.Sprintf(
			"if [ -f %s/scripts/box.service ]; then %s/scripts/box.service stop; elif [ -f %s/scripts/clash.service ]; then %s/scripts/clash.service stop; else killall mihomo clash 2>/dev/null || true; fi",
			boxBasePath, boxBasePath, clashBasePath, clashBasePath,
		)
	case "restart":
		cmdStr = fmt.Sprintf(
			"if [ -f %s/scripts/box.service ]; then %s/scripts/box.service restart; elif [ -f %s/scripts/clash.service ]; then %s/scripts/clash.service restart; else killall mihomo clash 2>/dev/null; sleep 1; mihomo -d %s/bin/ & fi",
			boxBasePath, boxBasePath, clashBasePath, clashBasePath, boxBasePath,
		)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	out, err := exec.Command(config.SUBin, "-c", cmdStr).CombinedOutput()
	if err != nil {
		return fmt.Errorf("control error: %v, out: %s", err, string(out))
	}
	return nil
}

func BroadcastLog(msg string) {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	for ch := range hub.listeners {
		select {
		case ch <- msg:
		default:
		}
	}
}

func StreamLogs(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	logChan := make(chan string, 50)
	hub.mu.Lock()
	hub.listeners[logChan] = true
	hub.mu.Unlock()

	defer func() {
		hub.mu.Lock()
		delete(hub.listeners, logChan)
		hub.mu.Unlock()
		close(logChan)
	}()

	// L-4: use env-overridable base path for log file detection.
	logFile := boxBasePath + "/run/runs.log"
	if _, err := os.Stat(logFile); err != nil {
		logFile = clashBasePath + "/run/runs.log"
	}

	cmd := exec.Command(config.SUBin, "-c", "tail -n 50 -f "+logFile)
	stdout, err := cmd.StdoutPipe()
	if err == nil {
		if err := cmd.Start(); err == nil {
			defer func() {
				_ = cmd.Process.Kill()
			}()
			go func() {
				scanner := bufio.NewScanner(stdout)
				for scanner.Scan() {
					BroadcastLog(scanner.Text())
				}
			}()
		}
	}

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case msg := <-logChan:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		}
	}
}

func GetMode() string {
	// M-7: use dedicated clashHTTPClient with timeout.
	resp, err := clashHTTPClient.Get(config.ClashAPI + "/configs")
	if err != nil {
		return "Rule"
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(body), `"mode":"Global"`) {
		return "Global"
	}
	if strings.Contains(string(body), `"mode":"Direct"`) {
		return "Direct"
	}
	return "Rule"
}

func SetMode(mode string) error {
	// M-2/B-1: use json.Marshal for the payload instead of fmt.Sprintf.
	payload, err := json.Marshal(map[string]string{"mode": mode})
	if err != nil {
		return fmt.Errorf("failed to marshal mode payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPatch, config.ClashAPI+"/configs", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	// M-7: use dedicated clashHTTPClient with timeout.
	resp, err := clashHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func DetectConfigPath() string {
	// L-4: allow overriding base paths via environment variables.
	paths := []string{
		boxBasePath + "/clash/config.yaml",
		boxBasePath + "/config.yaml",
		clashBasePath + "/config.yaml",
		"/data/adb/modules/box4magisk/config.yaml",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return paths[0]
}

func ReadConfig() (string, string, error) {
	p := DetectConfigPath()
	data, err := os.ReadFile(p)
	if err != nil {
		return p, "", err
	}
	return p, string(data), nil
}

func SaveConfig(content string) error {
	p := DetectConfigPath()
	dir := filepath.Dir(p)
	_ = os.MkdirAll(dir, 0755)
	return os.WriteFile(p, []byte(content), 0644)
}
