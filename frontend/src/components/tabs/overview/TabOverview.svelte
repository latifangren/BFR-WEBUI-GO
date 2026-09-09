<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import {
    Cpu,
    HardDrive,
    Battery,
    BatteryCharging,
    Activity,
    Wifi,
    Power,
    Shield,
    Zap,
    ArrowRight,
  } from '@lucide/svelte'
  import { sysinfoStore } from '../../../stores/sysinfo.svelte'
  import { navigationStore } from '../../../stores/navigation.svelte'
  import Card from '../../ui/Card.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Button from '../../ui/Button.svelte'

  onMount(() => {
    sysinfoStore.startPolling(2000)
  })

  onDestroy(() => {
    sysinfoStore.stopPolling()
  })

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
  <!-- Bento Quick Gauges (5 Cards) -->
  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
    <!-- CPU Gauge -->
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="neo-card bg-card p-4 rounded flex flex-col justify-between cursor-pointer hover:border-accent transition-colors"
      onclick={() => navigationStore.setTab('sysinfo')}
    >
      <div class="flex items-center justify-between text-muted mb-2 font-mono text-xs">
        <span class="font-bold uppercase tracking-wider">CPU Load</span>
        <Cpu class="w-4 h-4 text-accent" />
      </div>
      <div class="flex items-baseline justify-between">
        <span class="text-2xl font-black font-mono text-foreground">
          {stats ? stats.cpu_usage.toFixed(0) : '0'}%
        </span>
        {#if stats?.cpu_temp}
          <span class="text-xs font-mono text-amber-400 font-bold">
            {stats.cpu_temp.toFixed(0)}°C
          </span>
        {/if}
      </div>
      <div class="w-full bg-card-sub border border-border h-1.5 rounded overflow-hidden mt-3">
        <div
          class="h-full bg-accent transition-all duration-300"
          style="width: {stats ? Math.min(stats.cpu_usage, 100) : 0}%"
        ></div>
      </div>
    </div>

    <!-- RAM Gauge -->
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="neo-card bg-card p-4 rounded flex flex-col justify-between cursor-pointer hover:border-accent transition-colors"
      onclick={() => navigationStore.setTab('sysinfo')}
    >
      <div class="flex items-center justify-between text-muted mb-2 font-mono text-xs">
        <span class="font-bold uppercase tracking-wider">RAM Usage</span>
        <Activity class="w-4 h-4 text-purple-400" />
      </div>
      <div class="flex items-baseline justify-between">
        <span class="text-2xl font-black font-mono text-foreground">
          {stats ? stats.mem_used_pct.toFixed(0) : '0'}%
        </span>
        <span class="text-xs font-mono text-muted">
          {stats ? formatBytes(stats.mem_used) : '0 B'}
        </span>
      </div>
      <div class="w-full bg-card-sub border border-border h-1.5 rounded overflow-hidden mt-3">
        <div
          class="h-full bg-purple-500 transition-all duration-300"
          style="width: {stats ? Math.min(stats.mem_used_pct, 100) : 0}%"
        ></div>
      </div>
    </div>

    <!-- Storage Gauge -->
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="neo-card bg-card p-4 rounded flex flex-col justify-between cursor-pointer hover:border-accent transition-colors"
      onclick={() => navigationStore.setTab('sysinfo')}
    >
      <div class="flex items-center justify-between text-muted mb-2 font-mono text-xs">
        <span class="font-bold uppercase tracking-wider">Storage</span>
        <HardDrive class="w-4 h-4 text-blue-400" />
      </div>
      <div class="flex items-baseline justify-between">
        <span class="text-2xl font-black font-mono text-foreground">
          {stats ? stats.disk_used_pct.toFixed(0) : '0'}%
        </span>
        <span class="text-xs font-mono text-muted">
          {stats ? formatBytes(stats.disk_used) : '0 B'}
        </span>
      </div>
      <div class="w-full bg-card-sub border border-border h-1.5 rounded overflow-hidden mt-3">
        <div
          class="h-full bg-blue-500 transition-all duration-300"
          style="width: {stats ? Math.min(stats.disk_used_pct, 100) : 0}%"
        ></div>
      </div>
    </div>

    <!-- Battery Gauge -->
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="neo-card bg-card p-4 rounded flex flex-col justify-between cursor-pointer hover:border-accent transition-colors"
      onclick={() => navigationStore.setTab('charger')}
    >
      <div class="flex items-center justify-between text-muted mb-2 font-mono text-xs">
        <span class="font-bold uppercase tracking-wider">Battery</span>
        {#if stats?.battery_status === 'Charging'}
          <BatteryCharging class="w-4 h-4 text-emerald-400" />
        {:else}
          <Battery class="w-4 h-4 text-emerald-400" />
        {/if}
      </div>
      <div class="flex items-baseline justify-between">
        <span class="text-2xl font-black font-mono text-foreground">
          {stats ? stats.battery_level : '0'}%
        </span>
        <span class="text-xs font-mono text-emerald-400 font-bold">
          {stats?.battery_status || 'OK'}
        </span>
      </div>
      <div class="w-full bg-card-sub border border-border h-1.5 rounded overflow-hidden mt-3">
        <div
          class="h-full bg-emerald-500 transition-all duration-300"
          style="width: {stats ? stats.battery_level : 0}%"
        ></div>
      </div>
    </div>

    <!-- Uptime Gauge -->
    <div class="neo-card bg-card p-4 rounded flex flex-col justify-between font-mono">
      <div class="flex items-center justify-between text-muted mb-2 text-xs">
        <span class="font-bold uppercase tracking-wider">Uptime</span>
        <Zap class="w-4 h-4 text-cyan-400" />
      </div>
      <div class="flex items-baseline justify-between">
        <span class="text-xl font-black text-foreground">
          {stats ? formatUptime(stats.uptime) : '0m'}
        </span>
      </div>
      <div class="text-[10px] text-muted mt-3 truncate">
        Kernel: {stats ? stats.kernel : 'Linux'}
      </div>
    </div>
  </div>

  <!-- Quick Action Jump Cards -->
  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
    <button
      type="button"
      class="neo-button bg-card hover:bg-card-sub p-4 rounded border-2 border-border text-left flex items-center justify-between cursor-pointer group"
      onclick={() => navigationStore.setTab('network')}
    >
      <div class="flex items-center gap-3">
        <div class="p-2.5 rounded bg-card-sub border border-border text-accent group-hover:scale-105 transition-transform">
          <Activity class="w-5 h-5" />
        </div>
        <div>
          <h4 class="font-mono font-bold text-xs uppercase text-foreground">Network & DNS</h4>
          <p class="text-[11px] font-mono text-muted">DNS, TTL & sysctl</p>
        </div>
      </div>
      <ArrowRight class="w-4 h-4 text-muted group-hover:text-foreground transition-colors" />
    </button>

    <button
      type="button"
      class="neo-button bg-card hover:bg-card-sub p-4 rounded border-2 border-border text-left flex items-center justify-between cursor-pointer group"
      onclick={() => navigationStore.setTab('hotspot')}
    >
      <div class="flex items-center gap-3">
        <div class="p-2.5 rounded bg-card-sub border border-border text-emerald-400 group-hover:scale-105 transition-transform">
          <Wifi class="w-5 h-5" />
        </div>
        <div>
          <h4 class="font-mono font-bold text-xs uppercase text-foreground">SoftAP Hotspot</h4>
          <p class="text-[11px] font-mono text-muted">Clients & Tethering</p>
        </div>
      </div>
      <ArrowRight class="w-4 h-4 text-muted group-hover:text-foreground transition-colors" />
    </button>

    <button
      type="button"
      class="neo-button bg-card hover:bg-card-sub p-4 rounded border-2 border-border text-left flex items-center justify-between cursor-pointer group"
      onclick={() => navigationStore.setTab('proxy')}
    >
      <div class="flex items-center gap-3">
        <div class="p-2.5 rounded bg-card-sub border border-border text-purple-400 group-hover:scale-105 transition-transform">
          <Shield class="w-5 h-5" />
        </div>
        <div>
          <h4 class="font-mono font-bold text-xs uppercase text-foreground">Clash / Mihomo</h4>
          <p class="text-[11px] font-mono text-muted">Proxy core & routing</p>
        </div>
      </div>
      <ArrowRight class="w-4 h-4 text-muted group-hover:text-foreground transition-colors" />
    </button>

    <button
      type="button"
      class="neo-button bg-card hover:bg-card-sub p-4 rounded border-2 border-border text-left flex items-center justify-between cursor-pointer group"
      onclick={() => navigationStore.setTab('power')}
    >
      <div class="flex items-center gap-3">
        <div class="p-2.5 rounded bg-card-sub border border-border text-red-400 group-hover:scale-105 transition-transform">
          <Power class="w-5 h-5" />
        </div>
        <div>
          <h4 class="font-mono font-bold text-xs uppercase text-foreground">Power Actions</h4>
          <p class="text-[11px] font-mono text-muted">Reboot, Recovery</p>
        </div>
      </div>
      <ArrowRight class="w-4 h-4 text-muted group-hover:text-foreground transition-colors" />
    </button>
  </div>

  <!-- System Summary Card -->
  <Card title="Device Status & Core Daemons">
    <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-3 font-mono text-xs">
      <div class="bg-card-sub border border-border p-3 rounded space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold">Device</span>
        <p class="font-bold text-foreground truncate">{stats?.model || 'Android Device'}</p>
      </div>

      <div class="bg-card-sub border border-border p-3 rounded space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold">Android / SDK</span>
        <p class="font-bold text-foreground">
          Android {stats?.android_ver || 'N/A'} (API {stats?.sdk_ver || 'N/A'})
        </p>
      </div>

      <div class="bg-card-sub border border-border p-3 rounded space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold">Security Patch</span>
        <p class="font-bold text-foreground">{stats?.security_patch || 'N/A'}</p>
      </div>

      <div class="bg-card-sub border border-border p-3 rounded space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold">Active Daemons</span>
        <p class="font-bold text-emerald-400">
          {stats?.active_services?.filter((s) => s.running).length || 0} Running
        </p>
      </div>
    </div>
  </Card>
</div>
