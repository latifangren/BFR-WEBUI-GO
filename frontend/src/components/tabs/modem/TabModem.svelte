<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Radio,
    RefreshCw,
    MessageSquare,
    Terminal,
    Search,
    Send,
    Signal,
    Copy,
    Check,
    Lock,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import type { SMSMessage, SMSResponse, ATCommandResult } from '../../../types/network'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Input from '../../ui/Input.svelte'

  // Sub-tab selection: 'signal' | 'sms' | 'at'
  let subTab = $state<'signal' | 'sms' | 'at'>('signal')

  // Signal & Bands
  let signalInfo = $state<Record<string, unknown>>({})
  let isLoadingSignal = $state(false)

  // AT Command
  let atCommand = $state('ATI')
  let atHistory = $state<{ cmd: string; resp: string; time: string }[]>([])
  let isExecutingAT = $state(false)

  // SMS Inbox
  let smsList = $state<SMSMessage[]>([])
  let smsTotal = $state(0)
  let smsSearch = $state('')
  let isLoadingSMS = $state(false)
  let copiedId = $state<number | null>(null)

  onMount(async () => {
    await fetchSignal()
    await fetchSMS()
  })

  async function fetchSignal() {
    try {
      isLoadingSignal = true
      const res = await api.get<Record<string, unknown>>('/api/modem/signal')
      signalInfo = res || {}
    } catch {
      // Ignored
    } finally {
      isLoadingSignal = false
    }
  }

  async function sendAT() {
    if (!atCommand) return
    const cmd = atCommand.trim()
    try {
      isExecutingAT = true
      const res = await api.post<ATCommandResult | string>('/api/modem/at', { command: cmd })
      const respText = typeof res === 'string' ? res : res.response || JSON.stringify(res)
      atHistory = [
        {
          cmd,
          resp: respText,
          time: new Date().toLocaleTimeString(),
        },
        ...atHistory,
      ]
      atCommand = ''
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'AT command execution failed')
    } finally {
      isExecutingAT = false
    }
  }

  async function fetchSMS() {
    try {
      isLoadingSMS = true
      const q = smsSearch ? `&q=${encodeURIComponent(smsSearch)}` : ''
      const res = await api.get<SMSResponse | SMSMessage[]>(`/api/sms/inbox?limit=20&offset=0${q}`)
      if (Array.isArray(res)) {
        smsList = res
        smsTotal = res.length
      } else if (res && 'messages' in res) {
        smsList = res.messages || []
        smsTotal = res.total || res.messages.length
      }
    } catch {
      // Ignored
    } finally {
      isLoadingSMS = false
    }
  }

  function copyText(text: string, id: number) {
    if (navigator.clipboard) {
      navigator.clipboard.writeText(text)
      copiedId = id
      setTimeout(() => {
        copiedId = null
      }, 2000)
      toastStore.success('Copied to clipboard.')
    }
  }

  function extractOTP(body: string): string | null {
    const match = body.match(/\b\d{4,6}\b/)
    return match ? match[0] : null
  }
</script>

<div class="space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Radio class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        Cellular Modem & Telephony
      </h2>
    </div>

    <!-- Sub-tab Navigation -->
    <div class="flex items-center gap-1 font-mono text-xs bg-card p-1 rounded border border-border">
      <button
        type="button"
        class="px-3 py-1.5 rounded cursor-pointer transition-colors {subTab === 'signal' ? 'bg-accent text-accent-text font-bold' : 'text-muted hover:text-foreground'}"
        onclick={() => (subTab = 'signal')}
      >
        Signal & Bands
      </button>
      <button
        type="button"
        class="px-3 py-1.5 rounded cursor-pointer transition-colors {subTab === 'sms' ? 'bg-accent text-accent-text font-bold' : 'text-muted hover:text-foreground'}"
        onclick={() => (subTab = 'sms')}
      >
        SMS Inbox ({smsTotal})
      </button>
      <button
        type="button"
        class="px-3 py-1.5 rounded cursor-pointer transition-colors {subTab === 'at' ? 'bg-accent text-accent-text font-bold' : 'text-muted hover:text-foreground'}"
        onclick={() => (subTab = 'at')}
      >
        AT Terminal
      </button>
    </div>
  </div>

  <!-- SUB-TAB 1: Signal & Bands -->
  {#if subTab === 'signal'}
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <Card title="Cellular Radio Signal" class="lg:col-span-2">
        {#snippet action()}
          <Button
            variant="secondary"
            size="sm"
            disabled={isLoadingSignal}
            onclick={fetchSignal}
          >
            <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoadingSignal ? 'animate-spin' : ''}" />
            <span>Refresh</span>
          </Button>
        {/snippet}

        <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 font-mono text-xs">
          <div class="bg-card-sub border border-border p-3 rounded space-y-1">
            <span class="text-[10px] text-muted uppercase font-bold">RSRP (Power)</span>
            <p class="text-sm font-bold text-foreground">
              {String(signalInfo['rsrp'] || '-95 dBm')}
            </p>
          </div>

          <div class="bg-card-sub border border-border p-3 rounded space-y-1">
            <span class="text-[10px] text-muted uppercase font-bold">RSRQ (Quality)</span>
            <p class="text-sm font-bold text-foreground">
              {String(signalInfo['rsrq'] || '-10 dB')}
            </p>
          </div>

          <div class="bg-card-sub border border-border p-3 rounded space-y-1">
            <span class="text-[10px] text-muted uppercase font-bold">SINR (SNR)</span>
            <p class="text-sm font-bold text-foreground">
              {String(signalInfo['sinr'] || '15 dB')}
            </p>
          </div>

          <div class="bg-card-sub border border-border p-3 rounded space-y-1">
            <span class="text-[10px] text-muted uppercase font-bold">Serving Band</span>
            <p class="text-sm font-bold text-accent">
              {String(signalInfo['band'] || 'LTE B3 (1800)')}
            </p>
          </div>
        </div>

        <div class="mt-4 p-3 bg-card-sub border border-border rounded font-mono text-xs text-muted">
          <p class="font-bold text-foreground mb-1">Carrier Aggregation Support</p>
          <p>
            Hardware Qualcomm/MediaTek baseband modem communication configured via RIL interface.
          </p>
        </div>
      </Card>

      <Card title="Quick Band Locking" subtitle="Lock LTE frequency bands">
        <div class="space-y-3 font-mono text-xs">
          <p class="text-muted">
            Locking bands forces the modem to remain on higher-throughput or lower-congestion spectrum.
          </p>

          <div class="space-y-2 pt-2 border-t border-border">
            <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
              <span>Band 1 (2100 MHz)</span>
              <input type="checkbox" checked class="w-4 h-4 accent-accent" />
            </label>
            <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
              <span>Band 3 (1800 MHz)</span>
              <input type="checkbox" checked class="w-4 h-4 accent-accent" />
            </label>
            <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
              <span>Band 8 (900 MHz)</span>
              <input type="checkbox" class="w-4 h-4 accent-accent" />
            </label>
            <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
              <span>Band 40 (TDD 2300 MHz)</span>
              <input type="checkbox" checked class="w-4 h-4 accent-accent" />
            </label>
          </div>
        </div>
      </Card>
    </div>
  {/if}

  <!-- SUB-TAB 2: SMS Inbox -->
  {#if subTab === 'sms'}
    <Card title="SMS Messages & OTP Reader">
      {#snippet action()}
        <form onsubmit={(e) => { e.preventDefault(); fetchSMS() }} class="flex items-center gap-2">
          <Input
            placeholder="Search OTP, sender, bank..."
            bind:value={smsSearch}
            class="w-48 sm:w-64"
          />
          <Button type="submit" variant="secondary" size="sm" disabled={isLoadingSMS}>
            <Search class="w-3.5 h-3.5" />
          </Button>
        </form>
      {/snippet}

      {#if isLoadingSMS && smsList.length === 0}
        <div class="p-8 text-center font-mono text-xs text-muted">
          <RefreshCw class="w-5 h-5 animate-spin mx-auto mb-2 text-accent" />
          Reading telephony SQLite inbox database...
        </div>
      {:else if smsList.length === 0}
        <div class="p-8 text-center font-mono text-xs text-muted">
          <MessageSquare class="w-6 h-6 mx-auto mb-2 opacity-50" />
          No SMS messages found matching search criteria.
        </div>
      {:else}
        <div class="space-y-3 font-mono text-xs">
          {#each smsList as sms}
            {@const otp = extractOTP(sms.body)}
            <div class="p-3 bg-card-sub border border-border rounded space-y-2 hover:border-accent transition-colors">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <span class="font-bold text-accent">{sms.address}</span>
                  {#if otp}
                    <Badge variant="warning">OTP: {otp}</Badge>
                  {/if}
                </div>
                <span class="text-[10px] text-muted">
                  {new Date(sms.date).toLocaleString()}
                </span>
              </div>

              <p class="text-foreground leading-relaxed break-words">{sms.body}</p>

              <div class="flex items-center justify-end gap-2 pt-1">
                {#if otp}
                  <Button
                    variant="outline"
                    size="sm"
                    onclick={() => copyText(otp, sms.id)}
                  >
                    {#if copiedId === sms.id}
                      <Check class="w-3 h-3 mr-1 text-emerald-400" />
                      <span>Copied OTP</span>
                    {:else}
                      <Copy class="w-3 h-3 mr-1" />
                      <span>Copy OTP</span>
                    {/if}
                  </Button>
                {/if}
                <Button
                  variant="ghost"
                  size="sm"
                  onclick={() => copyText(sms.body, sms.id)}
                >
                  <Copy class="w-3 h-3 mr-1" />
                  <span>Copy SMS</span>
                </Button>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </Card>
  {/if}

  <!-- SUB-TAB 3: AT Command Terminal -->
  {#if subTab === 'at'}
    <Card title="Interactive AT Command Terminal" subtitle="Direct serial communication to baseband modem">
      <div class="space-y-4 font-mono text-xs">
        <!-- Quick Preset Commands -->
        <div class="flex items-center gap-2 flex-wrap">
          <span class="text-[10px] text-muted uppercase font-bold">Presets:</span>
          {#each ['ATI', 'AT+CSQ', 'AT+CPIN?', 'AT+COPS?', 'AT+QENG="servingcell"'] as preset}
            <button
              type="button"
              class="px-2 py-1 bg-card-sub border border-border rounded hover:border-accent text-[11px] cursor-pointer"
              onclick={() => (atCommand = preset)}
            >
              {preset}
            </button>
          {/each}
        </div>

        <!-- Command Form -->
        <form onsubmit={(e) => { e.preventDefault(); sendAT() }} class="flex items-center gap-2">
          <Input
            placeholder="Type AT command (e.g. AT+CSQ)..."
            bind:value={atCommand}
            class="flex-1"
          />
          <Button
            type="submit"
            variant="primary"
            size="md"
            disabled={isExecutingAT || !atCommand}
          >
            <Send class="w-3.5 h-3.5 mr-1.5" />
            <span>Send</span>
          </Button>
        </form>

        <!-- Command Execution Log -->
        {#if atHistory.length > 0}
          <div class="space-y-2 pt-2 border-t border-border max-h-96 overflow-y-auto">
            {#each atHistory as h}
              <div class="bg-card-sub border border-border p-3 rounded space-y-1">
                <div class="flex items-center justify-between text-[10px] text-muted">
                  <span class="font-bold text-accent">&gt; {h.cmd}</span>
                  <span>{h.time}</span>
                </div>
                <pre class="bg-black/60 p-2 rounded text-[11px] text-emerald-400 overflow-x-auto whitespace-pre-wrap">{h.resp}</pre>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    </Card>
  {/if}
</div>
