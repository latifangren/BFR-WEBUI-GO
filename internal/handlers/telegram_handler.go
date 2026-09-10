package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"bfr-webui-go/internal/logger"
	"bfr-webui-go/internal/telegram"
)

func HandleTelegramStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	manager := telegram.GetManager()
	st := manager.GetStatus()

	resp := map[string]interface{}{
		"config":      st.Config,
		"running":     st.Running,
		"bot_name":    st.BotName,
		"enabled":     st.Config.Enabled,
		"has_token":   st.Config.BotToken != "",
		"has_chat_id": len(st.Config.AllowedChatIDs) > 0,
	}
	if len(st.Config.AllowedChatIDs) > 0 {
		resp["chat_id"] = fmt.Sprintf("%d", st.Config.AllowedChatIDs[0])
	} else {
		resp["chat_id"] = ""
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func HandleTelegramConfig(w http.ResponseWriter, r *http.Request) {
	manager := telegram.GetManager()

	if r.Method == http.MethodGet {
		st := manager.GetStatus()
		resp := map[string]interface{}{
			"config":           st.Config,
			"running":          st.Running,
			"bot_name":         st.BotName,
			"enabled":          st.Config.Enabled,
			"bot_token":        st.Config.BotToken,
			"allowed_chat_ids": st.Config.AllowedChatIDs,
			"notify_on_boot":   st.Config.NotifyOnBoot,
			"notifications":    st.Config.Notifications,
		}
		if len(st.Config.AllowedChatIDs) > 0 {
			resp["chat_id"] = fmt.Sprintf("%d", st.Config.AllowedChatIDs[0])
		} else {
			resp["chat_id"] = ""
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Failed to read request body"})
		return
	}
	defer r.Body.Close()

	var input struct {
		Enabled            *bool                        `json:"enabled"`
		BotToken           *string                      `json:"bot_token"`
		ChatID             interface{}                  `json:"chat_id"`
		AllowedChatIDs     []int64                      `json:"allowed_chat_ids"`
		AllowShellCommands *bool                        `json:"allow_shell_commands"`
		NotifyOnBoot       *bool                        `json:"notify_on_boot"`
		NotifyOnIPChange   *bool                        `json:"notify_on_ip_change"`
		Notifications      *telegram.NotificationConfig `json:"notifications"`
		Config             *telegram.Config             `json:"config"`
	}

	if err := json.Unmarshal(body, &input); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Invalid request payload"})
		return
	}

	st := manager.GetStatus()
	cfg := st.Config

	if input.Config != nil {
		cfg = *input.Config
	} else {
		_ = json.Unmarshal(body, &cfg)
	}

	if input.Enabled != nil {
		cfg.Enabled = *input.Enabled
	}
	if input.BotToken != nil {
		cfg.BotToken = *input.BotToken
	}
	if input.AllowShellCommands != nil {
		cfg.AllowShellCommands = *input.AllowShellCommands
	}
	if input.NotifyOnBoot != nil {
		cfg.NotifyOnBoot = *input.NotifyOnBoot
	}
	if input.Notifications != nil {
		cfg.Notifications = *input.Notifications
	}
	if input.NotifyOnIPChange != nil {
		cfg.Notifications.IPChange = *input.NotifyOnIPChange
	}

	if len(input.AllowedChatIDs) > 0 {
		cfg.AllowedChatIDs = input.AllowedChatIDs
	} else if input.ChatID != nil {
		switch v := input.ChatID.(type) {
		case float64:
			cfg.AllowedChatIDs = []int64{int64(v)}
		case int64:
			cfg.AllowedChatIDs = []int64{v}
		case int:
			cfg.AllowedChatIDs = []int64{int64(v)}
		case string:
			clean := strings.TrimSpace(v)
			if clean != "" {
				if parsed, parseErr := strconv.ParseInt(clean, 10, 64); parseErr == nil {
					cfg.AllowedChatIDs = []int64{parsed}
				}
			} else {
				cfg.AllowedChatIDs = []int64{}
			}
		}
	}

	status, err := manager.SaveConfig(cfg)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	logger.Get().Infof("Telegram", "Telegram configuration saved (Enabled: %v)", cfg.Enabled)

	resp := map[string]interface{}{
		"success":          true,
		"error":            nil,
		"config":           status.Config,
		"running":          status.Running,
		"bot_name":         status.BotName,
		"enabled":          status.Config.Enabled,
		"bot_token":        status.Config.BotToken,
		"allowed_chat_ids": status.Config.AllowedChatIDs,
		"notify_on_boot":   status.Config.NotifyOnBoot,
		"notifications":    status.Config.Notifications,
	}
	if len(status.Config.AllowedChatIDs) > 0 {
		resp["chat_id"] = fmt.Sprintf("%d", status.Config.AllowedChatIDs[0])
	} else {
		resp["chat_id"] = ""
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func HandleTelegramControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Failed to read request body"})
		return
	}
	defer r.Body.Close()

	var req struct {
		Action string `json:"action"`
	}

	if err := json.Unmarshal(body, &req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request payload"})
		return
	}

	manager := telegram.GetManager()
	logger.Get().Infof("Telegram", "Telegram service action requested: %s", req.Action)

	var errAction error
	switch req.Action {
	case "start":
		errAction = manager.Start()
	case "stop":
		errAction = manager.Stop()
	case "restart":
		_ = manager.Stop()
		errAction = manager.Start()
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Unknown control action"})
		return
	}

	if errAction != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": errAction.Error()})
		return
	}

	status := manager.GetStatus()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}
