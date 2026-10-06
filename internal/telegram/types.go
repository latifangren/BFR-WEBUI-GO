package telegram

type NotificationConfig struct {
	BatteryGuard    bool `json:"battery_guard"`
	BatteryOverheat bool `json:"battery_overheat"`
	SSHStatus       bool `json:"ssh_status"`
	IPChange        bool `json:"ip_change"`
	HotspotClient   bool `json:"hotspot_client"`
	ForwardSMS      bool `json:"forward_sms"`
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

type ReplyKeyboardButton struct {
	Text string `json:"text"`
}

type ReplyKeyboardMarkup struct {
	Keyboard       [][]ReplyKeyboardButton `json:"keyboard"`
	ResizeKeyboard bool                    `json:"resize_keyboard"`
	Persistent     bool                    `json:"is_persistent"`
}

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type telegramCallbackQuery struct {
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
}
