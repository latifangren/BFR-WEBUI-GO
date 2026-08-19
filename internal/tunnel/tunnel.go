package tunnel

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
)

type TunnelConfig struct {
	Engine            string `json:"engine"` // "cloudflare", "tailscale", "zerotier"
	Enabled           bool   `json:"enabled"`
	CloudflareToken   string `json:"cloudflare_token"`
	CloudflareQuick   bool   `json:"cloudflare_quick"`
	TailscaleAuthKey  string `json:"tailscale_auth_key"`
	ZeroTierNetworkID string `json:"zerotier_network_id"`
	NgrokAuthToken    string `json:"ngrok_auth_token"`
	PinggyToken       string `json:"pinggy_token"`
}

type TunnelStatus struct {
	Engine      string   `json:"engine"`
	Active      bool     `json:"active"`
	PublicURL   string   `json:"public_url"`
	IPAddress   string   `json:"ip_address"`
	Logs        []string `json:"logs"`
	BinaryFound bool     `json:"binary_found"`
	BinaryPath  string   `json:"binary_path"`
}

type Manager struct {
	mu          sync.RWMutex
	dataPath    string
	config      TunnelConfig
	active      bool
	publicURL   string
	ipAddress   string
	logs        []string
	cmd         *exec.Cmd
	cmdCancel   context.CancelFunc
	activeEngine string
}

var (
	globalManager *Manager
	once          sync.Once

	reURL       = regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)
	reNgrokURL  = regexp.MustCompile(`https://[a-zA-Z0-9-]+\.ngrok-free\.app`)
	rePinggyURL = regexp.MustCompile(`https://[a-zA-Z0-9-]+\.pinggy\.link|https://[a-zA-Z0-9-]+\.a\.pinggy\.online`)
)

func getStoragePath() string {
	return config.GetPersistentFilePath("tunnel.json")
}

func newManager() *Manager {
	m := &Manager{
		dataPath: getStoragePath(),
		config: TunnelConfig{
			Engine:          "cloudflare",
			Enabled:         false,
			CloudflareQuick: true,
		},
		logs: make([]string, 0),
	}
	_, _ = m.LoadConfig()
	return m
}

func GetManager() *Manager {
	once.Do(func() {
		globalManager = newManager()
	})
	return globalManager
}

func (m *Manager) log(msg string) {
	entry := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg)
	m.logs = append(m.logs, entry)
	if len(m.logs) > 50 {
		m.logs = m.logs[len(m.logs)-50:]
	}
}

func (m *Manager) LoadConfig() (*TunnelConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	buf, err := os.ReadFile(m.dataPath)
	if err != nil && m.dataPath != "tunnel.json" {
		buf, err = os.ReadFile("tunnel.json")
	}

	if err == nil {
		buf = bytes.TrimPrefix(buf, []byte("\xef\xbb\xbf"))
		var cfg TunnelConfig
		if err := json.Unmarshal(buf, &cfg); err == nil {
			if cfg.Engine == "" {
				cfg.Engine = "cloudflare"
			}
			m.config = cfg
			return &m.config, nil
		}
	}

	return &m.config, nil
}

func (m *Manager) SaveConfig(cfg *TunnelConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}

	m.config = *cfg

	buf, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(m.dataPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		m.dataPath = "tunnel.json"
	}

	if err := os.WriteFile(m.dataPath, buf, 0644); err != nil {
		if m.dataPath != "tunnel.json" {
			_ = os.WriteFile("tunnel.json", buf, 0644)
		}
		return err
	}

	return nil
}

func GetBinDir() string {
	dir := filepath.Join(config.GetPersistentDataDir(), "bin")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

// FindBinary searches for the tunnel binary in PATH or Magisk/KernelSU/APatch module directories.
func FindBinary(engine string) string {
	var names []string
	switch strings.ToLower(engine) {
	case "cloudflare":
		names = []string{"cloudflared"}
	case "tailscale":
		names = []string{"tailscale", "tailscaled"}
	case "zerotier":
		names = []string{"zerotier-one", "zerotier-cli"}
	case "ngrok":
		names = []string{"ngrok"}
	case "pinggy":
		names = []string{"ssh"}
	default:
		return ""
	}

	// 0. Search in persistent bin dir first
	binDir := GetBinDir()
	for _, name := range names {
		p := filepath.Join(binDir, name)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}

	// 1. Search in PATH
	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}

	// 2. Search in standard Android / Magisk module binary locations
	searchDirs := []string{
		"/data/adb/modules/bfr_webui_go/bin",
		"/data/adb/modules/cloudflared/bin",
		"/data/adb/modules/tailscale/bin",
		"/data/adb/modules/zerotier/bin",
		"/data/adb/ap/bin",
		"/data/adb/ksu/bin",
		"/data/local/tmp",
		"/system/bin",
		"/system/xbin",
	}

	for _, dir := range searchDirs {
		for _, name := range names {
			p := filepath.Join(dir, name)
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
				return p
			}
		}
	}

	return ""
}

// DownloadBinary streams and saves a tunnel binary file into GetBinDir().
func DownloadBinary(engine string, reader io.Reader, fileName string) (string, error) {
	if reader == nil {
		return "", fmt.Errorf("reader cannot be nil")
	}

	engineLower := strings.ToLower(engine)
	targetName := ""
	switch engineLower {
	case "cloudflare":
		targetName = "cloudflared"
	case "ngrok":
		targetName = "ngrok"
	case "tailscale":
		targetName = "tailscale"
	case "zerotier":
		targetName = "zerotier-one"
	default:
		targetName = filepath.Base(fileName)
	}

	targetPath := filepath.Join(GetBinDir(), targetName)
	outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return "", fmt.Errorf("failed creating binary target path: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, reader); err != nil {
		_ = os.Remove(targetPath)
		return "", fmt.Errorf("failed saving binary file: %w", err)
	}

	// Grant executable permissions
	_ = os.Chmod(targetPath, 0755)
	logger.Get().Infof("Tunnel", "Binary %s installed successfully to %s", targetName, targetPath)
	return targetPath, nil
}

func (m *Manager) StopTunnel() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmdCancel != nil {
		m.cmdCancel()
		m.cmdCancel = nil
	}

	if m.cmd != nil && m.cmd.Process != nil {
		_ = m.cmd.Process.Kill()
		m.cmd = nil
	}

	m.active = false
	m.publicURL = ""
	m.ipAddress = ""
	m.activeEngine = ""
	m.log("Tunnel service stopped")
	logger.Get().Infof("Tunnel", "Tunnel service stopped")
	return nil
}

func (m *Manager) StartTunnel(cfg TunnelConfig) error {
	if err := m.SaveConfig(&cfg); err != nil {
		return err
	}

	if !cfg.Enabled {
		return m.StopTunnel()
	}

	_ = m.StopTunnel()

	m.mu.Lock()
	defer m.mu.Unlock()

	binPath := FindBinary(cfg.Engine)
	if binPath == "" {
		m.log(fmt.Sprintf("Error: Binary for engine %s not found", cfg.Engine))
		return fmt.Errorf("binary for engine %s not found", cfg.Engine)
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cmdCancel = cancel
	m.activeEngine = cfg.Engine

	switch strings.ToLower(cfg.Engine) {
	case "cloudflare":
		var args []string
		if cfg.CloudflareToken != "" && !cfg.CloudflareQuick {
			args = []string{"tunnel", "run", "--token", cfg.CloudflareToken}
		} else {
			args = []string{"tunnel", "--url", "http://localhost:8080"}
		}

		cmd := exec.CommandContext(ctx, binPath, args...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			cancel()
			return fmt.Errorf("failed creating stdout pipe: %w", err)
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			cancel()
			return fmt.Errorf("failed creating stderr pipe: %w", err)
		}

		if err := cmd.Start(); err != nil {
			cancel()
			return fmt.Errorf("failed starting cloudflared: %w", err)
		}

		m.cmd = cmd
		m.active = true
		m.log(fmt.Sprintf("Started Cloudflare Tunnel (%s)", binPath))

		// Monitor output for quick tunnel URL parsing
		reader := io.MultiReader(stdout, stderr)
		go func() {
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				line := scanner.Text()
				if match := reURL.FindString(line); match != "" {
					m.mu.Lock()
					m.publicURL = match
					m.log(fmt.Sprintf("Cloudflare Public URL: %s", match))
					m.mu.Unlock()
				}
			}
		}()

	case "tailscale":
		var args []string
		if cfg.TailscaleAuthKey != "" {
			args = []string{"up", "--authkey", cfg.TailscaleAuthKey}
		} else {
			args = []string{"up"}
		}

		out, err := config.ExecSuTimeout(15*time.Second, fmt.Sprintf("%s %s", binPath, strings.Join(args, " ")))
		if err != nil {
			cancel()
			return fmt.Errorf("failed starting Tailscale: %v, output: %s", err, string(out))
		}

		m.active = true
		m.log("Tailscale connected successfully")

		// Query Tailscale IPv4 address
		ipOut, err := config.ExecSuTimeout(5*time.Second, fmt.Sprintf("%s ip -4", binPath))
		if err == nil {
			m.ipAddress = strings.TrimSpace(string(ipOut))
		}

	case "zerotier":
		if cfg.ZeroTierNetworkID == "" {
			cancel()
			return fmt.Errorf("zerotier_network_id is required")
		}

		out, err := config.ExecSuTimeout(15*time.Second, fmt.Sprintf("%s join %s", binPath, cfg.ZeroTierNetworkID))
		if err != nil {
			cancel()
			return fmt.Errorf("failed joining ZeroTier network: %v, output: %s", err, string(out))
		}

		m.active = true
		m.log(fmt.Sprintf("ZeroTier joined network %s", cfg.ZeroTierNetworkID))

		// Query assigned ZeroTier IP
		listOut, err := config.ExecSuTimeout(5*time.Second, fmt.Sprintf("%s listnetworks", binPath))
		if err == nil {
			fields := strings.Fields(string(listOut))
			for _, f := range fields {
				if ip := net.ParseIP(strings.Split(f, "/")[0]); ip != nil {
					m.ipAddress = ip.String()
					break
				}
			}
		}

	case "ngrok":
		if cfg.NgrokAuthToken != "" {
			_ = exec.CommandContext(ctx, binPath, "config", "add-authtoken", cfg.NgrokAuthToken).Run()
		}
		cmd := exec.CommandContext(ctx, binPath, "http", "8080", "--log=stdout")
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			cancel()
			return fmt.Errorf("failed creating ngrok stdout pipe: %w", err)
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			cancel()
			return fmt.Errorf("failed creating ngrok stderr pipe: %w", err)
		}

		if err := cmd.Start(); err != nil {
			cancel()
			return fmt.Errorf("failed starting ngrok: %w", err)
		}

		m.cmd = cmd
		m.active = true
		m.log(fmt.Sprintf("Started Ngrok Tunnel (%s)", binPath))

		reader := io.MultiReader(stdout, stderr)
		go func() {
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				line := scanner.Text()
				if match := reNgrokURL.FindString(line); match != "" {
					m.mu.Lock()
					m.publicURL = match
					m.log(fmt.Sprintf("Ngrok Public URL: %s", match))
					m.mu.Unlock()
				}
			}
		}()

	case "pinggy":
		userHost := "a.pinggy.io"
		if cfg.PinggyToken != "" {
			userHost = cfg.PinggyToken + "@a.pinggy.io"
		}
		cmd := exec.CommandContext(ctx, binPath, "-p", "443", "-R", "0:localhost:8080", "-o", "StrictHostKeyChecking=no", "-o", "ServerAliveInterval=30", userHost)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			cancel()
			return fmt.Errorf("failed creating pinggy stdout pipe: %w", err)
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			cancel()
			return fmt.Errorf("failed creating pinggy stderr pipe: %w", err)
		}

		if err := cmd.Start(); err != nil {
			cancel()
			return fmt.Errorf("failed starting Pinggy: %w", err)
		}

		m.cmd = cmd
		m.active = true
		m.log(fmt.Sprintf("Started Pinggy SSH Tunnel (%s)", binPath))

		reader := io.MultiReader(stdout, stderr)
		go func() {
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				line := scanner.Text()
				if match := rePinggyURL.FindString(line); match != "" {
					m.mu.Lock()
					m.publicURL = match
					m.log(fmt.Sprintf("Pinggy Public URL: %s", match))
					m.mu.Unlock()
				}
			}
		}()
	}

	logger.Get().Infof("Tunnel", "Tunnel started successfully using engine %s", cfg.Engine)
	return nil
}

func (m *Manager) GetStatus() TunnelStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	engine := m.config.Engine
	if engine == "" {
		engine = "cloudflare"
	}

	binPath := FindBinary(engine)

	logsCopy := make([]string, len(m.logs))
	copy(logsCopy, m.logs)

	return TunnelStatus{
		Engine:      engine,
		Active:      m.active,
		PublicURL:   m.publicURL,
		IPAddress:   m.ipAddress,
		Logs:        logsCopy,
		BinaryFound: binPath != "",
		BinaryPath:  binPath,
	}
}
