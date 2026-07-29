package proxy

import (
	"bufio"
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

var (
	hub = &LogHub{
		listeners: make(map[chan string]bool),
	}
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	possibleCores = []string{
		"/data/adb/box/bin/mihomo",
		"/data/adb/clash/clash",
		"/data/adb/modules/box4magisk/bin/mihomo",
		"/data/adb/modules/clash_for_magisk/bin/clash",
		"/system/bin/mihomo",
		"/system/bin/clash",
	}

	watchdogEnabled = false
	watchdogMux     sync.Mutex
)

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

func checkRunning(name string) (int, bool) {
	out, err := exec.Command("su", "-c", "pidof "+name).Output()
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
	out, err := exec.Command("su", "-c", fmt.Sprintf("cat /proc/%d/status | grep RSS", pid)).Output()
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
		cmdStr = "if [ -f /data/adb/box/scripts/box.service ]; then /data/adb/box/scripts/box.service start; elif [ -f /data/adb/clash/scripts/clash.service ]; then /data/adb/clash/scripts/clash.service start; else su -c mihomo -d /data/adb/box/bin/ & fi"
	case "stop":
		SetWatchdog(false)
		cmdStr = "if [ -f /data/adb/box/scripts/box.service ]; then /data/adb/box/scripts/box.service stop; elif [ -f /data/adb/clash/scripts/clash.service ]; then /data/adb/clash/scripts/clash.service stop; else killall mihomo clash 2>/dev/null || true; fi"
	case "restart":
		cmdStr = "if [ -f /data/adb/box/scripts/box.service ]; then /data/adb/box/scripts/box.service restart; elif [ -f /data/adb/clash/scripts/clash.service ]; then /data/adb/clash/scripts/clash.service restart; else killall mihomo clash 2>/dev/null; sleep 1; mihomo -d /data/adb/box/bin/ & fi"
	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	out, err := exec.Command("su", "-c", cmdStr).CombinedOutput()
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

	logFile := "/data/adb/box/run/runs.log"
	if _, err := os.Stat(logFile); err != nil {
		logFile = "/data/adb/clash/run/runs.log"
	}

	cmd := exec.Command("su", "-c", "tail -n 50 -f "+logFile)
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
	resp, err := http.Get("http://127.0.0.1:9090/configs")
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
	req, err := http.NewRequest(http.MethodPatch, "http://127.0.0.1:9090/configs", strings.NewReader(fmt.Sprintf(`{"mode": "%s"}`, mode)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func DetectConfigPath() string {
	paths := []string{
		"/data/adb/box/clash/config.yaml",
		"/data/adb/box/config.yaml",
		"/data/adb/clash/config.yaml",
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
