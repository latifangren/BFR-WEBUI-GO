<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Power,
    RotateCcw,
    AlertTriangle,
    ShieldAlert,
    Smartphone,
    Clock,
    CalendarClock,
    CheckCircle,
    RefreshCw,
    Sliders,
    Timer,
    Activity,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import type { PowerAction, ScheduleStatus, ScheduleConfig } from '../../../types/network'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Modal from '../../ui/Modal.svelte'

  // Immediate Power Action State
  let confirmAction = $state<PowerAction | null>(null)
  let isExecuting = $state(false)

  // Scheduled Auto-Reboot State
  let scheduleStatus = $state<ScheduleStatus | null>(null)
  let enabled = $state(false)
  let selectedMode = $state<'interval' | 'clock'>('interval')
  let intervalHours = $state(4)
  let clockTime = $state('04:00')
  let countdownSec = $state(-1)
  let isLoadingSchedule = $state(false)
  let isSavingSchedule = $state(false)

  // Derived Values
  let formattedUptime = $derived(
    scheduleStatus?.uptime_seconds !== undefined
      ? formatDuration(scheduleStatus.uptime_seconds)
      : '0m'
  )

  let formattedCountdown = $derived(
    !enabled || countdownSec < 0
      ? 'Disabled'
      : countdownSec === 0
        ? 'Due now'
        : formatDuration(countdownSec)
  )

  let formattedNextReboot = $derived(
    !enabled
      ? 'Schedule Inactive'
      : scheduleStatus?.next_reboot_time ||
          (selectedMode === 'clock'
            ? `Daily at ${clockTime}`
            : `Every ${intervalHours} hours`)
  )

  // Svelte 5 Effect to keep countdownSec synced when scheduleStatus updates
  $effect(() => {
    if (scheduleStatus) {
      countdownSec = scheduleStatus.next_reboot_in_sec
    }
  })

  // Timer loop for local countdown decrement
  onMount(() => {
    fetchSchedule()
    const timer = setInterval(() => {
      if (countdownSec > 0) {
        countdownSec -= 1
      }
    }, 1000)
    return () => clearInterval(timer)
  })

  function formatDuration(totalSeconds: number): string {
    if (totalSeconds <= 0) return '0m'
    const d = Math.floor(totalSeconds / 86400)
    const h = Math.floor((totalSeconds % 86400) / 3600)
    const m = Math.floor((totalSeconds % 3600) / 60)
    const s = totalSeconds % 60

    const parts: string[] = []
    if (d > 0) parts.push(`${d}d`)
    if (h > 0 || d > 0) parts.push(`${h}h`)
    parts.push(`${m}m`)
    if (d === 0 && h === 0) parts.push(`${s}s`)
    return parts.join(' ')
  }

  async function fetchSchedule() {
    try {
      isLoadingSchedule = true
      const res = await api.get<ScheduleStatus>('/api/power/schedule')
      scheduleStatus = res
      if (res.config) {
        enabled = res.config.enabled
        if (res.config.mode === 'clock') {
          selectedMode = 'clock'
        } else if (res.config.mode === 'interval') {
          selectedMode = 'interval'
        }
        intervalHours = res.config.interval_hours || 4
        clockTime = res.config.clock_time || '04:00'
      }
      countdownSec = res.next_reboot_in_sec
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to fetch reboot schedule')
    } finally {
      isLoadingSchedule = false
    }
  }

  async function saveSchedule() {
    try {
      isSavingSchedule = true
      const payload: ScheduleConfig = {
        enabled,
        mode: enabled ? selectedMode : 'disabled',
        interval_hours: Math.max(1, Number(intervalHours) || 4),
        clock_time: clockTime || '04:00',
      }
      const res = await api.post<ScheduleStatus>('/api/power/schedule', payload)
      scheduleStatus = res
      countdownSec = res.next_reboot_in_sec
      toastStore.success('Reboot schedule saved successfully.')
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to save reboot schedule')
    } finally {
      isSavingSchedule = false
    }
  }

  const actionLabels: Record<PowerAction, { title: string; desc: string; danger: boolean }> = {
    reboot: {
      title: 'Reboot Device',
      desc: 'Performs standard Android OS system reboot via root execution.',
      danger: false,
    },
    recovery: {
      title: 'Reboot to Recovery',
      desc: 'Reboots into TWRP / OrangeFox / Custom Recovery environment.',
      danger: true,
    },
    bootloader: {
      title: 'Reboot to Bootloader',
      desc: 'Reboots into Fastboot / Download mode for flashing.',
      danger: true,
    },
    poweroff: {
      title: 'System Power Off',
      desc: 'Completely shuts down the Android device hardware.',
      danger: true,
    },
  }

  async function executePowerAction() {
    if (!confirmAction) return

    try {
      isExecuting = true
      await api.post('/api/power', { action: confirmAction })
      toastStore.success(`Power command '${confirmAction}' dispatched to Android kernel.`)
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to execute power action')
    } finally {
      isExecuting = false
      confirmAction = null
    }
  }
</script>

<div class="space-y-3 sm:space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Power class="w-5 h-5 text-red-500" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        Power & System Actions
      </h2>
    </div>
  </div>

  <!-- Safety Warning Banner -->
  <div class="bg-amber-950/50 border-2 border-amber-800 text-amber-300 p-4 rounded font-mono text-xs flex items-center gap-3">
    <AlertTriangle class="w-5 h-5 text-amber-400 shrink-0" />
    <div>
      <p class="font-bold uppercase">Root Execution Warning</p>
      <p>
        Power actions execute system-level commands with superuser privileges (`su -c`).
        The WebUI daemon and active services will disconnect immediately upon reboot.
      </p>
    </div>
  </div>

  <!-- Scheduled Auto-Reboot Card -->
  <Card
    title="Scheduled Auto-Reboot"
    subtitle="Periodic or time-based automated reboot cycle"
    tone="lavender"
  >
    {#snippet action()}
      <Badge
        variant={enabled ? 'success' : 'default'}
        class="text-xs px-2.5 py-1 border-2 font-mono"
      >
        <span
          class="inline-block w-2 h-2 rounded-full mr-1.5 {enabled ? 'bg-emerald-400 animate-pulse' : 'bg-muted'}"
        ></span>
        {enabled ? 'SCHEDULE ACTIVE' : 'SCHEDULE DISABLED'}
      </Badge>
    {/snippet}

    <div class="space-y-5 font-mono text-xs">
      <!-- Master Toggle Switch -->
      <div
        class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 p-3.5 bg-card-sub border-2 border-border rounded-md shadow-neobrutal-sm"
      >
        <div class="flex items-start sm:items-center gap-3">
          <div
            class="p-2 rounded border-2 border-border shrink-0 {enabled
              ? 'bg-accent/20 text-accent'
              : 'bg-card text-muted'}"
          >
            <CalendarClock class="w-5 h-5" />
          </div>
          <div>
            <div class="flex items-center gap-2">
              <span class="font-bold text-foreground text-sm">Automated Reboot</span>
              <span
                class="text-[10px] font-mono px-1.5 py-0.5 rounded border border-border bg-card {enabled
                  ? 'text-emerald-400 font-bold'
                  : 'text-muted'}"
              >
                {enabled ? 'ENABLED' : 'DISABLED'}
              </span>
            </div>
            <p class="text-[11px] text-muted mt-0.5">
              Automates kernel reboot cycles to clear memory and refresh cellular modem stack.
            </p>
          </div>
        </div>

        <div class="flex items-center self-end sm:self-center">
          <button
            type="button"
            role="switch"
            aria-checked={enabled}
            onclick={() => (enabled = !enabled)}
            title={enabled ? 'Disable Auto-Reboot' : 'Enable Auto-Reboot'}
            class="relative inline-flex h-7 w-14 shrink-0 cursor-pointer rounded-full border-2 border-border transition-colors duration-200 ease-in-out focus:outline-none {enabled
              ? 'bg-accent shadow-neobrutal-sm'
              : 'bg-card'}"
          >
            <span
              class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white border border-black shadow transition duration-200 ease-in-out mt-[2px] {enabled
                ? 'translate-x-7'
                : 'translate-x-1'}"
            ></span>
          </button>
        </div>
      </div>

      <!-- Mode Selector -->
      <div class="space-y-1.5">
        <span class="block text-[11px] font-bold uppercase text-muted tracking-wider">
          Reboot Trigger Mode
        </span>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
          <button
            type="button"
            onclick={() => (selectedMode = 'interval')}
            class="neo-button flex items-center justify-center gap-2 p-2.5 rounded border-2 text-xs font-mono font-bold transition-all cursor-pointer {selectedMode === 'interval'
              ? 'border-accent bg-accent/15 text-accent shadow-neobrutal-sm'
              : 'border-border bg-card-sub text-muted hover:text-foreground'}"
          >
            <Timer class="w-4 h-4 {selectedMode === 'interval' ? 'text-accent' : 'text-muted'}" />
            <span>Interval sejak Boot</span>
          </button>
          <button
            type="button"
            onclick={() => (selectedMode = 'clock')}
            class="neo-button flex items-center justify-center gap-2 p-2.5 rounded border-2 text-xs font-mono font-bold transition-all cursor-pointer {selectedMode === 'clock'
              ? 'border-accent bg-accent/15 text-accent shadow-neobrutal-sm'
              : 'border-border bg-card-sub text-muted hover:text-foreground'}"
          >
            <Clock class="w-4 h-4 {selectedMode === 'clock' ? 'text-accent' : 'text-muted'}" />
            <span>Jam Tertentu (Setiap Hari)</span>
          </button>
        </div>
      </div>

      <!-- Mode Specific Configuration -->
      {#if selectedMode === 'interval'}
        <div class="space-y-4 p-4 bg-card-sub/60 border-2 border-border rounded-md">
          <div class="space-y-1.5">
            <span class="block text-[11px] font-bold uppercase text-muted tracking-wider">
              Quick Interval Presets
            </span>
            <div class="grid grid-cols-3 sm:grid-cols-6 gap-2">
              {#each [2, 4, 6, 8, 12, 24] as h}
                <button
                  type="button"
                  onclick={() => (intervalHours = h)}
                  class="neo-button py-2 px-1 text-center rounded border-2 font-mono text-xs font-bold transition-all cursor-pointer {intervalHours === h
                    ? 'border-accent bg-accent text-accent-text shadow-neobrutal-sm'
                    : 'border-border bg-card text-foreground hover:border-accent/60'}"
                >
                  {h} Jam
                </button>
              {/each}
            </div>
          </div>

          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pt-2 border-t border-border/60">
            <div class="flex items-center gap-2.5">
              <label for="custom-interval" class="text-xs text-muted whitespace-nowrap font-bold">
                Custom Interval:
              </label>
              <input
                id="custom-interval"
                type="number"
                min="1"
                max="720"
                bind:value={intervalHours}
                class="neo-input w-24 bg-card border-2 border-border rounded px-3 py-1.5 text-sm font-mono font-bold text-foreground focus:outline-none focus:border-accent"
              />
              <span class="text-xs text-muted">Jam Sekali</span>
            </div>
            <span class="text-[11px] text-muted">
              Cycle: setiap {intervalHours} jam dari waktu boot
            </span>
          </div>

          <!-- Interval Info Badges -->
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-2">
            <div class="flex items-center gap-3 p-3 bg-card border-2 border-border rounded">
              <Activity class="w-4 h-4 text-cyan-400 shrink-0" />
              <div>
                <span class="text-muted block text-[10px] uppercase font-bold tracking-wider">Current Uptime</span>
                <span class="font-bold text-foreground text-sm">{formattedUptime}</span>
              </div>
            </div>
            <div class="flex items-center gap-3 p-3 bg-card border-2 border-border rounded">
              <Timer class="w-4 h-4 text-amber-400 shrink-0" />
              <div>
                <span class="text-muted block text-[10px] uppercase font-bold tracking-wider">Next Reboot Countdown</span>
                <span class="font-bold text-sm {enabled && countdownSec > 0 ? 'text-amber-400' : 'text-muted'}">
                  {formattedCountdown}
                </span>
              </div>
            </div>
          </div>
        </div>
      {:else}
        <div class="space-y-4 p-4 bg-card-sub/60 border-2 border-border rounded-md">
          <div class="flex flex-col sm:flex-row sm:items-end gap-4">
            <div class="space-y-1.5">
              <label for="clock-time-input" class="block text-[11px] font-bold uppercase text-muted tracking-wider">
                Waktu Reboot (24 Jam)
              </label>
              <input
                id="clock-time-input"
                type="time"
                bind:value={clockTime}
                class="neo-input bg-card border-2 border-border rounded px-3 py-1.5 text-sm font-mono font-bold text-foreground focus:outline-none focus:border-accent"
              />
            </div>

            <div class="space-y-1.5 flex-1">
              <span class="block text-[11px] font-bold uppercase text-muted tracking-wider">
                Quick Time Presets
              </span>
              <div class="flex flex-wrap gap-2">
                {#each [
                  { time: '03:00', label: '03:00 Subuh' },
                  { time: '04:00', label: '04:00 Subuh' },
                  { time: '05:00', label: '05:00 Subuh' },
                  { time: '00:00', label: '00:00 Malam' }
                ] as chip}
                  <button
                    type="button"
                    onclick={() => (clockTime = chip.time)}
                    class="neo-button text-xs font-mono px-3 py-1.5 rounded border-2 transition-all cursor-pointer {clockTime === chip.time
                      ? 'border-accent bg-accent text-accent-text font-bold shadow-neobrutal-sm'
                      : 'border-border bg-card text-foreground hover:border-accent/60'}"
                  >
                    {chip.label}
                  </button>
                {/each}
              </div>
            </div>
          </div>

          <!-- Clock Info Badges -->
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-2">
            <div class="flex items-center gap-3 p-3 bg-card border-2 border-border rounded">
              <Clock class="w-4 h-4 text-cyan-400 shrink-0" />
              <div>
                <span class="text-muted block text-[10px] uppercase font-bold tracking-wider">Estimated Next Reboot</span>
                <span class="font-bold text-sm {enabled ? 'text-accent' : 'text-muted'}">
                  {formattedNextReboot}
                </span>
              </div>
            </div>
            <div class="flex items-center gap-3 p-3 bg-card border-2 border-border rounded">
              <Timer class="w-4 h-4 text-amber-400 shrink-0" />
              <div>
                <span class="text-muted block text-[10px] uppercase font-bold tracking-wider">Countdown</span>
                <span class="font-bold text-sm {enabled && countdownSec > 0 ? 'text-amber-400' : 'text-muted'}">
                  {formattedCountdown}
                </span>
              </div>
            </div>
          </div>
        </div>
      {/if}

      <!-- Action Footer -->
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pt-3 border-t border-border">
        <div class="flex items-center gap-2">
          {#if scheduleStatus?.description}
            <span class="text-xs text-muted flex items-center gap-1.5">
              <CheckCircle class="w-4 h-4 text-emerald-400 shrink-0" />
              <span>{scheduleStatus.description}</span>
            </span>
          {:else}
            <span class="text-xs text-muted">Configure your preferred auto-reboot interval or time.</span>
          {/if}
        </div>

        <div class="flex items-center gap-2 self-end sm:self-auto">
          <Button
            variant="outline"
            size="md"
            disabled={isLoadingSchedule || isSavingSchedule}
            onclick={fetchSchedule}
            title="Refresh Schedule Status"
          >
            <RefreshCw class="w-4 h-4 mr-2 {isLoadingSchedule ? 'animate-spin' : ''}" />
            <span>Refresh</span>
          </Button>

          <Button
            variant="primary"
            size="md"
            disabled={isLoadingSchedule || isSavingSchedule}
            onclick={saveSchedule}
          >
            {#if isSavingSchedule}
              <RefreshCw class="w-4 h-4 mr-2 animate-spin" />
              <span>Saving...</span>
            {:else}
              <Sliders class="w-4 h-4 mr-2" />
              <span>Save Schedule</span>
            {/if}
          </Button>
        </div>
      </div>
    </div>
  </Card>

  <!-- Power Actions Grid -->
  <div class="grid grid-cols-2 sm:grid-cols-2 lg:grid-cols-4 gap-2.5 sm:gap-4 font-mono text-xs">
    <!-- Standard Reboot -->
    <div class="neo-card neo-organic bg-card p-3 sm:p-4 rounded border-2 border-border flex flex-col justify-between space-y-3">
      <div>
        <div class="flex items-center gap-2 mb-1">
          <RotateCcw class="w-4 h-4 text-accent shrink-0" />
          <h3 class="font-bold text-xs sm:text-sm text-foreground uppercase truncate">Reboot</h3>
        </div>
        <p class="text-[10px] sm:text-[11px] text-muted line-clamp-2">
          Restart Android OS and reload root daemons.
        </p>
      </div>
      <Button
        variant="restart"
        size="sm"
        fullWidth={true}
        onclick={() => (confirmAction = 'reboot')}
      >
        <span>Reboot System</span>
      </Button>
    </div>

    <!-- Recovery Reboot -->
    <div class="neo-card neo-organic bg-card p-3 sm:p-4 rounded border-2 border-border flex flex-col justify-between space-y-3">
      <div>
        <div class="flex items-center gap-2 mb-1">
          <Smartphone class="w-4 h-4 text-amber-400 shrink-0" />
          <h3 class="font-bold text-xs sm:text-sm text-foreground uppercase truncate">Recovery</h3>
        </div>
        <p class="text-[10px] sm:text-[11px] text-muted line-clamp-2">
          Enter custom recovery (TWRP/OrangeFox).
        </p>
      </div>
      <Button
        variant="cyan"
        size="sm"
        fullWidth={true}
        onclick={() => (confirmAction = 'recovery')}
      >
        <span>Recovery Mode</span>
      </Button>
    </div>

    <!-- Bootloader / Fastboot -->
    <div class="neo-card neo-organic bg-card p-3 sm:p-4 rounded border-2 border-border flex flex-col justify-between space-y-3">
      <div>
        <div class="flex items-center gap-2 mb-1">
          <ShieldAlert class="w-4 h-4 text-purple-400 shrink-0" />
          <h3 class="font-bold text-xs sm:text-sm text-foreground uppercase truncate">Fastboot</h3>
        </div>
        <p class="text-[10px] sm:text-[11px] text-muted line-clamp-2">
          Enter bootloader partition for PC USB flashing.
        </p>
      </div>
      <Button
        variant="purple"
        size="sm"
        fullWidth={true}
        onclick={() => (confirmAction = 'bootloader')}
      >
        <span>Fastboot Mode</span>
      </Button>
    </div>

    <!-- Power Off -->
    <div class="neo-card neo-organic bg-card p-3 sm:p-4 rounded border-2 border-border flex flex-col justify-between space-y-3">
      <div>
        <div class="flex items-center gap-2 mb-1">
          <Power class="w-4 h-4 text-red-500 shrink-0" />
          <h3 class="font-bold text-xs sm:text-sm text-foreground uppercase truncate">Power Off</h3>
        </div>
        <p class="text-[10px] sm:text-[11px] text-muted line-clamp-2">
          Halt kernel and turn off device completely.
        </p>
      </div>
      <Button
        variant="stop"
        size="sm"
        fullWidth={true}
        onclick={() => (confirmAction = 'poweroff')}
      >
        <span>Shut Down</span>
      </Button>
    </div>
  </div>

  <!-- Explicit Confirmation Modal -->
  {#if confirmAction}
    <Modal
      open={true}
      title="Confirm Power Action"
      onclose={() => (confirmAction = null)}
    >
      <div class="space-y-4 font-mono text-xs">
        <div class="p-3 bg-red-950/50 border border-red-800 rounded text-red-200">
          <p class="font-bold uppercase text-sm mb-1">
            {actionLabels[confirmAction].title}
          </p>
          <p>{actionLabels[confirmAction].desc}</p>
        </div>

        <p class="text-muted">
          Are you sure you want to proceed? Any unsaved changes in running processes may be lost.
        </p>

        <div class="flex items-center justify-end gap-2 pt-2 border-t border-border">
          <Button
            variant="ghost"
            size="md"
            disabled={isExecuting}
            onclick={() => (confirmAction = null)}
          >
            Cancel
          </Button>
          <Button
            variant="danger"
            size="md"
            disabled={isExecuting}
            onclick={executePowerAction}
          >
            {isExecuting ? 'Executing...' : 'Yes, Proceed'}
          </Button>
        </div>
      </div>
    </Modal>
  {/if}
</div>
