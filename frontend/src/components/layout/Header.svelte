<script lang="ts">
  import {
    Menu,
    X,
    Sun,
    Moon,
    LogOut,
    Battery,
    BatteryCharging,
    Thermometer,
    Sparkles,
  } from '@lucide/svelte'
  import { themeStore } from '../../stores/theme.svelte'
  import { authStore } from '../../stores/auth.svelte'
  import { sysinfoStore } from '../../stores/sysinfo.svelte'
  import { navigationStore } from '../../stores/navigation.svelte'
  import type { ThemeMode, UIStyle } from '../../types/common'

  const themeOptions: { id: ThemeMode; label: string }[] = [
    { id: 'dark', label: 'Dark' },
    { id: 'amoled', label: 'AMOLED' },
    { id: 'light', label: 'Light' },
    { id: 'cyberpunk', label: 'Cyberpunk' },
    { id: 'dracula', label: 'Dracula' },
    { id: 'nord', label: 'Nord' },
  ]

  let showThemeDropdown = $state(false)

  function handleThemeSelect(theme: ThemeMode) {
    themeStore.setTheme(theme)
    showThemeDropdown = false
  }

  function toggleUIStyle() {
    const nextStyle: UIStyle = themeStore.currentStyle === 'neobrutal' ? 'modern' : 'neobrutal'
    themeStore.setStyle(nextStyle)
  }
</script>

<header class="sticky top-0 z-40 bg-card border-b-2 border-border shadow-neobrutal-sm">
  <div class="max-w-7xl mx-auto px-4 h-14 flex items-center justify-between gap-2">
    <!-- Left: Mobile Menu Toggle & Title -->
    <div class="flex items-center gap-3">
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

    <!-- Center: Quick Telemetry Pills (Visible on sm+) -->
    <div class="hidden sm:flex items-center gap-2 font-mono text-xs">
      {#if sysinfoStore.stats && sysinfoStore.stats.battery_level !== undefined}
        <div class="flex items-center gap-1.5 px-2 py-1 rounded bg-card-sub border border-border text-foreground">
          {#if sysinfoStore.stats.battery_status === 'Charging'}
            <BatteryCharging class="w-3.5 h-3.5 text-emerald-400" />
          {:else}
            <Battery class="w-3.5 h-3.5 text-muted" />
          {/if}
          <span>{sysinfoStore.stats.battery_level}%</span>
        </div>
      {/if}

      {#if sysinfoStore.stats?.cpu_temp}
        <div class="flex items-center gap-1.5 px-2 py-1 rounded bg-card-sub border border-border text-foreground">
          <Thermometer class="w-3.5 h-3.5 text-amber-400" />
          <span>{sysinfoStore.stats.cpu_temp.toFixed(1)}°C</span>
        </div>
      {/if}
    </div>

    <!-- Right: Theme, Style, and Auth Actions -->
    <div class="flex items-center gap-2">
      <!-- UI Style Toggle (Neobrutal / Modern) -->
      <button
        type="button"
        class="hidden sm:flex items-center gap-1.5 px-2.5 py-1 text-xs font-mono font-bold rounded bg-card-sub border border-border text-foreground hover:border-accent cursor-pointer transition-colors"
        onclick={toggleUIStyle}
        title="Toggle UI Style"
      >
        <Sparkles class="w-3.5 h-3.5 text-accent" />
        <span class="uppercase text-[11px]">{themeStore.currentStyle}</span>
      </button>

      <!-- Theme Dropdown -->
      <div class="relative">
        <button
          type="button"
          class="p-1.5 rounded bg-card-sub border border-border text-foreground hover:border-accent cursor-pointer"
          onclick={() => (showThemeDropdown = !showThemeDropdown)}
          aria-label="Select theme"
        >
          {#if themeStore.currentTheme === 'light'}
            <Sun class="w-4 h-4 text-amber-400" />
          {:else}
            <Moon class="w-4 h-4 text-blue-400" />
          {/if}
        </button>

        {#if showThemeDropdown}
          <!-- svelte-ignore a11y_click_events_have_key_events -->
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <div
            class="fixed inset-0 z-40"
            onclick={() => (showThemeDropdown = false)}
          ></div>
          <div class="absolute right-0 mt-2 w-36 bg-card border-2 border-border shadow-neobrutal rounded p-1 z-50 font-mono text-xs space-y-0.5">
            {#each themeOptions as opt}
              <button
                type="button"
                class="w-full text-left px-2.5 py-1.5 rounded hover:bg-card-sub flex items-center justify-between cursor-pointer {themeStore.currentTheme === opt.id ? 'font-bold text-accent' : 'text-foreground'}"
                onclick={() => handleThemeSelect(opt.id)}
              >
                <span>{opt.label}</span>
                {#if themeStore.currentTheme === opt.id}
                  <span class="w-1.5 h-1.5 rounded-full bg-accent"></span>
                {/if}
              </button>
            {/each}
          </div>
        {/if}
      </div>

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
