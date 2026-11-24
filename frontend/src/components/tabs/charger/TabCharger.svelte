<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Battery,
    BatteryCharging,
    BatteryWarning,
    Zap,
    ZapOff,
    RefreshCw,
    Sliders,
    Power,
    Check,
    AlertCircle,
    AlertTriangle,
    Flame,
    Thermometer,
    Activity,
    Cpu,
    Search,
    Layers,
    Info,
    ArrowRight,
    CheckCircle2,
    XCircle,
    Terminal,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import { LL } from '../../../i18n/i18n-svelte'
  import { sysinfoStore } from '../../../stores/sysinfo.svelte'
  import type {
    ChargerConfig,
    ChargerStatusResponse,
    ProbeResult,
  } from '../../../types/network'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Input from '../../ui/Input.svelte'
  import { themeStore } from '../../../stores/theme.svelte'
  import SakugaToggle from '../../ui/SakugaToggle.svelte'
  import SakugaSlider from '../../ui/SakugaSlider.svelte'
  import { fireSakugaFx } from '../../../lib/sakuga-fx'

  let status = $state<ChargerStatusResponse>({
    config: {
      enabled: false,
      start_percent: 70,
      stop_percent: 80,
      selected_switch: '',
      custom_path: '',
      custom_on: '1',
      custom_off: '0',
    },
    detected_path: '',
    detected_type: '',
    charging_disabled: false,
    current_level: 0,
    battery: {
      capacity: 0,
      status: 'Unknown',
      current_now: 0,
      voltage_now: 0,
      temperature: 0,
      power_supply: 'unknown',
      is_online: false,
      is_idle: false,
    },
    available_switches: [],
    probe_results: [],
    logs: [],
  })

  let config = $state<ChargerConfig>({
    enabled: false,
    start_percent: 70,
    stop_percent: 80,
    selected_switch: '',
    custom_path: '',
    custom_on: '1',
    custom_off: '0',
  })

  let isLoading = $state(false)
  let isSaving = $state(false)
  let isToggling = $state(false)
  let isProbing = $state(false)
  let lastUpdated = $state<string>('')

  // Derived states for telemetry
  let isIdleBypass = $derived(
    status.battery.is_idle ||
      (status.battery.is_online &&
        status.charging_disabled &&
        Math.abs(status.battery.current_now) <= 30)
  )

  let isCharging = $derived(
    !status.charging_disabled &&
      (status.battery.status?.toLowerCase().includes('charge') ||
        (status.battery.is_online && status.battery.current_now > 15))
  )

  let isDischarging = $derived(
    status.battery.status?.toLowerCase().includes('discharg') ||
      (!status.battery.is_online && status.battery.current_now < -15)
  )

  let isOverheating = $derived(status.battery.temperature > 42)

  // Sakuga FX: burst once when charging genuinely starts (a state
  // transition, never per poll). The null guard skips the first run, so
  // opening the tab mid-charge stays silent.
  let batteryCardEl: HTMLDivElement | null = $state(null)
  let prevCharging: boolean | null = $state(null)
  $effect(() => {
    const now = isCharging
    if (prevCharging === false && now && batteryCardEl) {
      fireSakugaFx(batteryCardEl, 'charge')
    }
    prevCharging = now
  })

  let currentRateColor = $derived(
    isIdleBypass
      ? 'text-emerald-400'
      : isCharging
        ? 'text-amber-400'
        : status.battery.current_now < -800
          ? 'text-red-400'
          : 'text-zinc-200'
  )

  let currentRateLabel = $derived(
    isIdleBypass
      ? $LL.charger.rate.idleBypass()
      : isCharging
        ? $LL.charger.rate.charging()
        : isDischarging
          ? $LL.charger.rate.discharging()
          : $LL.charger.rate.standby()
  )

  let activeProfileObj = $derived(
    status.available_switches?.find((s) => s.id === config.selected_switch) ||
      status.active_profile
  )

  let successfulProbes = $derived(
    status.probe_results?.filter((r) => r.success) ?? []
  )

  onMount(() => {
    fetchChargerStatus(true)
    const timer = setInterval(() => {
      fetchChargerStatus(false)
    }, 4000)

    return () => {
      clearInterval(timer)
    }
  })

  async function fetchChargerStatus(initial = false) {
    try {
      if (initial) isLoading = true
      const res = await api.get<ChargerStatusResponse>('/api/charger/config')
      status = res

      if (initial) {
        config = {
          enabled: res.config?.enabled ?? false,
          start_percent: res.config?.start_percent ?? 70,
          stop_percent: res.config?.stop_percent ?? 80,
          selected_switch:
            res.config?.selected_switch ?? res.active_profile?.id ?? '',
          custom_path: res.config?.custom_path ?? '',
          custom_on: res.config?.custom_on ?? '1',
          custom_off: res.config?.custom_off ?? '0',
        }
      }
      lastUpdated = new Date().toLocaleTimeString()
    } catch (err: unknown) {
      if (initial) {
        toastStore.error(
          err instanceof Error ? err.message : $LL.charger.toasts.fetchFailed()
        )
      }
    } finally {
      if (initial) isLoading = false
    }
  }

  async function saveConfig() {
    try {
      isSaving = true
      if (config.start_percent >= config.stop_percent) {
        config.start_percent = Math.max(0, config.stop_percent - 5)
      }

      const res = await api.post<ChargerStatusResponse>('/api/charger/toggle', config)
      status = res
      config = {
        enabled: res.config?.enabled ?? false,
        start_percent: res.config?.start_percent ?? 70,
        stop_percent: res.config?.stop_percent ?? 80,
        selected_switch:
          res.config?.selected_switch ?? res.active_profile?.id ?? '',
        custom_path: res.config?.custom_path ?? '',
        custom_on: res.config?.custom_on ?? '1',
        custom_off: res.config?.custom_off ?? '0',
      }
      toastStore.success($LL.charger.toasts.saved())
      await sysinfoStore.refresh()
    } catch (err: unknown) {
      toastStore.error(
        err instanceof Error ? err.message : $LL.charger.toasts.saveFailed()
      )
    } finally {
      isSaving = false
    }
  }

  async function toggleMasterEnabled() {
    config.enabled = !config.enabled
    await saveConfig()
  }

  async function toggleManualCut() {
    try {
      isToggling = true
      const res = await api.post<ChargerStatusResponse>(
        '/api/charger/toggle?action=manual',
        {}
      )
      status = res
      if (res.charging_disabled) {
        toastStore.warning(
          $LL.charger.toasts.cutoffApplied()
        )
      } else {
        toastStore.success($LL.charger.toasts.restored())
      }
      await sysinfoStore.refresh()
    } catch (err: unknown) {
      toastStore.error(
        err instanceof Error ? err.message : $LL.charger.toasts.toggleFailed()
      )
    } finally {
      isToggling = false
    }
  }

  async function runProbe() {
    try {
      isProbing = true
      toastStore.info($LL.charger.toasts.probing())
      const results = await api.post<ProbeResult[]>('/api/charger/probe', {})
      status.probe_results = results
      const passCount = results.filter((r) => r.success).length
      if (passCount > 0) {
        toastStore.success(
          $LL.charger.toasts.probeComplete({ pass: passCount, total: results.length })
        )
      } else {
        toastStore.warning(
          $LL.charger.toasts.probeNone({ total: results.length })
        )
      }
      await fetchChargerStatus(false)
    } catch (err: unknown) {
      toastStore.error(
        err instanceof Error ? err.message : $LL.charger.toasts.probeFailed()
      )
    } finally {
      isProbing = false
    }
  }

  function applyPreset(stop: number, start: number) {
    config.stop_percent = stop
    config.start_percent = start
  }

  function onStopPercentChange(val: number) {
    config.stop_percent = Math.min(100, Math.max(20, val))
    if (config.start_percent >= config.stop_percent) {
      config.start_percent = Math.max(0, config.stop_percent - 5)
    }
  }

  function onStartPercentChange(val: number) {
    config.start_percent = Math.min(95, Math.max(0, val))
    if (config.start_percent >= config.stop_percent) {
      config.stop_percent = Math.min(100, config.start_percent + 5)
    }
  }

  async function selectProfile(profileId: string) {
    config.selected_switch = profileId
    await saveConfig()
  }

  function formatCurrent(ma: number): string {
    if (Math.abs(ma) < 1) return '0 mA'
    const sign = ma > 0 ? '+' : ''
    return `${sign}${Math.round(ma)} mA`
  }
</script>

<div class="space-y-3 sm:space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
    <div class="flex items-center gap-2.5 flex-wrap">
      <div
        class="w-9 h-9 rounded-lg bg-accent/20 border-2 border-accent flex items-center justify-center text-accent shadow-[2px_2px_0px_#000] dark:shadow-[2px_2px_0px_rgba(255,255,255,0.05)]"
      >
        <BatteryCharging class="w-5 h-5" />
      </div>
      <div>
        <div class="flex items-center gap-2">
          <h2
            class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground"
          >
            {$LL.charger.title()}
          </h2>
          <Badge variant="warning">Hardware ACC</Badge>
          {#if activeProfileObj?.is_bypass}
            <Badge variant="success">{$LL.charger.bypassReady()}</Badge>
          {/if}
        </div>
        <p class="text-xs text-muted font-mono">
          {$LL.charger.subtitle()}
        </p>
      </div>
    </div>

    <div class="flex items-center gap-2">
      {#if lastUpdated}
        <span class="text-[10px] font-mono text-muted hidden md:inline-block">
          {$LL.charger.synced({ time: lastUpdated })}
        </span>
      {/if}
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoading}
        onclick={() => fetchChargerStatus(true)}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoading ? 'animate-spin' : ''}" />
        <span>{$LL.common.refresh()}</span>
      </Button>
    </div>
  </div>

  <!-- CARD 1: Live Battery & Hardware Telemetry -->
  <Card
    title={$LL.charger.telemetry.title()}
    subtitle={$LL.charger.telemetry.subtitle()}
    tone="ice"
  >
    {#snippet action()}
      <div class="flex items-center gap-1.5">
        {#if isIdleBypass}
          <Badge variant="success" class="flex items-center gap-1">
            <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
            {$LL.charger.telemetry.idleBypassBadge()}
          </Badge>
        {:else if isCharging}
          <Badge variant="warning" class="flex items-center gap-1">
            <Zap class="w-3 h-3 text-amber-400" />
            {$LL.charger.telemetry.charging()}
          </Badge>
        {:else if isDischarging}
          <Badge variant="info">{$LL.charger.telemetry.discharging()}</Badge>
        {:else}
          <Badge variant="default">{status.battery.status || $LL.charger.telemetry.standby()}</Badge>
        {/if}

        {#if status.charging_disabled}
          <Badge variant="danger">{$LL.charger.telemetry.cutoffActive()}</Badge>
        {/if}
      </div>
    {/snippet}

    <div class="space-y-5">
      <!-- Battery Meter Graphic & Percentage -->
      <div
        bind:this={batteryCardEl}
        class="p-4 bg-card-sub border-2 border-border rounded-lg shadow-[2px_2px_0px_#000] dark:shadow-[2px_2px_0px_rgba(255,255,255,0.05)] space-y-3"
      >
        <div class="flex items-end justify-between">
          <div class="flex items-center gap-3">
            <div class="relative">
              {#if isCharging}
                <BatteryCharging class="w-10 h-10 text-amber-400 animate-pulse" />
              {:else if status.battery.capacity <= 20}
                <BatteryWarning class="w-10 h-10 text-red-500" />
              {:else}
                <Battery class="w-10 h-10 text-foreground" />
              {/if}
            </div>
            <div>
              <div class="flex items-baseline gap-2">
                <span class="text-4xl sm:text-5xl font-mono font-black text-foreground tracking-tight">
                  {status.battery.capacity >= 0 ? status.battery.capacity : '--'}
                </span>
                <span class="text-xl font-mono font-bold text-muted">%</span>
              </div>
              <span class="text-[11px] font-mono text-muted uppercase font-bold tracking-wider">
                {$LL.charger.telemetry.batteryLevel()}
              </span>
            </div>
          </div>

          <!-- Quick status indicators -->
          <div class="text-right space-y-1">
            <div class="text-xs font-mono font-bold text-foreground">
              {status.battery.is_online ? $LL.charger.telemetry.acPlugged() : $LL.charger.telemetry.batteryOnly()}
            </div>
            <div class="text-[10px] font-mono text-muted uppercase">
              {$LL.charger.telemetry.gateLabel()} {status.charging_disabled ? $LL.charger.telemetry.gateForcedOff() : $LL.charger.telemetry.gateNormal()}
            </div>
          </div>
        </div>

        <!-- Visual Progress Bar with Threshold Markers -->
        <div class="space-y-1 pt-1">
          <div
            class="h-8 w-full bg-black/50 border-2 border-border rounded overflow-hidden relative shadow-[inset_0_2px_4px_rgba(0,0,0,0.5)] flex items-center"
          >
            <!-- Start Limit Indicator Line -->
            {#if config.start_percent > 0 && config.start_percent < 100}
              <div
                class="absolute top-0 bottom-0 w-0.5 bg-emerald-400 z-20 pointer-events-none"
                style="left: {config.start_percent}%;"
                title={$LL.charger.telemetry.resumeTitle({ value: config.start_percent })}
              >
                <div class="w-2 h-2 -ml-0.75 -mt-0.5 bg-emerald-400 rotate-45"></div>
              </div>
            {/if}

            <!-- Stop Limit Indicator Line -->
            {#if config.stop_percent > 0 && config.stop_percent <= 100}
              <div
                class="absolute top-0 bottom-0 w-0.5 bg-amber-400 z-20 pointer-events-none"
                style="left: {config.stop_percent}%;"
                title={$LL.charger.telemetry.stopTitle({ value: config.stop_percent })}
              >
                <div class="w-2 h-2 -ml-0.75 -mt-0.5 bg-amber-400 rotate-45"></div>
              </div>
            {/if}

            <!-- Inner Fill Bar -->
            <div
              class="h-full transition-all duration-500 ease-out {isIdleBypass
                ? 'bg-emerald-500'
                : isCharging
                  ? 'bg-gradient-to-r from-amber-500 via-yellow-400 to-amber-300'
                  : status.battery.capacity <= 20
                    ? 'bg-red-500'
                    : 'bg-accent'}"
              style="width: {Math.max(0, Math.min(100, status.battery.capacity))}%;"
            ></div>

            <!-- Overlay Text on Bar -->
            <div
              class="absolute inset-0 flex items-center justify-between px-3 text-[10px] font-mono font-bold text-white drop-shadow pointer-events-none"
            >
              <span>0%</span>
              <span class="text-zinc-300 font-normal">
                {$LL.charger.telemetry.overlay({ start: config.start_percent, stop: config.stop_percent })}
              </span>
              <span>100%</span>
            </div>
          </div>

          <div class="flex justify-between text-[10px] font-mono text-muted px-1">
            <span class="flex items-center gap-1 text-emerald-400">
              <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
              {$LL.charger.telemetry.startThreshold({ value: config.start_percent })}
            </span>
            <span class="flex items-center gap-1 text-amber-400">
              <span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
              {$LL.charger.telemetry.stopThreshold({ value: config.stop_percent })}
            </span>
          </div>
        </div>
      </div>

      <!-- Overheating Alert Banner (if > 42°C) -->
      {#if isOverheating}
        <div
          class="p-3 bg-red-950/50 border-2 border-red-500 rounded-lg text-red-200 font-mono text-xs flex items-center justify-between animate-pulse shadow-[2px_2px_0px_#000]"
        >
          <div class="flex items-center gap-2.5">
            <Flame class="w-5 h-5 text-red-500 shrink-0" />
            <div>
              <span class="font-bold text-red-400 uppercase tracking-wider block">
                {$LL.charger.telemetry.thermalTitle({ temp: status.battery.temperature.toFixed(1) })}
              </span>
              <span class="text-[11px] text-red-300">
                {$LL.charger.telemetry.thermalBody()}
              </span>
            </div>
          </div>
          {#if !status.charging_disabled}
            <Button
              variant="danger"
              size="sm"
              onclick={toggleManualCut}
              disabled={isToggling}
            >
              {$LL.charger.telemetry.cutPowerNow()}
            </Button>
          {/if}
        </div>
      {/if}

      <!-- 4 Metrik Grid -->
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <!-- Metric 1: Current Now -->
        <div
          class="p-3 bg-card-sub border-2 border-border rounded-lg shadow-[2px_2px_0px_#000] dark:shadow-[2px_2px_0px_rgba(255,255,255,0.05)]"
        >
          <div
            class="flex items-center justify-between text-[10px] font-mono text-muted uppercase font-bold"
          >
            <span>{$LL.charger.telemetry.currentFlow()}</span>
            <Activity class="w-3.5 h-3.5 text-accent" />
          </div>
          <div class="text-xl sm:text-2xl font-mono font-black mt-1 {currentRateColor}">
            {formatCurrent(status.battery.current_now)}
          </div>
          <div class="text-[10px] font-mono text-muted mt-0.5 truncate">
            {currentRateLabel}
          </div>
        </div>

        <!-- Metric 2: Voltage Now -->
        <div
          class="p-3 bg-card-sub border-2 border-border rounded-lg shadow-[2px_2px_0px_#000] dark:shadow-[2px_2px_0px_rgba(255,255,255,0.05)]"
        >
          <div
            class="flex items-center justify-between text-[10px] font-mono text-muted uppercase font-bold"
          >
            <span>{$LL.charger.telemetry.voltage()}</span>
            <Zap class="w-3.5 h-3.5 text-cyan-400" />
          </div>
          <div class="text-xl sm:text-2xl font-mono font-black text-cyan-400 mt-1">
            {status.battery.voltage_now > 0
              ? status.battery.voltage_now.toFixed(2)
              : '--'} V
          </div>
          <div class="text-[10px] font-mono text-muted mt-0.5 truncate">
            {$LL.charger.telemetry.busPotential()}
          </div>
        </div>

        <!-- Metric 3: Temperature -->
        <div
          class="p-3 bg-card-sub border-2 border-border rounded-lg shadow-[2px_2px_0px_#000] dark:shadow-[2px_2px_0px_rgba(255,255,255,0.05)] {isOverheating
            ? 'border-red-500 bg-red-950/20'
            : ''}"
        >
          <div
            class="flex items-center justify-between text-[10px] font-mono text-muted uppercase font-bold"
          >
            <span>{$LL.charger.telemetry.temperature()}</span>
            <Thermometer
              class="w-3.5 h-3.5 {isOverheating
                ? 'text-red-500 animate-bounce'
                : 'text-amber-400'}"
            />
          </div>
          <div
            class="text-xl sm:text-2xl font-mono font-black mt-1 {isOverheating
              ? 'text-red-500'
              : 'text-amber-400'}"
          >
            {status.battery.temperature > 0
              ? status.battery.temperature.toFixed(1)
              : '--'} °C
          </div>
          <div class="text-[10px] font-mono text-muted mt-0.5 truncate">
            {isOverheating ? $LL.charger.telemetry.criticalHigh() : $LL.charger.telemetry.normalOperating()}
          </div>
        </div>

        <!-- Metric 4: Power Supply -->
        <div
          class="p-3 bg-card-sub border-2 border-border rounded-lg shadow-[2px_2px_0px_#000] dark:shadow-[2px_2px_0px_rgba(255,255,255,0.05)]"
        >
          <div
            class="flex items-center justify-between text-[10px] font-mono text-muted uppercase font-bold"
          >
            <span>{$LL.charger.telemetry.powerSource()}</span>
            <Cpu class="w-3.5 h-3.5 text-accent" />
          </div>
          <div class="text-base sm:text-lg font-mono font-black text-foreground mt-1 truncate">
            {status.battery.power_supply ? status.battery.power_supply.toUpperCase() : 'BATTERY'}
          </div>
          <div class="text-[10px] font-mono text-muted mt-0.5 truncate">
            {status.battery.is_online ? $LL.charger.telemetry.externalPowered() : $LL.charger.telemetry.unplugged()}
          </div>
        </div>
      </div>

      <!-- Active Switch Detection Strip -->
      <div
        class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-2 p-2.5 bg-black/30 border border-border rounded font-mono text-xs"
      >
        <div class="flex items-center gap-2 overflow-hidden">
          <Layers class="w-4 h-4 text-accent shrink-0" />
          <span class="text-muted">{$LL.charger.telemetry.activeSwitch()}</span>
          <span class="font-bold text-foreground truncate">
            {activeProfileObj?.name || status.detected_type || $LL.charger.telemetry.autoDetection()}
          </span>
          {#if activeProfileObj?.is_bypass}
            <Badge variant="success" class="text-[9px]">{$LL.charger.telemetry.bypass()}</Badge>
          {/if}
        </div>
        {#if status.detected_path}
          <span class="text-[10px] text-muted truncate max-w-xs font-mono">
            {status.detected_path}
          </span>
        {/if}
      </div>
    </div>
  </Card>

  <div class="grid grid-cols-1 lg:grid-cols-2 gap-3 sm:gap-6">
    <!-- CARD 2: Limiter Threshold Settings (Compact) -->
    <Card
      title={$LL.charger.limiter.title()}
      subtitle={$LL.charger.limiter.subtitle()}
      tone="peach"
    >
      {#snippet action()}
        <div class="flex items-center gap-2">
          <Badge variant={config.enabled ? 'success' : 'default'}>
            {config.enabled ? $LL.charger.limiter.autoOn() : $LL.charger.limiter.disabled()}
          </Badge>
          {#if themeStore.useSakugaControls()}
            <SakugaToggle checked={config.enabled} label={$LL.charger.limiter.title()} onchange={toggleMasterEnabled} />
          {:else}
          <button
            type="button"
            onclick={toggleMasterEnabled}
            class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-border transition-colors duration-200 ease-in-out {config.enabled
              ? 'bg-accent border-accent'
              : 'bg-zinc-800'}"
            title={$LL.charger.limiter.toggleTitle()}
          >
            <span
              class="pointer-events-none inline-block h-3.5 w-3.5 transform rounded-full bg-white shadow-md ring-0 transition duration-200 ease-in-out {config.enabled
                ? 'translate-x-4'
                : 'translate-x-0'}"
            ></span>
          </button>
          {/if}
        </div>
      {/snippet}

      <div class="space-y-3.5 font-mono text-xs">
        <!-- Compact Bento: Stop & Resume Sliders Side-by-Side -->
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
          <!-- Stop Threshold -->
          <div class="p-2.5 sm:p-3 bg-card-sub border-2 border-border rounded-lg space-y-1.5 shadow-[2px_2px_0px_var(--neo-shadow)]">
            <div class="flex items-center justify-between">
              <span class="font-bold text-[11px] uppercase text-amber-400 flex items-center gap-1.5">
                <span class="w-2 h-2 rounded-full bg-amber-400 animate-pulse"></span>
                {$LL.charger.limiter.stopCutoff()}
              </span>
              <div class="flex items-center gap-1">
                <input
                  type="number"
                  min="30"
                  max="100"
                  class="w-12 px-1.5 py-0.5 text-center font-black text-xs bg-card border border-border rounded text-foreground font-mono focus:outline-none focus:border-accent"
                  bind:value={config.stop_percent}
                  oninput={(e) => onStopPercentChange(Number((e.target as HTMLInputElement).value))}
                />
                <span class="font-bold text-muted text-[10px]">%</span>
              </div>
            </div>

            {#if themeStore.useSakugaControls()}
              <SakugaSlider bind:value={config.stop_percent} min={50} max={95} step={1} oninput={(e) => onStopPercentChange(Number((e.target as HTMLInputElement).value))} />
            {:else}
            <input
              type="range"
              min="50"
              max="95"
              step="1"
              class="w-full accent-amber-400 cursor-pointer h-1.5 bg-background rounded-lg border border-border"
              bind:value={config.stop_percent}
              oninput={(e) => onStopPercentChange(Number((e.target as HTMLInputElement).value))}
            />
            {/if}
            <div class="flex items-center justify-between text-[9px] text-muted">
              <span>{$LL.charger.limiter.tripsAt({ value: config.stop_percent })}</span>
              <span>{$LL.charger.limiter.bypassAc()}</span>
            </div>
          </div>

          <!-- Resume Threshold -->
          <div class="p-2.5 sm:p-3 bg-card-sub border-2 border-border rounded-lg space-y-1.5 shadow-[2px_2px_0px_var(--neo-shadow)]">
            <div class="flex items-center justify-between">
              <span class="font-bold text-[11px] uppercase text-emerald-400 flex items-center gap-1.5">
                <span class="w-2 h-2 rounded-full bg-emerald-400"></span>
                {$LL.charger.limiter.resumeCharge()}
              </span>
              <div class="flex items-center gap-1">
                <input
                  type="number"
                  min="20"
                  max="95"
                  class="w-12 px-1.5 py-0.5 text-center font-black text-xs bg-card border border-border rounded text-foreground font-mono focus:outline-none focus:border-accent"
                  bind:value={config.start_percent}
                  oninput={(e) => onStartPercentChange(Number((e.target as HTMLInputElement).value))}
                />
                <span class="font-bold text-muted text-[10px]">%</span>
              </div>
            </div>

            {#if themeStore.useSakugaControls()}
              <SakugaSlider bind:value={config.start_percent} min={20} max={90} step={1} oninput={(e) => onStartPercentChange(Number((e.target as HTMLInputElement).value))} />
            {:else}
            <input
              type="range"
              min="20"
              max="90"
              step="1"
              class="w-full accent-emerald-400 cursor-pointer h-1.5 bg-background rounded-lg border border-border"
              bind:value={config.start_percent}
              oninput={(e) => onStartPercentChange(Number((e.target as HTMLInputElement).value))}
            />
            {/if}
            <div class="flex items-center justify-between text-[9px] text-muted">
              <span>{$LL.charger.limiter.reopensAt({ value: config.start_percent })}</span>
              <span>{$LL.charger.limiter.hysteresis({ value: config.stop_percent - config.start_percent })}</span>
            </div>
          </div>
        </div>

        <!-- Quick Presets Row & Save Action -->
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 pt-1 border-t border-border/60">
          <div class="flex items-center gap-1.5 flex-wrap">
            <span class="text-[9px] text-muted uppercase font-bold mr-0.5">{$LL.charger.limiter.presets()}</span>
            <button
              type="button"
              class="px-2 py-1 bg-card-sub hover:bg-card border border-border rounded text-[10px] font-mono font-bold text-foreground transition cursor-pointer {config.stop_percent === 80 && config.start_percent === 70 ? 'border-accent text-accent shadow-[1px_1px_0px_var(--neo-shadow)]' : ''}"
              onclick={() => applyPreset(80, 70)}
            >
              80% / 70% <span class="text-[8px] text-muted">{$LL.charger.limiter.presetDaily()}</span>
            </button>
            <button
              type="button"
              class="px-2 py-1 bg-card-sub hover:bg-card border border-border rounded text-[10px] font-mono font-bold text-foreground transition cursor-pointer {config.stop_percent === 85 && config.start_percent === 75 ? 'border-accent text-accent shadow-[1px_1px_0px_var(--neo-shadow)]' : ''}"
              onclick={() => applyPreset(85, 75)}
            >
              85% / 75% <span class="text-[8px] text-muted">{$LL.charger.limiter.presetMax()}</span>
            </button>
            <button
              type="button"
              class="px-2 py-1 bg-card-sub hover:bg-card border border-border rounded text-[10px] font-mono font-bold text-foreground transition cursor-pointer {config.stop_percent === 65 && config.start_percent === 60 ? 'border-accent text-accent shadow-[1px_1px_0px_var(--neo-shadow)]' : ''}"
              onclick={() => applyPreset(65, 60)}
            >
              65% / 60% <span class="text-[8px] text-muted">{$LL.charger.limiter.presetDesk()}</span>
            </button>
          </div>

          <Button
            variant="primary"
            size="sm"
            disabled={isSaving}
            onclick={saveConfig}
            class="shadow-[2px_2px_0px_var(--neo-shadow)] shrink-0"
          >
            {#if isSaving}
              <RefreshCw class="w-3.5 h-3.5 mr-1.5 animate-spin" />
              <span>{$LL.charger.limiter.saving()}</span>
            {:else}
              <Check class="w-3.5 h-3.5 mr-1.5" />
              <span>{$LL.charger.limiter.saveLimits()}</span>
            {/if}
          </Button>
        </div>

        <!-- Compact Manual Hardware Override Strip -->
        <div class="p-2.5 bg-card-sub/60 border border-border rounded-lg flex items-center justify-between gap-2.5 font-mono">
          <div class="flex items-center gap-2 min-w-0">
            <Zap class="w-4 h-4 text-accent shrink-0" />
            <div class="truncate">
              <div class="flex items-center gap-1.5">
                <span class="font-bold text-[11px] text-foreground uppercase">{$LL.charger.limiter.manualGate()}</span>
                {#if status.charging_disabled}
                  <Badge variant="danger">{$LL.charger.limiter.forcedOff()}</Badge>
                {:else}
                  <Badge variant="success">{$LL.charger.limiter.normalGate()}</Badge>
                {/if}
              </div>
              <span class="text-[9px] text-muted block truncate sm:max-w-none max-w-[180px]">
                {$LL.charger.limiter.manualDesc()}
              </span>
            </div>
          </div>

          <div>
            {#if status.charging_disabled}
              <Button
                variant="primary"
                size="sm"
                disabled={isToggling}
                onclick={toggleManualCut}
                class="shadow-[1px_1px_0px_var(--neo-shadow)] shrink-0 text-xs"
              >
                <Zap class="w-3.5 h-3.5 mr-1" />
                <span>{$LL.charger.limiter.restore()}</span>
              </Button>
            {:else}
              <Button
                variant="danger"
                size="sm"
                disabled={isToggling}
                onclick={toggleManualCut}
                class="shadow-[1px_1px_0px_var(--neo-shadow)] shrink-0 text-xs"
              >
                <ZapOff class="w-3.5 h-3.5 mr-1" />
                <span>{$LL.charger.limiter.cutOff()}</span>
              </Button>
            {/if}
          </div>
        </div>
      </div>
    </Card>

    <!-- CARD 3: Hardware Switch Probing & ACC Profile Selection -->
    <Card
      title={$LL.charger.switches.title()}
      subtitle={$LL.charger.switches.subtitle()}
      tone="lavender"
    >
      {#snippet action()}
        <Button
          variant="outline"
          size="sm"
          disabled={isProbing}
          onclick={runProbe}
        >
          <Search class="w-3.5 h-3.5 mr-1.5 {isProbing ? 'animate-spin' : ''}" />
          <span>{isProbing ? $LL.charger.switches.probing() : $LL.charger.switches.probeSwitches()}</span>
        </Button>
      {/snippet}

      <div class="space-y-5 font-mono text-xs">
        <!-- Switch Selection Dropdown -->
        <div class="space-y-1.5">
          <label
            for="switch-profile-select"
            class="block text-muted uppercase font-bold tracking-wider text-[11px]"
          >
            {$LL.charger.switches.profileLabel()}
          </label>
          <select
            id="switch-profile-select"
            class="neo-input w-full bg-card-sub border-2 border-border rounded px-3 py-2 text-xs font-mono text-foreground focus:outline-none focus:border-accent"
            bind:value={config.selected_switch}
            onchange={saveConfig}
          >
            <option value="">{$LL.charger.switches.autoDefault()}</option>
            {#each status.available_switches as sw}
              <option value={sw.id}>
                {sw.name} {sw.is_bypass ? '★ [BYPASS]' : ''}
              </option>
            {/each}
            <option value="custom">{$LL.charger.switches.customNode()}</option>
          </select>
          <p class="text-[10px] text-muted">
            {$LL.charger.switches.hint()}
          </p>
        </div>

        <!-- Custom Switch Fields -->
        {#if config.selected_switch === 'custom'}
          <div
            class="p-3 bg-card-sub border-2 border-accent/40 rounded-lg space-y-3"
          >
            <div class="flex items-center gap-1.5 text-accent font-bold">
              <Sliders class="w-4 h-4" />
              <span>{$LL.charger.switches.customTitle()}</span>
            </div>

            <div class="space-y-2">
              <Input
                label={$LL.charger.switches.sysfsPath()}
                placeholder="/sys/class/power_supply/battery/charging_enabled"
                bind:value={config.custom_path}
              />
              <div class="grid grid-cols-2 gap-2">
                <Input
                  label={$LL.charger.switches.resumeOnValue()}
                  placeholder="1"
                  bind:value={config.custom_on}
                />
                <Input
                  label={$LL.charger.switches.cutoffOffValue()}
                  placeholder="0"
                  bind:value={config.custom_off}
                />
              </div>
            </div>

            <Button
              variant="secondary"
              size="sm"
              fullWidth
              onclick={saveConfig}
              disabled={isSaving}
            >
              {$LL.charger.switches.applyCustom()}
            </Button>
          </div>
        {:else if activeProfileObj}
          <!-- Profile Details Box -->
          <div class="p-3 bg-card-sub border border-border rounded-lg space-y-2">
            <div class="flex items-center justify-between">
              <span class="font-bold text-foreground">
                {activeProfileObj.name}
              </span>
              {#if activeProfileObj.is_bypass}
                <Badge variant="success">{$LL.charger.switches.trueBypass()}</Badge>
              {:else}
                <Badge variant="default">{$LL.charger.switches.standardCutoff()}</Badge>
              {/if}
            </div>
            <p class="text-[11px] text-muted">
              {activeProfileObj.description}
            </p>
            {#if activeProfileObj.nodes && activeProfileObj.nodes.length > 0}
              <div class="space-y-1 pt-1">
                <span class="text-[9px] uppercase font-bold text-muted block">
                  {$LL.charger.switches.sysfsNodes({ count: activeProfileObj.nodes.length })}
                </span>
                {#each activeProfileObj.nodes as node}
                  <div
                    class="p-1.5 bg-black/40 border border-border rounded text-[10px] text-zinc-300 font-mono flex items-center justify-between"
                  >
                    <span class="truncate max-w-[240px]">{node.path}</span>
                    <span class="text-[9px] text-muted font-bold">
                      ON:{node.on_value || '1'} | OFF:{node.off_value || '0'}
                    </span>
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        {/if}

        <!-- Probe Test Results History -->
        <div class="space-y-2 pt-1">
          <div class="flex items-center justify-between">
            <span class="text-[11px] font-bold uppercase text-muted tracking-wider">
              {$LL.charger.switches.probeResults()}
            </span>
            {#if status.probe_results && status.probe_results.length > 0}
              <span class="text-[10px] text-accent font-bold">
                {$LL.charger.switches.functional({ pass: successfulProbes.length, total: status.probe_results.length })}
              </span>
            {/if}
          </div>

          {#if status.probe_results && status.probe_results.length > 0}
            <div class="space-y-2 max-h-56 overflow-y-auto pr-1">
              {#each status.probe_results as probe}
                <div
                  class="p-2.5 bg-card-sub border-2 {probe.success
                    ? 'border-emerald-600/60 bg-emerald-950/10'
                    : 'border-border'} rounded-lg space-y-1.5 shadow-[1px_1px_0px_#000]"
                >
                  <div class="flex items-center justify-between gap-2">
                    <span class="font-bold text-foreground truncate">
                      {probe.name}
                    </span>
                    {#if probe.success}
                      <Badge variant="success" class="shrink-0 flex items-center gap-1">
                        <CheckCircle2 class="w-3 h-3 text-emerald-400" />
                        {$LL.charger.switches.pass()}
                      </Badge>
                    {:else}
                      <Badge variant="default" class="shrink-0 flex items-center gap-1">
                        <XCircle class="w-3 h-3 text-muted" />
                        {$LL.charger.switches.fail()}
                      </Badge>
                    {/if}
                  </div>

                  <div class="flex items-center justify-between text-[10px] text-muted">
                    <span class="flex items-center gap-1">
                      <span>{$LL.charger.switches.flow()}</span>
                      <strong class="text-foreground">{formatCurrent(probe.current_before)}</strong>
                      <ArrowRight class="w-2.5 h-2.5 text-muted inline" />
                      <strong class={probe.success ? 'text-emerald-400' : 'text-foreground'}>
                        {formatCurrent(probe.current_after)}
                      </strong>
                    </span>
                    {#if probe.success}
                      <span class="text-emerald-400 font-bold">
                        Δ {Math.round(probe.current_after - probe.current_before)} mA
                      </span>
                    {/if}
                  </div>

                  {#if probe.message}
                    <p class="text-[10px] text-muted italic">
                      {probe.message}
                    </p>
                  {/if}

                  {#if probe.success && config.selected_switch !== probe.profile_id}
                    <div class="pt-1">
                      <Button
                        variant="secondary"
                        size="sm"
                        onclick={() => selectProfile(probe.profile_id)}
                      >
                        {$LL.charger.switches.useProfile()}
                      </Button>
                    </div>
                  {/if}
                </div>
              {/each}
            </div>
          {:else}
            <!-- Empty state -->
            <div
              class="p-4 bg-card-sub border-2 border-dashed border-border rounded-lg text-center space-y-2"
            >
              <Search class="w-6 h-6 text-muted mx-auto" />
              <p class="text-xs text-muted">
                {$LL.charger.switches.noProbes()}
              </p>
              <Button
                variant="outline"
                size="sm"
                onclick={runProbe}
                disabled={isProbing}
              >
                {$LL.charger.switches.runBenchmark()}
              </Button>
            </div>
          {/if}
        </div>
      </div>
    </Card>
  </div>

  <!-- CARD 4: Log Controller -->
  <Card
    title={$LL.charger.log.title()}
    subtitle={$LL.charger.log.subtitle()}
    tone="none"
  >
    {#snippet action()}
      <div class="flex items-center gap-2">
        <Badge variant="default" class="text-[10px]">
          {$LL.charger.log.events({ count: status.logs ? status.logs.length : 0 })}
        </Badge>
        <Button
          variant="ghost"
          size="sm"
          onclick={() => fetchChargerStatus(false)}
        >
          <RefreshCw class="w-3.5 h-3.5 mr-1" />
          <span>{$LL.charger.log.refreshLogs()}</span>
        </Button>
      </div>
    {/snippet}

    <div class="space-y-2">
      <div
        class="bg-black/90 border-2 border-border rounded-lg p-3 sm:p-4 font-mono text-xs max-h-52 overflow-y-auto space-y-1.5 shadow-[inset_0_2px_4px_rgba(0,0,0,0.6)]"
      >
        {#if status.logs && status.logs.length > 0}
          {#each status.logs as log}
            <div class="flex items-start gap-2 leading-relaxed">
              <span class="text-zinc-600 select-none">&gt;</span>
              <span
                class={log.includes('Stop') || log.includes('stop')
                  ? 'text-amber-400 font-bold'
                  : log.includes('Resum') || log.includes('start')
                    ? 'text-emerald-400 font-bold'
                    : log.includes('Error') || log.includes('fail')
                      ? 'text-red-400 font-bold'
                      : 'text-zinc-300'}
              >
                {log}
              </span>
            </div>
          {/each}
        {:else}
          <div class="text-muted text-center py-6 text-xs italic">
            {$LL.charger.log.empty()}
          </div>
        {/if}
      </div>
      <div class="flex items-center justify-between text-[10px] font-mono text-muted px-1">
        <span class="flex items-center gap-1">
          <Terminal class="w-3 h-3 text-accent" />
          {$LL.charger.log.service()}
        </span>
        <span>{$LL.charger.log.autoSync()}</span>
      </div>
    </div>
  </Card>
</div>
