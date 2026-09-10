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
    RotateCcw,
    Layers,
    AlertTriangle,
    Zap,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import type { SMSMessage, SMSResponse, ATCommandResult } from '../../../types/network'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Input from '../../ui/Input.svelte'
  import Modal from '../../ui/Modal.svelte'

  interface BandConfig {
    engine: 'universal' | 'qualcomm_at' | 'intent'
    preferred_rat: 'hybrid' | '5g_only' | '4g_only' | '3g_only'
    lte_bands: number[]
    nr_bands: number[]
    hex_bitmask?: string
  }

  // Sub-tab selection: 'signal' | 'bands' | 'sms' | 'at'
  let subTab = $state<'signal' | 'bands' | 'sms' | 'at'>('signal')

  // Signal & Info
  let signalInfo = $state<Record<string, unknown>>({})
  let isLoadingSignal = $state(false)

  // Band Locking
  let bandConfig = $state<BandConfig>({
    engine: 'universal',
    preferred_rat: 'hybrid',
    lte_bands: [1, 3, 8, 40],
    nr_bands: [1, 3, 40],
    hex_bitmask: '',
  })
  let isLoadingBands = $state(false)
  let isApplyingBands = $state(false)
  let isResettingModem = $state(false)
  let resetModalOpen = $state(false)

  // Pre-configured band definitions
  const availableLTEBands = [
    { band: 1, label: 'B1', name: '2100 MHz FDD', default: true },
    { band: 3, label: 'B3', name: '1800 MHz FDD', default: true },
    { band: 5, label: 'B5', name: '850 MHz FDD', default: false },
    { band: 7, label: 'B7', name: '2600 MHz FDD', default: false },
    { band: 8, label: 'B8', name: '900 MHz FDD', default: true },
    { band: 20, label: 'B20', name: '800 MHz EU', default: false },
    { band: 28, label: 'B28', name: '700 MHz APT', default: false },
    { band: 38, label: 'B38', name: '2600 MHz TDD', default: false },
    { band: 40, label: 'B40', name: '2300 MHz TDD', default: true },
    { band: 41, label: 'B41', name: '2500 MHz TDD', default: false },
  ]

  const availableNRBands = [
    { band: 1, label: 'n1', name: '2100 MHz Sub-6' },
    { band: 3, label: 'n3', name: '1800 MHz Sub-6' },
    { band: 28, label: 'n28', name: '700 MHz Sub-6' },
    { band: 40, label: 'n40', name: '2300 MHz Sub-6' },
    { band: 77, label: 'n77', name: '3700 MHz C-Band' },
    { band: 78, label: 'n78', name: '3500 MHz Mid-Band' },
  ]

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
    await fetchBands()
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

  async function fetchBands() {
    try {
      isLoadingBands = true
      const res = await api.get<BandConfig>('/api/modem/bands')
      if (res) {
        bandConfig = {
          engine: res.engine || 'universal',
          preferred_rat: res.preferred_rat || 'hybrid',
          lte_bands: Array.isArray(res.lte_bands) ? res.lte_bands : [1, 3, 8, 40],
          nr_bands: Array.isArray(res.nr_bands) ? res.nr_bands : [1, 3, 40],
          hex_bitmask: res.hex_bitmask || '',
        }
      }
    } catch {
      // Keep defaults
    } finally {
      isLoadingBands = false
    }
  }

  async function applyBandLock() {
    try {
      isApplyingBands = true
      const res = await api.post<{ success: boolean; error?: string }>('/api/modem/bands', {
        engine: bandConfig.engine,
        preferred_rat: bandConfig.preferred_rat,
        lte_bands: bandConfig.lte_bands,
        nr_bands: bandConfig.nr_bands,
        hex_bitmask: bandConfig.hex_bitmask || '',
      })
      if (res && res.success) {
        toastStore.success('Band lock configuration applied. Radio cycle triggered.')
        await fetchSignal()
      } else {
        toastStore.error(res?.error || 'Failed to apply band locking')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Band lock request failed')
    } finally {
      isApplyingBands = false
    }
  }

  async function resetModemSettings() {
    try {
      isResettingModem = true
      const res = await api.post<{ success: boolean; error?: string }>('/api/modem/reset')
      if (res && res.success) {
        toastStore.success('Modem radio and band settings reset to factory defaults.')
        resetModalOpen = false
        await fetchBands()
        await fetchSignal()
      } else {
        toastStore.error(res?.error || 'Failed to reset modem settings')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Modem reset failed')
    } finally {
      isResettingModem = false
    }
  }

  function toggleLTEBand(b: number) {
    if (bandConfig.lte_bands.includes(b)) {
      bandConfig.lte_bands = bandConfig.lte_bands.filter((x) => x !== b)
    } else {
      bandConfig.lte_bands = [...bandConfig.lte_bands, b].sort((a, b) => a - b)
    }
  }

  function toggleNRBand(b: number) {
    if (bandConfig.nr_bands.includes(b)) {
      bandConfig.nr_bands = bandConfig.nr_bands.filter((x) => x !== b)
    } else {
      bandConfig.nr_bands = [...bandConfig.nr_bands, b].sort((a, b) => a - b)
    }
  }

  function selectAllLTE() {
    bandConfig.lte_bands = availableLTEBands.map((b) => b.band)
  }

  function clearAllLTE() {
    bandConfig.lte_bands = []
  }

  async function sendAT() {
    if (!atCommand) return
    const cmd = atCommand.trim()
    try {
      isExecutingAT = true
      const res = await api.post<ATCommandResult | string>('/api/modem/at', { command: cmd })
      const text = typeof res === 'string' ? res : res.response || ''
      atHistory = [
        {
          cmd,
          resp: text,
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
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
    <div class="flex items-center gap-2">
      <Radio class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        Cellular Modem & Telephony
      </h2>
    </div>

    <div class="flex flex-wrap items-center gap-2">
      <!-- Reset Modem Button with confirmation -->
      <Button
        variant="danger"
        size="sm"
        onclick={() => (resetModalOpen = true)}
      >
        <RotateCcw class="w-3.5 h-3.5 mr-1" />
        <span>Reset Modem</span>
      </Button>

      <!-- Sub-tab Navigation -->
      <div class="flex items-center gap-1 font-mono text-xs bg-card p-1 rounded border border-border">
        <button
          type="button"
          class="px-2.5 py-1.5 rounded cursor-pointer transition-colors {subTab === 'signal' ? 'bg-accent text-accent-text font-bold' : 'text-muted hover:text-foreground'}"
          onclick={() => (subTab = 'signal')}
        >
          Signal
        </button>
        <button
          type="button"
          class="px-2.5 py-1.5 rounded cursor-pointer transition-colors {subTab === 'bands' ? 'bg-accent text-accent-text font-bold' : 'text-muted hover:text-foreground'}"
          onclick={() => (subTab = 'bands')}
        >
          Bands
        </button>
        <button
          type="button"
          class="px-2.5 py-1.5 rounded cursor-pointer transition-colors {subTab === 'sms' ? 'bg-accent text-accent-text font-bold' : 'text-muted hover:text-foreground'}"
          onclick={() => (subTab = 'sms')}
        >
          SMS ({smsTotal})
        </button>
        <button
          type="button"
          class="px-2.5 py-1.5 rounded cursor-pointer transition-colors {subTab === 'at' ? 'bg-accent text-accent-text font-bold' : 'text-muted hover:text-foreground'}"
          onclick={() => (subTab = 'at')}
        >
          AT
        </button>
      </div>
    </div>
  </div>

  <!-- SUB-TAB 1: Signal & Radio -->
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
            <span class="text-[10px] text-muted uppercase font-bold">Network Mode</span>
            <p class="text-sm font-bold text-emerald-400">
              {String(signalInfo['network_type'] || '4G LTE-A')}
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

      <Card title="Cell Identity & Info" subtitle="Radio mast details">
        <div class="space-y-3 font-mono text-xs">
          <div class="flex items-center justify-between p-2 rounded bg-card-sub border border-border">
            <span class="text-muted">Operator</span>
            <span class="font-bold text-foreground">{String(signalInfo['operator'] || 'Telkomsel')}</span>
          </div>
          <div class="flex items-center justify-between p-2 rounded bg-card-sub border border-border">
            <span class="text-muted">Active Band</span>
            <span class="font-bold text-accent">{String(signalInfo['band'] || 'B3 (1800 MHz)')}</span>
          </div>
          <div class="flex items-center justify-between p-2 rounded bg-card-sub border border-border">
            <span class="text-muted">Cell ID</span>
            <span class="font-bold text-foreground">{String(signalInfo['cell_id'] || '1348123')}</span>
          </div>
          <div class="flex items-center justify-between p-2 rounded bg-card-sub border border-border">
            <span class="text-muted">TAC / PCI</span>
            <span class="font-bold text-foreground">{String(signalInfo['tac'] || '4201')} / {String(signalInfo['pci'] || '189')}</span>
          </div>
        </div>
      </Card>
    </div>
  {/if}

  <!-- SUB-TAB 2: Band Locking (NEW) -->
  {#if subTab === 'bands'}
    <div class="space-y-6">
      <Card title="Band Locking & Radio Technology (RAT)" subtitle="Lock cellular frequency bands to boost throughput and stabilize ping">
        {#snippet action()}
          <div class="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={isLoadingBands}
              onclick={fetchBands}
            >
              <RefreshCw class="w-3 h-3 mr-1.5 {isLoadingBands ? 'animate-spin' : ''}" />
              <span>Reload</span>
            </Button>
            <Button
              variant="primary"
              size="sm"
              disabled={isApplyingBands}
              onclick={applyBandLock}
            >
              <Zap class="w-3.5 h-3.5 mr-1 text-amber-400" />
              <span>{isApplyingBands ? 'Applying...' : 'Apply Band Lock'}</span>
            </Button>
          </div>
        {/snippet}

        <div class="space-y-6 font-mono text-xs">
          <!-- Engine & RAT Selector -->
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <!-- Engine Selection -->
            <div class="space-y-2 p-3 bg-card-sub border border-border rounded">
              <span class="text-[10px] text-muted uppercase font-bold block">Modem Control Engine</span>
              <div class="grid grid-cols-3 gap-2">
                {#each [
                  { id: 'universal', label: 'Universal', desc: 'Sys Settings' },
                  { id: 'qualcomm_at', label: 'Qualcomm AT', desc: 'Direct AT' },
                  { id: 'intent', label: 'Intent Mode', desc: 'RIL Broadcast' }
                ] as eng}
                  <button
                    type="button"
                    class="neo-button p-2 text-center rounded border transition-all cursor-pointer {bandConfig.engine === eng.id ? 'border-accent bg-accent/15 text-accent font-bold shadow-neobrutal-sm' : 'border-border bg-card text-muted hover:text-foreground'}"
                    onclick={() => (bandConfig.engine = eng.id as BandConfig['engine'])}
                  >
                    <div class="font-bold text-xs">{eng.label}</div>
                    <div class="text-[9px] text-muted">{eng.desc}</div>
                  </button>
                {/each}
              </div>
            </div>

            <!-- Preferred Network (RAT) Selection -->
            <div class="space-y-2 p-3 bg-card-sub border border-border rounded">
              <span class="text-[10px] text-muted uppercase font-bold block">Preferred Network Mode (RAT)</span>
              <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
                {#each [
                  { id: 'hybrid', label: 'Hybrid (Auto)', badge: '5G/4G/3G' },
                  { id: '5g_only', label: '5G NR Only', badge: 'SA/NSA' },
                  { id: '4g_only', label: '4G LTE Only', badge: 'LTE-A' },
                  { id: '3g_only', label: '3G Only', badge: 'WCDMA' }
                ] as rat}
                  <button
                    type="button"
                    class="neo-button p-2 text-center rounded border transition-all cursor-pointer {bandConfig.preferred_rat === rat.id ? 'border-accent bg-accent/15 text-accent font-bold shadow-neobrutal-sm' : 'border-border bg-card text-muted hover:text-foreground'}"
                    onclick={() => (bandConfig.preferred_rat = rat.id as BandConfig['preferred_rat'])}
                  >
                    <div class="font-bold text-xs">{rat.label}</div>
                    <div class="text-[9px] text-muted">{rat.badge}</div>
                  </button>
                {/each}
              </div>
            </div>
          </div>

          <!-- LTE Bands Checklist -->
          <div class="space-y-3">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <span class="font-bold text-sm text-foreground uppercase tracking-wider">LTE 4G Frequency Bands</span>
                <Badge variant="default">{bandConfig.lte_bands.length} Bands Active</Badge>
              </div>
              <div class="flex items-center gap-1.5">
                <button
                  type="button"
                  class="px-2 py-0.5 text-[10px] font-bold border border-border rounded bg-card-sub hover:border-accent text-muted hover:text-foreground cursor-pointer"
                  onclick={selectAllLTE}
                >
                  Select All
                </button>
                <button
                  type="button"
                  class="px-2 py-0.5 text-[10px] font-bold border border-border rounded bg-card-sub hover:border-red-500 text-muted hover:text-red-400 cursor-pointer"
                  onclick={clearAllLTE}
                >
                  Clear All
                </button>
              </div>
            </div>

            <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 gap-2.5">
              {#each availableLTEBands as b}
                {@const isChecked = bandConfig.lte_bands.includes(b.band)}
                <button
                  type="button"
                  class="p-2.5 rounded-md border-2 transition-all text-left cursor-pointer flex flex-col justify-between {isChecked ? 'border-accent bg-accent/10 text-foreground shadow-neobrutal-sm' : 'border-border bg-card-sub text-muted hover:border-muted-foreground'}"
                  onclick={() => toggleLTEBand(b.band)}
                >
                  <div class="flex items-center justify-between">
                    <span class="font-black text-sm {isChecked ? 'text-accent' : 'text-muted'}">{b.label}</span>
                    <span class="w-4 h-4 rounded border flex items-center justify-center {isChecked ? 'bg-accent border-black text-black' : 'border-border bg-card'}">
                      {#if isChecked}
                        <Check class="w-3 h-3 stroke-[3]" />
                      {/if}
                    </span>
                  </div>
                  <div class="text-[10px] text-muted mt-1 leading-tight">{b.name}</div>
                </button>
              {/each}
            </div>
          </div>

          <!-- 5G NR Bands Checklist -->
          <div class="space-y-3 pt-4 border-t border-border">
            <div class="flex items-center gap-2">
              <span class="font-bold text-sm text-foreground uppercase tracking-wider">5G NR Frequency Bands</span>
              <Badge variant="info">{bandConfig.nr_bands.length} Bands Active</Badge>
            </div>

            <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-6 gap-2.5">
              {#each availableNRBands as b}
                {@const isChecked = bandConfig.nr_bands.includes(b.band)}
                <button
                  type="button"
                  class="p-2.5 rounded-md border-2 transition-all text-left cursor-pointer flex flex-col justify-between {isChecked ? 'border-cyan-500 bg-cyan-950/20 text-foreground shadow-neobrutal-sm' : 'border-border bg-card-sub text-muted hover:border-muted-foreground'}"
                  onclick={() => toggleNRBand(b.band)}
                >
                  <div class="flex items-center justify-between">
                    <span class="font-black text-sm {isChecked ? 'text-cyan-400' : 'text-muted'}">{b.label}</span>
                    <span class="w-4 h-4 rounded border flex items-center justify-center {isChecked ? 'bg-cyan-500 border-black text-black' : 'border-border bg-card'}">
                      {#if isChecked}
                        <Check class="w-3 h-3 stroke-[3]" />
                      {/if}
                    </span>
                  </div>
                  <div class="text-[10px] text-muted mt-1 leading-tight">{b.name}</div>
                </button>
              {/each}
            </div>
          </div>

          <!-- Custom Bitmask & Quick Reset Bar -->
          <div class="pt-4 border-t border-border flex flex-col sm:flex-row items-center justify-between gap-3">
            <div class="flex items-center gap-2 w-full sm:w-auto">
              <span class="text-[11px] text-muted whitespace-nowrap">Hex Bitmask Override:</span>
              <input
                type="text"
                placeholder="Auto-calculated (e.g. 0x8000000085)"
                bind:value={bandConfig.hex_bitmask}
                class="neo-input w-full sm:w-64 bg-card-sub border border-border rounded px-2.5 py-1 text-xs font-mono text-foreground focus:outline-none focus:border-accent"
              />
            </div>

            <div class="flex items-center gap-2 w-full sm:w-auto justify-end">
              <Button
                variant="outline"
                size="sm"
                onclick={() => (resetModalOpen = true)}
              >
                <RotateCcw class="w-3.5 h-3.5 mr-1" />
                <span>Reset Bands</span>
              </Button>
              <Button
                variant="primary"
                size="sm"
                disabled={isApplyingBands}
                onclick={applyBandLock}
              >
                <Lock class="w-3.5 h-3.5 mr-1 text-accent-text" />
                <span>Apply Band Lock</span>
              </Button>
            </div>
          </div>
        </div>
      </Card>
    </div>
  {/if}

  <!-- SUB-TAB 3: SMS Inbox -->
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

              <p class="text-foreground select-text whitespace-pre-wrap">{sms.body}</p>

              <div class="flex items-center justify-end gap-2 pt-1">
                {#if otp}
                  <Button
                    variant="outline"
                    size="sm"
                    onclick={() => copyText(otp, sms.id)}
                  >
                    {#if copiedId === sms.id}
                      <Check class="w-3 h-3 mr-1 text-emerald-400" />
                      <span class="text-emerald-400">Copied</span>
                    {:else}
                      <Copy class="w-3 h-3 mr-1 text-amber-400" />
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
                  <span>Copy Body</span>
                </Button>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </Card>
  {/if}

  <!-- SUB-TAB 4: AT Terminal -->
  {#if subTab === 'at'}
    <Card title="Modem Serial AT Command Console">
      <div class="space-y-4 font-mono text-xs">
        <form onsubmit={(e) => { e.preventDefault(); sendAT() }} class="flex items-center gap-2">
          <Input
            placeholder="Enter AT command (e.g. ATI, AT+CSQ, AT+QENG=?)..."
            bind:value={atCommand}
            disabled={isExecutingAT}
          />
          <Button type="submit" variant="primary" size="md" disabled={isExecutingAT || !atCommand}>
            <Send class="w-3.5 h-3.5 mr-1" />
            <span>Send</span>
          </Button>
        </form>

        <div class="bg-card-sub border border-border p-3 rounded space-y-2 max-h-96 overflow-y-auto">
          {#if atHistory.length === 0}
            <div class="text-center text-muted p-4">
              <Terminal class="w-6 h-6 mx-auto mb-2 opacity-50" />
              No commands executed yet. Default probe port: /dev/smd11 or /dev/ttyUSB2.
            </div>
          {:else}
            {#each atHistory as h}
              <div class="space-y-1 border-b border-border/50 pb-2 last:border-0 last:pb-0">
                <div class="flex items-center justify-between text-muted text-[10px]">
                  <span class="font-bold text-accent">&gt; {h.cmd}</span>
                  <span>{h.time}</span>
                </div>
                <pre class="bg-black/40 p-2 rounded text-[11px] text-foreground overflow-x-auto whitespace-pre-wrap">{h.resp}</pre>
              </div>
            {/each}
          {/if}
        </div>
      </div>
    </Card>
  {/if}
</div>

<!-- Reset Modem Confirmation Modal -->
<Modal
  open={resetModalOpen}
  title="Reset Cellular Modem & Band Settings"
  onclose={() => (resetModalOpen = false)}
>
  <div class="space-y-3 font-mono text-xs">
    <div class="flex items-start gap-3 p-3 bg-red-950/30 border border-red-800 rounded text-red-200">
      <AlertTriangle class="w-5 h-5 text-red-400 shrink-0 mt-0.5" />
      <div>
        <p class="font-bold text-red-400">Restore All Bands & Network Modes?</p>
        <p class="text-[11px] mt-1 text-red-300/80">
          This will wipe all active band lock overrides, restore preferred network mode to default (Auto 5G/4G/3G), and trigger a radio interface restart.
        </p>
      </div>
    </div>
  </div>

  {#snippet footer()}
    <Button
      variant="outline"
      size="sm"
      disabled={isResettingModem}
      onclick={() => (resetModalOpen = false)}
    >
      Cancel
    </Button>
    <Button
      variant="danger"
      size="sm"
      disabled={isResettingModem}
      onclick={resetModemSettings}
    >
      <RotateCcw class="w-3.5 h-3.5 mr-1" />
      <span>{isResettingModem ? 'Resetting...' : 'Reset Modem Now'}</span>
    </Button>
  {/snippet}
</Modal>
