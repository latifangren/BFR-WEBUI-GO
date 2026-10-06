package telegram

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"bfr-webui-go/internal/charger"
	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/hotspot"
	"bfr-webui-go/internal/logger"
	"bfr-webui-go/internal/modules"
	"bfr-webui-go/internal/proxy"
	"bfr-webui-go/internal/smsviewer"
	"bfr-webui-go/internal/ssh"
	"bfr-webui-go/internal/sysinfo"
)

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
	case "🔄 Auto Reboot":
		text = "/autoreboot"
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
			"/sms [count] - Read recent SMS inbox messages\n" +
			"/charger [limit N] - Battery status & limiter\n" +
			"/ssh [start|stop] - SSH daemon control\n" +
			"/proxy [restart] - Proxy status & restart\n" +
			"/hotspot - Connected hotspot clients\n" +
			"/modules - Active system modules\n" +
			"/ip - Public IP address\n" +
			"/autoreboot - Auto-reboot scheduler settings\n" +
			"/cmd <command> - Execute shell command\n" +
			"/reboot - Reboot device"
		m.sendMessageFull(chatID, msg, getMainReplyKeyboard())

	case "/autoreboot", "/reboot_schedule":
		msg, kb := buildAutoRebootMessageAndKeyboard()
		m.sendMessageFull(chatID, msg, kb)

	case "/sms":
		count := 5
		if len(parts) >= 2 {
			if n, err := strconv.Atoi(parts[1]); err == nil && n > 0 && n <= 20 {
				count = n
			}
		}
		resp, err := smsviewer.ReadSMSInbox(count, 0, "")
		if err != nil || len(resp.Messages) == 0 {
			m.sendMessage(chatID, "📥 *SMS Inbox*: No messages found or failed to read SMS database.")
			return
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("📥 *Recent SMS Inbox (%d)*\n\n", len(resp.Messages)))
		for i, sms := range resp.Messages {
			tStr := "Unknown"
			if sms.Date > 0 {
				tStr = time.Unix(sms.Date, 0).Format("2006-01-02 15:04:05")
			}
			sb.WriteString(fmt.Sprintf("*%d. Sender:* `%s`\n🕒 *Date:* %s\n💬 *Body:* %s\n\n", i+1, sms.Address, tStr, sms.Body))
		}
		m.sendMessage(chatID, sb.String())

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
		msg := "⚠️ *System Reboot Confirmation*\nAre you sure you want to reboot the device?"
		kb := InlineKeyboardMarkup{
			InlineKeyboard: [][]InlineKeyboardButton{
				{
					{Text: "🔴 Yes, Reboot Device", CallbackData: "cb_reboot_confirm"},
					{Text: "❌ Cancel", CallbackData: "cb_reboot_cancel"},
				},
			},
		}
		m.sendMessageFull(chatID, msg, kb)

	case "/tweak":
		m.sendMessage(chatID, "ℹ️ *Dynamic Optimizations & tweaks.json* have been removed from the system.")
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
