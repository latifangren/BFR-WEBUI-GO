<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Radio,
    RefreshCw,
    Terminal,
    ArrowLeftRight,
    Thermometer,
    Flame,
    ShieldCheck,
    AlertTriangle,
    Cpu,
    Smartphone,
    Layers,
    Zap,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import { navigationStore } from '../../../stores/navigation.svelte'
  import type { ATCommandResult } from '../../../types/network'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Input from '../../ui/Input.svelte'

  interface SimSlotInfo {
    slot_index: number
    sub_id: number
    state: string
    is_active: boolean
    is_esim?: boolean
    carrier?: string
    label: string
  }

  interface SimStatus {
    active_slot: number
    active_sub_id: number
    slots: SimSlotInfo[]
  }

  interface ThermalSensors {
    pa1_sub6_c: number
    pa2_sub6_c: number
    cell_front_cf_c: number
    modem_skin_c: number
    rf_pmic_c: number
    modem_baseband_c: number
    ap_soc_c: number
    battery_c: number
  }

  interface ThermalThrottlers {
    modem_pa_fr1_dsc: number
    modem_skin_nr_dsc: number
    modem_pa_fr1: number
  }

  interface Thermal5GStatus {
    available: boolean
    network_type: string
    is_5g_disconnect: boolean
    nr_band: string
    nr_rsrp: number
    traffic_mbps: number
    thermal_hal_status: number
    thermal_hal_desc: string
    sensors: ThermalSensors
    throttlers: ThermalThrottlers
    is_throttled: boolean
    culprit: string
    culprit_detail: string
    error?: string
  }

  // Sub-tab selection
  let subTab = $state<'signal' | 'sim' | 'thermal' | 'at'>('signal')

  // Signal & Info
  let signalInfo = $state<Record<string, unknown>>({})
  let isLoadingSignal = $state(false)

  // SIM Status & Switcher
  let simStatus = $state<SimStatus | null>(null)
  let isLoadingSim = $state(false)
  let isSwitchingSim = $state(false)

  // Thermal Telemetry
  let thermalData = $state<Thermal5GStatus | null>(null)
  let isLoadingThermal = $state(false)

  // AT Command Console
  let atCommand = $state('ATI')
  let atLogs = $state<ATCommandResult[]>([])
  let isExecutingAT = $state(false)

  onMount(() => {
    fetchSignal()
    fetchSim()
    fetchThermal()
  })

  async function fetchSignal() {
    try {
      isLoadingSignal = true
      const res = await api.get<Record<string, unknown>>('/api/modem/signal')
      if (res) signalInfo = res
    } catch {
      // Retain previous state
    } finally {
      isLoadingSignal = false
    }
  }

  async function fetchSim() {
    try {
      isLoadingSim = true
      const res = await api.get<SimStatus>('/api/modem/sim/status')
      if (res && Array.isArray(res.slots)) {
        simStatus = res
      }
    } catch {
      // Retain previous state
    } finally {
      isLoadingSim = false
    }
  }

  async function switchSim(slot: number) {
    try {
      isSwitchingSim = true
      const res = await api.post<{ success: boolean; error?: string; active_slot?: number }>('/api/modem/sim/switch', { slot })
      if (res && res.success) {
        toastStore.success(`Cellular data route switched to SIM ${slot + 1}.`)
        await fetchSim()
        await fetchSignal()
      } else {
        toastStore.error(res?.error || 'Failed to switch mobile data slot')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Switch request failed')
    } finally {
      isSwitchingSim = false
    }
  }

  async function fetchThermal() {
    try {
      isLoadingThermal = true
      const res = await api.get<Thermal5GStatus>('/api/modem/thermal')
      if (res) thermalData = res
    } catch {
      // Retain previous state
    } finally {
      isLoadingThermal = false
    }
  }

  async function executeAT(cmdToRun?: string) {
    const cmd = cmdToRun || atCommand.trim()
    if (!cmd) return

    try {
      isExecutingAT = true
      const res = await api.post<ATCommandResult>('/api/modem/at', { command: cmd })
      if (res) {
        atLogs = [res, ...atLogs].slice(0, 50)
        if (!cmdToRun) atCommand = ''
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'AT command execution failed')
    } finally {
      isExecutingAT = false
    }
  }

  const sensorList = $derived(
    thermalData?.sensors
      ? [
          { name: 'PA1 Sub-6', val: thermalData.sensors.pa1_sub6_c, desc: 'Sub-6GHz Power Amp 1', limit: 48.0 },
          { name: 'PA2 Sub-6', val: thermalData.sensors.pa2_sub6_c, desc: 'Sub-6GHz Power Amp 2', limit: 48.0 },
          { name: 'Modem Skin', val: thermalData.sensors.modem_skin_c, desc: 'Modem Board Thermistor', limit: 42.0 },
          { name: 'Cell Front (CF)', val: thermalData.sensors.cell_front_cf_c, desc: 'Cellular Frontend', limit: 45.0 },
          { name: 'RF PMIC', val: thermalData.sensors.rf_pmic_c, desc: 'RF Power IC', limit: 55.0 },
          { name: 'Modem DSP', val: thermalData.sensors.modem_baseband_c, desc: 'Baseband Subsystem', limit: 60.0 },
          { name: 'AP SoC', val: thermalData.sensors.ap_soc_c, desc: 'Application Processor', limit: 65.0 },
          { name: 'Battery', val: thermalData.sensors.battery_c, desc: 'Battery Pack', limit: 45.0 },
        ]
      : []
  )
</script>

<div class="space-y-3 sm:space-y-6 pb-12 font-mono">
  <!-- TOP HERO & REFRESH -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b-2 border-border pb-4">
    <div>
      <div class="flex items-center gap-2">
        <Radio class="w-6 h-6 text-accent" />
        <h1 class="text-xl sm:text-2xl font-black uppercase tracking-wider text-foreground">
          Cellular & Radio Telemetry
        </h1>
        <Badge variant="info" class="text-[10px]">Universal</Badge>
      </div>
      <p class="text-xs text-muted mt-1">
        General Android radio telemetry, signal power measurements, Dual-SIM routing, and thermal telemetry.
      </p>
    </div>

    <div class="flex items-center gap-2">
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoadingSignal || isLoadingSim || isLoadingThermal}
        onclick={() => { fetchSignal(); fetchSim(); fetchThermal(); }}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoadingSignal || isLoadingSim || isLoadingThermal ? 'animate-spin' : ''}" />
        <span>Refresh All</span>
      </Button>
    </div>
  </div>

  <!-- TARGET ARCHITECTURE & HARDWARE SPECIFICATIONS -->
  <div class="p-4 bg-card-sub border-2 border-border rounded-lg space-y-3 shadow-neobrutal-sm">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b-2 border-border/60 pb-3">
      <div class="flex items-center gap-2">
        <Cpu class="w-4 h-4 text-accent" />
        <span class="text-xs font-bold uppercase tracking-wider text-foreground">
          Target Architecture & Hardware Compatibility
        </span>
      </div>
      <Badge variant="success" class="flex items-center gap-1.5 w-fit">
        <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
        <span>COMPATIBLE (UNIVERSAL CELLULAR DASHBOARD)</span>
      </Badge>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 text-xs">
      <div class="p-2.5 bg-card border-2 border-border rounded">
        <span class="text-[10px] text-muted uppercase font-bold block mb-1">Target Device</span>
        <p class="font-bold text-foreground">All Android Smartphones (Universal / Brand-Agnostic)</p>
      </div>
      <div class="p-2.5 bg-card border-2 border-border rounded">
        <span class="text-[10px] text-muted uppercase font-bold block mb-1">Target ROM</span>
        <p class="font-bold text-foreground">All Android 8.0 to Android 15+ System ROMs</p>
      </div>
      <div class="p-2.5 bg-card border-2 border-border rounded">
        <span class="text-[10px] text-muted uppercase font-bold block mb-1">Target Chipset</span>
        <p class="font-bold text-foreground">Qualcomm, MediaTek, Exynos, Google Tensor, Unisoc</p>
      </div>
      <div class="p-2.5 bg-card border-2 border-border rounded">
        <span class="text-[10px] text-muted uppercase font-bold block mb-1">Target Modem</span>
        <p class="font-bold text-foreground">Android Telephony Subsystem & Telemetry Services</p>
      </div>
    </div>
  </div>

  <!-- DEDICATED BAND LOCKING NAVIGATION BANNER -->
  <div class="p-4 bg-card-sub border-2 border-border rounded-lg space-y-3">
    <div class="flex items-center justify-between">
      <span class="text-xs font-bold uppercase text-foreground">Looking for Hardware Band Locking?</span>
      <span class="text-[10px] text-muted font-bold">SELECT TARGET VENDOR TAB</span>
    </div>
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
      <button
        type="button"
        class="neo-button p-3 rounded-lg border-2 border-emerald-500/40 bg-emerald-950/20 text-left hover:border-emerald-500 transition-all cursor-pointer shadow-neobrutal-sm"
        onclick={() => (navigationStore.activeTab = 'qualcomm')}
      >
        <div class="flex items-center justify-between mb-1">
          <span class="font-bold text-emerald-400 text-xs flex items-center gap-1.5">
            <Zap class="w-3.5 h-3.5" />
            <span>Snapdragon QMI</span>
          </span>
          <Badge variant="success" class="text-[9px]">QRTR</Badge>
        </div>
        <p class="text-[10px] text-muted">Pixel 1-5, Xiaomi, POCO, OnePlus, AOSP Snapdragon</p>
      </button>

      <button
        type="button"
        class="neo-button p-3 rounded-lg border-2 border-cyan-500/40 bg-cyan-950/20 text-left hover:border-cyan-500 transition-all cursor-pointer shadow-neobrutal-sm"
        onclick={() => (navigationStore.activeTab = 'samsung')}
      >
        <div class="flex items-center justify-between mb-1">
          <span class="font-bold text-cyan-400 text-xs flex items-center gap-1.5">
            <Smartphone class="w-3.5 h-3.5" />
            <span>Samsung OneUI</span>
          </span>
          <Badge variant="info" class="text-[9px]">SecRIL</Badge>
        </div>
        <p class="text-[10px] text-muted">Galaxy Z Flip 3-6, Fold, S20-S24 (Samsung OneUI)</p>
      </button>

      <button
        type="button"
        class="neo-button p-3 rounded-lg border-2 border-amber-500/40 bg-amber-950/20 text-left hover:border-amber-500 transition-all cursor-pointer shadow-neobrutal-sm"
        onclick={() => (navigationStore.activeTab = 'shannon')}
      >
        <div class="flex items-center justify-between mb-1">
          <span class="font-bold text-amber-400 text-xs flex items-center gap-1.5">
            <Layers class="w-3.5 h-3.5" />
            <span>Google Tensor</span>
          </span>
          <Badge variant="warning" class="text-[9px]">Shannon</Badge>
        </div>
        <p class="text-[10px] text-muted">Pixel 6, 7, 8, 9 Series (/dev/umts_router)</p>
      </button>
    </div>
  </div>

  <!-- SUB-TAB SWITCHER -->
  <div class="flex items-center gap-2 border-b-2 border-border pb-2 overflow-x-auto text-xs">
    {#each [
      { id: 'signal', label: 'Signal & Radio', icon: Radio },
      { id: 'sim', label: 'Dual SIM Routing', icon: ArrowLeftRight },
      { id: 'thermal', label: '5G Thermal Telemetry', icon: Thermometer },
      { id: 'at', label: 'AT Console Terminal', icon: Terminal },
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

  <!-- SUB-TAB 1: Signal & Radio -->
  {#if subTab === 'signal'}
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-3 sm:gap-6">
      <Card title="Cellular Radio Signal" tone="mint" class="lg:col-span-2">
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

        <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
          <div class="bg-card-sub border-2 border-border p-3 rounded space-y-1">
            <span class="text-[10px] text-muted uppercase font-bold">RSRP (Power)</span>
            <p class="text-sm font-bold text-foreground">
              {signalInfo['rsrp'] !== undefined && signalInfo['rsrp'] !== null && signalInfo['rsrp'] !== -999 ? `${signalInfo['rsrp']} dBm` : '--'}
            </p>
          </div>

          <div class="bg-card-sub border-2 border-border p-3 rounded space-y-1">
            <span class="text-[10px] text-muted uppercase font-bold">RSRQ (Quality)</span>
            <p class="text-sm font-bold text-foreground">
              {signalInfo['rsrq'] !== undefined && signalInfo['rsrq'] !== null && signalInfo['rsrq'] !== -999 ? `${signalInfo['rsrq']} dB` : '--'}
            </p>
          </div>

          <div class="bg-card-sub border-2 border-border p-3 rounded space-y-1">
            <span class="text-[10px] text-muted uppercase font-bold">SINR (SNR)</span>
            <p class="text-sm font-bold text-foreground">
              {signalInfo['sinr'] !== undefined && signalInfo['sinr'] !== null && signalInfo['sinr'] !== -999 ? `${signalInfo['sinr']} dB` : 'N/A'}
            </p>
            {#if signalInfo['sinr'] === -999 || signalInfo['sinr'] === undefined}
              <span class="text-[9px] text-muted block">No SNR broadcast</span>
            {/if}
          </div>

          <div class="bg-card-sub border-2 border-border p-3 rounded space-y-1">
            <span class="text-[10px] text-muted uppercase font-bold">Network Mode</span>
            <p class="text-sm font-bold text-emerald-400">
              {String(signalInfo['network_type'] || '4G LTE-A')}
            </p>
          </div>
        </div>

        <div class="mt-4 p-3 bg-card-sub border-2 border-border rounded text-xs text-muted">
          <p class="font-bold text-foreground mb-1">Carrier Telemetry Integration</p>
          <p>
            Modem radio measurements parsed live via Android Telephony service registries.
          </p>
        </div>
      </Card>

      <Card title="Cell Identity & Info" subtitle="Radio mast details" tone="mint">
        <div class="space-y-3 text-xs">
          <div class="flex items-center justify-between p-2 rounded bg-card-sub border-2 border-border">
            <span class="text-muted">Operator</span>
            <span class="font-bold text-foreground">{String(signalInfo['operator'] || 'Auto-Detected')}</span>
          </div>
          <div class="flex items-center justify-between p-2 rounded bg-card-sub border-2 border-border">
            <span class="text-muted">Active Band</span>
            <span class="font-bold text-accent">{String(signalInfo['band'] || 'Auto')}</span>
          </div>
          <div class="flex items-center justify-between p-2 rounded bg-card-sub border-2 border-border">
            <span class="text-muted">Bandwidth</span>
            <span class="font-bold text-foreground">{String(signalInfo['bandwidth'] || '--')}</span>
          </div>
          <div class="flex items-center justify-between p-2 rounded bg-card-sub border-2 border-border">
            <span class="text-muted">EARFCN (Channel)</span>
            <span class="font-bold text-accent">{String(signalInfo['earfcn'] && signalInfo['earfcn'] !== 0 ? signalInfo['earfcn'] : '--')}</span>
          </div>
          <div class="flex items-center justify-between p-2 rounded bg-card-sub border-2 border-border">
            <span class="text-muted">Physical Cell ID (PCI)</span>
            <span class="font-bold text-emerald-400">{String(signalInfo['pci'] && signalInfo['pci'] !== 'Unknown' ? signalInfo['pci'] : '--')}</span>
          </div>
          <div class="flex items-center justify-between p-2 rounded bg-card-sub border-2 border-border">
            <span class="text-muted">Tower Distance (TA)</span>
            <span class="font-bold text-foreground">
              {#if signalInfo['distance_str']}
                <span class="text-cyan-400 font-mono font-bold">{signalInfo['distance_str']}</span>
              {:else if signalInfo['timing_advance'] && Number(signalInfo['timing_advance']) > 0}
                <span class="text-cyan-400 font-mono font-bold">TA: {signalInfo['timing_advance']}</span>
              {:else}
                <span class="text-muted font-normal">N/A (No TA broadcast)</span>
              {/if}
            </span>
          </div>
          <div class="flex items-center justify-between p-2 rounded bg-card-sub border-2 border-border">
            <span class="text-muted">Cell ID / TAC</span>
            <span class="font-bold text-foreground">
              {signalInfo['cell_id'] && signalInfo['cell_id'] !== 'Unknown' && signalInfo['cell_id'] !== '0' && signalInfo['cell_id'] !== 'Protected' ? signalInfo['cell_id'] : '--'} / {signalInfo['tac'] && signalInfo['tac'] !== 'Unknown' && signalInfo['tac'] !== '0' && signalInfo['tac'] !== 'Protected' ? signalInfo['tac'] : '--'}
              <span class="text-[9px] font-normal text-muted ml-1.5 px-1.5 py-0.5 rounded bg-amber-950/40 border border-amber-800/60 text-amber-300" title="Android 16 privacy policy masks global Cell ID and TAC in telephony dumpsys">Android 16 Masked</span>
            </span>
          </div>
        </div>
      </Card>
    </div>
  {/if}

  <!-- SUB-TAB 2: DUAL SIM & MOBILE DATA ROUTING -->
  {#if subTab === 'sim'}
    <div class="space-y-3 sm:space-y-6">
      <Card
        title="Universal Dual-SIM & Mobile Data Routing"
        subtitle="Manage active cellular data slot across physical SIMs and eSIMs"
        tone="mint"
      >
        {#snippet action()}
          <Button
            variant="secondary"
            size="sm"
            disabled={isLoadingSim}
            onclick={fetchSim}
          >
            <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoadingSim ? 'animate-spin' : ''}" />
            <span>Refresh</span>
          </Button>
        {/snippet}

        <div class="space-y-4">
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            {#each (simStatus?.slots || [
              { slot_index: 0, sub_id: 1, state: 'READY', is_active: true, is_esim: false, carrier: 'Auto', label: 'SIM 1 (Slot 0)' },
              { slot_index: 1, sub_id: 2, state: 'READY', is_active: false, is_esim: true, carrier: 'Auto', label: 'SIM 2 (Slot 1)' },
            ]) as slot}
              {@const isActive = simStatus ? simStatus.active_slot === slot.slot_index : slot.is_active}
              {@const isAbsent = slot.state === 'ABSENT'}
              <div
                class="p-4 rounded-lg border-2 transition-all flex flex-col justify-between gap-4 {isActive
                  ? 'border-emerald-500 bg-emerald-950/25 shadow-neobrutal-sm ring-1 ring-emerald-500/50'
                  : 'border-border bg-card-sub'}"
              >
                <div class="flex items-start justify-between gap-2">
                  <div class="flex items-center gap-3">
                    <div
                      class="w-9 h-9 rounded border-2 flex items-center justify-center font-bold text-sm {isActive
                        ? 'border-emerald-500 bg-emerald-500/20 text-emerald-300'
                        : 'border-border bg-card text-muted'}"
                    >
                      {slot.slot_index + 1}
                    </div>
                    <div>
                      <div class="flex items-center gap-1.5">
                        <span class="text-sm font-bold text-foreground">
                          SIM {slot.slot_index + 1}
                        </span>
                        <span
                          class="px-1.5 py-0.2 text-[9px] font-bold rounded border uppercase {slot.is_esim
                            ? 'border-purple-600 bg-purple-950/40 text-purple-300'
                            : 'border-blue-600 bg-blue-950/40 text-blue-300'}"
                        >
                          {slot.is_esim ? 'eSIM' : 'pSIM'}
                        </span>
                      </div>
                      <span class="text-xs font-semibold text-foreground/90 block mt-0.5">
                        {slot.carrier || (isAbsent ? 'No SIM Card Inserted' : 'Unknown Operator')}
                      </span>
                      <span class="text-[11px] text-muted block">
                        SubID: {slot.sub_id > 0 ? slot.sub_id : '--'} | Status: <strong class="text-foreground">{slot.state}</strong>
                      </span>
                    </div>
                  </div>

                  {#if isActive}
                    <Badge variant="success" class="flex items-center gap-1.5 shadow-sm">
                      <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
                      <span>ACTIVE DATA</span>
                    </Badge>
                  {:else if isAbsent}
                    <Badge variant="danger">ABSENT</Badge>
                  {:else}
                    <Badge variant="default">STANDBY</Badge>
                  {/if}
                </div>

                <div class="flex items-center justify-between gap-2 pt-3 border-t-2 border-border/50">
                  <span class="text-xs {isActive ? 'text-emerald-300 font-semibold' : 'text-muted'}">
                    {isActive ? '● Active Mobile Data Route' : isAbsent ? 'Slot Empty' : 'Standby for Data'}
                  </span>

                  {#if !isActive}
                    <Button
                      variant="primary"
                      size="sm"
                      disabled={isSwitchingSim || isAbsent}
                      onclick={() => switchSim(slot.slot_index)}
                    >
                      {#if isSwitchingSim}
                        <RefreshCw class="w-3.5 h-3.5 mr-1 animate-spin" />
                        <span>Switching...</span>
                      {:else if isAbsent}
                        <span>Slot Empty</span>
                      {:else}
                        <ArrowLeftRight class="w-3.5 h-3.5 mr-1" />
                        <span>Switch Data to SIM {slot.slot_index + 1}</span>
                      {/if}
                    </Button>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        </div>
      </Card>
    </div>
  {/if}

  <!-- SUB-TAB 3: MODEM & 5G THERMAL TELEMETRY -->
  {#if subTab === 'thermal'}
    <div class="space-y-3 sm:space-y-6">
      <Card
        title="Universal Modem & 5G Thermal Telemetry"
        subtitle="Real-time thermal zone sensor telemetry, Power Amps (PA), and Thermal HAL state"
        tone="ice"
      >
        {#snippet action()}
          <div class="flex items-center gap-2">
            {#if thermalData}
              {#if thermalData.is_throttled}
                <Badge variant="danger" class="flex items-center gap-1">
                  <Flame class="w-3 h-3" />
                  <span>THROTTLED</span>
                </Badge>
              {:else}
                <Badge variant="success" class="flex items-center gap-1">
                  <ShieldCheck class="w-3 h-3" />
                  <span>SAFE</span>
                </Badge>
              {/if}
            {/if}
            <Button
              variant="secondary"
              size="sm"
              disabled={isLoadingThermal}
              onclick={fetchThermal}
            >
              <RefreshCw class="w-3.5 h-3.5 mr-1 {isLoadingThermal ? 'animate-spin' : ''}" />
              <span>Refresh</span>
            </Button>
          </div>
        {/snippet}

        <div class="space-y-4">
          <!-- Sensors Temperature Grid -->
          <div>
            <div class="flex items-center justify-between mb-2">
              <span class="text-[10px] text-muted uppercase font-bold tracking-wider flex items-center gap-1.5">
                <Thermometer class="w-3.5 h-3.5 text-accent" />
                <span>Hardware Sensors Telemetry Grid (°C)</span>
              </span>
            </div>

            <div class="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
              {#each sensorList as s}
                {@const isHot = s.val >= s.limit}
                {@const isWarm = s.val >= s.limit - 3}
                <div class="p-3 rounded border-2 border-border bg-card-sub flex flex-col justify-between">
                  <div class="flex items-center justify-between text-[10px]">
                    <span class="text-muted font-bold">{s.name}</span>
                    <span class="text-[9px] text-muted font-mono">Limit: {s.limit}°C</span>
                  </div>
                  <div class="text-lg font-black font-mono my-1 {isHot ? 'text-red-400' : isWarm ? 'text-amber-400' : 'text-emerald-400'}">
                    {s.val > 0 ? s.val.toFixed(1) : '--'}°C
                  </div>
                  <div class="text-[9px] text-muted truncate">{s.desc}</div>
                </div>
              {/each}
            </div>
          </div>
        </div>
      </Card>
    </div>
  {/if}

  <!-- SUB-TAB 4: AT Command Console -->
  {#if subTab === 'at'}
    <Card title="Modem AT Command Terminal" subtitle="Direct AT serial interaction with baseband modem" tone="ice">
      <div class="space-y-4 text-xs">
        <form onsubmit={(e) => { e.preventDefault(); executeAT(); }} class="flex items-center gap-2">
          <Input
            placeholder="e.g. ATI, AT+CSQ, AT+CPIN?, AT+CGREG?"
            bind:value={atCommand}
            class="flex-1 font-mono uppercase"
          />
          <Button
            type="submit"
            variant="primary"
            disabled={isExecutingAT || !atCommand.trim()}
          >
            <Terminal class="w-4 h-4 mr-1.5" />
            <span>Send Command</span>
          </Button>
        </form>

        <div class="flex flex-wrap gap-1.5">
          {#each ['ATI', 'AT+CSQ', 'AT+CPIN?', 'AT+CGREG?', 'AT+COPS?'] as cmd}
            <button
              type="button"
              class="px-2 py-1 text-[11px] font-mono border-2 border-border rounded bg-card-sub hover:border-accent cursor-pointer"
              onclick={() => executeAT(cmd)}
            >
              {cmd}
            </button>
          {/each}
        </div>

        <div class="h-64 overflow-y-auto bg-black border-2 border-border rounded p-3 font-mono text-[11px] text-emerald-400 space-y-2">
          {#if atLogs.length === 0}
            <div class="text-muted italic">No AT commands sent yet. Type a command above to query the modem.</div>
          {:else}
            {#each atLogs as log}
              <div class="border-b border-zinc-800 pb-2">
                <div class="text-cyan-400 font-bold">&gt; {log.command}</div>
                <pre class="whitespace-pre-wrap text-zinc-300 mt-1">{log.response}</pre>
              </div>
            {/each}
          {/if}
        </div>
      </div>
    </Card>
  {/if}
</div>
