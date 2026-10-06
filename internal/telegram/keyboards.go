package telegram

import (
	"fmt"
	"strings"

	"bfr-webui-go/internal/charger"
	"bfr-webui-go/internal/power"
	"bfr-webui-go/internal/proxy"
	"bfr-webui-go/internal/ssh"
	"bfr-webui-go/internal/sysinfo"
)

func getMainReplyKeyboard() ReplyKeyboardMarkup {
	return ReplyKeyboardMarkup{
		Keyboard: [][]ReplyKeyboardButton{
			{{Text: "📊 Status Sistem"}, {Text: "🔋 Battery Guard"}},
			{{Text: "🔑 Kontrol SSH"}, {Text: "🌐 Status Proxy"}},
			{{Text: "📶 Klien Hotspot"}, {Text: "🧩 Modul Root"}},
			{{Text: "🔄 Auto Reboot"}, {Text: "🌐 Cek IP Publik"}},
			{{Text: "🔄 Refresh Menu"}},
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

func formatDuration(sec int64) string {
	if sec < 60 {
		return fmt.Sprintf("%ds", sec)
	}
	m := sec / 60
	h := m / 60
	m = m % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func buildAutoRebootMessageAndKeyboard() (string, InlineKeyboardMarkup) {
	st := power.GetScheduler().GetStatus()

	enabledStr := "Disabled 🔴"
	if st.Config.Enabled {
		enabledStr = "Enabled 🟢"
	}

	modeStr := "Disabled"
	if st.Config.Mode == power.ScheduleModeInterval {
		modeStr = fmt.Sprintf("Interval (%d Hours)", st.Config.IntervalHours)
	} else if st.Config.Mode == power.ScheduleModeClock {
		modeStr = fmt.Sprintf("Clock Time (%s)", st.Config.ClockTime)
	}

	uptimeMins := st.UptimeSeconds / 60
	uptimeHours := uptimeMins / 60
	uptimeMins = uptimeMins % 60

	nextReboot := "Not scheduled"
	if st.NextRebootInSec >= 0 {
		nextReboot = fmt.Sprintf("%s (in %s)", st.NextRebootTime, formatDuration(st.NextRebootInSec))
	}

	msg := fmt.Sprintf("🔄 *Auto-Reboot Scheduler*\n\n"+
		"• *Status:* %s\n"+
		"• *Mode:* %s\n"+
		"• *Device Uptime:* %dh %dm\n"+
		"• *Next Reboot Schedule:* %s\n"+
		"• *Details:* %s\n\n"+
		"💡 _Select an option below to configure automatic reboot schedule._",
		enabledStr, modeStr, uptimeHours, uptimeMins, nextReboot, st.Description,
	)

	kb := InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "⏱ 2 Hours", CallbackData: "cb_autoreboot_interval_2"},
				{Text: "⏱ 4 Hours", CallbackData: "cb_autoreboot_interval_4"},
				{Text: "⏱ 6 Hours", CallbackData: "cb_autoreboot_interval_6"},
			},
			{
				{Text: "⏱ 8 Hours", CallbackData: "cb_autoreboot_interval_8"},
				{Text: "⏱ 12 Hours", CallbackData: "cb_autoreboot_interval_12"},
				{Text: "⏱ 24 Hours", CallbackData: "cb_autoreboot_interval_24"},
			},
			{
				{Text: "⏰ 03:00", CallbackData: "cb_autoreboot_clock_03:00"},
				{Text: "⏰ 04:00", CallbackData: "cb_autoreboot_clock_04:00"},
				{Text: "⏰ 05:00", CallbackData: "cb_autoreboot_clock_05:00"},
			},
			{
				{Text: "❌ Disable Auto Reboot", CallbackData: "cb_autoreboot_disable"},
				{Text: "🔄 Refresh", CallbackData: "cb_autoreboot_refresh"},
			},
		},
	}
	return msg, kb
}

