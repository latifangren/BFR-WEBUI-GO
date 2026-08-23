package ssh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/config"
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
	return config.GetPersistentFilePath("ssh_config.json")
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

func (m *Manager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Enabled
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
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
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
	return config.WriteFileAtomic(m.dataPath, data, 0644)
}

func (m *Manager) detectSshBinary() string {
	// Only standalone dropbear binaries or Termux OpenSSH are accepted.
	// Stock Android /system/bin/sshd and /product/bin/sshd are vendor RIL/debug
	// daemons that exit with error when invoked from CLI — excluded explicitly.
	paths := []string{
		filepath.Join(config.ModuleDir, "bin/arm64/dropbear"),
		filepath.Join(config.ModuleDir, "bin/dropbear"),
		"/data/adb/modules/bfr-webui/bin/arm64/dropbear",
		"/data/adb/modules/bfr-webui/bin/dropbear",
		"bin/arm64/dropbear",
		"bin/dropbear",
		"/data/adb/modules/bfr_webui_go/bin/arm64/dropbear",
		"/data/adb/modules/bfr_webui_go/bin/dropbear",
		"/system/bin/dropbear",
		"/product/bin/dropbear",
		"/vendor/bin/dropbear",
		"/data/adb/magisk/dropbear",
		"/data/adb/apatch/dropbear",
		"/data/local/tmp/dropbear",
		"/data/local/tmp/bin/arm64/dropbear",
		"/data/data/com.termux/files/usr/bin/dropbear",
		// Termux OpenSSH is a real standalone sshd; stock Android sshd is NOT.
		"/data/data/com.termux/files/usr/bin/sshd",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Fallback: search PATH via root shell — only accept dropbear (safe).
	// Never accept 'which sshd' here because it may resolve to a stock Android daemon.
	for _, bin := range []string{"dropbear"} {
		out, err := exec.Command(config.SUBin, "-c", "which "+bin).Output()
		if err == nil {
			p := strings.TrimSpace(string(out))
			if p != "" && !strings.Contains(p, "not found") {
				return p
			}
		}
		if path, err := exec.LookPath(bin); err == nil {
			return path
		}
	}
	return ""
}

func ensurePasswd() {
	_ = exec.Command(config.SUBin, "-c", "mkdir -p /data/ssh").Run()

	// Default root SSH password is set to "bfr" (MD5 crypt hash for "bfr" with salt "12345678")
	// Shell /bin/sh is natively accepted by Dropbear (musl) default getusershell list.
	passwdEntry := "root:$1$12345678$AnOalxy58Pt6NMyuN0T54.:0:0:root:/data/local/tmp:/bin/sh"
	cmdPasswd := fmt.Sprintf("echo '%s' > /data/ssh/passwd && chmod 644 /data/ssh/passwd && mount -o bind /data/ssh/passwd /etc/passwd 2>/dev/null", passwdEntry)
	_ = exec.Command(config.SUBin, "-c", cmdPasswd).Run()
}

func ensureDropbearHostKeys(bin string) string {
	keyPath := "/data/ssh/dropbear_ecdsa_host_key"
	_ = exec.Command(config.SUBin, "-c", "mkdir -p /data/ssh").Run()

	checkCmd := exec.Command(config.SUBin, "-c", "test -f "+keyPath)
	if checkCmd.Run() == nil {
		return keyPath
	}

	dropbearkey := filepath.Join(filepath.Dir(bin), "dropbearkey")
	if _, err := os.Stat(dropbearkey); err != nil {
		dropbearkey = filepath.Join(config.ModuleDir, "bin/arm64/dropbearkey")
	}
	if _, err := os.Stat(dropbearkey); err != nil {
		dropbearkey = "dropbearkey"
	}

	genCmd := fmt.Sprintf("%s -t ecdsa -f %s", dropbearkey, keyPath)
	_ = exec.Command(config.SUBin, "-c", genCmd).Run()

	return keyPath
}

// ensureOpenSSHHostKeys ensures /data/ssh/ssh_host_rsa_key exists for use with
// Termux OpenSSH (sshd). Generates via ssh-keygen if missing.
func ensureOpenSSHHostKeys() {
	keyPath := "/data/ssh/ssh_host_rsa_key"
	_ = exec.Command(config.SUBin, "-c", "mkdir -p /data/ssh").Run()

	checkCmd := exec.Command(config.SUBin, "-c", "test -f "+keyPath)
	if checkCmd.Run() == nil {
		return // key already present
	}

	genCmd := fmt.Sprintf("ssh-keygen -t rsa -f %s -N \"\"", keyPath)
	out, err := exec.Command(config.SUBin, "-c", genCmd).CombinedOutput()
	logger.Get().Infof("ssh", "ensureOpenSSHHostKeys ssh-keygen out=%s err=%v", strings.TrimSpace(string(out)), err)
}

func (m *Manager) getRunningProcessForPort(port int) (string, int) {
	candidates := []string{"dropbear", "sshd"}
	for _, c := range candidates {
		cmdStr := fmt.Sprintf("pgrep -f \"%s.*%d\"", c, port)
		out, err := exec.Command(config.SUBin, "-c", cmdStr).Output()
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
	cmd := exec.Command(config.SUBin, "-c", fmt.Sprintf("kill -0 %d", pid))
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
	defer func() {
		if r := recover(); r != nil {
			logger.Get().Errorf("ssh", "Recovered from SSH Start panic: %v", r)
		}
	}()

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
		ensurePasswd()
		keyPath := ensureDropbearHostKeys(bin)
		var addr string
		if m.config.Bind == "" || m.config.Bind == "0.0.0.0" {
			addr = fmt.Sprintf("%d", m.config.Port)
		} else {
			addr = fmt.Sprintf("%s:%d", m.config.Bind, m.config.Port)
		}
		cmdStr = fmt.Sprintf("%s -p %s -r %s", bin, addr, keyPath)
	} else {
		// Termux OpenSSH sshd — ensure RSA host key exists before calling sshd.
		ensureOpenSSHHostKeys()
		cmdStr = fmt.Sprintf("%s -h /data/ssh/ssh_host_rsa_key -p %d -o \"ListenAddress %s\" -o \"PasswordAuthentication yes\"", bin, m.config.Port, m.config.Bind)
	}

	// Use nohup for clean daemon detach — prevents SIGHUP kill when su exits
	fullCmd := fmt.Sprintf("nohup %s > /dev/null 2>&1 &", cmdStr)
	cmd := exec.Command(config.SUBin, "-c", fullCmd)
	if err := cmd.Run(); err != nil {
		l := logger.Get()
		l.Errorf("ssh", "failed to launch ssh daemon binary=%s err=%v", bin, err)
		return fmt.Errorf("failed to launch ssh daemon: %v", err)
	}

	m.config.Enabled = true
	_ = m.saveConfig()

	// Post-start verification: wait 1200ms then check if daemon actually survived
	time.Sleep(1200 * time.Millisecond)
	_, verifiedPid := m.getRunningProcessForPort(m.config.Port)
	if verifiedPid == 0 {
		// Fallback pgrep for dropbear or sshd without strict port regex
		for _, c := range []string{"dropbear", "sshd"} {
			out, err := exec.Command(config.SUBin, "-c", "pgrep -f "+c).Output()
			if err == nil {
				for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
					if p, err := strconv.Atoi(strings.TrimSpace(line)); err == nil && p > 0 {
						if checkPidRunning(p) {
							verifiedPid = p
							break
						}
					}
				}
			}
			if verifiedPid > 0 {
				break
			}
		}
	}

	l := logger.Get()
	if verifiedPid == 0 {
		l.Errorf("ssh", "daemon exited immediately after start binary=%s cmd=%s", bin, fullCmd)
		return fmt.Errorf("ssh daemon started but exited immediately. binary=%s cmd=%s", bin, fullCmd)
	}

	l.Infof("ssh", "daemon started pid=%d port=%d bind=%s binary=%s", verifiedPid, m.config.Port, m.config.Bind, bin)
	return nil
}

func (m *Manager) Stop() error {
	defer func() {
		if r := recover(); r != nil {
			logger.Get().Errorf("ssh", "Recovered from SSH Stop panic: %v", r)
		}
	}()

	m.mu.Lock()
	defer m.mu.Unlock()

	_, pid := m.getRunningProcessForPort(m.config.Port)
	if pid > 0 {
		_ = exec.Command(config.SUBin, "-c", fmt.Sprintf("kill -15 %d", pid)).Run()
	}

	l := logger.Get()
	l.Infof("ssh", "daemon stopped pid=%d", pid)

	m.config.Enabled = false
	_ = m.saveConfig()

	return nil
}
