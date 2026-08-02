package telegram

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
	"bfr-webui-go/internal/network"
	"bfr-webui-go/internal/sysinfo"
)

func getHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: 4 * time.Second}
				if !strings.Contains(address, "127.0.0.1") && !strings.Contains(address, "[::1]") && !strings.Contains(address, "localhost") {
					if conn, err := d.DialContext(ctx, network, address); err == nil {
						return conn, nil
					}
				}
				for _, dnsServer := range []string{"1.1.1.1:53", "8.8.8.8:53", "9.9.9.9:53"} {
					if conn, err := d.DialContext(ctx, "udp", dnsServer); err == nil {
						return conn, nil
					}
				}
				return d.DialContext(ctx, "udp", "1.1.1.1:53")
			},
		},
	}

	tr := &http.Transport{
		DialContext:     dialer.DialContext,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Proxy:           http.ProxyFromEnvironment,
	}

	return &http.Client{
		Transport: tr,
		Timeout:   timeout,
	}
}

type Config struct {
	Enabled            bool    `json:"enabled"`
	BotToken           string  `json:"bot_token"`
	AllowedChatIDs     []int64 `json:"allowed_chat_ids"`
	AllowShellCommands bool    `json:"allow_shell_commands"`
	NotifyOnBoot       bool    `json:"notify_on_boot"`
}

type StatusResponse struct {
	Config  Config `json:"config"`
	Running bool   `json:"running"`
	BotName string `json:"bot_name,omitempty"`
}

type Manager struct {
	mu         sync.RWMutex
	config     Config
	dataPath   string
	running    bool
	botName    string
	cancelFunc context.CancelFunc
	ctx        context.Context
}

var (
	globalManager *Manager
	once          sync.Once
)

func getStoragePath() string {
	magiskDir := config.ModuleDir
	if magiskDir != "" {
		_ = os.MkdirAll(magiskDir, 0755)
		return filepath.Join(magiskDir, "telegram_config.json")
	}
	return "telegram_config.json"
}

func newManager() *Manager {
	m := &Manager{
		dataPath: getStoragePath(),
		config: Config{
			Enabled:            false,
			BotToken:           "",
			AllowedChatIDs:     []int64{},
			AllowShellCommands: false,
			NotifyOnBoot:       true,
		},
	}
	m.loadConfig()
	return m
}

func GetManager() *Manager {
	once.Do(func() {
		globalManager = newManager()
	})
	return globalManager
}

func isValidBotToken(token string) bool {
	token = strings.TrimSpace(token)
	if token == "" {
		return false
	}
	parts := strings.Split(token, ":")
	if len(parts) != 2 {
		return false
	}
	if parts[0] == "" || parts[1] == "" {
		return false
	}
	for _, c := range parts[0] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func verifyBotToken(token string) (string, error) {
	if !isValidBotToken(token) {
		return "", fmt.Errorf("invalid bot token format (must be numeric_id:secret)")
	}

	reqURL := fmt.Sprintf("https://api.telegram.org/bot%s/getMe", token)
	client := getHTTPClient(10 * time.Second)
	resp, err := client.Get(reqURL)
	if err != nil {
		return "", fmt.Errorf("telegram getMe API connection error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("telegram getMe API returned status %d", resp.StatusCode)
	}

	var res struct {
		Ok     bool `json:"ok"`
		Result struct {
			Username  string `json:"username"`
			FirstName string `json:"first_name"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("failed to parse getMe response: %w", err)
	}

	if !res.Ok {
		return "", fmt.Errorf("telegram API returned ok=false")
	}

	botName := res.Result.Username
	if botName == "" {
		botName = res.Result.FirstName
	}
	return botName, nil
}

func (m *Manager) loadConfig() {
	if _, err := os.Stat(m.dataPath); os.IsNotExist(err) {
		return
	}
	data, err := os.ReadFile(m.dataPath)
	if err != nil {
		logger.Get().Warnf("Telegram", "Failed to read config file %s: %v", m.dataPath, err)
		return
	}

	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		logger.Get().Warnf("Telegram", "Failed to unmarshal telegram config: %v", err)
		return
	}

	m.config = cfg
}

func (m *Manager) saveConfigFileLocked() error {
	dir := filepath.Dir(m.dataPath)
	_ = os.MkdirAll(dir, 0755)

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.dataPath, data, 0644)
}

func (m *Manager) saveConfigFile() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	dir := filepath.Dir(m.dataPath)
	_ = os.MkdirAll(dir, 0755)

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.dataPath, data, 0644)
}

func (m *Manager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Enabled
}

func (m *Manager) GetStatus() StatusResponse {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return StatusResponse{
		Config:  m.config,
		Running: m.running,
		BotName: m.botName,
	}
}

func (m *Manager) SaveConfig(cfg Config) (StatusResponse, error) {
	m.mu.Lock()
	oldEnabled := m.config.Enabled
	oldToken := m.config.BotToken

	token := strings.TrimSpace(cfg.BotToken)
	if token == "" {
		token = oldToken
	}
	if token == "" {
		m.mu.Unlock()
		return StatusResponse{Config: m.config, Running: m.running, BotName: m.botName}, fmt.Errorf("bot token cannot be empty")
	}
	if !isValidBotToken(token) {
		m.mu.Unlock()
		return StatusResponse{Config: m.config, Running: m.running, BotName: m.botName}, fmt.Errorf("invalid bot token format (must be numeric_id:secret)")
	}

	var cleanIDs []int64
	seen := make(map[int64]bool)
	for _, id := range cfg.AllowedChatIDs {
		if id != 0 && !seen[id] {
			cleanIDs = append(cleanIDs, id)
			seen[id] = true
		}
	}

	m.config.BotToken = token
	m.config.AllowedChatIDs = cleanIDs
	m.config.AllowShellCommands = cfg.AllowShellCommands
	m.config.NotifyOnBoot = cfg.NotifyOnBoot
	m.config.Enabled = oldEnabled

	err := m.saveConfigFileLocked()
	m.mu.Unlock()

	if err != nil {
		logger.Get().Errorf("Telegram", "Failed to save telegram_config.json: %v", err)
		return m.GetStatus(), err
	}

	logger.Get().Infof("Telegram", "Telegram configuration updated (Enabled: %v)", oldEnabled)

	if oldEnabled && oldToken != token {
		_ = m.Stop()
		_ = m.Start()
	}

	return m.GetStatus(), nil
}

func (m *Manager) Start() error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	token := strings.TrimSpace(m.config.BotToken)
	if token == "" {
		m.mu.Unlock()
		return fmt.Errorf("bot token is empty. Please enter your Telegram Bot Token and save settings first")
	}

	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.config.Enabled = true
	_ = m.saveConfigFileLocked()
	m.running = true
	m.ctx = ctx
	m.cancelFunc = cancel
	m.mu.Unlock()

	logger.Get().Infof("Telegram", "Starting Telegram bot daemon in background...")

	go m.startDaemonLoop(ctx, token)

	return nil
}

func (m *Manager) startDaemonLoop(ctx context.Context, token string) {
	var botName string
	var err error

	// Retry loop for early boot network initialization (up to 10 retries, 3s apart)
	for retries := 0; retries < 10; retries++ {
		botName, err = verifyBotToken(token)
		if err == nil {
			break
		}
		logger.Get().Warnf("Telegram", "Bot token verification attempt %d/10 failed (waiting for network): %v", retries+1, err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(3 * time.Second):
		}
	}

	if err != nil {
		logger.Get().Errorf("Telegram", "Bot token verification failed: %v", err)
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()
		return
	}

	m.mu.Lock()
	m.botName = botName
	notifyBoot := m.config.NotifyOnBoot
	m.mu.Unlock()

	logger.Get().Infof("Telegram", "Telegram bot daemon connected (@%s)", botName)

	if notifyBoot {
		m.SendMessageToAllowed("🚀 *BFR WebUI Telegram Bot Started*\nDevice service is online and operational.")
	}

	m.pollLoop(ctx)
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	if m.cancelFunc != nil {
		m.cancelFunc()
	}
	m.running = false
	m.botName = ""
	m.config.Enabled = false
	_ = m.saveConfigFileLocked()
	m.mu.Unlock()

	logger.Get().Infof("Telegram", "Telegram bot daemon stopped")
	return nil
}

func (m *Manager) isChatAllowed(chatID int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.config.AllowedChatIDs) == 0 {
		return false
	}
	for _, id := range m.config.AllowedChatIDs {
		if id == chatID {
			return true
		}
	}
	return false
}

func (m *Manager) sendMessage(chatID int64, text string) {
	m.mu.RLock()
	token := m.config.BotToken
	m.mu.RUnlock()

	if token == "" {
		return
	}

	reqURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	client := getHTTPClient(10 * time.Second)
	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewBuffer(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err == nil {
		if resp.StatusCode != http.StatusOK {
			delete(payload, "parse_mode")
			dataRetry, _ := json.Marshal(payload)
			reqRetry, _ := http.NewRequest(http.MethodPost, reqURL, bytes.NewBuffer(dataRetry))
			if reqRetry != nil {
				reqRetry.Header.Set("Content-Type", "application/json")
				respRetry, errRetry := client.Do(reqRetry)
				if errRetry == nil {
					_ = respRetry.Body.Close()
				}
			}
		}
		_ = resp.Body.Close()
	}
}

func (m *Manager) SendMessageToAllowed(text string) {
	m.mu.RLock()
	allowed := make([]int64, len(m.config.AllowedChatIDs))
	copy(allowed, m.config.AllowedChatIDs)
	m.mu.RUnlock()

	for _, chatID := range allowed {
		m.sendMessage(chatID, text)
	}
}

func (m *Manager) NotifyBattery(message string) {
	m.SendMessageToAllowed("🔋 " + message)
}

func NotifyBattery(message string) {
	if globalManager != nil {
		globalManager.NotifyBattery(message)
	}
}

func (m *Manager) pollLoop(ctx context.Context) {
	var offset int64 = 0
	client := getHTTPClient(35 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		m.mu.RLock()
		token := m.config.BotToken
		enabled := m.config.Enabled
		m.mu.RUnlock()

		if !enabled || token == "" {
			time.Sleep(2 * time.Second)
			continue
		}

		reqURL := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=20", token, offset)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			time.Sleep(3 * time.Second)
			continue
		}

		var tgResp struct {
			Ok          bool   `json:"ok"`
			Description string `json:"description"`
			Result      []struct {
				UpdateID int64 `json:"update_id"`
				Message  *struct {
					MessageID int64 `json:"message_id"`
					From      *struct {
						ID        int64  `json:"id"`
						FirstName string `json:"first_name"`
						Username  string `json:"username"`
					} `json:"from"`
					Chat struct {
						ID   int64  `json:"id"`
						Type string `json:"type"`
					} `json:"chat"`
					Text string `json:"text"`
					Date int64  `json:"date"`
				} `json:"message"`
			} `json:"result"`
		}

		err = json.NewDecoder(resp.Body).Decode(&tgResp)
		resp.Body.Close()

		if err != nil || !tgResp.Ok {
			time.Sleep(3 * time.Second)
			continue
		}

		for _, update := range tgResp.Result {
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}

			if update.Message != nil && update.Message.Text != "" {
				m.handleMessage(update.Message.Chat.ID, update.Message.Text)
			}
		}
	}
}

func (m *Manager) handleMessage(chatID int64, text string) {
	if !m.isChatAllowed(chatID) {
		logger.Get().Warnf("Telegram", "Rejected unauthorized message from Chat ID %d", chatID)
		m.sendMessage(chatID, fmt.Sprintf("❌ Unauthorized sender chat ID (%d).", chatID))
		return
	}

	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return
	}

	parts := strings.Fields(text)
	cmd := parts[0]
	if idx := strings.Index(cmd, "@"); idx != -1 {
		cmd = cmd[:idx]
	}

	switch cmd {
	case "/start":
		msg := "🤖 *BFR WebUI Telegram Bot*\n\nStatus: Online\n\n*Available Commands:*\n/stats - System diagnostics\n/reboot - Reboot device\n/tweak - Tweaks status\n/cmd <command> - Execute shell command"
		m.sendMessage(chatID, msg)

	case "/stats":
		st, err := sysinfo.GetStats()
		if err != nil {
			m.sendMessage(chatID, fmt.Sprintf("❌ Error fetching sysinfo stats: %v", err))
			return
		}
		memUsedMB := st.MemUsed / (1024 * 1024)
		memTotalMB := st.MemTotal / (1024 * 1024)
		uptimeMins := int(st.Uptime) / 60
		uptimeHours := uptimeMins / 60
		uptimeMins = uptimeMins % 60

		msg := fmt.Sprintf("📊 *System Statistics*\n\n"+
			"📱 *Model:* %s\n"+
			"⚡ *CPU Usage:* %.1f%%\n"+
			"🌡️ *CPU Temp:* %.1f°C\n"+
			"💾 *Memory:* %d MB / %d MB (%.1f%%)\n"+
			"🔋 *Battery:* %d%% (%s, %.1f°C)\n"+
			"⏱️ *Uptime:* %dh %dm",
			st.Model,
			st.CPUUsage,
			st.CPUTemp,
			memUsedMB, memTotalMB, st.MemUsedPct,
			st.BatteryLevel, st.BatteryStatus, st.BatteryTemp,
			uptimeHours, uptimeMins,
		)
		m.sendMessage(chatID, msg)

	case "/reboot":
		m.sendMessage(chatID, "🔄 *Rebooting system...*")
		logger.Get().Warnf("Telegram", "Reboot command received from Telegram Chat ID %d", chatID)
		go func() {
			time.Sleep(1 * time.Second)
			_, _ = config.ExecSuTimeout(5*time.Second, "reboot")
		}()

	case "/tweak":
		tweaks, err := network.LoadTweaks()
		if err != nil {
			m.sendMessage(chatID, fmt.Sprintf("❌ Error loading tweaks.json: %v", err))
			return
		}
		formatBool := func(v bool) string {
			if v {
				return "ENABLED"
			}
			return "DISABLED"
		}
		msg := fmt.Sprintf("⚙️ *Tweaks Status (tweaks.json)*\n\n"+
			"• LTE Carrier Aggregation: %s\n"+
			"• TCP Buffer Optimization: %s\n"+
			"• BBR2 Congestion Control: %s\n"+
			"• Sysctl Buffers Opt: %s\n"+
			"• Dalvik Responsiveness: %s\n"+
			"• Settings Global Tweaks: %s\n"+
			"• TTL Spoofing: %s\n"+
			"• Packet Steering RPS: %s\n"+
			"• MTU Tuning: %s",
			formatBool(tweaks.LTECarrierAggregation),
			formatBool(tweaks.TCPBufferOptimization),
			formatBool(tweaks.BBR2CongestionControl),
			formatBool(tweaks.SysctlBuffersOpt),
			formatBool(tweaks.DalvikResponsiveness),
			formatBool(tweaks.SettingsGlobalTweaks),
			formatBool(tweaks.TTLSpoofing),
			formatBool(tweaks.PacketSteeringRPS),
			formatBool(tweaks.MTUTuning),
		)
		m.sendMessage(chatID, msg)

	case "/cmd":
		m.mu.RLock()
		allowCmd := m.config.AllowShellCommands
		m.mu.RUnlock()

		if !allowCmd {
			m.sendMessage(chatID, "⚠️ Unauthorized command execution. `allow_shell_commands` is disabled.")
			return
		}

		cmdStr := strings.TrimSpace(strings.TrimPrefix(text, parts[0]))
		if cmdStr == "" {
			m.sendMessage(chatID, "Usage: `/cmd <command>`")
			return
		}

		logger.Get().Infof("Telegram", "Executing shell command from Telegram chat %d: %s", chatID, cmdStr)
		out, err := config.ExecSuTimeout(15*time.Second, cmdStr)
		respText := string(out)
		if respText == "" {
			if err != nil {
				respText = fmt.Sprintf("Error: %v", err)
			} else {
				respText = "(No output)"
			}
		}
		if len(respText) > 3500 {
			respText = respText[:3500] + "\n...(truncated)"
		}
		m.sendMessage(chatID, fmt.Sprintf("💻 *Command:* `%s`\n\n```\n%s\n```", cmdStr, respText))

	default:
		m.sendMessage(chatID, "❓ Unknown command. Send /start to see available commands.")
	}
}
