<script lang="ts">
  import { fly } from 'svelte/transition'
  import { cubicOut } from 'svelte/easing'
  import { onMount } from 'svelte'
  import Header from './components/layout/Header.svelte'
  import Sidebar from './components/layout/Sidebar.svelte'
  import BottomPopoverNav from './components/layout/BottomPopoverNav.svelte'
  import LoginCard from './components/layout/LoginCard.svelte'
  import { TAB_COMPONENTS, DEFAULT_TAB_COMPONENT } from './components/tabs/registry'
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

  const ActiveTab = $derived(
    TAB_COMPONENTS[navigationStore.activeTab] ?? DEFAULT_TAB_COMPONENT
  )
</script>

{#snippet tabContent()}
  {#key navigationStore.activeTab}
    <div in:fly={{ y: 8, duration: 160, easing: cubicOut }} class="w-full">
      <ActiveTab />
    </div>
  {/key}
{/snippet}

{#if !authStore.authenticated}
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
