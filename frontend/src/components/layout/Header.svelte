<script lang="ts">
  import {
    Menu,
    X,
    LogOut,
    Battery,
    BatteryCharging,
    Thermometer,
    Palette,
    Cpu,
    Clock,
    KeyRound,
  } from '@lucide/svelte'
  import { themeStore } from '../../stores/theme.svelte'
  import { authStore } from '../../stores/auth.svelte'
  import { sysinfoStore } from '../../stores/sysinfo.svelte'
  import { navigationStore } from '../../stores/navigation.svelte'
  import AppearanceModal from './AppearanceModal.svelte'
  import ChangePasswordModal from '../modals/ChangePasswordModal.svelte'
  import TopNav from './TopNav.svelte'

  let showAppearanceModal = $state(false)
  let showChangePasswordModal = $state(false)

  const freeRamText = $derived.by(() => {
    const stats = sysinfoStore.stats
    if (!stats) return ''
    const freeBytes = stats.mem_available || stats.mem_free
    if (!freeBytes) {
      if (stats.mem_used_pct !== undefined) {
        return `RAM: ${(100 - stats.mem_used_pct).toFixed(0)}% Free`
      }
      return ''
    }
    const gb = freeBytes / (1024 * 1024 * 1024)
    if (gb >= 1) return `RAM: ${gb.toFixed(1)} GB Free`
    const mb = freeBytes / (1024 * 1024)
    return `RAM: ${mb.toFixed(0)} MB Free`
  })

  const uptimeText = $derived.by(() => {
    const seconds = sysinfoStore.stats?.uptime
    if (!seconds || seconds <= 0) return ''
    const d = Math.floor(seconds / (3600 * 24))
    const h = Math.floor((seconds % (3600 * 24)) / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    const parts: string[] = []
    if (d > 0) parts.push(`${d}d`)
    if (h > 0) parts.push(`${h}h`)
    if (parts.length === 0 || (d === 0 && m > 0)) parts.push(`${m}m`)
    return parts.join(' ')
  })
</script>

<header class="sticky top-0 z-40 bg-card border-b-2 border-border shadow-neobrutal-sm">
  <div class="max-w-7xl mx-auto px-4 h-14 flex items-center justify-between gap-2">
    <!-- Left: Mobile Menu Toggle & Title -->
    <div class="flex items-center gap-3">
      {#if navigationStore.layout === 'sidebar'}
        <button
          type="button"
          class="md:hidden p-1.5 rounded border border-border text-foreground hover:bg-card-sub cursor-pointer"
          onclick={() => navigationStore.toggleSidebar()}
          aria-label="Toggle navigation"
        >
          {#if navigationStore.sidebarOpen}
            <X class="w-5 h-5" />
          {:else}
            <Menu class="w-5 h-5" />
          {/if}
        </button>
      {/if}

      <div class="flex items-center gap-2">
        <div class="w-7 h-7 bg-accent text-accent-text flex items-center justify-center font-mono font-black text-sm rounded border border-border">
          B
        </div>
        <div>
          <span class="font-mono font-black text-sm tracking-wider uppercase text-foreground">
            BFR WEBUI
          </span>
          <span class="hidden sm:inline-block ml-1 text-[10px] font-mono text-accent font-bold px-1.5 py-0.2 bg-card-sub rounded border border-border">
            GO
          </span>
        </div>
      </div>
    </div>

    <!-- Center Left: TopNav Bar (Desktop, only in topbar layout mode) -->
    {#if navigationStore.layout === 'topbar'}
      <TopNav />
    {/if}

    <!-- Center: Quick Telemetry Pills (Progressive Responsive Display) -->
    <div class="hidden sm:flex items-center gap-2 font-mono text-xs">
      <!-- Battery Pill (sm+) -->
      {#if sysinfoStore.stats && sysinfoStore.stats.battery_level !== undefined}
        <div
          class="flex items-center gap-1.5 px-2 py-1 rounded bg-card-sub border border-border text-foreground select-none"
          title="Battery: {sysinfoStore.stats.battery_level}% ({sysinfoStore.stats.battery_status || 'Discharging'})"
        >
          {#if sysinfoStore.stats.battery_status === 'Charging'}
            <BatteryCharging class="w-3.5 h-3.5 text-emerald-400 animate-pulse" />
          {:else}
            <Battery class="w-3.5 h-3.5 text-muted" />
          {/if}
          <span>{sysinfoStore.stats.battery_level}%</span>
        </div>
      {/if}

      <!-- CPU Temp Pill (sm+) -->
      {#if sysinfoStore.stats?.cpu_temp}
        <div
          class="flex items-center gap-1.5 px-2 py-1 rounded bg-card-sub border border-border text-foreground select-none"
          title="CPU Temperature"
        >
          <Thermometer class="w-3.5 h-3.5 text-amber-400" />
          <span>{sysinfoStore.stats.cpu_temp.toFixed(1)}°C</span>
        </div>
      {/if}

      <!-- Free RAM Pill (md+) -->
      {#if freeRamText}
        <div
          class="hidden md:flex items-center gap-1.5 px-2 py-1 rounded bg-card-sub border border-border text-foreground select-none"
          title="Available Physical RAM"
        >
          <Cpu class="w-3.5 h-3.5 text-accent" />
          <span>{freeRamText}</span>
        </div>
      {/if}

      <!-- Uptime Pill (lg+) -->
      {#if uptimeText}
        <div
          class="hidden lg:flex items-center gap-1.5 px-2 py-1 rounded bg-card-sub border border-border text-foreground select-none"
          title="System Uptime"
        >
          <Clock class="w-3.5 h-3.5 text-blue-400" />
          <span>Up: {uptimeText}</span>
        </div>
      {/if}
    </div>

    <!-- Right: Theme, Style, and Auth Actions -->
    <div class="flex items-center gap-2">
      <!-- Appearance Studio Button -->
      <button
        type="button"
        class="flex items-center gap-1.5 px-2.5 py-1 text-xs font-mono font-bold rounded bg-card-sub border border-border text-foreground hover:border-accent cursor-pointer transition-colors"
        onclick={() => (showAppearanceModal = true)}
        title="Appearance Studio (Theme & Style)"
      >
        <Palette class="w-3.5 h-3.5 text-accent" />
        <span class="hidden sm:inline uppercase text-[11px]">Theme</span>
      </button>

      <!-- Change Password Button -->
      {#if authStore.authenticated}
        <button
          type="button"
          class="p-1.5 rounded bg-card-sub border border-border text-muted hover:text-accent hover:border-accent cursor-pointer transition-colors"
          onclick={() => (showChangePasswordModal = true)}
          title="Change Admin Password"
          aria-label="Change Admin Password"
        >
          <KeyRound class="w-4 h-4" />
        </button>
      {/if}

      <!-- Logout Button (if authenticated) -->
      {#if authStore.authenticated}
        <button
          type="button"
          class="p-1.5 rounded bg-card-sub border border-border text-muted hover:text-red-400 hover:border-red-500 cursor-pointer transition-colors"
          onclick={() => authStore.logout()}
          title="Logout"
          aria-label="Logout"
        >
          <LogOut class="w-4 h-4" />
        </button>
      {/if}
    </div>
  </div>
</header>

<AppearanceModal
  open={showAppearanceModal}
  onclose={() => (showAppearanceModal = false)}
/>
<ChangePasswordModal
  open={showChangePasswordModal}
  onclose={() => (showChangePasswordModal = false)}
/>
