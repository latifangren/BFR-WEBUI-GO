package ssh

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/logger"
)

type Config struct {
	Enabled     bool   `json:"enabled"`
	Port        int    `json:"port"`
	Bind        string `json:"bind"`
	KeyAuthOnly bool   `json:"key_auth_only"`
}

type StatusResponse struct {
	Config     Config `json:"config"`
	Running    bool   `json:"running"`
	Pid        int    `json:"pid"`
	BinaryPath string `json:"binary_path"`
}

type Manager struct {
	mu         sync.RWMutex
	config     Config
	dataPath   string
	binaryPath string
}

var (
	globalManager *Manager
	once          sync.Once
)

func getStoragePath() string {
	magiskDir := "/data/adb/modules/bfr_webui_go"
	magiskPath := filepath.Join(magiskDir, "ssh_config.json")
	if _, err := os.Stat(magiskDir); err == nil {
		return magiskPath
	}
	return "ssh_config.json"
}

func newManager() *Manager {
	m := &Manager{
		dataPath: getStoragePath(),
		config: Config{
			Enabled:     false,
			Port:        2222,
			Bind:        "127.0.0.1",
			KeyAuthOnly: true,
		},
	}
	m.loadConfig()
	m.binaryPath = m.detectSshBinary()
	return m
}

func GetManager() *Manager {
	once.Do(func() {
		globalManager = newManager()
	})
	return globalManager
}

func (m *Manager) loadConfig() {
	if _, err := os.Stat(m.dataPath); os.IsNotExist(err) {
		return
	}
	data, err := os.ReadFile(m.dataPath)
	if err != nil {
		return
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err == nil {
		m.config = cfg
	}
}

func (m *Manager) saveConfig() error {
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.dataPath, data, 0644)
}

func (m *Manager) detectSshBinary() string {
	paths := []string{
		"/system/bin/dropbear",
		"/system/bin/sshd",
		"/data/adb/magisk/dropbear",
		"/data/adb/apatch/dropbear",
		"/data/data/com.termux/files/usr/bin/sshd",
		"/data/data/com.termux/files/usr/sbin/sshd",
		"/data/data/com.termux/files/usr/bin/dropbear",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Fallback to searching PATH
	for _, bin := range []string{"dropbear", "sshd"} {
		if path, err := exec.LookPath(bin); err == nil {
			return path
		}
	}
	return ""
}

func (m *Manager) getRunningProcessForPort(port int) (string, int) {
	candidates := []string{"dropbear", "sshd"}
	for _, c := range candidates {
		// Specifically target sshd/dropbear instances bound to our custom port
		cmd := exec.Command("su", "-c", fmt.Sprintf("pgrep -f \"%s.*-p %d\"", c, port))
		out, err := cmd.Output()
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			for _, line := range lines {
				if pid, err := strconv.Atoi(strings.TrimSpace(line)); err == nil && pid > 0 {
					if checkPidRunning(pid) {
						return c, pid
					}
				}
			}
		}
	}
	return "", 0
}

func (m *Manager) getRunningProcess() (string, int) {
	m.mu.RLock()
	port := m.config.Port
	m.mu.RUnlock()
	return m.getRunningProcessForPort(port)
}

func checkPidRunning(pid int) bool {
	cmd := exec.Command("su", "-c", fmt.Sprintf("kill -0 %d", pid))
	return cmd.Run() == nil
}

// M-5: GetStatus does not hold m.mu.RLock() while executing su/pgrep commands.
func (m *Manager) GetStatus() StatusResponse {
	m.mu.RLock()
	cfg := m.config
	bin := m.binaryPath
	m.mu.RUnlock()

	_, pid := m.getRunningProcessForPort(cfg.Port)
	running := pid > 0
	if bin == "" {
		bin = m.detectSshBinary()
	}

	return StatusResponse{
		Config:     cfg,
		Running:    running,
		Pid:        pid,
		BinaryPath: bin,
	}
}

// M-12: SaveConfig validates cfg.Port (1 to 65535).
func (m *Manager) SaveConfig(cfg Config) (StatusResponse, error) {
	if cfg.Port < 1 || cfg.Port > 65535 {
		return m.GetStatus(), fmt.Errorf("invalid port: must be between 1 and 65535")
	}

	m.mu.Lock()
	m.config = cfg
	_ = m.saveConfig()
	m.mu.Unlock()

	return m.GetStatus(), nil
}

func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, pid := m.getRunningProcessForPort(m.config.Port)
	if pid > 0 {
		return nil // already running
	}

	bin := m.binaryPath
	if bin == "" {
		bin = m.detectSshBinary()
		m.binaryPath = bin
	}
	if bin == "" {
		l := logger.Get()
		l.Errorf("ssh", "no ssh daemon binary found on system")
		return fmt.Errorf("no SSH daemon binary found on target system")
	}

	isDropbear := strings.Contains(strings.ToLower(bin), "dropbear")
	var cmdStr string

	if isDropbear {
		addr := fmt.Sprintf("%s:%d", m.config.Bind, m.config.Port)
		cmdStr = fmt.Sprintf("%s -p %s -R", bin, addr)
	} else {
		cmdStr = fmt.Sprintf("%s -h /data/ssh/ssh_host_rsa_key -p %d -o \"ListenAddress %s\" -o \"PasswordAuthentication yes\"", bin, m.config.Port, m.config.Bind)
	}

	// Use nohup for clean daemon detach — prevents SIGHUP kill when su exits
	fullCmd := fmt.Sprintf("nohup %s > /dev/null 2>&1 &", cmdStr)
	cmd := exec.Command("su", "-c", fullCmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to launch ssh daemon: %v", err)
	}

	m.config.Enabled = true
	_ = m.saveConfig()

	// Post-start verification: wait briefly then check if daemon actually survived
	time.Sleep(500 * time.Millisecond)
	_, verifiedPid := m.getRunningProcessForPort(m.config.Port)
	l := logger.Get()
	if verifiedPid == 0 {
		l.Errorf("ssh", "daemon exited immediately after start binary=%s cmd=%s", bin, fullCmd)
		return fmt.Errorf("ssh daemon started but exited immediately. binary=%s cmd=%s", bin, fullCmd)
	}

	l.Infof("ssh", "daemon started pid=%d port=%d bind=%s binary=%s", verifiedPid, m.config.Port, m.config.Bind, bin)
	return nil
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, pid := m.getRunningProcessForPort(m.config.Port)
	if pid > 0 {
		_ = exec.Command("su", "-c", fmt.Sprintf("kill -15 %d", pid)).Run()
	}

	l := logger.Get()
	l.Infof("ssh", "daemon stopped pid=%d", pid)

	m.config.Enabled = false
	_ = m.saveConfig()

	return nil
}
