<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Send,
    RefreshCw,
    Play,
    Square,
    Check,
    Bell,
    MessageCircle,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Input from '../../ui/Input.svelte'

  interface TelegramConfig {
    enabled: boolean
    bot_token: string
    chat_id: string
    notify_on_boot?: boolean
    notify_on_login?: boolean
    notify_on_ip_change?: boolean
  }

  interface TelegramStatus {
    running: boolean
    enabled: boolean
    has_token?: boolean
    has_chat_id?: boolean
  }

  let config = $state<TelegramConfig>({
    enabled: false,
    bot_token: '',
    chat_id: '',
    notify_on_boot: true,
    notify_on_login: true,
    notify_on_ip_change: false,
  })
  let status = $state<TelegramStatus>({ running: false, enabled: false })
  let isLoading = $state(false)
  let isSaving = $state(false)
  let isTesting = $state(false)

  onMount(async () => {
    await fetchTelegram()
  })

  async function fetchTelegram() {
    try {
      isLoading = true
      const s = await api.get<TelegramStatus>('/api/telegram/status')
      if (s) status = s

      const c = await api.get<TelegramConfig>('/api/telegram/config')
      if (c) config = { ...config, ...c }
    } catch {
      // Ignored
    } finally {
      isLoading = false
    }
  }

  async function saveConfig() {
    try {
      isSaving = true
      await api.post('/api/telegram/config', config)
      toastStore.success('Telegram bot configuration saved.')
      await fetchTelegram()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to save config')
    } finally {
      isSaving = false
    }
  }

  async function sendTestMessage() {
    try {
      isTesting = true
      await api.post('/api/telegram/control', { action: 'test' })
      toastStore.success('Test message sent to Telegram chat.')
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to send test message')
    } finally {
      isTesting = false
    }
  }
</script>

<div class="space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Send class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        Telegram Bot Notifications
      </h2>
      <Badge variant={status.running ? 'success' : 'default'}>
        {status.running ? 'Bot Active' : 'Disabled'}
      </Badge>
    </div>

    <div class="flex items-center gap-2">
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoading}
        onclick={fetchTelegram}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoading ? 'animate-spin' : ''}" />
        <span>Refresh</span>
      </Button>

      <Button
        variant="outline"
        size="sm"
        disabled={isTesting || !config.bot_token || !config.chat_id}
        onclick={sendTestMessage}
      >
        <MessageCircle class="w-3.5 h-3.5 mr-1 text-accent" />
        <span>Test Alert</span>
      </Button>
    </div>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <!-- Configuration Card -->
    <Card
      title="Bot Credentials & Events"
      subtitle="Receive real-time alerts when Android system events occur"
      class="lg:col-span-2"
    >
      {#snippet action()}
        <Button
          variant="primary"
          size="sm"
          disabled={isSaving}
          onclick={saveConfig}
        >
          <Check class="w-3.5 h-3.5 mr-1" />
          <span>Save Config</span>
        </Button>
      {/snippet}

      <div class="space-y-4 font-mono text-xs">
        <label class="flex items-center justify-between p-3 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
          <div>
            <span class="font-bold text-foreground block">Enable Telegram Bot Service</span>
            <span class="text-[10px] text-muted">Activate background alert dispatcher</span>
          </div>
          <input
            type="checkbox"
            bind:checked={config.enabled}
            class="w-4 h-4 accent-accent"
          />
        </label>

        <div class="space-y-3 pt-2">
          <Input
            type="password"
            label="Telegram Bot API Token (from @BotFather)"
            placeholder="123456789:ABCdefGhIJKlmNoPQRstuVWXyz..."
            bind:value={config.bot_token}
          />

          <Input
            label="Target Chat ID / Channel ID"
            placeholder="987654321 or -100123456789"
            bind:value={config.chat_id}
          />
        </div>

        <div class="space-y-2 pt-2 border-t border-border">
          <span class="text-[10px] text-muted uppercase font-bold block">
            Automated Alert Event Triggers
          </span>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
            <label class="flex items-center justify-between p-2.5 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
              <div class="flex items-center gap-2">
                <Bell class="w-4 h-4 text-emerald-400" />
                <span>Notify On System Boot</span>
              </div>
              <input
                type="checkbox"
                bind:checked={config.notify_on_boot}
                class="w-4 h-4 accent-accent"
              />
            </label>

            <label class="flex items-center justify-between p-2.5 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
              <div class="flex items-center gap-2">
                <Bell class="w-4 h-4 text-amber-400" />
                <span>Notify On Admin Login</span>
              </div>
              <input
                type="checkbox"
                bind:checked={config.notify_on_login}
                class="w-4 h-4 accent-accent"
              />
            </label>
          </div>
        </div>
      </div>
    </Card>

    <!-- Help Card -->
    <Card title="Quick Setup Guide" subtitle="How to create a Telegram bot">
      <div class="space-y-3 font-mono text-xs text-muted leading-relaxed">
        <ol class="list-decimal list-inside space-y-2">
          <li>Open Telegram and message <strong class="text-foreground">@BotFather</strong>.</li>
          <li>Send command <code class="text-accent font-bold">/newbot</code> and copy the HTTP API token.</li>
          <li>Message <strong class="text-foreground">@userinfobot</strong> to obtain your personal numeric Chat ID.</li>
          <li>Paste the token and Chat ID, then click <strong class="text-foreground">Test Alert</strong>.</li>
        </ol>
      </div>
    </Card>
  </div>
</div>
