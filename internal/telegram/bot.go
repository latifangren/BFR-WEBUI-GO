package telegram

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/charger"
	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/hotspot"
	"bfr-webui-go/internal/logger"
	"bfr-webui-go/internal/modules"
	"bfr-webui-go/internal/network"
	"bfr-webui-go/internal/proxy"
	"bfr-webui-go/internal/ssh"
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

type NotificationConfig struct {
	BatteryGuard    bool `json:"battery_guard"`
	BatteryOverheat bool `json:"battery_overheat"`
	SSHStatus       bool `json:"ssh_status"`
	IPChange        bool `json:"ip_change"`
	HotspotClient   bool `json:"hotspot_client"`
}

type Config struct {
	Enabled            bool               `json:"enabled"`
	BotToken           string             `json:"bot_token"`
	AllowedChatIDs     []int64            `json:"allowed_chat_ids"`
	AllowShellCommands bool               `json:"allow_shell_commands"`
	NotifyOnBoot       bool               `json:"notify_on_boot"`
	Notifications      NotificationConfig `json:"notifications"`
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
	m.config.Notifications = cfg.Notifications
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

	go m.startNotificationWorker(ctx)
	m.pollLoop(ctx)
}

func fetchPublicIP() (string, error) {
	client := getHTTPClient(10 * time.Second)
	endpoints := []string{
		"https://api.ipify.org",
		"https://icanhazip.com",
		"https://ifconfig.me/ip",
	}
	for _, url := range endpoints {
		resp, err := client.Get(url)
		if err == nil {
			body, err := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err == nil {
				ip := strings.TrimSpace(string(body))
				if net.ParseIP(ip) != nil {
					return ip, nil
				}
			}
		}
	}
	return "", fmt.Errorf("failed to fetch public IP")
}

func (m *Manager) startNotificationWorker(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	var (
		overheatAlerted      bool
		lastChargingDisabled *bool
		lastSSHRunning       *bool
		lastClientCount      int = -1
		lastPublicIP         string
		ipCheckCounter       int = 0
	)

	if initialIP, err := fetchPublicIP(); err == nil {
		lastPublicIP = initialIP
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.mu.RLock()
			enabled := m.config.Enabled
			notif := m.config.Notifications
			m.mu.RUnlock()

			if !enabled {
				continue
			}

			// 1. Battery Overheat alert (temp > 45°C)
			st, err := sysinfo.GetStats()
			if err == nil {
				if notif.BatteryOverheat {
					if st.BatteryTemp > 45.0 {
						if !overheatAlerted {
							overheatAlerted = true
							m.SendMessageToAllowed(fmt.Sprintf("⚠️ *Battery Overheat Warning!*\nTemperature: %.1f°C (Level: %d%%)", st.BatteryTemp, st.BatteryLevel))
						}
					} else if st.BatteryTemp <= 42.0 {
						overheatAlerted = false
					}
				}
			}

			// 2. Battery Guard alert (charging disabled/enabled state change)
			chStatus := charger.GetManager().GetStatus()
			if lastChargingDisabled == nil {
				val := chStatus.ChargingDisabled
				lastChargingDisabled = &val
			} else if *lastChargingDisabled != chStatus.ChargingDisabled {
				*lastChargingDisabled = chStatus.ChargingDisabled
				if notif.BatteryGuard {
					stateStr := "Enabled"
					if chStatus.ChargingDisabled {
						stateStr = "Disabled / Limited"
					}
					m.SendMessageToAllowed(fmt.Sprintf("⚡ *Battery Guard Alert*\nCharging state changed to: *%s* (Level: %d%%)", stateStr, chStatus.CurrentLevel))
				}
			}

			// 3. SSH Status alert
			sshStatus := ssh.GetManager().GetStatus()
			if lastSSHRunning == nil {
				val := sshStatus.Running
				lastSSHRunning = &val
			} else if *lastSSHRunning != sshStatus.Running {
				*lastSSHRunning = sshStatus.Running
				if notif.SSHStatus {
					stateStr := "Stopped"
					if sshStatus.Running {
						stateStr = fmt.Sprintf("Running (PID: %d, Port: %d)", sshStatus.Pid, sshStatus.Config.Port)
					}
					m.SendMessageToAllowed(fmt.Sprintf("🔑 *SSH Status Alert*\nSSH daemon state changed to: *%s*", stateStr))
				}
			}

			// 4. Hotspot Client alert
			clients, err := hotspot.GetConnectedClients()
			if err == nil {
				currCount := len(clients)
				if lastClientCount == -1 {
					lastClientCount = currCount
				} else if lastClientCount != currCount {
					lastClientCount = currCount
					if notif.HotspotClient {
						m.SendMessageToAllowed(fmt.Sprintf("📶 *Hotspot Client Alert*\nConnected clients count changed to: *%d*", currCount))
					}
				}
			}

			// 5. Public IP Change alert (every 5 mins = 20 * 15s ticks)
			ipCheckCounter++
			if ipCheckCounter >= 20 {
				ipCheckCounter = 0
				if notif.IPChange {
					if currentIP, err := fetchPublicIP(); err == nil && currentIP != "" {
						if lastPublicIP != "" && lastPublicIP != currentIP {
							m.SendMessageToAllowed(fmt.Sprintf("🌐 *Public IP Changed*\nOld IP: `%s`\nNew IP: `%s`", lastPublicIP, currentIP))
						}
						lastPublicIP = currentIP
					}
				}
			}
		}
	}
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

type ReplyKeyboardButton struct {
	Text string `json:"text"`
}

type ReplyKeyboardMarkup struct {
	Keyboard       [][]ReplyKeyboardButton `json:"keyboard"`
	ResizeKeyboard bool                  `json:"resize_keyboard"`
	Persistent     bool                  `json:"is_persistent"`
}

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

func getMainReplyKeyboard() ReplyKeyboardMarkup {
	return ReplyKeyboardMarkup{
		Keyboard: [][]ReplyKeyboardButton{
			{{Text: "📊 Status Sistem"}, {Text: "🔋 Battery Guard"}},
			{{Text: "🔑 Kontrol SSH"}, {Text: "🌐 Status Proxy"}},
			{{Text: "📶 Klien Hotspot"}, {Text: "🧩 Modul Root"}},
			{{Text: "🌐 Cek IP Publik"}, {Text: "🔄 Refresh Menu"}},
		},
		ResizeKeyboard: true,
		Persistent:     true,
	}
}

func buildChargerMessageAndKeyboard() (string, InlineKeyboardMarkup) {
	chStatus := charger.GetManager().GetStatus()
	st, _ := sysinfo.GetStats()

	chState := "Enabled (Charging Active)"
	if chStatus.ChargingDisabled {
		chState = "Disabled / Limited"
	}
	msg := fmt.Sprintf("🔋 *Charger Control & Status*\n\n"+
		"• *Battery Level:* %d%%\n"+
		"• *Status:* %s\n"+
		"• *Temperature:* %.1f°C\n"+
		"• *Limiter Enabled:* %v\n"+
		"• *Start / Stop Limit:* %d%% / %d%%\n"+
		"• *Charging State:* %s\n\n"+
		"💡 _Set limit: `/charger limit 80`_",
		chStatus.CurrentLevel, st.BatteryStatus, st.BatteryTemp,
		chStatus.Config.Enabled, chStatus.Config.StartPercent, chStatus.Config.StopPercent,
		chState,
	)

	kb := InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "🔋 Limit ke 80%", CallbackData: "cb_charger_80"},
				{Text: "⚡ Nonaktifkan Limit", CallbackData: "cb_charger_off"},
			},
			{
				{Text: "🔄 Refresh Status", CallbackData: "cb_charger_refresh"},
			},
		},
	}
	return msg, kb
}

func buildSSHMessageAndKeyboard() (string, InlineKeyboardMarkup) {
	st := ssh.GetManager().GetStatus()
	statusStr := "Stopped 🛑"
	btnText := "▶️ Start SSH"
	if st.Running {
		statusStr = fmt.Sprintf("Running 🟢 (PID: %d)", st.Pid)
		btnText = "🛑 Stop SSH"
	}
	msg := fmt.Sprintf("🔑 *SSH Daemon Status*\n\n"+
		"• *Status:* %s\n"+
		"• *Port:* %d\n"+
		"• *Bind Address:* %s\n"+
		"• *Binary:* `%s`\n\n"+
		"💡 _Commands: `/ssh start` | `/ssh stop`_",
		statusStr, st.Config.Port, st.Config.Bind, st.BinaryPath,
	)

	kb := InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: btnText, CallbackData: "cb_ssh_toggle"},
			},
			{
				{Text: "🔄 Refresh Status", CallbackData: "cb_ssh_refresh"},
			},
		},
	}
	return msg, kb
}

func buildProxyMessageAndKeyboard() (string, InlineKeyboardMarkup) {
	cores := proxy.DetectCores()
	mode := proxy.GetMode()
	watchdog := proxy.GetWatchdog()

	var coreLines []string
	for _, c := range cores {
		statusStr := "Stopped 🛑"
		if c.Running {
			statusStr = fmt.Sprintf("Running 🟢 (PID: %d, Mem: %s)", c.PID, c.Memory)
		}
		coreLines = append(coreLines, fmt.Sprintf("• *%s:* %s", c.Name, statusStr))
	}
	if len(coreLines) == 0 {
		coreLines = append(coreLines, "• No proxy cores detected")
	}

	msg := fmt.Sprintf("🛡️ *Proxy Status*\n\n"+
		"%s\n"+
		"• *Mode:* %s\n"+
		"• *Watchdog:* %v\n\n"+
		"💡 _Restart proxy: `/proxy restart`_",
		strings.Join(coreLines, "\n"), mode, watchdog,
	)

	kb := InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "🔄 Restart Proxy", CallbackData: "cb_proxy_restart"},
			},
		},
	}
	return msg, kb
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

func (m *Manager) sendMessageFull(chatID int64, text string, replyMarkup interface{}) {
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
	if replyMarkup != nil {
		payload["reply_markup"] = replyMarkup
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

func (m *Manager) sendMessage(chatID int64, text string) {
	m.sendMessageFull(chatID, text, nil)
}

func (m *Manager) editMessageText(chatID int64, messageID int64, text string, replyMarkup interface{}) {
	m.mu.RLock()
	token := m.config.BotToken
	m.mu.RUnlock()

	if token == "" || messageID == 0 {
		return
	}

	reqURL := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageText", token)
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
		"text":       text,
		"parse_mode": "Markdown",
	}
	if replyMarkup != nil {
		payload["reply_markup"] = replyMarkup
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
		_ = resp.Body.Close()
	}
}

func (m *Manager) answerCallbackQuery(callbackQueryID string, text string) {
	m.mu.RLock()
	token := m.config.BotToken
	m.mu.RUnlock()

	if token == "" || callbackQueryID == "" {
		return
	}

	reqURL := fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", token)
	payload := map[string]interface{}{
		"callback_query_id": callbackQueryID,
	}
	if text != "" {
		payload["text"] = text
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
				CallbackQuery *struct {
					ID   string `json:"id"`
					From *struct {
						ID int64 `json:"id"`
					} `json:"from"`
					Message *struct {
						MessageID int64 `json:"message_id"`
						Chat      struct {
							ID int64 `json:"id"`
						} `json:"chat"`
						Text string `json:"text"`
					} `json:"message"`
					Data string `json:"data"`
				} `json:"callback_query"`
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
			} else if update.CallbackQuery != nil {
				m.handleCallbackQuery(update.CallbackQuery)
			}
		}
	}
}

func (m *Manager) handleCallbackQuery(cb *struct {
	ID   string `json:"id"`
	From *struct {
		ID int64 `json:"id"`
	} `json:"from"`
	Message *struct {
		MessageID int64 `json:"message_id"`
		Chat      struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		Text string `json:"text"`
	} `json:"message"`
	Data string `json:"data"`
}) {
	if cb == nil || cb.From == nil {
		return
	}
	chatID := cb.From.ID
	if cb.Message != nil && cb.Message.Chat.ID != 0 {
		chatID = cb.Message.Chat.ID
	}

	if !m.isChatAllowed(chatID) {
		m.answerCallbackQuery(cb.ID, "❌ Unauthorized")
		return
	}

	m.answerCallbackQuery(cb.ID, "")

	msgID := int64(0)
	if cb.Message != nil {
		msgID = cb.Message.MessageID
	}

	switch cb.Data {
	case "cb_charger_80":
		cfg := charger.GetManager().GetStatus().Config
		cfg.Enabled = true
		cfg.StopPercent = 80
		if cfg.StartPercent >= cfg.StopPercent {
			cfg.StartPercent = 75
		}
		_ = charger.UpdateConfig(cfg)
		msg, kb := buildChargerMessageAndKeyboard()
		if msgID > 0 {
			m.editMessageText(chatID, msgID, msg, kb)
		} else {
			m.sendMessageFull(chatID, msg, kb)
		}

	case "cb_charger_off":
		cfg := charger.GetManager().GetStatus().Config
		cfg.Enabled = false
		_ = charger.UpdateConfig(cfg)
		msg, kb := buildChargerMessageAndKeyboard()
		if msgID > 0 {
			m.editMessageText(chatID, msgID, msg, kb)
		} else {
			m.sendMessageFull(chatID, msg, kb)
		}

	case "cb_charger_refresh":
		msg, kb := buildChargerMessageAndKeyboard()
		if msgID > 0 {
			m.editMessageText(chatID, msgID, msg, kb)
		} else {
			m.sendMessageFull(chatID, msg, kb)
		}

	case "cb_ssh_toggle":
		st := ssh.GetManager().GetStatus()
		if st.Running {
			_ = ssh.GetManager().Stop()
		} else {
			_ = ssh.GetManager().Start()
		}
		time.Sleep(500 * time.Millisecond)
		msg, kb := buildSSHMessageAndKeyboard()
		if msgID > 0 {
			m.editMessageText(chatID, msgID, msg, kb)
		} else {
			m.sendMessageFull(chatID, msg, kb)
		}

	case "cb_ssh_refresh":
		msg, kb := buildSSHMessageAndKeyboard()
		if msgID > 0 {
			m.editMessageText(chatID, msgID, msg, kb)
		} else {
			m.sendMessageFull(chatID, msg, kb)
		}

	case "cb_proxy_restart":
		if msgID > 0 {
			m.editMessageText(chatID, msgID, "🔄 *Restarting proxy service...*", nil)
		} else {
			m.sendMessage(chatID, "🔄 *Restarting proxy service...*")
		}
		go func() {
			err := proxy.ControlService("restart")
			time.Sleep(1 * time.Second)
			msg, kb := buildProxyMessageAndKeyboard()
			if err != nil {
				msg = fmt.Sprintf("❌ Failed to restart proxy: %v\n\n%s", err, msg)
			}
			m.sendMessageFull(chatID, msg, kb)
		}()

	case "cb_reboot_confirm":
		if msgID > 0 {
			m.editMessageText(chatID, msgID, "🔄 *Rebooting system...*", nil)
		} else {
			m.sendMessage(chatID, "🔄 *Rebooting system...*")
		}
		logger.Get().Warnf("Telegram", "Reboot confirmed from Telegram Chat ID %d", chatID)
		go func() {
			time.Sleep(1 * time.Second)
			_, _ = config.ExecSuTimeout(5*time.Second, "reboot")
		}()

	case "cb_reboot_cancel":
		if msgID > 0 {
			m.editMessageText(chatID, msgID, "❌ Reboot dibatalkan.", nil)
		} else {
			m.sendMessage(chatID, "❌ Reboot dibatalkan.")
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

	switch text {
	case "📊 Status Sistem":
		text = "/stats"
	case "🔋 Battery Guard":
		text = "/charger"
	case "🔑 Kontrol SSH":
		text = "/ssh"
	case "🌐 Status Proxy":
		text = "/proxy"
	case "📶 Klien Hotspot":
		text = "/hotspot"
	case "🧩 Modul Root":
		text = "/modules"
	case "🌐 Cek IP Publik":
		text = "/ip"
	case "🔄 Refresh Menu":
		text = "/start"
	}

	if !strings.HasPrefix(text, "/") {
		return
	}

	parts := strings.Fields(text)
	cmd := parts[0]
	if idx := strings.Index(cmd, "@"); idx != -1 {
		cmd = cmd[:idx]
	}

	switch cmd {
	case "/start", "/help":
		msg := "🤖 *BFR WebUI Telegram Bot*\n\nStatus: Online\n\n*Available Commands:*\n" +
			"/stats - System diagnostics\n" +
			"/charger [limit N] - Battery status & limiter\n" +
			"/ssh [start|stop] - SSH daemon control\n" +
			"/proxy [restart] - Proxy status & restart\n" +
			"/hotspot - Connected hotspot clients\n" +
			"/modules - Active system modules\n" +
			"/ip - Public IP address\n" +
			"/tweak - Tweaks status\n" +
			"/cmd <command> - Execute shell command\n" +
			"/reboot - Reboot device"
		m.sendMessageFull(chatID, msg, getMainReplyKeyboard())

	case "/charger":
		chStatus := charger.GetManager().GetStatus()
		if len(parts) >= 2 {
			valStr := parts[len(parts)-1]
			if limit, err := strconv.Atoi(valStr); err == nil && limit >= 20 && limit <= 100 {
				cfg := chStatus.Config
				cfg.Enabled = true
				cfg.StopPercent = limit
				if cfg.StartPercent >= cfg.StopPercent {
					cfg.StartPercent = cfg.StopPercent - 5
				}
				_ = charger.UpdateConfig(cfg)
			}
		}
		msg, kb := buildChargerMessageAndKeyboard()
		m.sendMessageFull(chatID, msg, kb)

	case "/ssh":
		if len(parts) >= 2 {
			subCmd := strings.ToLower(parts[1])
			if subCmd == "start" {
				_ = ssh.GetManager().Start()
			} else if subCmd == "stop" {
				_ = ssh.GetManager().Stop()
			}
			time.Sleep(500 * time.Millisecond)
		}
		msg, kb := buildSSHMessageAndKeyboard()
		m.sendMessageFull(chatID, msg, kb)

	case "/proxy":
		if len(parts) >= 2 && strings.ToLower(parts[1]) == "restart" {
			m.sendMessage(chatID, "🔄 *Restarting proxy service...*")
			go func() {
				err := proxy.ControlService("restart")
				time.Sleep(1 * time.Second)
				msg, kb := buildProxyMessageAndKeyboard()
				if err != nil {
					msg = fmt.Sprintf("❌ Failed to restart proxy: %v\n\n%s", err, msg)
				}
				m.sendMessageFull(chatID, msg, kb)
			}()
			return
		}
		msg, kb := buildProxyMessageAndKeyboard()
		m.sendMessageFull(chatID, msg, kb)

	case "/hotspot":
		st := hotspot.GetHotspotStatus()
		clients, err := hotspot.GetConnectedClients()

		statusStr := "Disabled 🛑"
		if st.Enabled {
			statusStr = "Enabled 🟢"
		}

		clientLines := []string{}
		if err == nil && len(clients) > 0 {
			for _, c := range clients {
				deviceStr := c.Device
				if deviceStr == "" {
					deviceStr = "Unknown"
				}
				clientLines = append(clientLines, fmt.Sprintf("📱 *%s*\n  IP: `%s` | MAC: `%s`", deviceStr, c.IP, c.MAC))
			}
		} else {
			clientLines = append(clientLines, "_No connected clients_")
		}

		msg := fmt.Sprintf("📶 *Hotspot Status*\n\n"+
			"• *Status:* %s\n"+
			"• *SSID:* `%s`\n"+
			"• *Connected Clients (%d):*\n%s",
			statusStr, st.SSID, len(clients), strings.Join(clientLines, "\n"),
		)
		m.sendMessage(chatID, msg)

	case "/modules":
		mods, err := modules.ListModules()
		if err != nil {
			m.sendMessage(chatID, fmt.Sprintf("❌ Error listing modules: %v", err))
			return
		}

		var lines []string
		activeCount := 0
		for _, mod := range mods {
			statusStr := "Disabled 🔴"
			if mod.Enabled {
				statusStr = "Active 🟢"
				activeCount++
			}
			lines = append(lines, fmt.Sprintf("• *%s* v%s (%s)\n  ID: `%s`", mod.Name, mod.Version, statusStr, mod.ID))
		}

		if len(lines) == 0 {
			lines = append(lines, "_No system modules found_")
		}

		msg := fmt.Sprintf("🧩 *System Modules (%d Active / %d Total)*\n\n%s",
			activeCount, len(mods), strings.Join(lines, "\n"))
		m.sendMessage(chatID, msg)

	case "/ip":
		m.sendMessage(chatID, "🔍 *Fetching public IP...*")
		ip, err := fetchPublicIP()
		if err != nil {
			m.sendMessage(chatID, fmt.Sprintf("❌ Error fetching public IP: %v", err))
		} else {
			m.sendMessage(chatID, fmt.Sprintf("🌐 *Public IP Address*\n\n`%s`", ip))
		}

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
		msg := "⚠️ *Konfirmasi Reboot System*\nApakah Anda yakin ingin melakukan reboot perangkat?"
		kb := InlineKeyboardMarkup{
			InlineKeyboard: [][]InlineKeyboardButton{
				{
					{Text: "🔴 Ya, Reboot HP", CallbackData: "cb_reboot_confirm"},
					{Text: "❌ Batal", CallbackData: "cb_reboot_cancel"},
				},
			},
		}
		m.sendMessageFull(chatID, msg, kb)

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
