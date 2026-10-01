<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Layers,
    RefreshCw,
    Lock,
    RotateCcw,
    AlertTriangle,
    ShieldCheck,
    Check,
    Cpu,
    Radio,
    Sliders,
    PhoneCall,
    Activity,
    Info,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Modal from '../../ui/Modal.svelte'

  interface ShannonStatus {
    is_shannon: boolean
    device_model: string
    soc: string
    modem_type?: string
    router_available: boolean
    dm_available: boolean
    active_slot: number
    mode: string // 'both' | 'sa' | 'nsa' | 'lte_only'
    lte_manual_enabled: boolean
    active_lte_bands: number[]
    sa_manual_enabled: boolean
    active_sa_bands: number[]
    nsa_manual_enabled: boolean
    active_nsa_bands: number[]
    wcdma_manual_enabled: boolean
    active_wcdma_bands: number[]
    gsm_manual_enabled: boolean
    active_gsm_bands: number[]
    active_rats: string[]
    vonr_state: number
    supported_lte: number[]
    supported_sa: number[]
    supported_wcdma: number[]
    supported_gsm: number[]
    nsg_running: boolean
    status_message?: string
  }

  let subTab = $state<'bands' | 'rat_mode' | 'vonr_diag'>('bands')
  let bandCategory = $state<'lte' | 'sa' | 'nsa' | 'wcdma' | 'gsm'>('lte')

  let status = $state<ShannonStatus>({
    is_shannon: false,
    device_model: '',
    soc: '',
    router_available: false,
    dm_available: false,
    active_slot: 0,
    mode: 'both',
    lte_manual_enabled: false,
    active_lte_bands: [],
    sa_manual_enabled: false,
    active_sa_bands: [],
    nsa_manual_enabled: false,
    active_nsa_bands: [],
    wcdma_manual_enabled: false,
    active_wcdma_bands: [],
    gsm_manual_enabled: false,
    active_gsm_bands: [],
    active_rats: [],
    vonr_state: -2,
    supported_lte: [],
    supported_sa: [],
    supported_wcdma: [],
    supported_gsm: [],
    nsg_running: false,
  })

  let selectedSlot = $state<number>(0)
  let selectedMode = $state<string>('both')
  let selectedVoNR = $state<number>(-1)

  // Selection states
  let selectedLTEBands = $state<number[]>([1, 3, 8])
  let selectedSABands = $state<number[]>([1, 3, 28, 40, 77, 78])
  let selectedNSABands = $state<number[]>([1, 3, 28, 40, 77, 78])
  let selectedWCDMABands = $state<number[]>([1, 8])
  let selectedGSMBands = $state<number[]>([900, 1800])
  let selectedRATs = $state<string[]>(['GSM', 'WCDMA', 'LTE', 'NR'])

  let isLoading = $state(false)
  let isApplying = $state(false)
  let isResetting = $state(false)
  let showResetModal = $state(false)

  // Standard Band Definitions
  const availableLTEBands = [
    { band: 1, label: 'B1', name: '2100 MHz FDD' },
    { band: 2, label: 'B2', name: '1900 MHz PCS' },
    { band: 3, label: 'B3', name: '1800 MHz FDD' },
    { band: 4, label: 'B4', name: '1700/2100 AWS' },
    { band: 5, label: 'B5', name: '850 MHz FDD' },
    { band: 7, label: 'B7', name: '2600 MHz FDD' },
    { band: 8, label: 'B8', name: '900 MHz FDD' },
    { band: 12, label: 'B12', name: '700 MHz Lower' },
    { band: 13, label: 'B13', name: '700 MHz Upper' },
    { band: 14, label: 'B14', name: '700 MHz FirstNet' },
    { band: 17, label: 'B17', name: '700 MHz Lower' },
    { band: 18, label: 'B18', name: '800 MHz Japan' },
    { band: 19, label: 'B19', name: '800 MHz Japan' },
    { band: 20, label: 'B20', name: '800 MHz EU' },
    { band: 25, label: 'B25', name: '1900 MHz Ext' },
    { band: 26, label: 'B26', name: '850 MHz Ext' },
    { band: 28, label: 'B28', name: '700 MHz APT' },
    { band: 29, label: 'B29', name: '700 MHz SDL' },
    { band: 30, label: 'B30', name: '2300 MHz WCS' },
    { band: 32, label: 'B32', name: '1500 MHz L-Band' },
    { band: 38, label: 'B38', name: '2600 MHz TDD' },
    { band: 39, label: 'B39', name: '1900 MHz TDD' },
    { band: 40, label: 'B40', name: '2300 MHz TDD' },
    { band: 41, label: 'B41', name: '2500 MHz TDD' },
    { band: 46, label: 'B46', name: '5200 MHz LAA' },
    { band: 48, label: 'B48', name: '3600 MHz CBRS' },
    { band: 66, label: 'B66', name: '1700/2100 Ext' },
    { band: 71, label: 'B71', name: '600 MHz US' },
  ]

  const availableNRBands = [
    { band: 1, label: 'n1', name: '2100 MHz Sub-6' },
    { band: 2, label: 'n2', name: '1900 MHz PCS' },
    { band: 3, label: 'n3', name: '1800 MHz Sub-6' },
    { band: 5, label: 'n5', name: '850 MHz Sub-6' },
    { band: 7, label: 'n7', name: '2600 MHz Sub-6' },
    { band: 8, label: 'n8', name: '900 MHz Sub-6' },
    { band: 12, label: 'n12', name: '700 MHz Sub-6' },
    { band: 20, label: 'n20', name: '800 MHz Sub-6' },
    { band: 25, label: 'n25', name: '1900 MHz Sub-6' },
    { band: 26, label: 'n26', name: '850 MHz Sub-6' },
    { band: 28, label: 'n28', name: '700 MHz Sub-6' },
    { band: 29, label: 'n29', name: '700 MHz SDL' },
    { band: 30, label: 'n30', name: '2300 MHz WCS' },
    { band: 38, label: 'n38', name: '2600 MHz Sub-6' },
    { band: 40, label: 'n40', name: '2300 MHz Sub-6' },
    { band: 41, label: 'n41', name: '2500 MHz Sub-6' },
    { band: 48, label: 'n48', name: '3600 MHz CBRS' },
    { band: 66, label: 'n66', name: '1700/2100 Sub-6' },
    { band: 70, label: 'n70', name: 'AWS-4' },
    { band: 71, label: 'n71', name: '600 MHz Sub-6' },
    { band: 75, label: 'n75', name: '1500 MHz SDL' },
    { band: 77, label: 'n77', name: '3700 MHz C-Band' },
    { band: 78, label: 'n78', name: '3500 MHz Mid-Band' },
    { band: 79, label: 'n79', name: '4700 MHz Sub-6' },
    { band: 257, label: 'n257', name: '28 GHz mmWave' },
    { band: 258, label: 'n258', name: '26 GHz mmWave' },
    { band: 260, label: 'n260', name: '39 GHz mmWave' },
    { band: 261, label: 'n261', name: '28 GHz mmWave' },
  ]

  const availableWCDMABands = [
    { band: 1, label: 'B1', name: '2100 MHz IMT' },
    { band: 2, label: 'B2', name: '1900 MHz PCS' },
    { band: 4, label: 'B4', name: '1700/2100 AWS' },
    { band: 5, label: 'B5', name: '850 MHz CLR' },
    { band: 8, label: 'B8', name: '900 MHz GSM' },
  ]

  const availableGSMBands = [
    { band: 850, label: 'GSM 850', name: '850 MHz Cellular' },
    { band: 900, label: 'GSM 900', name: '900 MHz E-GSM' },
    { band: 1800, label: 'DCS 1800', name: '1800 MHz DCS' },
    { band: 1900, label: 'PCS 1900', name: '1900 MHz PCS' },
  ]

  onMount(() => {
    fetchStatus()
  })

  async function fetchStatus() {
    try {
      isLoading = true
      const res = await api.get<ShannonStatus>('/api/shannon/status')
      if (res) {
        status = res
        selectedSlot = res.active_slot ?? 0
        selectedMode = res.mode || 'both'
        selectedVoNR = res.vonr_state ?? -1
        if (Array.isArray(res.active_lte_bands) && res.active_lte_bands.length > 0) {
          selectedLTEBands = [...res.active_lte_bands]
        }
        if (Array.isArray(res.active_sa_bands) && res.active_sa_bands.length > 0) {
          selectedSABands = [...res.active_sa_bands]
        }
        if (Array.isArray(res.active_nsa_bands) && res.active_nsa_bands.length > 0) {
          selectedNSABands = [...res.active_nsa_bands]
        }
        if (Array.isArray(res.active_wcdma_bands) && res.active_wcdma_bands.length > 0) {
          selectedWCDMABands = [...res.active_wcdma_bands]
        }
        if (Array.isArray(res.active_gsm_bands) && res.active_gsm_bands.length > 0) {
          selectedGSMBands = [...res.active_gsm_bands]
        }
        if (Array.isArray(res.active_rats) && res.active_rats.length > 0) {
          selectedRATs = [...res.active_rats]
        }
      }
    } catch {
      // Retain previous state
    } finally {
      isLoading = false
    }
  }

  async function applyChanges() {
    if (!status.is_shannon) {
      toastStore.warning(`Shannon modem not detected on ${status.device_model || 'this device'}. Action blocked for safety.`)
      return
    }
    try {
      isApplying = true
      const payload = {
        slot: selectedSlot,
        mode: selectedMode,
        apply_mode: true,
        lte_bands: selectedLTEBands,
        apply_lte: true,
        lte_all: false,
        sa_bands: selectedSABands,
        apply_sa: true,
        sa_all: false,
        nsa_bands: selectedNSABands,
        apply_nsa: true,
        nsa_all: false,
        wcdma_bands: selectedWCDMABands,
        apply_wcdma: true,
        wcdma_all: false,
        gsm_bands: selectedGSMBands,
        apply_gsm: true,
        gsm_all: false,
        rats: selectedRATs,
        apply_rat: true,
        vonr_state: selectedVoNR,
        apply_vonr: selectedVoNR !== -2,
      }

      const res = await api.post<{ success: boolean; error?: string }>('/api/shannon/lock', payload)
      if (res && res.success) {
        toastStore.success('Google Tensor Shannon settings applied via DM menu (zero CFUN drop).')
        await fetchStatus()
      } else {
        toastStore.error(res?.error || 'Failed to apply Shannon settings')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Settings application failed')
    } finally {
      isApplying = false
    }
  }

  async function confirmReset() {
    if (!status.is_shannon) {
      toastStore.warning(`Shannon modem not detected on ${status.device_model || 'this device'}.`)
      showResetModal = false
      return
    }
    try {
      isResetting = true
      const res = await api.post<{ success: boolean; error?: string }>('/api/shannon/reset', {
        slot: selectedSlot,
      })
      if (res && res.success) {
        toastStore.success('Shannon modem restored to factory default bands & modes.')
        showResetModal = false
        await fetchStatus()
      } else {
        toastStore.error(res?.error || 'Failed to reset Shannon modem')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Reset request failed')
    } finally {
      isResetting = false
    }
  }

  function toggleBand(targetArray: number[], bandNum: number): number[] {
    if (targetArray.includes(bandNum)) {
      return targetArray.filter((x) => x !== bandNum)
    }
    return [...targetArray, bandNum].sort((a, b) => a - b)
  }

  function toggleRAT(rat: string) {
    if (selectedRATs.includes(rat)) {
      selectedRATs = selectedRATs.filter((r) => r !== rat)
    } else {
      selectedRATs = [...selectedRATs, rat]
    }
  }

  function applyRATPreset(preset: '5g_only' | '4g_only' | '3g_only' | '2g_only' | 'lte_nr' | 'auto') {
    switch (preset) {
      case '5g_only':
        selectedRATs = ['NR']
        selectedMode = 'sa'
        break
      case '4g_only':
        selectedRATs = ['LTE']
        selectedMode = 'lte_only'
        break
      case '3g_only':
        selectedRATs = ['WCDMA']
        selectedMode = 'lte_only'
        break
      case '2g_only':
        selectedRATs = ['GSM']
        selectedMode = 'lte_only'
        break
      case 'lte_nr':
        selectedRATs = ['LTE', 'NR']
        selectedMode = 'both'
        break
      case 'auto':
        selectedRATs = ['GSM', 'WCDMA', 'LTE', 'NR']
        selectedMode = 'both'
        break
    }
  }
</script>

<div class="space-y-6 pb-12 font-mono">
  <!-- TOP HERO & REFRESH -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b-2 border-border pb-4">
    <div>
      <div class="flex items-center gap-2">
        <Layers class="w-6 h-6 text-accent" />
        <h1 class="text-xl sm:text-2xl font-black uppercase tracking-wider text-foreground">
          Google Tensor Shannon Control
        </h1>
        <Badge variant="info" class="text-[10px]">Pixel 6/7/8/9</Badge>
        {#if status.dm_available}
          <Badge variant="success" class="text-[9px]">CFUN-Free Apply</Badge>
        {/if}
      </div>
      <p class="text-xs text-muted mt-1">
        Direct Samsung Shannon modem AT/NV modulation via /dev/umts_router & instant DM service menu keypresses.
      </p>
    </div>

    <div class="flex items-center gap-2">
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoading || isApplying}
        onclick={fetchStatus}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoading ? 'animate-spin' : ''}" />
        <span>Refresh</span>
      </Button>
    </div>
  </div>

  <!-- NON-SHANNON HARDWARE NOTICE BANNER -->
  {#if !status.is_shannon}
    <div class="p-4 bg-amber-950/30 border-2 border-amber-600/70 rounded-lg flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 text-xs text-amber-200 shadow-neobrutal-sm">
      <div class="flex items-start gap-3">
        <AlertTriangle class="w-5 h-5 text-amber-400 shrink-0 mt-0.5" />
        <div class="space-y-1">
          <div class="flex items-center gap-2">
            <span class="font-bold text-amber-300 uppercase tracking-wide">
              Non-Google Tensor Device Detected (Shannon Baseband Unavailable)
            </span>
            <Badge variant="warning" class="text-[9px]">Preview Mode</Badge>
          </div>
          <p class="text-[11px] text-amber-200/90 leading-relaxed">
            Detected Hardware: <strong class="text-white font-bold">{status.device_model || 'Unknown'}</strong> (SoC: <strong class="text-white font-bold">{status.soc || 'Qualcomm'}</strong> — {status.modem_type || 'Qualcomm Snapdragon'}). This Shannon modem control suite is strictly engineered for <strong>Google Pixel 6/7/8/9 Series (Google Tensor GS101/GS201/G3/G4)</strong> utilizing Samsung Shannon baseband NV architecture.
          </p>
          <p class="text-[10px] text-amber-300/80 font-mono">
            ℹ️ All modem NV write actions are locked to prevent baseband radio corruption. To configure bands or lock cell towers on this device, please switch to the <strong>Qualcomm Snapdragon</strong> tab.
          </p>
        </div>
      </div>
    </div>
  {/if}

  <!-- NSG CONFLICT WARNING (ONLY ON REAL SHANNON HARDWARE) -->
  {#if status.is_shannon && status.nsg_running}
    <div class="p-3 bg-amber-950/40 border-2 border-amber-600 rounded-lg flex items-center gap-3 text-xs text-amber-200">
      <AlertTriangle class="w-5 h-5 text-amber-400 shrink-0" />
      <div>
        <p class="font-bold">Network Signal Guru (NSG) Detected Running!</p>
        <p class="text-[11px] text-amber-300/80">
          NSG actively communicates with the Shannon DM channel. Close NSG or pause test sessions before applying NV changes to avoid port collision.
        </p>
      </div>
    </div>
  {/if}

  <!-- HARDWARE STATUS BAR -->
  <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
    <div class="p-2.5 bg-card-sub border-2 border-border rounded">
      <span class="text-[10px] text-muted uppercase font-bold block mb-1">Detected Hardware</span>
      <p class="font-bold text-foreground truncate">{status.device_model || 'Unknown'}</p>
      <span class="text-[10px] text-muted">{status.soc || 'Generic'}</span>
    </div>
    <div class="p-2.5 bg-card-sub border-2 border-border rounded">
      <span class="text-[10px] text-muted uppercase font-bold block mb-1">Modem Architecture</span>
      <p class="font-bold {status.is_shannon ? 'text-emerald-400' : 'text-amber-400'} truncate">
        {status.modem_type || (status.is_shannon ? 'Samsung Shannon' : 'Non-Shannon')}
      </p>
      <span class="text-[10px] {status.is_shannon ? 'text-emerald-300/80' : 'text-amber-300/80'}">
        {status.is_shannon ? 'Tensor Baseband OK' : 'Incompatible (Preview)'}
      </span>
    </div>
    <div class="p-2.5 bg-card-sub border-2 border-border rounded">
      <span class="text-[10px] text-muted uppercase font-bold block mb-1">Shannon Router Port</span>
      <p class="font-bold {status.router_available ? 'text-emerald-400' : 'text-red-400'}">
        {status.router_available ? '/dev/umts_router (OK)' : 'Unavailable'}
      </p>
      <span class="text-[10px] text-muted">AT / NV Modulation</span>
    </div>
    <div class="p-2.5 bg-card-sub border-2 border-border rounded">
      <span class="text-[10px] text-muted uppercase font-bold block mb-1">DM Keypress Port</span>
      <p class="font-bold {status.dm_available ? 'text-cyan-400' : (status.is_shannon ? 'text-amber-400' : 'text-muted')}">
        {status.dm_available ? '/dev/umts_dm0 (OK)' : (status.is_shannon ? 'AT Fallback' : 'N/A')}
      </p>
      <span class="text-[10px] text-muted">Zero-Drop Reload</span>
    </div>
  </div>

  <!-- SUB-TAB SWITCHER -->
  <div class="flex items-center gap-2 border-b-2 border-border pb-2 overflow-x-auto text-xs">
    {#each [
      { id: 'bands', label: 'Band Lock (All RATs)', icon: Radio },
      { id: 'rat_mode', label: 'RAT Lock & 5G Mode', icon: Sliders },
      { id: 'vonr_diag', label: 'VoNR & Modem State', icon: PhoneCall },
    ] as tab}
      <button
        type="button"
        class="neo-button flex items-center gap-2 px-3 py-2 rounded-t font-bold transition-all cursor-pointer {subTab === tab.id
          ? 'border-2 border-b-0 border-accent bg-accent/15 text-accent shadow-neobrutal-sm'
          : 'border-2 border-transparent text-muted hover:text-foreground'}"
        onclick={() => (subTab = tab.id as typeof subTab)}
      >
        <tab.icon class="w-3.5 h-3.5" />
        <span>{tab.label}</span>
      </button>
    {/each}
  </div>

  <!-- SUB-TAB 1: FREQUENCY BANDS -->
  {#if subTab === 'bands'}
    <Card title="Multi-RAT Frequency Band Allocation" subtitle="Lock manual frequency carriers across 5G NR, 4G LTE, 3G WCDMA, and 2G GSM" tone="ice">
      {#snippet action()}
        <div class="flex items-center gap-1.5">
          <Button
            variant="outline"
            size="sm"
            disabled={!status.is_shannon || isResetting || isApplying}
            onclick={() => (showResetModal = true)}
          >
            <RotateCcw class="w-3.5 h-3.5 mr-1" />
            <span>Reset All</span>
          </Button>
          <Button
            variant="primary"
            size="sm"
            disabled={!status.is_shannon || isApplying}
            onclick={applyChanges}
          >
            <Lock class="w-3.5 h-3.5 mr-1 text-accent-text" />
            <span>{status.is_shannon ? (isApplying ? 'Applying...' : 'Apply Shannon Lock') : 'Locked (Tensor Only)'}</span>
          </Button>
        </div>
      {/snippet}

      <div class="space-y-4 text-xs">
        <!-- RAT Category Switcher -->
        <div class="flex items-center gap-2 flex-wrap border-b border-border/60 pb-3">
          {#each [
            { id: 'lte', label: '4G LTE', count: selectedLTEBands.length, total: availableLTEBands.length },
            { id: 'sa', label: '5G NR-SA', count: selectedSABands.length, total: availableNRBands.length },
            { id: 'nsa', label: '5G NR-NSA', count: selectedNSABands.length, total: availableNRBands.length },
            { id: 'wcdma', label: '3G WCDMA', count: selectedWCDMABands.length, total: availableWCDMABands.length },
            { id: 'gsm', label: '2G GSM', count: selectedGSMBands.length, total: availableGSMBands.length },
          ] as cat}
            <button
              type="button"
              class="px-3 py-1.5 rounded-md border-2 text-xs font-bold transition-all cursor-pointer flex items-center gap-1.5 {bandCategory === cat.id
                ? 'border-accent bg-accent/20 text-accent shadow-neobrutal-sm'
                : 'border-border bg-card-sub text-muted hover:text-foreground'}"
              onclick={() => (bandCategory = cat.id as typeof bandCategory)}
            >
              <span>{cat.label}</span>
              <span class="text-[10px] px-1.5 py-0.2 rounded bg-card text-muted">
                {cat.count}/{cat.total}
              </span>
            </button>
          {/each}
        </div>

        <!-- 1. LTE Category -->
        {#if bandCategory === 'lte'}
          <div class="space-y-3">
            <div class="flex items-center justify-between">
              <span class="font-bold text-foreground uppercase tracking-wider">LTE Frequency Bands</span>
              <div class="flex items-center gap-1.5">
                <button
                  type="button"
                  class="px-2 py-0.5 text-[10px] font-bold border-2 border-border rounded bg-card-sub hover:border-accent text-muted hover:text-foreground cursor-pointer"
                  onclick={() => (selectedLTEBands = availableLTEBands.map((b) => b.band))}
                >
                  Select All
                </button>
                <button
                  type="button"
                  class="px-2 py-0.5 text-[10px] font-bold border-2 border-border rounded bg-card-sub hover:border-red-500 text-muted hover:text-red-400 cursor-pointer"
                  onclick={() => (selectedLTEBands = [])}
                >
                  Clear All
                </button>
              </div>
            </div>

            <div class="grid grid-cols-2 sm:grid-cols-4 md:grid-cols-6 gap-2.5">
              {#each availableLTEBands as b}
                {@const isChecked = selectedLTEBands.includes(b.band)}
                <button
                  type="button"
                  class="p-2 rounded-md border-2 transition-all text-left cursor-pointer flex flex-col justify-between {isChecked ? 'border-accent bg-accent/10 text-foreground shadow-neobrutal-sm' : 'border-border bg-card-sub text-muted hover:border-muted-foreground'}"
                  onclick={() => (selectedLTEBands = toggleBand(selectedLTEBands, b.band))}
                >
                  <div class="flex items-center justify-between">
                    <span class="font-black text-sm {isChecked ? 'text-accent' : 'text-muted'}">{b.label}</span>
                    <span class="w-4 h-4 rounded border flex items-center justify-center {isChecked ? 'bg-accent border-black text-black' : 'border-border bg-card'}">
                      {#if isChecked}
                        <Check class="w-3 h-3 stroke-[3]" />
                      {/if}
                    </span>
                  </div>
                  <div class="text-[9px] text-muted mt-1 leading-tight">{b.name}</div>
                </button>
              {/each}
            </div>
          </div>
        {/if}

        <!-- 2. NR-SA Category -->
        {#if bandCategory === 'sa'}
          <div class="space-y-3">
            <div class="flex items-center justify-between">
              <div>
                <span class="font-bold text-foreground uppercase tracking-wider">5G NR Standalone (SA) Bands</span>
                <span class="text-[10px] text-muted ml-2">Written to !NRRRC.MANUAL_NR_BAND_LIST</span>
              </div>
              <div class="flex items-center gap-1.5">
                <button
                  type="button"
                  class="px-2 py-0.5 text-[10px] font-bold border-2 border-border rounded bg-card-sub hover:border-cyan-500 text-muted hover:text-foreground cursor-pointer"
                  onclick={() => (selectedSABands = availableNRBands.map((b) => b.band))}
                >
                  Select All
                </button>
                <button
                  type="button"
                  class="px-2 py-0.5 text-[10px] font-bold border-2 border-border rounded bg-card-sub hover:border-red-500 text-muted hover:text-red-400 cursor-pointer"
                  onclick={() => (selectedSABands = [])}
                >
                  Clear All
                </button>
              </div>
            </div>

            <div class="grid grid-cols-2 sm:grid-cols-4 md:grid-cols-6 gap-2.5">
              {#each availableNRBands as b}
                {@const isChecked = selectedSABands.includes(b.band)}
                <button
                  type="button"
                  class="p-2 rounded-md border-2 transition-all text-left cursor-pointer flex flex-col justify-between {isChecked ? 'border-cyan-500 bg-cyan-950/20 text-foreground shadow-neobrutal-sm' : 'border-border bg-card-sub text-muted hover:border-muted-foreground'}"
                  onclick={() => (selectedSABands = toggleBand(selectedSABands, b.band))}
                >
                  <div class="flex items-center justify-between">
                    <span class="font-black text-sm {isChecked ? 'text-cyan-400' : 'text-muted'}">{b.label}</span>
                    <span class="w-4 h-4 rounded border flex items-center justify-center {isChecked ? 'bg-cyan-500 border-black text-black' : 'border-border bg-card'}">
                      {#if isChecked}
                        <Check class="w-3 h-3 stroke-[3]" />
                      {/if}
                    </span>
                  </div>
                  <div class="text-[9px] text-muted mt-1 leading-tight">{b.name}</div>
                </button>
              {/each}
            </div>
          </div>
        {/if}

        <!-- 3. NR-NSA Category -->
        {#if bandCategory === 'nsa'}
          <div class="space-y-3">
            <div class="flex items-center justify-between">
              <div>
                <span class="font-bold text-foreground uppercase tracking-wider">5G NR Non-Standalone (NSA) Bands</span>
                <span class="text-[10px] text-muted ml-2">Written to !LTE.NR Manual Band List</span>
              </div>
              <div class="flex items-center gap-1.5">
                <button
                  type="button"
                  class="px-2 py-0.5 text-[10px] font-bold border-2 border-border rounded bg-card-sub hover:border-cyan-500 text-muted hover:text-foreground cursor-pointer"
                  onclick={() => (selectedNSABands = availableNRBands.map((b) => b.band))}
                >
                  Select All
                </button>
                <button
                  type="button"
                  class="px-2 py-0.5 text-[10px] font-bold border-2 border-border rounded bg-card-sub hover:border-red-500 text-muted hover:text-red-400 cursor-pointer"
                  onclick={() => (selectedNSABands = [])}
                >
                  Clear All
                </button>
              </div>
            </div>

            <div class="grid grid-cols-2 sm:grid-cols-4 md:grid-cols-6 gap-2.5">
              {#each availableNRBands as b}
                {@const isChecked = selectedNSABands.includes(b.band)}
                <button
                  type="button"
                  class="p-2 rounded-md border-2 transition-all text-left cursor-pointer flex flex-col justify-between {isChecked ? 'border-cyan-500 bg-cyan-950/20 text-foreground shadow-neobrutal-sm' : 'border-border bg-card-sub text-muted hover:border-muted-foreground'}"
                  onclick={() => (selectedNSABands = toggleBand(selectedNSABands, b.band))}
                >
                  <div class="flex items-center justify-between">
                    <span class="font-black text-sm {isChecked ? 'text-cyan-400' : 'text-muted'}">{b.label}</span>
                    <span class="w-4 h-4 rounded border flex items-center justify-center {isChecked ? 'bg-cyan-500 border-black text-black' : 'border-border bg-card'}">
                      {#if isChecked}
                        <Check class="w-3 h-3 stroke-[3]" />
                      {/if}
                    </span>
                  </div>
                  <div class="text-[9px] text-muted mt-1 leading-tight">{b.name}</div>
                </button>
              {/each}
            </div>
          </div>
        {/if}

        <!-- 4. WCDMA Category -->
        {#if bandCategory === 'wcdma'}
          <div class="space-y-3">
            <div class="flex items-center justify-between">
              <span class="font-bold text-foreground uppercase tracking-wider">3G WCDMA / UMTS Bands</span>
              <div class="flex items-center gap-1.5">
                <button
                  type="button"
                  class="px-2 py-0.5 text-[10px] font-bold border-2 border-border rounded bg-card-sub hover:border-accent text-muted hover:text-foreground cursor-pointer"
                  onclick={() => (selectedWCDMABands = availableWCDMABands.map((b) => b.band))}
                >
                  Select All
                </button>
                <button
                  type="button"
                  class="px-2 py-0.5 text-[10px] font-bold border-2 border-border rounded bg-card-sub hover:border-red-500 text-muted hover:text-red-400 cursor-pointer"
                  onclick={() => (selectedWCDMABands = [])}
                >
                  Clear All
                </button>
              </div>
            </div>

            <div class="grid grid-cols-2 sm:grid-cols-5 gap-2.5">
              {#each availableWCDMABands as b}
                {@const isChecked = selectedWCDMABands.includes(b.band)}
                <button
                  type="button"
                  class="p-2.5 rounded-md border-2 transition-all text-left cursor-pointer flex flex-col justify-between {isChecked ? 'border-amber-500 bg-amber-950/20 text-foreground shadow-neobrutal-sm' : 'border-border bg-card-sub text-muted hover:border-muted-foreground'}"
                  onclick={() => (selectedWCDMABands = toggleBand(selectedWCDMABands, b.band))}
                >
                  <div class="flex items-center justify-between">
                    <span class="font-black text-sm {isChecked ? 'text-amber-400' : 'text-muted'}">{b.label}</span>
                    <span class="w-4 h-4 rounded border flex items-center justify-center {isChecked ? 'bg-amber-500 border-black text-black' : 'border-border bg-card'}">
                      {#if isChecked}
                        <Check class="w-3 h-3 stroke-[3]" />
                      {/if}
                    </span>
                  </div>
                  <div class="text-[9px] text-muted mt-1 leading-tight">{b.name}</div>
                </button>
              {/each}
            </div>
          </div>
        {/if}

        <!-- 5. GSM Category -->
        {#if bandCategory === 'gsm'}
          <div class="space-y-3">
            <div class="flex items-center justify-between">
              <span class="font-bold text-foreground uppercase tracking-wider">2G GSM / EDGE Bands</span>
              <div class="flex items-center gap-1.5">
                <button
                  type="button"
                  class="px-2 py-0.5 text-[10px] font-bold border-2 border-border rounded bg-card-sub hover:border-accent text-muted hover:text-foreground cursor-pointer"
                  onclick={() => (selectedGSMBands = availableGSMBands.map((b) => b.band))}
                >
                  Select All
                </button>
                <button
                  type="button"
                  class="px-2 py-0.5 text-[10px] font-bold border-2 border-border rounded bg-card-sub hover:border-red-500 text-muted hover:text-red-400 cursor-pointer"
                  onclick={() => (selectedGSMBands = [])}
                >
                  Clear All
                </button>
              </div>
            </div>

            <div class="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
              {#each availableGSMBands as b}
                {@const isChecked = selectedGSMBands.includes(b.band)}
                <button
                  type="button"
                  class="p-2.5 rounded-md border-2 transition-all text-left cursor-pointer flex flex-col justify-between {isChecked ? 'border-orange-500 bg-orange-950/20 text-foreground shadow-neobrutal-sm' : 'border-border bg-card-sub text-muted hover:border-muted-foreground'}"
                  onclick={() => (selectedGSMBands = toggleBand(selectedGSMBands, b.band))}
                >
                  <div class="flex items-center justify-between">
                    <span class="font-black text-sm {isChecked ? 'text-orange-400' : 'text-muted'}">{b.label}</span>
                    <span class="w-4 h-4 rounded border flex items-center justify-center {isChecked ? 'bg-orange-500 border-black text-black' : 'border-border bg-card'}">
                      {#if isChecked}
                        <Check class="w-3 h-3 stroke-[3]" />
                      {/if}
                    </span>
                  </div>
                  <div class="text-[9px] text-muted mt-1 leading-tight">{b.name}</div>
                </button>
              {/each}
            </div>
          </div>
        {/if}
      </div>
    </Card>
  {/if}

  <!-- SUB-TAB 2: RAT LOCK & 5G NR MODE -->
  {#if subTab === 'rat_mode'}
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- RAT Lock Card -->
      <Card title="Radio Access Technology (RAT) Lock" subtitle="Disable unwanted network generations independently" tone="ice">
        <div class="space-y-4 text-xs">
          <!-- Quick Presets -->
          <div>
            <span class="text-[10px] text-muted uppercase font-bold block mb-2">Quick 1-Click RAT Presets</span>
            <div class="grid grid-cols-2 sm:grid-cols-3 gap-2">
              <button
                type="button"
                class="p-2 rounded border-2 border-border bg-card-sub hover:border-accent text-center font-bold text-foreground cursor-pointer"
                onclick={() => applyRATPreset('5g_only')}
              >
                5G Only (NR SA)
              </button>
              <button
                type="button"
                class="p-2 rounded border-2 border-border bg-card-sub hover:border-accent text-center font-bold text-foreground cursor-pointer"
                onclick={() => applyRATPreset('4g_only')}
              >
                4G Only (LTE)
              </button>
              <button
                type="button"
                class="p-2 rounded border-2 border-border bg-card-sub hover:border-accent text-center font-bold text-foreground cursor-pointer"
                onclick={() => applyRATPreset('lte_nr')}
              >
                LTE + 5G (Dual)
              </button>
              <button
                type="button"
                class="p-2 rounded border-2 border-border bg-card-sub hover:border-accent text-center font-bold text-foreground cursor-pointer"
                onclick={() => applyRATPreset('3g_only')}
              >
                3G Only
              </button>
              <button
                type="button"
                class="p-2 rounded border-2 border-border bg-card-sub hover:border-accent text-center font-bold text-foreground cursor-pointer"
                onclick={() => applyRATPreset('2g_only')}
              >
                2G Only
              </button>
              <button
                type="button"
                class="p-2 rounded border-2 border-border bg-card-sub hover:border-accent text-center font-bold text-accent cursor-pointer"
                onclick={() => applyRATPreset('auto')}
              >
                Full Auto (All)
              </button>
            </div>
          </div>

          <!-- Active RAT Switches -->
          <div class="pt-3 border-t border-border/60">
            <span class="text-[10px] text-muted uppercase font-bold block mb-2">Individual RAT Toggles</span>
            <div class="grid grid-cols-2 gap-2.5">
              {#each [
                { id: 'NR', label: '5G New Radio', desc: 'SA & NSA 5G carriers' },
                { id: 'LTE', label: '4G LTE', desc: 'LTE-FDD & LTE-TDD' },
                { id: 'WCDMA', label: '3G WCDMA / UMTS', desc: 'HSPA+, 3G voice & data' },
                { id: 'GSM', label: '2G GSM / EDGE', desc: 'Legacy voice & 2G fallback' },
              ] as rat}
                {@const isEnabled = selectedRATs.includes(rat.id)}
                <button
                  type="button"
                  class="p-3 rounded-md border-2 transition-all text-left cursor-pointer flex items-center justify-between {isEnabled ? 'border-accent bg-accent/15 text-foreground shadow-neobrutal-sm' : 'border-border bg-card-sub text-muted'}"
                  onclick={() => toggleRAT(rat.id)}
                >
                  <div>
                    <strong class="block {isEnabled ? 'text-accent' : 'text-muted'}">{rat.label}</strong>
                    <span class="text-[10px] text-muted">{rat.desc}</span>
                  </div>
                  <span class="w-5 h-5 rounded border flex items-center justify-center {isEnabled ? 'bg-accent border-black text-black' : 'border-border bg-card'}">
                    {#if isEnabled}
                      <Check class="w-3.5 h-3.5 stroke-[3]" />
                    {/if}
                  </span>
                </button>
              {/each}
            </div>
          </div>
        </div>
      </Card>

      <!-- 5G NR Mode Card -->
      <Card title="5G NR Operating Mode" subtitle="Controls NR.CONFIG.MODE NV register" tone="ice">
        <div class="space-y-4 text-xs">
          <span class="text-[10px] text-muted uppercase font-bold block">Network Mode Options</span>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            {#each [
              { id: 'both', label: 'SA + NSA (0x11)', desc: 'Full 5G connectivity across Standalone and Non-Standalone anchor' },
              { id: 'sa', label: 'SA Only (0x10)', desc: 'Forces pure 5G Core (5GC) without LTE anchor tower requirement' },
              { id: 'nsa', label: 'NSA Only (0x01)', desc: 'Requires 4G LTE anchor with secondary 5G NR carrier attachment' },
              { id: 'lte_only', label: 'Disabled (0x00)', desc: 'Completely disables 5G NR stack to conserve battery power' },
            ] as m}
              <button
                type="button"
                class="p-3 rounded-md border-2 transition-all text-left cursor-pointer flex flex-col justify-between {selectedMode === m.id ? 'border-cyan-500 bg-cyan-950/25 text-foreground shadow-neobrutal-sm' : 'border-border bg-card-sub text-muted hover:text-foreground'}"
                onclick={() => (selectedMode = m.id)}
              >
                <div>
                  <strong class="block text-sm {selectedMode === m.id ? 'text-cyan-400' : 'text-foreground'}">{m.label}</strong>
                  <p class="text-[10px] text-muted mt-1 leading-snug">{m.desc}</p>
                </div>
              </button>
            {/each}
          </div>

          <div class="pt-4 border-t border-border/60 flex items-center justify-end">
            <Button
              variant="primary"
              size="sm"
              disabled={!status.is_shannon || isApplying}
              onclick={applyChanges}
            >
              <Lock class="w-3.5 h-3.5 mr-1 text-accent-text" />
              <span>{status.is_shannon ? (isApplying ? 'Applying Mode...' : 'Save & Reload Modem') : 'Locked (Tensor Only)'}</span>
            </Button>
          </div>
        </div>
      </Card>
    </div>
  {/if}

  <!-- SUB-TAB 3: VONR & DIAGNOSTICS -->
  {#if subTab === 'vonr_diag'}
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- VoNR Configuration -->
      <Card title="VoNR (Voice over New Radio)" subtitle="Native HD voice calls over 5G Standalone infrastructure" tone="ice">
        <div class="space-y-4 text-xs">
          <p class="text-muted text-[11px]">
            Controls <code>OEM_GFEATURE_QMS_VONR_SAVE_AP_VALUE</code> and <code>OMC.BASED.VONR.PLMN.ENABLE</code> NV registers.
          </p>

          <div class="space-y-2">
            {#each [
              { val: 1, label: 'Forced ON (0x01)', desc: 'Overrides carrier SIM policy and forces VoNR activation' },
              { val: 0, label: 'Forced OFF (0x00)', desc: 'Disables VoNR; voice calls will fallback to VoLTE' },
              { val: -1, label: 'Carrier Default (0xFF)', desc: 'Allows SIM card carrier configuration to determine VoNR policy' },
            ] as v}
              <button
                type="button"
                class="w-full p-3 rounded-md border-2 transition-all text-left cursor-pointer flex items-center justify-between {selectedVoNR === v.val ? 'border-accent bg-accent/15 text-foreground shadow-neobrutal-sm' : 'border-border bg-card-sub text-muted hover:text-foreground'}"
                onclick={() => (selectedVoNR = v.val)}
              >
                <div>
                  <strong class="block {selectedVoNR === v.val ? 'text-accent' : 'text-foreground'}">{v.label}</strong>
                  <span class="text-[10px] text-muted">{v.desc}</span>
                </div>
                <span class="w-4 h-4 rounded-full border flex items-center justify-center {selectedVoNR === v.val ? 'border-accent bg-accent' : 'border-border'}">
                  {#if selectedVoNR === v.val}
                    <span class="w-2 h-2 rounded-full bg-black"></span>
                  {/if}
                </span>
              </button>
            {/each}
          </div>

          <div class="pt-3 border-t border-border/60 flex items-center justify-end">
            <Button
              variant="primary"
              size="sm"
              disabled={!status.is_shannon || isApplying}
              onclick={applyChanges}
            >
              <Lock class="w-3.5 h-3.5 mr-1 text-accent-text" />
              <span>{status.is_shannon ? (isApplying ? 'Applying...' : 'Apply VoNR Policy') : 'Locked (Tensor Only)'}</span>
            </Button>
          </div>
        </div>
      </Card>

      <!-- Live NV Inspector & Diagnostics -->
      <Card title="Shannon NV Memory State" subtitle="Hardware ground-truth values read via AT+GOOGGETNV" tone="ice">
        <div class="space-y-3 text-xs">
          <div class="flex items-center justify-between p-2 rounded bg-card-sub border border-border">
            <span class="text-muted">LTE Manual Selection NV</span>
            <span class="font-bold {status.lte_manual_enabled ? 'text-amber-400' : 'text-emerald-400'}">
              {status.lte_manual_enabled ? 'Enabled (Manual Active)' : 'Disabled (All Auto)'}
            </span>
          </div>

          <div class="flex items-center justify-between p-2 rounded bg-card-sub border border-border">
            <span class="text-muted">NR-SA Manual Selection NV</span>
            <span class="font-bold {status.sa_manual_enabled ? 'text-amber-400' : 'text-emerald-400'}">
              {status.sa_manual_enabled ? 'Enabled (Manual Active)' : 'Disabled (All Auto)'}
            </span>
          </div>

          <div class="flex items-center justify-between p-2 rounded bg-card-sub border border-border">
            <span class="text-muted">NR-NSA Manual Selection NV</span>
            <span class="font-bold {status.nsa_manual_enabled ? 'text-amber-400' : 'text-emerald-400'}">
              {status.nsa_manual_enabled ? 'Enabled (Manual Active)' : 'Disabled (All Auto)'}
            </span>
          </div>

          <div class="flex items-center justify-between p-2 rounded bg-card-sub border border-border">
            <span class="text-muted">Active RAT Ensembles</span>
            <span class="font-bold text-accent">
              {status.active_rats && status.active_rats.length > 0 ? status.active_rats.join(' • ') : 'None'}
            </span>
          </div>

          <div class="flex items-center justify-between p-2 rounded bg-card-sub border border-border">
            <span class="text-muted">VoNR Memory Status</span>
            <span class="font-bold text-foreground">
              {status.vonr_state === 1 ? 'Forced ON (0x01)' : status.vonr_state === 0 ? 'Forced OFF (0x00)' : status.vonr_state === -1 ? 'Default (0xFF)' : 'Unknown'}
            </span>
          </div>

          <div class="flex items-center justify-between p-2 rounded bg-card-sub border border-border">
            <span class="text-muted">NSG Conflict Status</span>
            <span class="font-bold {status.nsg_running ? 'text-red-400' : 'text-emerald-400'}">
              {status.nsg_running ? 'Running (Potential Conflict)' : 'Clear (No Conflict)'}
            </span>
          </div>
        </div>
      </Card>
    </div>
  {/if}
</div>

<!-- RESET CONFIRMATION MODAL -->
<Modal
  open={showResetModal}
  title="Restore All Shannon Factory Bands"
  onclose={() => (showResetModal = false)}
>
  <div class="space-y-3 text-xs">
    <div class="flex items-start gap-3 p-3 bg-red-950/30 border-2 border-red-800 rounded text-red-200">
      <AlertTriangle class="w-5 h-5 text-red-400 shrink-0 mt-0.5" />
      <div>
        <p class="font-bold text-red-400">Restore Shannon Modem to Defaults?</p>
        <p class="text-[11px] mt-1 text-red-300/80">
          This will wipe manual LTE, NR-SA, NR-NSA, WCDMA, and GSM overrides, re-enable all RATs, set NR mode back to SA+NSA (0x11), and trigger an instant modem service menu reload without dropping cellular connection.
        </p>
      </div>
    </div>
  </div>

  {#snippet footer()}
    <Button
      variant="outline"
      size="sm"
      disabled={isResetting}
      onclick={() => (showResetModal = false)}
    >
      Cancel
    </Button>
    <Button
      variant="danger"
      size="sm"
      disabled={isResetting}
      onclick={confirmReset}
    >
      <RotateCcw class="w-3.5 h-3.5 mr-1" />
      <span>{isResetting ? 'Restoring...' : 'Restore Factory Defaults'}</span>
    </Button>
  {/snippet}
</Modal>
