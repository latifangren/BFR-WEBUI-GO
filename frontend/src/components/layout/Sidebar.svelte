<script lang="ts">
  import { onMount } from 'svelte'
  import { slide } from 'svelte/transition'
  import {
    LayoutDashboard,
    Cpu,
    Network,
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
    Wrench,
    Send,
    Info,
    X,
    MessageSquare,
    Box,
    ChevronDown,
  } from '@lucide/svelte'
  import { navigationStore, AVAILABLE_TABS, type TabItem } from '../../stores/navigation.svelte'
  import { themeStore } from '../../stores/theme.svelte'

  // Explicit mapping of tab id to icon component
  // svelte-ignore non_reactive_update
  const iconComponents: Record<string, typeof Cpu> = {
    overview: LayoutDashboard,
    sysinfo: Cpu,
    network: Network,
    hotspot: Wifi,
    proxy: Shield,
    modem: Radio,
    samsung: Smartphone,
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

  const categories = [
    { key: 'core', label: 'Core Dashboard', icon: LayoutDashboard },
    { key: 'network', label: 'Network & Connectivity', icon: Network },
    { key: 'system', label: 'Hardware & System', icon: Cpu },
    { key: 'tools', label: 'Tools & Utilities', icon: Wrench },
  ] as const

  const BADGE_COLOR_MAP: Record<string, string> = {
    mint: 'bg-[#c8f5d0] text-black',
    peach: 'bg-[#ffd5cc] text-black',
    ice: 'bg-[#c2e7ff] text-black',
    yellow: 'bg-[#fff0a3] text-black',
    lavender: 'bg-[#e2daf9] text-black',
    amber: 'bg-[#f59e0b] text-black',
  }

  const DEFAULT_ACCORDIONS: Record<string, boolean> = {
    core: true,
    network: true,
    system: true,
    tools: true,
  }

  let accordions = $state<Record<string, boolean>>({ ...DEFAULT_ACCORDIONS })

  onMount(() => {
    try {
      const saved = localStorage.getItem('bfr_sidebar_accordions')
      if (saved) {
        const parsed = JSON.parse(saved)
        accordions = { ...DEFAULT_ACCORDIONS, ...parsed }
      }
    } catch {
      // Ignored
    }
    autoExpandActiveCategory(navigationStore.activeTab)
  })

  function toggleCategory(key: string) {
    accordions[key] = !accordions[key]
    try {
      localStorage.setItem('bfr_sidebar_accordions', JSON.stringify(accordions))
    } catch {
      // Ignored
    }
  }

  let prevActiveTab = $state(navigationStore.activeTab)

  function autoExpandActiveCategory(activeTab: string) {
    const targetCat = AVAILABLE_TABS.find((t) => t.id === activeTab)?.category
    if (targetCat && !accordions[targetCat]) {
      accordions[targetCat] = true
      try {
        localStorage.setItem('bfr_sidebar_accordions', JSON.stringify(accordions))
      } catch {
        // Ignored
      }
    }
  }

  $effect(() => {
    // Auto-expand category only when activeTab genuinely changes
    const current = navigationStore.activeTab
    if (current && current !== prevActiveTab) {
      prevActiveTab = current
      autoExpandActiveCategory(current)
    }
  })

  function getTabsByCategory(category: string): TabItem[] {
    return AVAILABLE_TABS.filter((t) => t.category === category)
  }
</script>

<!-- Mobile Drawer Backdrop -->
{#if navigationStore.sidebarOpen}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="fixed inset-0 bg-black/80 backdrop-blur-sm z-40 md:hidden transition-opacity"
    onclick={() => navigationStore.toggleSidebar()}
  ></div>
{/if}

<!-- Sidebar Container -->
<aside
  class="fixed md:sticky top-0 md:top-14 left-0 h-full md:h-[calc(100vh-3.5rem)] w-64 bg-card border-r-2 border-border z-50 md:z-30 flex flex-col transition-transform duration-200 ease-in-out {navigationStore.sidebarOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'}"
>
  <!-- Mobile Header in Drawer -->
  <div class="md:hidden flex items-center justify-between p-4 border-b border-border bg-card shrink-0">
    <div class="flex items-center gap-2.5">
      <div class="w-7 h-7 bg-accent text-accent-text flex items-center justify-center font-mono font-black text-sm rounded border border-border">
        B
      </div>
      <div>
        <div class="font-mono font-bold text-sm tracking-wider uppercase text-foreground leading-tight">
          BFR WEBUI
        </div>
        <span class="text-[9px] font-mono text-accent font-bold uppercase">System Panel</span>
      </div>
    </div>
    <button
      type="button"
      class="p-1.5 rounded border border-border text-foreground hover:bg-card-sub cursor-pointer transition-colors"
      onclick={() => navigationStore.toggleSidebar()}
      aria-label="Close navigation"
    >
      <X class="w-5 h-5" />
    </button>
  </div>

  <!-- Navigation Links with Collapsible Accordions -->
  <div class="flex-1 overflow-y-auto p-3 space-y-4 font-mono text-xs custom-scrollbar">
    {#each categories as cat}
      {@const CategoryIcon = cat.icon}
      {@const catTabs = getTabsByCategory(cat.key)}
      {@const hasActiveTab = catTabs.some((t) => t.id === navigationStore.activeTab)}
      <div class="space-y-1">
        <!-- Accordion Header Button -->
        <button
          type="button"
          class="w-full flex items-center justify-between px-2.5 py-1.5 rounded text-left transition-colors cursor-pointer group select-none hover:bg-card-sub"
          onclick={() => toggleCategory(cat.key)}
          aria-expanded={accordions[cat.key]}
          aria-controls="accordion-{cat.key}"
        >
          <div class="flex items-center gap-2 min-w-0">
            <CategoryIcon class="w-3.5 h-3.5 text-accent shrink-0" />
            <span class="text-[10px] font-mono font-bold uppercase tracking-wider text-muted group-hover:text-foreground transition-colors truncate">
              {cat.label}
            </span>
            {#if !accordions[cat.key] && hasActiveTab}
              <span
                class="w-2 h-2 rounded-full bg-accent animate-pulse shrink-0"
                title="Active tab inside collapsed category"
              ></span>
            {/if}
          </div>

          <div class="flex items-center gap-1.5 shrink-0">
            <span class="text-[9px] font-mono font-bold px-1.5 py-0.2 rounded bg-card-sub border border-border text-muted">
              {catTabs.length}
            </span>
            <ChevronDown
              class="w-3.5 h-3.5 text-muted transition-transform duration-200 {accordions[cat.key] ? '' : '-rotate-90'}"
            />
          </div>
        </button>

        <!-- Collapsible Content with slide transition -->
        {#if accordions[cat.key]}
          <div id="accordion-{cat.key}" class="space-y-1 pt-0.5" transition:slide={{ duration: 150 }}>
            {#each catTabs as tab}
              {@const IconComponent = iconComponents[tab.id]}
              {@const isActive = navigationStore.activeTab === tab.id}
              <button
                type="button"
                class="w-full flex items-center gap-2.5 px-3 py-2 font-mono text-xs font-bold uppercase tracking-wider cursor-pointer transition-all {themeStore.currentStyle === 'neobrutal' ? (isActive ? 'bg-accent text-accent-text border-2 border-border shadow-neobrutal-sm rounded' : 'text-muted hover:text-foreground hover:bg-card-sub border-2 border-transparent rounded') : (isActive ? 'bg-accent text-accent-text shadow-md rounded-xl ring-1 ring-accent' : 'text-muted hover:text-foreground hover:bg-card-sub/80 rounded-xl transition-colors')}"
                onclick={() => navigationStore.setTab(tab.id)}
              >
                {#if IconComponent}
                  <IconComponent class="w-4 h-4 shrink-0 {isActive ? '' : 'text-muted'}" />
                {/if}
                <span class="truncate">{tab.label}</span>
                {#if tab.badge}
                  {#if themeStore.currentStyle === 'neobrutal'}
                    <span class="border border-black shadow-[1px_1px_0px_#000] px-1.5 py-0.2 rounded text-[9px] font-mono font-black uppercase tracking-wider ml-auto shrink-0 select-none {BADGE_COLOR_MAP[tab.badgeColor || 'mint'] || 'bg-[#c8f5d0] text-black'}">
                      {tab.badge}
                    </span>
                  {:else}
                    <span class="bg-accent/15 text-accent border border-accent/30 px-1.5 py-0.2 rounded text-[9px] font-mono font-bold uppercase tracking-wider ml-auto shrink-0 select-none">
                      {tab.badge}
                    </span>
                  {/if}
                {:else if isActive}
                  <span class="ml-auto w-1.5 h-1.5 rounded-full bg-white shrink-0"></span>
                {/if}
              </button>
            {/each}
          </div>
        {/if}
      </div>
    {/each}
  </div>

  <!-- Footer Info inside Sidebar -->
  <div class="p-3 border-t border-border font-mono text-[10px] text-muted flex items-center justify-between shrink-0 bg-card">
    <span class="font-bold">BFR-WEBUI-GO</span>
    <span class="px-1.5 py-0.5 rounded bg-card-sub border border-border text-foreground font-bold">
      v1.2.3
    </span>
  </div>
</aside>
