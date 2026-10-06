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
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/charger"
	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/hotspot"
	"bfr-webui-go/internal/logger"
	"bfr-webui-go/internal/power"
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

func init() {
	power.GetScheduler().SetRebootNotifier(func(msg string) {
		m := GetManager()
		if m != nil && m.IsEnabled() {
			m.BroadcastMessage(msg)
		}
	})
}

func getStoragePath() string {
	return config.GetPersistentFilePath("telegram_config.json")
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

	return config.WriteFileAtomic(m.dataPath, data, 0644)
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
	defer func() {
		if r := recover(); r != nil {
			logger.Get().Errorf("Telegram", "Recovered from startDaemonLoop panic: %v", r)
		}
	}()
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
	defer func() {
		if r := recover(); r != nil {
			logger.Get().Errorf("Telegram", "Recovered from startNotificationWorker panic: %v", r)
		}
	}()
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

func (m *Manager) BroadcastMessage(text string) {
	m.SendMessageToAllowed(text)
}

func BroadcastMessage(text string) {
	if globalManager != nil {
		globalManager.BroadcastMessage(text)
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
	defer func() {
		if r := recover(); r != nil {
			logger.Get().Errorf("Telegram", "Recovered from pollLoop panic: %v", r)
		}
	}()
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
				CallbackQuery *telegramCallbackQuery `json:"callback_query"`
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
