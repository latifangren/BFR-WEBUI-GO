<script lang="ts">
  import { onDestroy } from 'svelte'
  import { fly } from 'svelte/transition'
  import {
    LayoutDashboard,
    Cpu,
    Network,
    Wifi,
    Shield,
    Radio,
    MessageSquare,
    Power,
    BatteryCharging,
    Box,
    Terminal,
    Smartphone,
    FolderOpen,
    FileText,
    Gauge,
    Activity,
    HardDrive,
    Globe,
    Zap,
    Wrench,
    Send,
    Info,
    ChevronDown,
  } from '@lucide/svelte'
  import {
    navigationStore,
    NAV_CATEGORIES,
    type TabItem,
  } from '../../stores/navigation.svelte'
  import { themeStore } from '../../stores/theme.svelte'

  const iconComponents: Record<string, typeof Cpu> = {
    overview: LayoutDashboard,
    sysinfo: Cpu,
    network: Network,
    hotspot: Wifi,
    proxy: Shield,
    modem: Radio,
    sms: MessageSquare,
    power: Power,
    charger: BatteryCharging,
    modules: Box,
    terminal: Terminal,
    ssh: Terminal,
    scrcpy: Smartphone,
    files: FolderOpen,
    logs: FileText,
    qos: Gauge,
    vnstat: Activity,
    nas: HardDrive,
    tunnel: Globe,
    speedtest: Zap,
    tools: Wrench,
    telegram: Send,
    about: Info,
  }

  const categoryIcons: Record<string, typeof Cpu> = {
    core: LayoutDashboard,
    network: Network,
    system: Cpu,
    tools: Wrench,
  }

  let openCategory = $state<string | null>(null)
  let closeTimer: ReturnType<typeof setTimeout> | null = null

  onDestroy(() => {
    if (closeTimer) {
      clearTimeout(closeTimer)
      closeTimer = null
    }
  })

  function handleMouseEnter(catId: string) {
    if (closeTimer) {
      clearTimeout(closeTimer)
      closeTimer = null
    }
    openCategory = catId
  }

  function handleMouseLeave() {
    if (closeTimer) clearTimeout(closeTimer)
    closeTimer = setTimeout(() => {
      openCategory = null
    }, 150)
  }

  function toggleCategory(catId: string) {
    if (openCategory === catId) {
      openCategory = null
    } else {
      openCategory = catId
    }
  }

  function selectTab(tabId: string) {
    navigationStore.setTab(tabId)
    openCategory = null
    if (closeTimer) {
      clearTimeout(closeTimer)
      closeTimer = null
    }
  }

  function handleWindowClick(e: MouseEvent) {
    const target = e.target as HTMLElement
    if (!target.closest('.top-nav-container')) {
      openCategory = null
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      openCategory = null
    }
  }
</script>

<svelte:window onclick={handleWindowClick} onkeydown={handleKeydown} />

<nav
  class="top-nav-container hidden md:flex items-center gap-1 sm:gap-1.5 font-mono text-xs select-none relative"
  aria-label="Desktop Top Flyout Navigation"
>
  {#each NAV_CATEGORIES as cat}
    {@const CatIcon = categoryIcons[cat.id] || LayoutDashboard}
    {@const isCatActive = navigationStore.currentCategory === cat.id}
    {@const isOpen = openCategory === cat.id}
    {@const tabs = navigationStore.getTabsForCategory(cat.id)}

    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="relative"
      onmouseenter={() => handleMouseEnter(cat.id)}
      onmouseleave={handleMouseLeave}
    >
      <!-- Category Pill Button -->
      <button
        type="button"
        class="flex items-center gap-1.5 px-3 py-1.5 font-bold uppercase tracking-wider transition-all cursor-pointer select-none {themeStore.currentStyle === 'neobrutal' ? (isCatActive ? 'bg-accent text-accent-text border-2 border-border shadow-neobrutal-sm rounded' : 'text-muted hover:text-foreground hover:bg-card-sub border-2 border-transparent rounded') : (isCatActive ? 'bg-accent text-accent-text shadow-sm rounded-xl ring-1 ring-accent' : 'text-muted hover:text-foreground hover:bg-card-sub/80 rounded-xl transition-colors')}"
        onclick={() => toggleCategory(cat.id)}
        aria-expanded={isOpen}
      >
        <CatIcon class="w-3.5 h-3.5 shrink-0 {isCatActive ? 'text-accent-text' : 'text-accent'}" />
        <span>{cat.label}</span>
        <ChevronDown
          class="w-3 h-3 transition-transform duration-200 opacity-70 {isOpen ? 'rotate-180' : ''}"
        />
      </button>

      <!-- Floating Flyout Menu -->
      {#if isOpen}
        <div
          class="absolute top-full left-0 mt-1.5 min-w-[210px] z-50 p-1.5 {themeStore.currentStyle === 'neobrutal' ? 'bg-card border-2 border-border shadow-[4px_4px_0px_0px_var(--neo-shadow)] rounded' : 'bg-card/95 backdrop-blur-md border border-border shadow-xl rounded-2xl'}"
          transition:fly={{ y: -6, duration: 150 }}
          role="menu"
        >
          <!-- Small Header inside Flyout -->
          <div class="px-2.5 py-1 text-[10px] font-bold uppercase text-muted tracking-wider border-b border-border/60 flex items-center justify-between mb-1">
            <span>{cat.label} Tabs</span>
            <span class="text-[9px] px-1.5 py-0.2 rounded bg-card-sub border border-border text-foreground font-bold">
              {tabs.length}
            </span>
          </div>

          <div class="space-y-0.5">
            {#each tabs as tab}
              {@const TabIcon = iconComponents[tab.id] || Info}
              {@const isTabActive = navigationStore.activeTab === tab.id}
              <button
                type="button"
                class="w-full flex items-center justify-between px-2.5 py-1.5 font-mono text-xs font-bold uppercase tracking-wider transition-all cursor-pointer rounded {isTabActive ? 'bg-accent text-accent-text' : 'text-muted hover:text-foreground hover:bg-card-sub'}"
                onclick={() => selectTab(tab.id)}
                role="menuitem"
              >
                <div class="flex items-center gap-2.5 truncate">
                  <TabIcon class="w-3.5 h-3.5 shrink-0 {isTabActive ? '' : 'text-accent'}" />
                  <span class="truncate">{tab.label}</span>
                </div>
                {#if isTabActive}
                  <span class="w-1.5 h-1.5 rounded-full bg-white shrink-0 ml-2"></span>
                {/if}
              </button>
            {/each}
          </div>
        </div>
      {/if}
    </div>
  {/each}
</nav>
