const TelegramModule = {
    telegramData: {
        config: {
            enabled: false,
            bot_token: "",
            allowed_chat_ids: [],
            allow_shell_commands: false,
            notify_on_boot: false,
            notifications: {
                battery_guard: false,
                battery_overheat: false,
                ssh_status: false,
                ip_change: false,
                hotspot_client: false
            }
        },
        running: false,
        bot_name: "",
        chat_ids_str: ""
    },

    updateNotificationsFromConfig(cfg) {
        const notifs = cfg.notifications || cfg.Notifications;
        if (notifs) {
            if (!this.telegramData.config.notifications) {
                this.telegramData.config.notifications = {};
            }
            this.telegramData.config.notifications.battery_guard = !!(notifs.battery_guard || notifs.BatteryGuard);
            this.telegramData.config.notifications.battery_overheat = !!(notifs.battery_overheat || notifs.BatteryOverheat);
            this.telegramData.config.notifications.ssh_status = !!(notifs.ssh_status || notifs.SSHStatus);
            this.telegramData.config.notifications.ip_change = !!(notifs.ip_change || notifs.IPChange);
            this.telegramData.config.notifications.hotspot_client = !!(notifs.hotspot_client || notifs.HotspotClient);
        }
    },

    async fetchTelegramStatus() {
        try {
            const res = await fetch('/api/telegram/status');
            if (res.ok) {
                const data = await res.json();
                const cfg = data.config || data.Config || {};
                
                this.telegramData.running = !!(data.running || data.Running);
                this.telegramData.bot_name = data.bot_name || data.BotName || '';
                
                this.telegramData.config.enabled = !!cfg.enabled;
                if (cfg.bot_token) {
                    this.telegramData.config.bot_token = cfg.bot_token;
                }

                if (cfg.allow_shell_commands !== undefined) {
                    this.telegramData.config.allow_shell_commands = !!cfg.allow_shell_commands;
                }
                if (cfg.notify_on_boot !== undefined) {
                    this.telegramData.config.notify_on_boot = !!cfg.notify_on_boot;
                }

                this.updateNotificationsFromConfig(cfg);

                const chatIds = cfg.allowed_chat_ids || cfg.AllowedChatIDs;
                if (Array.isArray(chatIds) && chatIds.length > 0) {
                    this.telegramData.config.allowed_chat_ids = chatIds;
                    if (!this.telegramData.chat_ids_str) {
                        this.telegramData.chat_ids_str = chatIds.join(', ');
                    }
                }
            }
        } catch (e) {
            console.error('Failed to fetch Telegram status:', e);
        }
    },

    async saveTelegramConfig() {
        try {
            const token = (this.telegramData.config.bot_token || '').trim();
            if (!token) {
                this.showToast('Telegram Error', 'Please enter your Telegram Bot Token before saving.', 'error');
                return false;
            }

            const rawStr = (this.telegramData.chat_ids_str || '').trim();
            let parsedIds = [];
            if (rawStr) {
                parsedIds = rawStr
                    .split(',')
                    .map(s => parseInt(s.trim(), 10))
                    .filter(n => !isNaN(n));
            }

            if (!this.telegramData.config) {
                this.telegramData.config = {};
            }
            this.telegramData.config.allowed_chat_ids = parsedIds;

            const res = await fetch('/api/telegram/config', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(this.telegramData.config)
            });

            if (res.ok) {
                const data = await res.json();
                const cfg = data.config || data.Config || {};
                
                this.telegramData.running = !!(data.running || data.Running);
                this.telegramData.bot_name = data.bot_name || data.BotName || '';
                this.telegramData.config.enabled = !!cfg.enabled;
                if (cfg.bot_token) {
                    this.telegramData.config.bot_token = cfg.bot_token;
                }
                this.telegramData.config.allow_shell_commands = !!cfg.allow_shell_commands;
                this.telegramData.config.notify_on_boot = !!cfg.notify_on_boot;

                this.updateNotificationsFromConfig(cfg);

                const savedChatIds = cfg.allowed_chat_ids || cfg.AllowedChatIDs;
                if (Array.isArray(savedChatIds) && savedChatIds.length > 0) {
                    this.telegramData.config.allowed_chat_ids = savedChatIds;
                    this.telegramData.chat_ids_str = savedChatIds.join(', ');
                }

                this.showToast('Telegram Config', 'Telegram bot settings saved successfully!', 'success');
                return true;
            } else {
                const errData = await res.json().catch(() => ({}));
                this.showToast('Telegram Error', errData.error || 'Failed to save settings.', 'error');
                return false;
            }
        } catch (e) {
            this.showToast('Telegram Error', 'Request failed.', 'error');
            return false;
        }
    },

    async controlTelegram(action) {
        try {
            if (action === 'start') {
                const saved = await this.saveTelegramConfig();
                if (!saved) return;
            }

            const res = await fetch('/api/telegram/control', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ action: action })
            });
            if (res.ok) {
                const data = await res.json();
                const cfg = data.config || data.Config || {};
                
                this.telegramData.running = !!(data.running || data.Running);
                this.telegramData.bot_name = data.bot_name || data.BotName || '';
                this.telegramData.config.enabled = !!cfg.enabled;
                if (cfg.bot_token) {
                    this.telegramData.config.bot_token = cfg.bot_token;
                }
                this.telegramData.config.allow_shell_commands = !!cfg.allow_shell_commands;
                this.telegramData.config.notify_on_boot = !!cfg.notify_on_boot;

                this.updateNotificationsFromConfig(cfg);

                const savedChatIds = cfg.allowed_chat_ids || cfg.AllowedChatIDs;
                if (Array.isArray(savedChatIds) && savedChatIds.length > 0) {
                    this.telegramData.config.allowed_chat_ids = savedChatIds;
                    this.telegramData.chat_ids_str = savedChatIds.join(', ');
                }
                this.showToast('Telegram Service', `Telegram bot ${action} successful!`, 'success');
            } else {
                const errData = await res.json().catch(() => ({}));
                this.showToast('Telegram Error', errData.error || `Action ${action} failed.`, 'error');
            }
        } catch (e) {
            this.showToast('Telegram Error', 'Connection failed.', 'error');
        }
    }
};