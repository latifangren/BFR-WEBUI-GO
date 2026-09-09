<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Cpu,
    Battery,
    BatteryCharging,
    Thermometer,
    RefreshCw,
    ShieldAlert,
    Sliders,
    Check,
    Activity,
  } from '@lucide/svelte'
  import { sysinfoStore } from '../../../stores/sysinfo.svelte'
  import { toastStore } from '../../../stores/toast.svelte'
  import { api } from '../../../api/client'
  import Card from '../../ui/Card.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Button from '../../ui/Button.svelte'

  interface GovernorData {
    current: string
    available: string[]
    cores?: { core: number; governor: string; cur_freq: string }[]
  }

  let governorData = $state<GovernorData>({ current: '', available: [] })
  let selectedGovernor = $state('')
  let isUpdatingGovernor = $state(false)

  onMount(() => {
    fetchGovernor()
  })

  async function fetchGovernor() {
    try {
      const res = await api.get<GovernorData>('/api/sysinfo/governor')
      if (res) {
        governorData = res
        selectedGovernor = res.current || ''
      }
    } catch {
      // Ignored
    }
  }

  async function applyGovernor() {
    if (!selectedGovernor) return
    try {
      isUpdatingGovernor = true
      const res = await api.post<{ success: boolean; error?: string }>('/api/sysinfo/governor', {
        governor: selectedGovernor,
      })
      if (res && res.success) {
        governorData.current = selectedGovernor
        toastStore.success(`CPU Governor successfully changed to ${selectedGovernor}`)
      } else {
        toastStore.error(res?.error || 'Failed to update CPU Governor')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Error updating governor')
    } finally {
      isUpdatingGovernor = false
    }
  }

  function formatBytes(bytes: number): string {
    if (!bytes || bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`
  }

  function formatUptime(seconds: number): string {
    if (!seconds || seconds <= 0) return '0m'
    const d = Math.floor(seconds / (3600 * 24))
    const h = Math.floor((seconds % (3600 * 24)) / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    const parts = []
    if (d > 0) parts.push(`${d}d`)
    if (h > 0) parts.push(`${h}h`)
    parts.push(`${m}m`)
    return parts.join(' ')
  }

  const stats = $derived(sysinfoStore.stats)
</script>

<div class="space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Cpu class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        Hardware & System Telemetry
      </h2>
    </div>
    <div class="flex items-center gap-2">
      <Button
        variant="secondary"
        size="sm"
        disabled={sysinfoStore.isLoading}
        onclick={() => sysinfoStore.refresh()}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {sysinfoStore.isLoading ? 'animate-spin' : ''}" />
        <span>Refresh</span>
      </Button>
    </div>
  </div>

  {#if sysinfoStore.error}
    <div class="bg-red-950/60 border-2 border-red-800 text-red-300 p-4 rounded font-mono text-xs flex items-center gap-3">
      <ShieldAlert class="w-5 h-5 text-red-500 shrink-0" />
      <div>
        <p class="font-bold uppercase">Telemetry Error</p>
        <p>{sysinfoStore.error}</p>
      </div>
    </div>
  {/if}

  {#if !stats && sysinfoStore.isLoading}
    <div class="neo-card bg-card p-12 text-center font-mono text-sm text-muted">
      <RefreshCw class="w-6 h-6 animate-spin mx-auto mb-3 text-accent" />
      Loading hardware counters and telemetry...
    </div>
  {:else if stats}
    <!-- Grid 1: Device Details & Quick Metrics -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Device Specs Bento Box -->
      <Card
        title="Device Specifications & Hardware Info"
        class="lg:col-span-2"
      >
        <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3 font-mono text-xs">
          <div class="bg-card-sub border border-border p-3 rounded space-y-1">
            <span class="text-[10px] text-muted uppercase font-bold">Device Model</span>
            <div class="text-xs font-bold text-foreground truncate" title={stats.model}>
              {stats.model || 'Android Device'}
            </div>
          </div>

          <div class="bg-card-sub border border-border p-3 rounded space-y-1">
            <span class="text-[10px] text-muted uppercase font-bold">Android Version</span>
            <div class="text-xs font-bold text-foreground">
              Android {stats.android_ver || 'N/A'} (API {stats.sdk_ver || 'N/A'})
            </div>
          </div>

          <div class="bg-card-sub border border-border p-3 rounded space-y-1">
            <span class="text-[10px] text-muted uppercase font-bold">Kernel</span>
            <div class="text-xs font-bold text-foreground truncate" title={stats.kernel}>
              {stats.kernel || 'Linux'}
            </div>
          </div>

          <div class="bg-card-sub border border-border p-3 rounded space-y-1">
            <span class="text-[10px] text-muted uppercase font-bold">Security Patch</span>
            <div class="text-xs font-bold text-foreground">
              {stats.security_patch || 'N/A'}
            </div>
          </div>

          <div class="bg-card-sub border border-border p-3 rounded space-y-1">
            <span class="text-[10px] text-muted uppercase font-bold">Resolution & Density</span>
            <div class="text-xs font-bold text-foreground">
              {stats.resolution || 'Unknown'} @ {stats.density || 'N/A'}
            </div>
          </div>

          <div class="bg-card-sub border border-border p-3 rounded space-y-1">
            <span class="text-[10px] text-muted uppercase font-bold">System Uptime</span>
            <div class="text-xs font-bold text-foreground">
              {formatUptime(stats.uptime)}
            </div>
          </div>
        </div>
      </Card>

      <!-- Battery & Power State -->
      <Card title="Battery & Power Subsystem">
        <div class="space-y-4 font-mono text-xs">
          <div class="flex items-center justify-between">
            <span class="text-muted">Charge Level</span>
            <div class="flex items-center gap-1.5">
              {#if stats.battery_status === 'Charging'}
                <BatteryCharging class="w-4 h-4 text-emerald-400" />
              {:else}
                <Battery class="w-4 h-4 text-muted" />
              {/if}
              <span class="text-sm font-bold text-foreground">{stats.battery_level}%</span>
            </div>
          </div>

          <!-- Progress bar -->
          <div class="w-full bg-card-sub border border-border h-3 rounded overflow-hidden">
            <div
              class="h-full transition-all duration-300 {stats.battery_level > 20 ? 'bg-emerald-500' : 'bg-red-500'}"
              style="width: {stats.battery_level}%"
            ></div>
          </div>

          <div class="grid grid-cols-2 gap-2 pt-2 border-t border-border">
            <div>
              <span class="text-[10px] text-muted uppercase font-bold">Status</span>
              <p class="font-bold text-foreground">{stats.battery_status || 'Discharging'}</p>
            </div>
            <div>
              <span class="text-[10px] text-muted uppercase font-bold">Health</span>
              <p class="font-bold text-emerald-400">{stats.battery_detail?.health || 'Good'}</p>
            </div>
            <div>
              <span class="text-[10px] text-muted uppercase font-bold">Voltage</span>
              <p class="font-bold text-foreground">
                {stats.battery_detail?.voltage_mv ? (stats.battery_detail.voltage_mv / 1000).toFixed(2) + ' V' : 'N/A'}
              </p>
            </div>
            <div>
              <span class="text-[10px] text-muted uppercase font-bold">Battery Temp</span>
              <p class="font-bold text-amber-400">
                {stats.battery_temp ? stats.battery_temp.toFixed(1) + '°C' : stats.battery_detail?.temp ? stats.battery_detail.temp.toFixed(1) + '°C' : 'N/A'}
              </p>
            </div>
          </div>
        </div>
      </Card>
    </div>

    <!-- Grid 2: CPU, Thermals & Memory -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- CPU & Per-Core Telemetry -->
      <Card title="CPU Core Activity & Thermals">
        <div class="space-y-4 font-mono text-xs">
          <!-- Overall CPU Usage -->
          <div class="flex items-center justify-between">
            <span class="text-muted">Total CPU Utilization</span>
            <div class="flex items-center gap-2">
              {#if stats.cpu_temp}
                <span class="text-amber-400 font-bold flex items-center gap-1">
                  <Thermometer class="w-3.5 h-3.5" />
                  {stats.cpu_temp.toFixed(1)}°C
                </span>
              {/if}
              <span class="text-sm font-bold text-foreground">{stats.cpu_usage.toFixed(1)}%</span>
            </div>
          </div>

          <div class="w-full bg-card-sub border border-border h-3 rounded overflow-hidden">
            <div
              class="h-full bg-accent transition-all duration-300"
              style="width: {Math.min(stats.cpu_usage, 100)}%"
            ></div>
          </div>

          <!-- Per Core Grid -->
          {#if stats.cpu_cores && stats.cpu_cores.length > 0}
            <div class="pt-2">
              <span class="text-[10px] text-muted uppercase font-bold block mb-2">
                Active Cores ({stats.cpu_cores.length} Threads)
              </span>
              <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
                {#each stats.cpu_cores as core}
                  <div class="bg-card-sub border border-border p-2 rounded space-y-1">
                    <div class="flex items-center justify-between text-[10px]">
                      <span class="font-bold text-foreground">Core {core.core}</span>
                      <span class="text-muted">{core.usage.toFixed(0)}%</span>
                    </div>
                    <div class="w-full bg-card h-1.5 rounded overflow-hidden">
                      <div
                        class="h-full bg-blue-500"
                        style="width: {Math.min(core.usage, 100)}%"
                      ></div>
                    </div>
                    <div class="text-[9px] text-muted truncate">
                      {core.freq_mhz > 0 ? `${core.freq_mhz.toFixed(0)} MHz` : 'Offline'}
                    </div>
                  </div>
                {/each}
              </div>
            </div>
          {/if}
        </div>
      </Card>

      <!-- Memory & Swap Utilization -->
      <Card title="RAM & Swap Memory">
        <div class="space-y-5 font-mono text-xs">
          <!-- Physical RAM -->
          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <span class="text-muted">Physical RAM ({stats.mem_used_pct.toFixed(1)}%)</span>
              <span class="font-bold text-foreground">
                {formatBytes(stats.mem_used)} / {formatBytes(stats.mem_total)}
              </span>
            </div>
            <div class="w-full bg-card-sub border border-border h-3 rounded overflow-hidden">
              <div
                class="h-full bg-purple-500 transition-all duration-300"
                style="width: {Math.min(stats.mem_used_pct, 100)}%"
              ></div>
            </div>
            <div class="flex items-center justify-between text-[10px] text-muted">
              <span>Free: {formatBytes(stats.mem_free)}</span>
              <span>Available: {formatBytes(stats.mem_available)}</span>
            </div>
          </div>

          <!-- Swap / ZRAM -->
          <div class="space-y-2 pt-3 border-t border-border">
            <div class="flex items-center justify-between">
              <span class="text-muted">ZRAM / Swap ({stats.swap_used_pct.toFixed(1)}%)</span>
              <span class="font-bold text-foreground">
                {formatBytes(stats.swap_used)} / {formatBytes(stats.swap_total)}
              </span>
            </div>
            <div class="w-full bg-card-sub border border-border h-3 rounded overflow-hidden">
              <div
                class="h-full bg-cyan-500 transition-all duration-300"
                style="width: {Math.min(stats.swap_used_pct, 100)}%"
              ></div>
            </div>
          </div>
        </div>
      </Card>
    </div>

    <!-- CPU Scaling Governor & Thermal Summary -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- CPU Scaling Governor -->
      <Card title="CPU Scaling Governor Policy">
        <div class="space-y-4 font-mono text-xs">
          <div class="flex items-center justify-between">
            <span class="text-muted">Active Governor Policy</span>
            <span class="px-2 py-0.5 rounded bg-accent/15 text-accent font-bold uppercase text-[10px] border border-accent/30 flex items-center gap-1">
              <Check class="w-3 h-3" />
              {governorData.current || stats.governor || 'schedutil'}
            </span>
          </div>

          <div class="space-y-1.5">
            <label for="governor-select" class="text-[10px] uppercase font-bold text-muted block">
              Available Governors
            </label>
            <div class="flex items-center gap-2">
              <select
                id="governor-select"
                bind:value={selectedGovernor}
                class="neo-input flex-1 bg-card-sub border border-border rounded px-3 py-2 text-xs font-mono text-foreground focus:outline-none focus:border-accent"
              >
                {#if governorData.available.length === 0}
                  <option value="schedutil">schedutil (Energy-Aware EAS)</option>
                  <option value="performance">performance (Max Clock)</option>
                  <option value="powersave">powersave (Battery Saver)</option>
                  <option value="conservative">conservative</option>
                {:else}
                  {#each governorData.available as gov}
                    <option value={gov}>{gov} {gov === (governorData.current || stats.governor) ? '(Active)' : ''}</option>
                  {/each}
                {/if}
              </select>

              <Button
                variant="primary"
                size="sm"
                disabled={isUpdatingGovernor || selectedGovernor === (governorData.current || stats.governor)}
                onclick={applyGovernor}
              >
                {isUpdatingGovernor ? 'Applying...' : 'Apply'}
              </Button>
            </div>
          </div>

          <p class="text-[10px] text-muted leading-relaxed">
            * <strong>schedutil</strong>: Dynamic Energy-Aware Scheduling.<br />
            * <strong>performance</strong>: Locks CPU to maximum frequency for lowest latency.<br />
            * <strong>powersave</strong>: Limits CPU clock to preserve battery and reduce heat.
          </p>
        </div>
      </Card>

      <!-- Thermal Sensors Monitor -->
      <Card title="Thermal Sensors Telemetry">
        <div class="space-y-3 font-mono text-xs">
          <div class="flex items-center justify-between">
            <span class="text-muted">Primary SoC Temp</span>
            <span class="font-bold text-sm {stats.cpu_temp > 65 ? 'text-red-400' : stats.cpu_temp > 50 ? 'text-amber-400' : 'text-emerald-400'}">
              {stats.cpu_temp ? stats.cpu_temp.toFixed(1) + '°C' : 'N/A'}
            </span>
          </div>

          {#if stats.thermals && stats.thermals.length > 0}
            <div class="grid grid-cols-2 gap-2 max-h-40 overflow-y-auto pr-1">
              {#each stats.thermals as tz}
                <div class="p-2 rounded bg-card-sub border border-border flex items-center justify-between">
                  <span class="text-[10px] text-muted truncate max-w-[120px]">{tz.name}</span>
                  <span class="text-[11px] font-bold {tz.temp > 65 ? 'text-red-400' : tz.temp > 50 ? 'text-amber-400' : 'text-emerald-400'}">
                    {tz.temp.toFixed(1)}°C
                  </span>
                </div>
              {/each}
            </div>
          {:else}
            <div class="p-4 text-center text-[11px] text-muted bg-card-sub rounded border border-border">
              Standard thermal zones active.
            </div>
          {/if}
        </div>
      </Card>
    </div>

    <!-- Grid 3: Storage Partitions & Services -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- Disks & Partitions -->
      <Card title="Storage Filesystems">
        {#if stats.disks && stats.disks.length > 0}
          <div class="space-y-3 font-mono text-xs">
            {#each stats.disks as disk}
              <div class="bg-card-sub border border-border p-3 rounded space-y-2">
                <div class="flex items-center justify-between">
                  <span class="font-bold text-foreground">{disk.path}</span>
                  <span class="text-muted text-[11px]">{disk.used_pct.toFixed(1)}% used</span>
                </div>
                <div class="w-full bg-card h-2 rounded overflow-hidden">
                  <div
                    class="h-full {disk.used_pct > 90 ? 'bg-red-500' : 'bg-emerald-500'}"
                    style="width: {Math.min(disk.used_pct, 100)}%"
                  ></div>
                </div>
                <div class="flex items-center justify-between text-[10px] text-muted">
                  <span>Used: {formatBytes(disk.used)}</span>
                  <span>Free: {formatBytes(disk.free)}</span>
                  <span>Total: {formatBytes(disk.total)}</span>
                </div>
              </div>
            {/each}
          </div>
        {:else}
          <p class="font-mono text-xs text-muted">No disk partitions mounted.</p>
        {/if}
      </Card>

      <!-- Background System Services -->
      <Card title="Root Daemon Services">
        {#if stats.active_services && stats.active_services.length > 0}
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-2 font-mono text-xs">
            {#each stats.active_services as svc}
              <div class="bg-card-sub border border-border p-2.5 rounded flex items-center justify-between">
                <div class="space-y-0.5">
                  <span class="font-bold text-foreground">{svc.name}</span>
                  {#if svc.detail}
                    <p class="text-[10px] text-muted truncate max-w-[120px]">{svc.detail}</p>
                  {/if}
                </div>
                <Badge variant={svc.running ? 'success' : 'default'}>
                  {svc.running ? 'Active' : 'Stopped'}
                </Badge>
              </div>
            {/each}
          </div>
        {:else}
          <p class="font-mono text-xs text-muted">No service hooks detected.</p>
        {/if}
      </Card>
    </div>
  {/if}
</div>
