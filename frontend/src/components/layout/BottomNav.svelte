<script lang="ts">
  import {
    LayoutDashboard,
    Network,
    Cpu,
    Wrench,
    Compass,
    Search,
    X,
    Wifi,
    Shield,
    Radio,
    Power,
    BatteryCharging,
    Terminal,
    Smartphone,
    FolderOpen,
    FileText,
    Gauge,
    Activity,
    HardDrive,
    Globe,
    Zap,
    Send,
    Info,
    MessageSquare,
    Box,
  } from '@lucide/svelte'
  import { navigationStore, AVAILABLE_TABS, type TabItem } from '../../stores/navigation.svelte'

  let showTabsSheet = $state(false)
  let searchQuery = $state('')

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

  const primaryTabs = [
    { id: 'overview', label: 'Overview', icon: LayoutDashboard },
    { id: 'network', label: 'Network', icon: Network },
    { id: 'sysinfo', label: 'System', icon: Cpu },
    { id: 'tools', label: 'Tools', icon: Wrench },
  ] as const

  const categories = [
    { key: 'core', label: 'Core Dashboard' },
    { key: 'network', label: 'Network & Connectivity' },
    { key: 'system', label: 'Hardware & System' },
    { key: 'tools', label: 'Tools & Utilities' },
  ] as const

  const isMoreActive = $derived(
    !primaryTabs.some((t) => t.id === navigationStore.activeTab)
  )

  const activeTabItem = $derived(
    AVAILABLE_TABS.find((t) => t.id === navigationStore.activeTab)
  )

  const filteredTabs = $derived.by(() => {
    const q = searchQuery.toLowerCase().trim()
    if (!q) return null
    return AVAILABLE_TABS.filter(
      (t) => t.label.toLowerCase().includes(q) || t.id.toLowerCase().includes(q)
    )
  })

  function getTabsByCategory(category: string): TabItem[] {
    return AVAILABLE_TABS.filter((t) => t.category === category)
  }

  function handleTabSelect(tabId: string) {
    navigationStore.setTab(tabId)
    showTabsSheet = false
    searchQuery = ''
    if (typeof window !== 'undefined') {
      window.scrollTo({ top: 0, behavior: 'smooth' })
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && showTabsSheet) {
      showTabsSheet = false
      searchQuery = ''
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<!-- Mobile Bottom Navigation Bar (Fixed Bottom) -->
<nav
  class="md:hidden fixed bottom-0 inset-x-0 z-40 bg-card border-t-2 border-border shadow-[0_-4px_20px_rgba(0,0,0,0.3)] font-mono select-none"
  aria-label="Mobile Navigation Bar"
>
  <div class="grid grid-cols-5 h-16 max-w-lg mx-auto px-1 items-center">
    {#each primaryTabs as tab}
      {@const Icon = tab.icon}
      {@const isActive = navigationStore.activeTab === tab.id}
      <button
        type="button"
        class="flex flex-col items-center justify-center h-full py-1 gap-1 text-[10px] font-bold transition-all active:scale-95 cursor-pointer relative {isActive ? 'text-accent' : 'text-muted hover:text-foreground'}"
        onclick={() => handleTabSelect(tab.id)}
      >
        <div class="p-1 rounded-md transition-colors {isActive ? 'bg-accent/15' : ''}">
          <Icon class="w-5 h-5 {isActive ? 'stroke-[2.5]' : 'stroke-2'}" />
        </div>
        <span class="truncate max-w-[56px] leading-tight">{tab.label}</span>
        {#if isActive}
          <span class="absolute top-1 w-1.5 h-1.5 rounded-full bg-accent"></span>
        {/if}
      </button>
    {/each}

    <!-- 5th Button: Action Sheet / More Tabs Drawer -->
    <button
      type="button"
      class="flex flex-col items-center justify-center h-full py-1 gap-1 text-[10px] font-bold transition-all active:scale-95 cursor-pointer relative {isMoreActive || showTabsSheet ? 'text-accent' : 'text-muted hover:text-foreground'}"
      onclick={() => (showTabsSheet = !showTabsSheet)}
      aria-label="Open tab drawer"
    >
      <div class="p-1 rounded-md transition-colors {isMoreActive || showTabsSheet ? 'bg-accent/15' : ''}">
        <Compass class="w-5 h-5 {isMoreActive || showTabsSheet ? 'stroke-[2.5]' : 'stroke-2'}" />
      </div>
      <span class="truncate max-w-[56px] leading-tight">
        {isMoreActive && activeTabItem ? activeTabItem.label : 'Tabs'}
      </span>
      {#if isMoreActive}
        <span class="absolute top-1 w-1.5 h-1.5 rounded-full bg-accent animate-pulse"></span>
      {/if}
    </button>
  </div>
</nav>

<!-- Action Sheet / Bottom Sheet for All Tabs -->
{#if showTabsSheet}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="fixed inset-0 z-50 md:hidden flex flex-col justify-end">
    <!-- Backdrop Overlay -->
    <div
      class="fixed inset-0 bg-black/80 backdrop-blur-sm transition-opacity"
      onclick={() => { showTabsSheet = false; searchQuery = ''; }}
    ></div>

    <!-- Bottom Sheet Dialog -->
    <div
      class="neo-card relative w-full bg-card border-t-2 border-border rounded-t-2xl shadow-2xl max-h-[85vh] flex flex-col z-10 font-mono text-foreground overflow-hidden"
    >
      <!-- Drag Handle Indicator -->
      <div class="w-12 h-1.5 bg-muted/40 rounded-full mx-auto my-2.5 shrink-0"></div>

      <!-- Header & Search Toolbar -->
      <div class="px-4 pb-3 border-b border-border space-y-3 shrink-0">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <Compass class="w-4 h-4 text-accent" />
            <h3 class="text-sm font-bold uppercase tracking-wider text-foreground">
              All System Tabs
            </h3>
            <span class="text-[10px] px-1.5 py-0.2 rounded bg-card-sub border border-border text-muted font-bold">
              {AVAILABLE_TABS.length} Tabs
            </span>
          </div>
          <button
            type="button"
            class="p-1 rounded border border-border text-muted hover:text-foreground hover:bg-card-sub cursor-pointer transition-colors"
            onclick={() => { showTabsSheet = false; searchQuery = ''; }}
            aria-label="Close drawer"
          >
            <X class="w-4 h-4" />
          </button>
        </div>

        <!-- Instant Filter Search Input -->
        <div class="relative w-full">
          <Search class="w-4 h-4 text-muted absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none" />
          <input
            type="text"
            bind:value={searchQuery}
            placeholder="Search tab (e.g. terminal, proxy, modem)..."
            class="neo-input w-full bg-card-sub border border-border rounded-lg pl-9 pr-8 py-2 text-xs font-mono text-foreground placeholder:text-muted focus:outline-none focus:border-accent transition-colors"
          />
          {#if searchQuery}
            <button
              type="button"
              class="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted hover:text-foreground cursor-pointer"
              onclick={() => (searchQuery = '')}
            >
              <X class="w-3.5 h-3.5" />
            </button>
          {/if}
        </div>
      </div>

      <!-- Scrollable Tab Grid -->
      <div class="flex-1 overflow-y-auto p-4 space-y-5">
        {#if filteredTabs !== null}
          <!-- Search Mode -->
          <div class="space-y-2">
            <div class="text-[10px] text-muted uppercase font-bold px-1">
              Search Results ({filteredTabs.length})
            </div>
            {#if filteredTabs.length === 0}
              <div class="py-8 text-center text-xs text-muted">
                No matching tabs found for "{searchQuery}".
              </div>
            {:else}
              <div class="grid grid-cols-2 gap-2">
                {#each filteredTabs as tab}
                  {@const Icon = iconComponents[tab.id] || Compass}
                  {@const isCurrent = navigationStore.activeTab === tab.id}
                  <button
                    type="button"
                    class="p-2.5 rounded border text-left flex items-center gap-2.5 cursor-pointer transition-all active:scale-95 {isCurrent ? 'border-accent bg-card-sub text-accent font-bold shadow-sm' : 'border-border bg-card hover:border-accent/40 text-foreground'}"
                    onclick={() => handleTabSelect(tab.id)}
                  >
                    <div class="p-1.5 rounded bg-card-sub border border-border shrink-0">
                      <Icon class="w-4 h-4 {isCurrent ? 'text-accent' : 'text-muted'}" />
                    </div>
                    <div class="min-w-0 flex-1">
                      <div class="text-xs truncate font-bold">{tab.label}</div>
                      <div class="text-[9px] text-muted uppercase truncate">{tab.category}</div>
                    </div>
                  </button>
                {/each}
              </div>
            {/if}
          </div>
        {:else}
          <!-- Categorized Mode -->
          {#each categories as cat}
            {@const tabsInCat = getTabsByCategory(cat.key)}
            <div class="space-y-2">
              <div class="text-[10px] text-muted uppercase font-bold tracking-wider px-1 flex items-center justify-between">
                <span>{cat.label}</span>
                <span class="text-[9px] text-muted">{tabsInCat.length}</span>
              </div>
              <div class="grid grid-cols-2 gap-2">
                {#each tabsInCat as tab}
                  {@const Icon = iconComponents[tab.id] || Compass}
                  {@const isCurrent = navigationStore.activeTab === tab.id}
                  <button
                    type="button"
                    class="p-2.5 rounded border text-left flex items-center gap-2.5 cursor-pointer transition-all active:scale-95 {isCurrent ? 'border-accent bg-card-sub text-accent font-bold shadow-sm' : 'border-border bg-card hover:border-accent/40 text-foreground'}"
                    onclick={() => handleTabSelect(tab.id)}
                  >
                    <div class="p-1.5 rounded bg-card-sub border border-border shrink-0">
                      <Icon class="w-4 h-4 {isCurrent ? 'text-accent' : 'text-muted'}" />
                    </div>
                    <div class="min-w-0 flex-1">
                      <div class="text-xs truncate font-bold">{tab.label}</div>
                      <div class="text-[9px] text-muted uppercase truncate">{tab.id}</div>
                    </div>
                    {#if isCurrent}
                      <span class="w-1.5 h-1.5 rounded-full bg-accent shrink-0"></span>
                    {/if}
                  </button>
                {/each}
              </div>
            </div>
          {/each}
        {/if}
      </div>
    </div>
  </div>
{/if}
