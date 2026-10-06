package telegram

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"bfr-webui-go/internal/charger"
	"bfr-webui-go/internal/config"
	"bfr-webui-go/internal/logger"
	"bfr-webui-go/internal/power"
	"bfr-webui-go/internal/proxy"
	"bfr-webui-go/internal/ssh"
)

func (m *Manager) handleCallbackQuery(cb *telegramCallbackQuery) {
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
			m.editMessageText(chatID, msgID, "❌ Reboot cancelled.", nil)
		} else {
			m.sendMessage(chatID, "❌ Reboot cancelled.")
		}

	case "cb_autoreboot_disable":
		_ = power.GetScheduler().UpdateConfig(power.ScheduleConfig{
			Enabled: false,
			Mode:    power.ScheduleModeDisabled,
		})
		msg, kb := buildAutoRebootMessageAndKeyboard()
		if msgID > 0 {
			m.editMessageText(chatID, msgID, msg, kb)
		} else {
			m.sendMessageFull(chatID, msg, kb)
		}

	case "cb_autoreboot_refresh":
		msg, kb := buildAutoRebootMessageAndKeyboard()
		if msgID > 0 {
			m.editMessageText(chatID, msgID, msg, kb)
		} else {
			m.sendMessageFull(chatID, msg, kb)
		}

	default:
		if strings.HasPrefix(cb.Data, "cb_autoreboot_interval_") {
			hoursStr := strings.TrimPrefix(cb.Data, "cb_autoreboot_interval_")
			if hours, err := strconv.Atoi(hoursStr); err == nil && hours >= 1 {
				_ = power.GetScheduler().UpdateConfig(power.ScheduleConfig{
					Enabled:       true,
					Mode:          power.ScheduleModeInterval,
					IntervalHours: hours,
				})
			}
			msg, kb := buildAutoRebootMessageAndKeyboard()
			if msgID > 0 {
				m.editMessageText(chatID, msgID, msg, kb)
			} else {
				m.sendMessageFull(chatID, msg, kb)
			}
		} else if strings.HasPrefix(cb.Data, "cb_autoreboot_clock_") {
			clockStr := strings.TrimPrefix(cb.Data, "cb_autoreboot_clock_")
			_ = power.GetScheduler().UpdateConfig(power.ScheduleConfig{
				Enabled:   true,
				Mode:      power.ScheduleModeClock,
				ClockTime: clockStr,
			})
			msg, kb := buildAutoRebootMessageAndKeyboard()
			if msgID > 0 {
				m.editMessageText(chatID, msgID, msg, kb)
			} else {
				m.sendMessageFull(chatID, msg, kb)
			}
		}
	}
}
