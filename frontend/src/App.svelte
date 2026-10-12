<script lang="ts">
  import { fly } from 'svelte/transition'
  import { cubicOut } from 'svelte/easing'
  import { onMount } from 'svelte'
  import Header from './components/layout/Header.svelte'
  import Sidebar from './components/layout/Sidebar.svelte'
  import BottomPopoverNav from './components/layout/BottomPopoverNav.svelte'
  import LoginCard from './components/layout/LoginCard.svelte'
  import { TAB_LOADERS, DEFAULT_TAB_LOADER, TERMINAL_TAB_LOADER } from './components/tabs/registry'
  import type { TabModule } from './components/tabs/registry'
  import ToastContainer from './components/ui/ToastContainer.svelte'
  import { authStore } from './stores/auth.svelte'
  import { navigationStore, AVAILABLE_TABS } from './stores/navigation.svelte'
  import { sysinfoStore } from './stores/sysinfo.svelte'
  import { themeStore } from './stores/theme.svelte'

  onMount(() => {
    authStore.checkStatus()
    sysinfoStore.init()
  })

  $effect(() => {
    const isFocused = navigationStore.activeTab === 'overview' || navigationStore.activeTab === 'sysinfo'
    sysinfoStore.setTelemetryFocus(isFocused)
  })

  const currentTab = $derived(
    AVAILABLE_TABS.find((t) => t.id === navigationStore.activeTab) || AVAILABLE_TABS[0]
  )

  function getWallpaperBackground(wallpaper: string): string {
    if (!wallpaper || wallpaper === 'preset:none') return 'none'
    if (wallpaper === 'preset:cyber-neon') {
      return 'linear-gradient(135deg, #0d0f18 0%, #1a102f 40%, #0d273a 70%, #05141b 100%)'
    }
    if (wallpaper === 'preset:aurora-mesh') {
      return 'radial-gradient(at 0% 0%, #120b29 0, transparent 50%), radial-gradient(at 50% 0%, #172554 0, transparent 50%), radial-gradient(at 100% 0%, #3b0764 0, transparent 50%), radial-gradient(at 0% 100%, #042f2e 0, transparent 50%), radial-gradient(at 100% 100%, #1e1b4b 0, transparent 50%), #090d16'
    }
    if (wallpaper === 'preset:sunset-glow') {
      return 'linear-gradient(135deg, #1b0a2a 0%, #3d1445 35%, #5e1c43 70%, #200a18 100%)'
    }
    if (wallpaper === 'preset:dark-velvet') {
      return 'linear-gradient(160deg, #18151e 0%, #2b1f38 50%, #110e16 100%)'
    }
    return `url('${wallpaper}')`
  }

  const isGlassWithWallpaper = $derived(
    themeStore.currentStyle === 'glass' &&
      !!themeStore.activeWallpaper &&
      themeStore.activeWallpaper !== 'preset:none'
  )

  // Memoize loader promises per tab id: a snippet re-render must never
  // restart a tab load (new promise identity would remount the component).
  const tabModuleCache = new Map<string, Promise<TabModule>>()
  const activeTabPromise = $derived.by(() => {
    const id = navigationStore.activeTab
    let p = tabModuleCache.get(id)
    if (!p) {
      p = (TAB_LOADERS[id] ?? DEFAULT_TAB_LOADER)()
      tabModuleCache.set(id, p)
    }
    return p
  })

  // Terminal keep-alive: mount the terminal on first visit and keep it mounted
  // (hidden via CSS) across tab switches so the PTY/WebSocket session survives.
  // The loader promise is memoized so the {#await} block below never resets.
  let terminalMounted = $state(false)
  let terminalModulePromise: Promise<TabModule> | null = $state(null)
  $effect(() => {
    if (navigationStore.activeTab === 'terminal' && !terminalModulePromise) {
      terminalMounted = true
      terminalModulePromise = TERMINAL_TAB_LOADER()
    }
  })
</script>

{#snippet tabContent()}
  {#if terminalMounted && terminalModulePromise}
    {#await terminalModulePromise then terminalModule}
      <div class="w-full" class:hidden={navigationStore.activeTab !== 'terminal'}>
        <svelte:component this={terminalModule.default} />
      </div>
    {/await}
  {/if}
  {#if navigationStore.activeTab !== 'terminal'}
    {#key navigationStore.activeTab}
      {#await activeTabPromise}
        <div class="w-full flex items-center justify-center py-16 text-muted">
          <span class="animate-pulse text-sm">Loading…</span>
        </div>
      {:then module}
        <div in:fly={{ y: 8, duration: 160, easing: cubicOut }} class="w-full">
          <svelte:component this={module.default} />
        </div>
      {:catch}
        <div class="w-full text-center py-16 text-red-400 text-sm">
          Failed to load this tab. Please try again.
        </div>
      {/await}
    {/key}
  {/if}
{/snippet}

{#if authStore.authenticated === null}
  <!-- Auth status still unknown: splash instead of flashing the dashboard -->
  <div class="min-h-screen flex items-center justify-center bg-background text-foreground">
    <div class="flex flex-col items-center gap-3">
      <div class="w-8 h-8 rounded-full border-2 border-accent border-t-transparent animate-spin"></div>
      <span class="text-sm text-muted">Connecting…</span>
    </div>
  </div>
{:else if authStore.authenticated === false}
  <!-- Full Screen Authentic Login Page (Matching Legacy WebUI) -->
  <LoginCard />
{:else if navigationStore.layout === 'sidebar'}
  <!-- Sidebar Rail Mode -->
  <div class="flex h-screen overflow-hidden {isGlassWithWallpaper ? 'bg-transparent' : 'bg-background'} text-foreground font-sans relative z-10">
    <Sidebar />
    <div class="flex-1 flex flex-col min-w-0 overflow-hidden">
      <Header />
      <main class="flex-1 overflow-y-auto p-2.5 sm:p-4 md:p-6 custom-scrollbar">
        {@render tabContent()}
      </main>
    </div>
  </div>
{:else}
  <!-- Full-Width TopBar Mode -->
  <div class="min-h-screen flex flex-col {isGlassWithWallpaper ? 'bg-transparent' : 'bg-background'} text-foreground font-sans relative z-10">
    <Header />
    <main class="flex-1 w-full max-w-7xl mx-auto p-2.5 sm:p-4 md:p-6 pb-20 md:pb-6">
      {@render tabContent()}
    </main>
    <BottomPopoverNav />
  </div>
{/if}

{#if isGlassWithWallpaper}
  <div
    class="glass-wallpaper-container fixed inset-0 pointer-events-none z-0"
    style="
      background-image: {getWallpaperBackground(themeStore.activeWallpaper)};
      filter: brightness({themeStore.glassDim});
    "
  ></div>
{/if}

<!-- Global Toast Notifications -->
<ToastContainer />
