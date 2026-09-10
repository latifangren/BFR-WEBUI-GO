<script lang="ts">
  import { onMount } from 'svelte'
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
    Plus,
    ExternalLink,
    Globe,
    Trash2,
    Edit3,
    Thermometer,
    Clock,
    Network,
    Radio,
  } from '@lucide/svelte'
  import { sysinfoStore } from '../../../stores/sysinfo.svelte'
  import { navigationStore } from '../../../stores/navigation.svelte'
  import { api } from '../../../api/client'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Modal from '../../ui/Modal.svelte'
  import Input from '../../ui/Input.svelte'
  import SparklineWave from '../../ui/SparklineWave.svelte'
  import NoticeBanner from '../../common/NoticeBanner.svelte'

  interface ShortcutItem {
    id: string
    title?: string
    name?: string
    url: string
    icon_url?: string
    icon?: string
    category?: string
    description?: string
    created_at?: number
  }

  // Telemetry Modals State
  let showCpuModal = $state(false)
  let showBatteryModal = $state(false)
  let showNetworkModal = $state(false)

  // Shortcuts State
  let shortcuts = $state<ShortcutItem[]>([])
  let isLoadingShortcuts = $state(false)
  let showAddShortcutModal = $state(false)
  let shortcutForm = $state({
    id: '',
    title: '',
    url: '',
    icon_url: '',
    description: '',
  })

  onMount(() => {
    fetchShortcuts()
  })

  async function fetchShortcuts() {
    try {
      isLoadingShortcuts = true
      const res = await api.get<ShortcutItem[] | { shortcuts: ShortcutItem[] }>('/api/shortcuts/list')
      if (Array.isArray(res)) {
        shortcuts = res
      } else if (res && Array.isArray((res as { shortcuts: ShortcutItem[] }).shortcuts)) {
        shortcuts = (res as { shortcuts: ShortcutItem[] }).shortcuts
      } else {
        shortcuts = []
      }
    } catch (e) {
      console.error('Failed to fetch shortcuts:', e)
    } finally {
      isLoadingShortcuts = false
    }
  }

  function openAddModal() {
    shortcutForm = { id: '', title: '', url: '', icon_url: '', description: '' }
    showAddShortcutModal = true
  }

  function openEditModal(item: ShortcutItem, e: MouseEvent) {
    e.stopPropagation()
    e.preventDefault()
    shortcutForm = {
      id: item.id,
      title: item.title || item.name || '',
      url: item.url,
      icon_url: item.icon_url || item.icon || '',
      description: item.description || '',
    }
    showAddShortcutModal = true
  }

  async function saveShortcut() {
    if (!shortcutForm.title.trim() || !shortcutForm.url.trim()) return
    try {
      let targetUrl = shortcutForm.url.trim()
      if (!/^https?:\/\//i.test(targetUrl)) {
        targetUrl = 'http://' + targetUrl
      }
      const payload = {
        id: shortcutForm.id || undefined,
        title: shortcutForm.title.trim(),
        url: targetUrl,
        icon_url: shortcutForm.icon_url.trim(),
      }
      const res = await api.post<{ success: boolean; shortcuts: ShortcutItem[] }>('/api/shortcuts/save', payload)
      if (res && Array.isArray(res.shortcuts)) {
        shortcuts = res.shortcuts
      } else {
        await fetchShortcuts()
      }
      showAddShortcutModal = false
      shortcutForm = { id: '', title: '', url: '', icon_url: '', description: '' }
    } catch (e) {
      console.error('Failed to save shortcut:', e)
    }
  }

  async function deleteShortcut(id: string, e: MouseEvent) {
    e.stopPropagation()
    e.preventDefault()
    if (!confirm('Are you sure you want to delete this shortcut?')) return
    try {
      const res = await api.post<{ success: boolean; shortcuts: ShortcutItem[] }>('/api/shortcuts/delete', { id })
      if (res && Array.isArray(res.shortcuts)) {
        shortcuts = res.shortcuts
      } else {
        await fetchShortcuts()
      }
    } catch (e) {
      console.error('Failed to delete shortcut:', e)
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

  function formatUrlDomain(urlStr: string): string {
    try {
      const parsed = new URL(urlStr)
      return parsed.host || urlStr
    } catch {
      return urlStr.replace(/^https?:\/\//i, '').split('/')[0] || urlStr
    }
  }

  function isImageUrl(val?: string): boolean {
    if (!val) return false
    return /^https?:\/\//i.test(val) || /^data:image/i.test(val) || /\.(png|jpe?g|svg|webp|ico)(\?.*)?$/i.test(val)
  }

  function isEmoji(val?: string): boolean {
    if (!val) return false
    const trimmed = val.trim()
    return trimmed.length <= 4 && !/^[a-zA-Z0-9_\-\.]+$/.test(trimmed)
  }

  const stats = $derived(sysinfoStore.stats)
</script>

<div class="space-y-6">
  <!-- Retro System Notice Banner -->
  <NoticeBanner
    title="KERNEL SUPERVISOR ONLINE"
    message={`Connected to Android root subsystem (${stats?.model || 'Android Device'} • Linux ${stats?.kernel || 'Kernel'}). Hardware telemetry & telemetry sensors active.`}
    icon="bulb"
  />

  <!-- Bento Quick Gauges (5 Cards) -->
  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
    <!-- CPU Gauge (Interactive: Click for Core & Thermal details) -->
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="neo-card neo-tone-ice bg-card p-4 rounded flex flex-col justify-between cursor-pointer hover:border-accent transition-colors group select-none overflow-hidden"
      onclick={() => (showCpuModal = true)}
      title="Click to inspect CPU Cores, Frequencies & Thermal sensors"
    >
      <div>
        <div class="flex items-center justify-between text-muted mb-2 font-mono text-xs">
          <span class="font-bold uppercase tracking-wider group-hover:text-accent transition-colors flex items-center gap-1">
            CPU Load
            <span class="text-[10px] text-accent font-mono opacity-80">↗</span>
          </span>
          <div class="neo-icon-box">
            <Cpu class="w-4 h-4 text-accent" />
          </div>
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
      <SparklineWave value={stats ? stats.cpu_usage : 0} color="var(--color-mint, var(--neo-accent, #3b82f6))" height={26} class="mt-2" />
    </div>

    <!-- RAM Gauge -->
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="neo-card neo-tone-lavender bg-card p-4 rounded flex flex-col justify-between cursor-pointer hover:border-accent transition-colors select-none overflow-hidden"
      onclick={() => navigationStore.setTab('sysinfo')}
      title="Click to open full System & RAM information"
    >
      <div>
        <div class="flex items-center justify-between text-muted mb-2 font-mono text-xs">
          <span class="font-bold uppercase tracking-wider">Memory</span>
          <div class="neo-icon-box">
            <HardDrive class="w-4 h-4 text-purple-400" />
          </div>
        </div>
        <div class="flex items-baseline justify-between">
          <span class="text-2xl font-black font-mono text-foreground">
            {stats ? stats.mem_used_pct.toFixed(0) : '0'}%
          </span>
          <div class="text-right font-mono">
            <div class="text-xs font-bold text-foreground">
              {stats ? formatBytes(stats.mem_used) : '0 B'} / {stats ? formatBytes(stats.mem_total) : '0 B'}
            </div>
            <div class="text-[10px] text-muted">
              {stats ? formatBytes(stats.mem_available || stats.mem_free) : '0 B'} free
            </div>
          </div>
        </div>
        <div class="w-full bg-card-sub border border-border h-1.5 rounded overflow-hidden mt-3">
          <div
            class="h-full bg-purple-500 transition-all duration-300"
            style="width: {stats ? Math.min(stats.mem_used_pct, 100) : 0}%"
          ></div>
        </div>
      </div>
      <SparklineWave value={stats ? stats.mem_used_pct : 0} color="var(--color-lavender, #c084fc)" height={26} class="mt-2" />
    </div>

    <!-- Storage Gauge -->
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="neo-card neo-tone-mint bg-card p-4 rounded flex flex-col justify-between cursor-pointer hover:border-accent transition-colors select-none overflow-hidden"
      onclick={() => navigationStore.setTab('files')}
      title="Click to open File Manager"
    >
      <div>
        <div class="flex items-center justify-between text-muted mb-2 font-mono text-xs">
          <span class="font-bold uppercase tracking-wider">Storage</span>
          <div class="neo-icon-box">
            <HardDrive class="w-4 h-4 text-blue-400" />
          </div>
        </div>
        <div class="flex items-baseline justify-between">
          <span class="text-2xl font-black font-mono text-foreground">
            {stats ? stats.disk_used_pct.toFixed(0) : '0'}%
          </span>
          <div class="text-right font-mono">
            <div class="text-xs font-bold text-foreground">
              {stats ? formatBytes(stats.disk_used) : '0 B'} / {stats ? formatBytes(stats.disk_total) : '0 B'}
            </div>
            <div class="text-[10px] text-muted">
              {stats ? formatBytes(stats.disk_free) : '0 B'} free
            </div>
          </div>
        </div>
        <div class="w-full bg-card-sub border border-border h-1.5 rounded overflow-hidden mt-3">
          <div
            class="h-full bg-blue-500 transition-all duration-300"
            style="width: {stats ? Math.min(stats.disk_used_pct, 100) : 0}%"
          ></div>
        </div>
      </div>
      <SparklineWave value={stats ? stats.disk_used_pct : 0} color="var(--color-ice, #60a5fa)" height={26} class="mt-2" />
    </div>

    <!-- Battery Gauge (Interactive: Click for Voltage, Current & Health) -->
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="neo-card neo-tone-peach bg-card p-4 rounded flex flex-col justify-between cursor-pointer hover:border-accent transition-colors group select-none overflow-hidden"
      onclick={() => (showBatteryModal = true)}
      title="Click to view Battery Voltage, Current & Health diagnostics"
    >
      <div>
        <div class="flex items-center justify-between text-muted mb-2 font-mono text-xs">
          <span class="font-bold uppercase tracking-wider group-hover:text-accent transition-colors flex items-center gap-1">
            Battery
            <span class="text-[10px] text-accent font-mono opacity-80">↗</span>
          </span>
          <div class="neo-icon-box">
            {#if stats?.battery_status === 'Charging'}
              <BatteryCharging class="w-4 h-4 text-emerald-400 animate-pulse" />
            {:else}
              <Battery class="w-4 h-4 text-emerald-400" />
            {/if}
          </div>
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
      <SparklineWave value={stats ? stats.battery_level : 0} color="var(--color-mint, #34d399)" height={26} class="mt-2" />
    </div>

    <!-- Uptime Gauge -->
    <div class="neo-card neo-tone-butter bg-card p-4 rounded flex flex-col justify-between font-mono select-none overflow-hidden">
      <div>
        <div class="flex items-center justify-between text-muted mb-2 text-xs">
          <span class="font-bold uppercase tracking-wider">Uptime</span>
          <div class="neo-icon-box">
            <Zap class="w-4 h-4 text-cyan-400" />
          </div>
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
      <SparklineWave value={stats ? Math.min((stats.uptime / (3600 * 24 * 7)) * 100, 100) : 50} color="var(--color-butter, #eab308)" height={26} class="mt-2" />
    </div>
  </div>

  <!-- Application Shortcuts (CasaOS Style) -->
  <Card title="Application Shortcuts" tone="ice">
    {#snippet action()}
      <Button
        variant="secondary"
        size="sm"
        onclick={openAddModal}
        title="Add new application shortcut"
      >
        <Plus class="w-3.5 h-3.5 mr-1" />
        <span>Add Shortcut</span>
      </Button>
    {/snippet}

    {#if isLoadingShortcuts && shortcuts.length === 0}
      <div class="p-8 text-center text-xs font-mono text-muted">
        <Activity class="w-5 h-5 animate-spin mx-auto mb-2 text-accent" />
        Loading application shortcuts...
      </div>
    {:else if shortcuts.length === 0}
      <div class="p-6 text-center border-2 border-dashed border-border rounded-lg space-y-3 font-mono">
        <div class="w-10 h-10 rounded-full bg-card-sub border border-border flex items-center justify-center mx-auto text-muted">
          <Globe class="w-5 h-5 text-accent" />
        </div>
        <div>
          <h4 class="text-xs font-bold uppercase text-foreground">No Shortcuts Configured</h4>
          <p class="text-[11px] text-muted max-w-sm mx-auto mt-1">
            Pin shortcuts to external services or web dashboards like Clash, AdGuard Home, OpenWrt, Portainer, or local network IPs.
          </p>
        </div>
        <Button variant="primary" size="sm" onclick={openAddModal}>
          <Plus class="w-3.5 h-3.5 mr-1" />
          <span>Add First Shortcut</span>
        </Button>
      </div>
    {:else}
      <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3 font-mono">
        {#each shortcuts as item}
          <div class="neo-card bg-card p-3 rounded flex flex-col justify-between items-center text-center relative group hover:border-accent transition-all h-36">
            <!-- Edit / Delete Floating Controls -->
            <div class="absolute top-1.5 right-1.5 flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity bg-card-sub/90 rounded p-0.5 border border-border z-10 shadow-sm">
              <button
                type="button"
                class="p-1 text-muted hover:text-foreground cursor-pointer transition-colors"
                title="Edit Shortcut"
                onclick={(e) => openEditModal(item, e)}
              >
                <Edit3 class="w-3 h-3" />
              </button>
              <button
                type="button"
                class="p-1 text-muted hover:text-red-400 cursor-pointer transition-colors"
                title="Delete Shortcut"
                onclick={(e) => deleteShortcut(item.id, e)}
              >
                <Trash2 class="w-3 h-3" />
              </button>
            </div>

            <!-- Clickable Link Surface -->
            <a
              href={item.url}
              target="_blank"
              rel="noopener noreferrer"
              class="flex flex-col items-center justify-center flex-1 w-full gap-2 cursor-pointer pt-1"
            >
              {#if isImageUrl(item.icon_url || item.icon)}
                <img
                  src={item.icon_url || item.icon}
                  alt={item.title || item.name}
                  class="w-10 h-10 object-contain rounded p-1 bg-card-sub border border-border"
                />
              {:else if isEmoji(item.icon_url || item.icon)}
                <div class="w-10 h-10 rounded bg-card-sub border border-border flex items-center justify-center text-xl">
                  {item.icon_url || item.icon}
                </div>
              {:else}
                <div class="w-10 h-10 rounded bg-card-sub border border-border flex items-center justify-center text-accent">
                  <Globe class="w-5 h-5" />
                </div>
              {/if}

              <div class="w-full">
                <span class="text-xs font-bold text-foreground truncate block group-hover:text-accent transition-colors">
                  {item.title || item.name || 'Shortcut'}
                </span>
                <span class="text-[10px] text-muted truncate block">
                  {formatUrlDomain(item.url)}
                </span>
              </div>
            </a>
          </div>
        {/each}
      </div>
    {/if}
  </Card>

  <!-- Quick Action Jump Cards -->
  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
    <!-- Network & Traffic Telemetry Card (Interactive) -->
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="neo-button bg-card hover:bg-card-sub p-4 rounded border-2 border-border text-left flex items-center justify-between cursor-pointer hover:border-accent group select-none transition-colors"
      onclick={() => (showNetworkModal = true)}
      title="Click to inspect Network Interface & Telemetry"
      role="button"
      tabindex="0"
    >
      <div class="flex items-center gap-3">
        <div class="p-2.5 rounded bg-card-sub border border-border text-accent group-hover:scale-105 transition-transform">
          <Activity class="w-5 h-5" />
        </div>
        <div>
          <h4 class="font-mono font-bold text-xs uppercase text-foreground flex items-center gap-1 group-hover:text-accent transition-colors">
            Network & Traffic
            <span class="text-[10px] text-accent font-mono opacity-80">↗</span>
          </h4>
          <p class="text-[11px] font-mono text-muted truncate max-w-[150px] sm:max-w-[180px]">
            {#if stats?.net_rx || stats?.net_tx}
              ↓ {formatBytes(stats.net_rx || 0)} ↑ {formatBytes(stats.net_tx || 0)}
            {:else if stats?.network_detail?.ip_addresses?.[0]}
              {stats.network_detail.ip_addresses[0]}
            {:else}
              DNS, TTL & sysctl
            {/if}
          </p>
        </div>
      </div>
      <ArrowRight class="w-4 h-4 text-muted group-hover:text-foreground transition-colors" />
    </div>

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
          <h4 class="font-mono font-bold text-xs uppercase text-foreground">Clash Core</h4>
          <p class="text-[11px] font-mono text-muted">Routing & Rules</p>
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
  <Card title="Device Status & Core Daemons" tone="butter">
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
        <span class="text-[10px] text-muted uppercase font-bold">SELinux Mode</span>
        <p class="font-bold text-foreground capitalize">{stats?.selinux || 'Enforcing'}</p>
      </div>
    </div>
  </Card>
</div>

<!-- CPU Detail Telemetry Modal -->
<Modal
  open={showCpuModal}
  title="CPU Architecture & Per-Core Telemetry"
  onclose={() => (showCpuModal = false)}
  class="!max-w-2xl"
>
  <div class="space-y-4 font-mono text-xs">
    <!-- Summary Header Cards -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
      <div class="p-2.5 rounded bg-card-sub border border-border space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold block">Overall Load</span>
        <span class="text-lg font-black text-accent">{stats ? stats.cpu_usage.toFixed(1) : 0}%</span>
      </div>
      <div class="p-2.5 rounded bg-card-sub border border-border space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold block">CPU Temp</span>
        <span class="text-lg font-black text-amber-400">{stats?.cpu_temp ? stats.cpu_temp.toFixed(1) + '°C' : 'N/A'}</span>
      </div>
      <div class="p-2.5 rounded bg-card-sub border border-border space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold block">Governor</span>
        <span class="text-xs font-bold text-foreground truncate block">{stats?.governor || 'schedutil'}</span>
      </div>
      <div class="p-2.5 rounded bg-card-sub border border-border space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold block">Chipset / SoC</span>
        <span class="text-xs font-bold text-foreground truncate block">{stats?.soc || 'ARM64'}</span>
      </div>
    </div>

    <!-- Load Average Strip -->
    {#if stats?.load_avg}
      <div class="p-2.5 rounded bg-card-sub border border-border flex items-center justify-between flex-wrap gap-2 text-[11px]">
        <span class="text-muted font-bold uppercase text-[10px]">Load Average:</span>
        <div class="flex items-center gap-3">
          <span>1m: <strong class="text-foreground">{stats.load_avg.one.toFixed(2)}</strong></span>
          <span>5m: <strong class="text-foreground">{stats.load_avg.five.toFixed(2)}</strong></span>
          <span>15m: <strong class="text-foreground">{stats.load_avg.fifteen.toFixed(2)}</strong></span>
        </div>
      </div>
    {/if}

    <!-- Per Core Frequency & Usage Grid -->
    <div class="space-y-2">
      <div class="flex items-center justify-between text-[11px] font-bold uppercase text-muted">
        <span>Active CPU Cores ({stats?.cpu_cores?.length || 0})</span>
        <span>Frequency & Load</span>
      </div>
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
        {#each stats?.cpu_cores || [] as core}
          <div class="p-2 rounded bg-card-sub border border-border space-y-1.5">
            <div class="flex items-center justify-between">
              <span class="font-bold text-accent">Core #{core.core}</span>
              <span class="text-[10px] text-muted font-bold">{(core.freq_mhz / 1000).toFixed(2)} GHz</span>
            </div>
            <div class="w-full bg-card border border-border h-1.5 rounded overflow-hidden">
              <div
                class="h-full bg-accent transition-all duration-300"
                style="width: {Math.min(core.usage, 100)}%"
              ></div>
            </div>
            <div class="flex items-center justify-between text-[10px] text-muted">
              <span>{core.freq_mhz} MHz</span>
              <span class="text-foreground font-bold">{core.usage.toFixed(0)}%</span>
            </div>
          </div>
        {/each}
      </div>
    </div>

    <!-- Thermal Sensors / Zones -->
    {#if stats?.thermals && stats.thermals.length > 0}
      <div class="space-y-2">
        <span class="text-[11px] font-bold uppercase text-muted block">Thermal Sensors</span>
        <div class="grid grid-cols-2 sm:grid-cols-3 gap-2 max-h-36 overflow-y-auto pr-1">
          {#each stats.thermals as tz}
            <div class="p-2 rounded bg-card-sub border border-border flex items-center justify-between">
              <span class="text-[10px] text-muted truncate max-w-[120px]">{tz.name}</span>
              <span class="text-[11px] font-bold {tz.temp > 65 ? 'text-red-400' : tz.temp > 50 ? 'text-amber-400' : 'text-emerald-400'}">
                {tz.temp.toFixed(1)}°C
              </span>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  </div>

  {#snippet footer()}
    <Button
      variant="secondary"
      size="sm"
      onclick={() => {
        showCpuModal = false
        navigationStore.setTab('sysinfo')
      }}
    >
      <span>Open Sysinfo Tab</span>
      <ArrowRight class="w-3.5 h-3.5 ml-1" />
    </Button>
    <Button variant="primary" size="sm" onclick={() => (showCpuModal = false)}>
      <span>Close</span>
    </Button>
  {/snippet}
</Modal>

<!-- Battery Detail Telemetry Modal -->
<Modal
  open={showBatteryModal}
  title="Battery Diagnostics & Health"
  onclose={() => (showBatteryModal = false)}
  class="!max-w-xl"
>
  <div class="space-y-4 font-mono text-xs">
    <!-- State Banner -->
    <div class="p-4 rounded bg-card-sub border border-border flex items-center justify-between">
      <div class="flex items-center gap-3">
        <div class="w-12 h-12 rounded bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 flex items-center justify-center">
          {#if stats?.battery_status === 'Charging'}
            <BatteryCharging class="w-7 h-7 animate-pulse" />
          {:else}
            <Battery class="w-7 h-7" />
          {/if}
        </div>
        <div>
          <div class="text-xl font-black text-foreground">{stats ? stats.battery_level : 0}%</div>
          <p class="text-[11px] text-muted">
            Status: <span class="text-emerald-400 font-bold">{stats?.battery_status || 'Discharging'}</span>
          </p>
        </div>
      </div>
      <div class="text-right">
        <span class="text-[10px] text-muted uppercase font-bold block">Health Condition</span>
        <span class="text-xs font-bold text-emerald-400 uppercase">
          {stats?.battery_detail?.health || 'Good'}
        </span>
      </div>
    </div>

    <!-- Battery Metrics Grid -->
    <div class="grid grid-cols-2 sm:grid-cols-3 gap-2.5">
      <div class="p-2.5 rounded bg-card-sub border border-border space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold block">Voltage</span>
        <span class="text-sm font-bold text-foreground">
          {stats?.battery_detail?.voltage_mv ? (stats.battery_detail.voltage_mv / 1000).toFixed(3) + ' V' : 'N/A'}
        </span>
      </div>

      <div class="p-2.5 rounded bg-card-sub border border-border space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold block">Current Flow</span>
        <span class="text-sm font-bold {stats?.battery_detail?.current_now && stats.battery_detail.current_now < 0 ? 'text-amber-400' : 'text-emerald-400'}">
          {stats?.battery_detail?.current_now !== undefined ? stats.battery_detail.current_now + ' mA' : 'N/A'}
        </span>
      </div>

      <div class="p-2.5 rounded bg-card-sub border border-border space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold block">Battery Temp</span>
        <span class="text-sm font-bold {stats?.battery_temp && stats.battery_temp > 42 ? 'text-red-400' : 'text-amber-400'}">
          {stats?.battery_temp ? stats.battery_temp.toFixed(1) + ' °C' : 'N/A'}
        </span>
      </div>

      <div class="p-2.5 rounded bg-card-sub border border-border space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold block">Technology</span>
        <span class="text-sm font-bold text-foreground">
          {stats?.battery_detail?.technology || 'Li-poly'}
        </span>
      </div>

      <div class="p-2.5 rounded bg-card-sub border border-border space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold block">Capacity</span>
        <span class="text-sm font-bold text-foreground">
          {stats?.battery_detail?.capacity || stats?.battery_level || 0}%
        </span>
      </div>

      <div class="p-2.5 rounded bg-card-sub border border-border space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold block">Full Charge Max</span>
        <span class="text-sm font-bold text-foreground">
          {stats?.battery_detail?.charge_full ? stats.battery_detail.charge_full + ' mAh' : 'N/A'}
        </span>
      </div>
    </div>

    <!-- Battery Saver / Charging Limit Note -->
    <div class="p-3 rounded bg-card-sub border border-border/80 flex items-start gap-2.5">
      <Zap class="w-4 h-4 text-accent shrink-0 mt-0.5" />
      <p class="text-[11px] text-muted leading-relaxed">
        BFR-WEBUI includes a hardware charging limitation controller (stop charging at 80% to protect against battery swelling during 24/7 router deployment).
      </p>
    </div>
  </div>

  {#snippet footer()}
    <Button
      variant="secondary"
      size="sm"
      onclick={() => {
        showBatteryModal = false
        navigationStore.setTab('charger')
      }}
    >
      <span>Charging Limit Controls</span>
      <ArrowRight class="w-3.5 h-3.5 ml-1" />
    </Button>
    <Button variant="primary" size="sm" onclick={() => (showBatteryModal = false)}>
      <span>Close</span>
    </Button>
  {/snippet}
</Modal>

<!-- Network Detail Telemetry Modal -->
<Modal
  open={showNetworkModal}
  title="Network Interface & Telemetry"
  onclose={() => (showNetworkModal = false)}
  class="!max-w-2xl"
>
  <div class="space-y-4 font-mono text-xs">
    <!-- Active Interface & Traffic Header -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
      <div class="p-2.5 rounded bg-card-sub border border-border space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold block">Active Interface</span>
        <span class="text-sm font-black text-accent truncate block">
          {stats?.network?.interface || stats?.network_detail?.interface || (stats?.network_detail?.wifi_ssid ? 'wlan0' : 'rmnet0')}
        </span>
      </div>

      <div class="p-2.5 rounded bg-card-sub border border-border space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold block">Network Mode</span>
        <span class="text-sm font-bold text-emerald-400 truncate block">
          {stats?.network_detail?.sim_slots?.[0]?.network_type || (stats?.network_detail?.wifi_ssid ? 'Wi-Fi 802.11' : 'Cellular LTE/5G')}
        </span>
      </div>

      <div class="p-2.5 rounded bg-card-sub border border-border space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold block">Total Download</span>
        <span class="text-sm font-black text-foreground truncate block">
          {formatBytes(stats?.net_rx || 0)}
        </span>
      </div>

      <div class="p-2.5 rounded bg-card-sub border border-border space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold block">Total Upload</span>
        <span class="text-sm font-black text-foreground truncate block">
          {formatBytes(stats?.net_tx || 0)}
        </span>
      </div>
    </div>

    <!-- IP Addresses & Gateway Details -->
    <div class="p-3 bg-card-sub border border-border rounded space-y-2">
      <div class="flex items-center justify-between border-b border-border/70 pb-1.5">
        <span class="font-bold text-foreground uppercase tracking-wider text-[11px]">IP & Routing Configuration</span>
        <Badge variant="info">IPv4 / IPv6</Badge>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 gap-2 text-[11px]">
        <div class="flex items-center justify-between p-2 rounded bg-card border border-border">
          <span class="text-muted">Primary IPv4:</span>
          <span class="font-bold text-accent select-text">
            {stats?.network_detail?.ip_addresses?.find((ip) => ip.includes('.')) || stats?.network_detail?.ip || '192.168.43.1'}
          </span>
        </div>

        <div class="flex items-center justify-between p-2 rounded bg-card border border-border">
          <span class="text-muted">Default Gateway:</span>
          <span class="font-bold text-foreground select-text">
            {stats?.network_detail?.gateway || '192.168.43.1'}
          </span>
        </div>

        <div class="flex items-center justify-between p-2 rounded bg-card border border-border">
          <span class="text-muted">Subnet Mask:</span>
          <span class="font-bold text-foreground select-text">
            255.255.255.0 (/24)
          </span>
        </div>

        <div class="flex items-center justify-between p-2 rounded bg-card border border-border">
          <span class="text-muted">System MTU:</span>
          <span class="font-bold text-foreground select-text">
            {stats?.mtu || '1500'}
          </span>
        </div>
      </div>

      <!-- IPv6 if present -->
      {#if stats?.network_detail?.ip_addresses?.some((ip) => ip.includes(':'))}
        <div class="p-2 rounded bg-card border border-border flex flex-col sm:flex-row sm:items-center justify-between gap-1 text-[10px]">
          <span class="text-muted">IPv6 Global/Link:</span>
          <span class="font-mono text-cyan-400 font-bold select-text break-all">
            {stats.network_detail.ip_addresses.filter((ip) => ip.includes(':')).join(', ')}
          </span>
        </div>
      {/if}
    </div>

    <!-- DNS & Carrier / Wi-Fi Strip -->
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
      <!-- DNS Resolvers -->
      <div class="p-3 bg-card-sub border border-border rounded space-y-2">
        <div class="flex items-center justify-between border-b border-border/70 pb-1.5">
          <span class="font-bold text-foreground uppercase tracking-wider text-[11px]">DNS Resolvers</span>
          <span class="text-[10px] text-muted">Port 53 / 1053</span>
        </div>
        <div class="space-y-1.5 text-[11px]">
          <div class="flex items-center justify-between">
            <span class="text-muted">Primary DNS:</span>
            <span class="font-bold text-emerald-400 select-text">{stats?.network_detail?.dns1 || stats?.network_detail?.dns?.[0] || '1.1.1.1'}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-muted">Secondary DNS:</span>
            <span class="font-bold text-emerald-400 select-text">{stats?.network_detail?.dns2 || stats?.network_detail?.dns?.[1] || '1.0.0.1'}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-muted">Default TTL:</span>
            <span class="font-bold text-accent">{stats?.default_ttl || '64'}</span>
          </div>
        </div>
      </div>

      <!-- Wi-Fi / Cellular Connection -->
      <div class="p-3 bg-card-sub border border-border rounded space-y-2">
        <div class="flex items-center justify-between border-b border-border/70 pb-1.5">
          <span class="font-bold text-foreground uppercase tracking-wider text-[11px]">Wireless Connection</span>
          <Badge variant="default">{stats?.network_detail?.wifi_ssid ? 'Wi-Fi' : 'Mobile Data'}</Badge>
        </div>
        <div class="space-y-1.5 text-[11px]">
          {#if stats?.network_detail?.wifi_ssid}
            <div class="flex items-center justify-between">
              <span class="text-muted">SSID:</span>
              <span class="font-bold text-accent">{stats.network_detail.wifi_ssid}</span>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-muted">Signal:</span>
              <span class="font-bold text-foreground">{stats.network_detail.wifi_signal || `${stats.network_detail.wifi_rssi || -60} dBm`}</span>
            </div>
          {:else}
            <div class="flex items-center justify-between">
              <span class="text-muted">Carrier:</span>
              <span class="font-bold text-accent">{stats?.network_detail?.sim_slots?.[0]?.operator || 'Active Operator'}</span>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-muted">Radio Mode:</span>
              <span class="font-bold text-foreground">{stats?.network_detail?.sim_slots?.[0]?.network_type || '4G LTE-A / 5G'}</span>
            </div>
          {/if}
          <div class="flex items-center justify-between">
            <span class="text-muted">Connected Clients:</span>
            <span class="font-bold text-cyan-400">{stats?.network_detail?.hotspot_clients ?? 0} devices</span>
          </div>
        </div>
      </div>
    </div>
  </div>

  {#snippet footer()}
    <div class="flex items-center justify-between w-full">
      <Button
        variant="outline"
        size="sm"
        onclick={() => {
          showNetworkModal = false
          navigationStore.setTab('network')
        }}
      >
        <ArrowRight class="w-3.5 h-3.5 mr-1" />
        <span>Open Network Tuning</span>
      </Button>

      <Button
        variant="primary"
        size="sm"
        onclick={() => (showNetworkModal = false)}
      >
        Close
      </Button>
    </div>
  {/snippet}
</Modal>

<!-- Add / Edit Shortcut Modal -->
<Modal
  open={showAddShortcutModal}
  title={shortcutForm.id ? 'Edit Shortcut' : 'Add App Shortcut'}
  onclose={() => (showAddShortcutModal = false)}
  class="!max-w-md"
>
  <form onsubmit={(e) => { e.preventDefault(); saveShortcut(); }} class="space-y-3 font-mono text-xs">
    <Input
      label="Application Name"
      placeholder="e.g. Clash Web, AdGuard, OpenWrt"
      bind:value={shortcutForm.title}
    />

    <Input
      label="Target URL"
      placeholder="http://192.168.43.1:9090 or https://..."
      bind:value={shortcutForm.url}
    />

    <Input
      label="Icon (Emoji or Image URL)"
      placeholder="e.g. 🌐, 🚀, 🛡️, ⚙️ or https://.../icon.png"
      bind:value={shortcutForm.icon_url}
    />

    <!-- Quick Emoji Presets -->
    <div class="space-y-1">
      <span class="text-[10px] font-bold text-muted uppercase block">Quick Icon Presets</span>
      <div class="flex items-center gap-1.5 flex-wrap">
        {#each ['🌐', '🛡️', '⚙️', '📊', '📡', '🚀', '⚡', '💻', '📁', '🏠'] as emoji}
          <button
            type="button"
            class="w-7 h-7 rounded bg-card-sub border border-border hover:border-accent flex items-center justify-center text-sm cursor-pointer transition-colors"
            onclick={() => (shortcutForm.icon_url = emoji)}
          >
            {emoji}
          </button>
        {/each}
      </div>
    </div>
  </form>

  {#snippet footer()}
    <Button
      variant="secondary"
      size="sm"
      onclick={() => (showAddShortcutModal = false)}
    >
      Cancel
    </Button>
    <Button
      variant="primary"
      size="sm"
      disabled={!shortcutForm.title.trim() || !shortcutForm.url.trim()}
      onclick={saveShortcut}
    >
      {shortcutForm.id ? 'Save Changes' : 'Add Shortcut'}
    </Button>
  {/snippet}
</Modal>
