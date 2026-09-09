<script lang="ts">
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
  } from '@lucide/svelte'
  import { navigationStore, AVAILABLE_TABS, type TabItem } from '../../stores/navigation.svelte'

  // Explicit mapping of tab id to icon component
  // svelte-ignore non_reactive_update
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

  const categories = [
    { key: 'core', label: 'Core Dashboard' },
    { key: 'network', label: 'Network & Connectivity' },
    { key: 'system', label: 'Hardware & System' },
    { key: 'tools', label: 'Tools & Utilities' },
  ] as const

  function getTabsByCategory(category: string): TabItem[] {
    return AVAILABLE_TABS.filter((t) => t.category === category)
  }
</script>

<!-- Mobile Drawer Backdrop -->
{#if navigationStore.sidebarOpen}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="fixed inset-0 bg-black/80 backdrop-blur-sm z-40 md:hidden"
    onclick={() => navigationStore.toggleSidebar()}
  ></div>
{/if}

<!-- Sidebar Container -->
<aside
  class="fixed md:sticky top-0 md:top-14 left-0 h-full md:h-[calc(100vh-3.5rem)] w-64 bg-card border-r-2 border-border z-50 md:z-30 flex flex-col transition-transform duration-200 ease-in-out {navigationStore.sidebarOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0'}"
>
  <!-- Mobile Header in Drawer -->
  <div class="md:hidden flex items-center justify-between p-4 border-b border-border">
    <div class="font-mono font-bold text-sm tracking-wider uppercase text-foreground">
      Navigation Menu
    </div>
    <button
      type="button"
      class="p-1 rounded border border-border text-foreground hover:bg-card-sub cursor-pointer"
      onclick={() => navigationStore.toggleSidebar()}
      aria-label="Close navigation"
    >
      <X class="w-5 h-5" />
    </button>
  </div>

  <!-- Navigation Links -->
  <div class="flex-1 overflow-y-auto p-3 space-y-5 font-mono text-xs">
    {#each categories as cat}
      <div class="space-y-1">
        <div class="px-2.5 py-1 text-[10px] font-bold uppercase text-muted tracking-widest">
          {cat.label}
        </div>
        {#each getTabsByCategory(cat.key) as tab}
          {@const IconComponent = iconComponents[tab.id]}
          <button
            type="button"
            class="w-full flex items-center gap-3 px-3 py-2 rounded font-mono text-xs font-bold uppercase tracking-wider transition-all cursor-pointer {navigationStore.activeTab === tab.id ? 'bg-accent text-accent-text border border-border shadow-neobrutal-sm' : 'text-muted hover:text-foreground hover:bg-card-sub'}"
            onclick={() => navigationStore.setTab(tab.id)}
          >
            {#if IconComponent}
              <IconComponent class="w-4 h-4 shrink-0" />
            {/if}
            <span class="truncate">{tab.label}</span>
          </button>
        {/each}
      </div>
    {/each}
  </div>

  <!-- Footer Info inside Sidebar -->
  <div class="p-3 border-t border-border font-mono text-[10px] text-muted flex items-center justify-between">
    <span>BFR-WEBUI-GO</span>
    <span class="px-1.5 py-0.5 rounded bg-card-sub border border-border text-foreground font-bold">
      v1.2.2
    </span>
  </div>
</aside>
