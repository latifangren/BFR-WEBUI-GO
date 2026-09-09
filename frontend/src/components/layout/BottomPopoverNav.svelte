<script lang="ts">
  import { fly, fade } from 'svelte/transition'
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
    Palette,
    X,
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

  const categoryLabels: Record<string, string> = {
    core: 'Core Dashboard',
    network: 'Network & Connectivity',
    system: 'Hardware & System',
    tools: 'Tools & Utilities',
  }

  const navItems = [
    { id: 'core', label: 'Core', icon: LayoutDashboard },
    { id: 'network', label: 'Network', icon: Network },
    { id: 'system', label: 'System', icon: Cpu },
    { id: 'tools', label: 'Tools', icon: Wrench },
    { id: 'style', label: 'Style', icon: Palette },
  ] as const

  let activePopoverCategory = $state<string | null>(null)

  function handleItemClick(itemId: string) {
    if (itemId === 'style') {
      activePopoverCategory = null
      themeStore.openAppearanceModal()
      return
    }

    if (activePopoverCategory === itemId) {
      activePopoverCategory = null
    } else {
      activePopoverCategory = itemId
    }
  }

  function selectTab(tabId: string) {
    navigationStore.setTab(tabId)
    activePopoverCategory = null
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      activePopoverCategory = null
    }
  }

  const activeCategoryGroup = $derived(
    NAV_CATEGORIES.find((c: { id: string }) => c.id === activePopoverCategory)
  )

  const currentCategoryTabs = $derived(
    activePopoverCategory
      ? navigationStore.getTabsForCategory(
          activePopoverCategory as 'core' | 'network' | 'system' | 'tools'
        )
      : []
  )
</script>

<svelte:window onkeydown={handleKeydown} />

<!-- Upward Popover Card & Backdrop -->
{#if activePopoverCategory && activeCategoryGroup}
  {@const CatIcon = categoryIcons[activePopoverCategory] || LayoutDashboard}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="fixed inset-0 bg-black/60 backdrop-blur-xs z-40 md:hidden transition-opacity"
    onclick={() => (activePopoverCategory = null)}
    transition:fade={{ duration: 120 }}
  ></div>

  <!-- Popover Container -->
  <div
    class="fixed bottom-16 left-3 right-3 z-50 max-h-[60vh] flex flex-col font-mono md:hidden select-none overflow-hidden {themeStore.currentStyle === 'neobrutal' ? 'bg-card border-2 border-border shadow-[4px_4px_0px_0px_var(--neo-shadow)] rounded-lg' : 'bg-card/95 backdrop-blur-md border border-border shadow-2xl rounded-2xl'}"
    transition:fly={{ y: 20, duration: 150 }}
    role="dialog"
    aria-label="Category Navigation"
  >
    <!-- Popover Header -->
    <div class="p-3 border-b border-border/80 flex items-center justify-between bg-card shrink-0">
      <div class="flex items-center gap-2 min-w-0">
        <div class="p-1 rounded bg-accent/15 text-accent border border-accent/30 shrink-0">
          <CatIcon class="w-4 h-4" />
        </div>
        <span class="text-xs font-bold uppercase tracking-wider text-foreground truncate">
          {categoryLabels[activePopoverCategory] || activeCategoryGroup.label}
        </span>
        <span class="text-[9px] font-bold px-1.5 py-0.2 rounded bg-card-sub border border-border text-muted shrink-0">
          {currentCategoryTabs.length}
        </span>
      </div>

      <button
        type="button"
        class="p-1 rounded border border-border text-muted hover:text-foreground hover:bg-card-sub cursor-pointer transition-colors shrink-0"
        onclick={() => (activePopoverCategory = null)}
        aria-label="Close popover"
      >
        <X class="w-4 h-4" />
      </button>
    </div>

    <!-- Sub-tabs Grid -->
    <div class="p-3 grid grid-cols-2 gap-2 overflow-y-auto max-h-[calc(60vh-3.5rem)]">
      {#each currentCategoryTabs as tab}
        {@const TabIcon = iconComponents[tab.id] || Info}
        {@const isTabActive = navigationStore.activeTab === tab.id}
        <button
          type="button"
          class="p-2.5 text-left flex items-center gap-2.5 cursor-pointer transition-all active:scale-95 font-mono {themeStore.currentStyle === 'neobrutal' ? (isTabActive ? 'border-2 border-border bg-accent text-accent-text shadow-neobrutal-sm font-bold rounded' : 'border-2 border-border bg-card-sub text-foreground hover:border-accent rounded') : (isTabActive ? 'border border-accent bg-accent text-accent-text shadow-sm font-bold rounded-xl ring-1 ring-accent' : 'border border-border/80 bg-card-sub text-muted hover:text-foreground rounded-xl transition-colors')}"
          onclick={() => selectTab(tab.id)}
        >
          <div class="p-1 rounded bg-card border border-border shrink-0">
            <TabIcon class="w-4 h-4 {isTabActive ? 'text-accent' : 'text-muted'}" />
          </div>
          <div class="min-w-0 flex-1">
            <div class="text-xs truncate font-bold leading-tight">{tab.label}</div>
            <div class="text-[9px] text-muted uppercase truncate leading-tight mt-0.5">{tab.id}</div>
          </div>
          {#if isTabActive}
            <span class="w-1.5 h-1.5 rounded-full bg-white shrink-0"></span>
          {/if}
        </button>
      {/each}
    </div>
  </div>
{/if}

<!-- Bottom Bar -->
<nav
  class="md:hidden fixed bottom-0 left-0 right-0 z-40 font-mono select-none {themeStore.currentStyle === 'neobrutal' ? 'border-t-2 border-border bg-card shadow-lg' : 'border-t border-border bg-card/90 backdrop-blur-md shadow-lg'}"
  aria-label="Mobile Bottom Popover Navigation"
>
  <div class="grid grid-cols-5 h-14 max-w-lg mx-auto px-1 items-center">
    {#each navItems as item}
      {@const Icon = item.icon}
      {@const isCurrentCategory = item.id !== 'style' && navigationStore.currentCategory === item.id}
      {@const isPopoverOpen = activePopoverCategory === item.id}
      {@const isHighlighted = isCurrentCategory || isPopoverOpen}

      <button
        type="button"
        class="flex flex-col items-center justify-center h-full py-1 gap-1 text-[10px] font-bold transition-all active:scale-95 cursor-pointer relative {isHighlighted ? 'text-accent' : 'text-muted hover:text-foreground'}"
        onclick={() => handleItemClick(item.id)}
        aria-expanded={isPopoverOpen}
      >
        <div class="p-1 rounded-md transition-colors {isHighlighted ? 'bg-accent/15' : ''}">
          <Icon class="w-4 h-4 {isHighlighted ? 'stroke-[2.5]' : 'stroke-2'}" />
        </div>
        <span class="truncate max-w-[56px] leading-tight text-[9px] uppercase tracking-wider">{item.label}</span>
        {#if isCurrentCategory}
          <span class="absolute top-1 w-1.5 h-1.5 rounded-full bg-accent"></span>
        {/if}
      </button>
    {/each}
  </div>
</nav>
