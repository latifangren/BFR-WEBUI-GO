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

func (m *Manager) getRunningProcess() (string, int) {
	candidates := []string{"dropbear", "sshd"}
	for _, c := range candidates {
		// Run via su -c pgrep to query root processes
		cmd := exec.Command("su", "-c", "pgrep -f "+c)
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

func checkPidRunning(pid int) bool {
	cmd := exec.Command("su", "-c", fmt.Sprintf("kill -0 %d", pid))
	return cmd.Run() == nil
}

func (m *Manager) GetStatus() StatusResponse {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, pid := m.getRunningProcess()
	running := pid > 0
	bin := m.binaryPath
	if bin == "" {
		bin = m.detectSshBinary()
	}

	return StatusResponse{
		Config:     m.config,
		Running:    running,
		Pid:        pid,
		BinaryPath: bin,
	}
}

func (m *Manager) SaveConfig(cfg Config) StatusResponse {
	m.mu.Lock()
	m.config = cfg
	_ = m.saveConfig()
	m.mu.Unlock()

	return m.GetStatus()
}

func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, pid := m.getRunningProcess()
	if pid > 0 {
		return nil // already running
	}

	bin := m.binaryPath
	if bin == "" {
		bin = m.detectSshBinary()
		m.binaryPath = bin
	}
	if bin == "" {
		return fmt.Errorf("no SSH daemon binary found on target system")
	}

	isDropbear := strings.Contains(strings.ToLower(bin), "dropbear")
	var cmdStr string

	if isDropbear {
		addr := fmt.Sprintf("%s:%d", m.config.Bind, m.config.Port)
		// Run dropbear as a daemon process in the background using `su`
		cmdStr = fmt.Sprintf("%s -p %s -R", bin, addr)
	} else {
		// Assume OpenSSH sshd; -D runs in foreground if we want but since we spawn as daemon we run blockingly in su shell background
		cmdStr = fmt.Sprintf("%s -h /data/ssh/ssh_host_rsa_key -p %d -o \"ListenAddress %s\" -o \"PasswordAuthentication yes\"", bin, m.config.Port, m.config.Bind)
	}

	// Launch in root shell context in the background using background execution syntax
	cmd := exec.Command("su", "-c", cmdStr+" &")
	err := cmd.Start()
	if err != nil {
		return err
	}

	m.config.Enabled = true
	_ = m.saveConfig()

	// Wait in goroutine to release system process resources
	go func() {
		_ = cmd.Wait()
	}()

	return nil
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	procName, pid := m.getRunningProcess()
	if pid > 0 {
		_ = exec.Command("su", "-c", fmt.Sprintf("kill -15 %d", pid)).Run()
	}

	// Generic backup kill tools
	if procName != "" {
		_ = exec.Command("su", "-c", "killall "+procName).Run()
		_ = exec.Command("su", "-c", "pkill -f "+procName).Run()
	}

	m.config.Enabled = false
	_ = m.saveConfig()

	return nil
}
