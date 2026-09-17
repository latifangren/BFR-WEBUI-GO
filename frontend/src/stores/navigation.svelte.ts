export type BadgeColor = 'mint' | 'peach' | 'ice' | 'yellow' | 'lavender' | 'amber'

export interface TabItem {
  id: string
  label: string
  icon: string
  category: 'core' | 'system' | 'network' | 'tools'
  badge?: string
  badgeColor?: BadgeColor
}

export type NavLayoutId = 'topbar' | 'sidebar'

export interface NavLayoutConfig {
  id: NavLayoutId
  label: string
  description: string
  icon: string
}

export const NAV_LAYOUT_REGISTRY: Record<NavLayoutId, NavLayoutConfig> = {
  topbar: {
    id: 'topbar',
    label: 'Classic Top Bar',
    description: 'Desktop header flyout dropdowns with 100% full-width cards; mobile bottom popovers.',
    icon: 'LayoutGrid',
  },
  sidebar: {
    id: 'sidebar',
    label: 'Modern Sidebar',
    description: 'Desktop collapsible accordion sidebar rail; mobile clean slide-over drawer.',
    icon: 'PanelLeft',
  },
}

export interface NavCategoryGroup {
  id: 'core' | 'network' | 'system' | 'tools'
  label: string
  icon: string
  tabs: string[]
}

export const NAV_CATEGORIES: NavCategoryGroup[] = [
  { id: 'core', label: 'Core', icon: 'LayoutDashboard', tabs: ['overview', 'sysinfo', 'about'] },
  { id: 'network', label: 'Network', icon: 'Network', tabs: ['network', 'hotspot', 'proxy', 'modem', 'samsung', 'sms', 'qos', 'vnstat', 'tunnel'] },
  { id: 'system', label: 'System', icon: 'Cpu', tabs: ['power', 'charger', 'modules'] },
  { id: 'tools', label: 'Tools', icon: 'Wrench', tabs: ['terminal', 'ssh', 'scrcpy', 'files', 'logs', 'nas', 'speedtest', 'tools', 'telegram'] },
]

export const AVAILABLE_TABS: TabItem[] = [
  { id: 'overview', label: 'Overview', icon: 'LayoutDashboard', category: 'core', badge: 'PRO', badgeColor: 'amber' },
  { id: 'sysinfo', label: 'Sysinfo', icon: 'Cpu', category: 'core', badge: 'CORE', badgeColor: 'mint' },
  { id: 'network', label: 'Network', icon: 'Network', category: 'network', badge: 'NET', badgeColor: 'ice' },
  { id: 'hotspot', label: 'Hotspot', icon: 'Wifi', category: 'network', badge: 'WIFI', badgeColor: 'mint' },
  { id: 'proxy', label: 'Proxy', icon: 'Shield', category: 'network', badge: 'CLASH', badgeColor: 'lavender' },
  { id: 'modem', label: 'Modem', icon: 'Radio', category: 'network', badge: 'LTE', badgeColor: 'ice' },
  { id: 'samsung', label: 'Samsung RIL', icon: 'Smartphone', category: 'network', badge: 'OEM', badgeColor: 'mint' },
  { id: 'sms', label: 'SMS', icon: 'MessageSquare', category: 'network', badge: 'SMS', badgeColor: 'peach' },
  { id: 'power', label: 'Power', icon: 'Power', category: 'system', badge: 'PWR', badgeColor: 'peach' },
  { id: 'charger', label: 'Charger', icon: 'BatteryCharging', category: 'system', badge: 'BATT', badgeColor: 'mint' },
  { id: 'modules', label: 'Modules', icon: 'Box', category: 'system', badge: 'ROOT', badgeColor: 'yellow' },
  { id: 'terminal', label: 'Terminal', icon: 'Terminal', category: 'tools', badge: 'PTY', badgeColor: 'peach' },
  { id: 'ssh', label: 'SSH', icon: 'Terminal', category: 'tools', badge: 'SSH', badgeColor: 'peach' },
  { id: 'scrcpy', label: 'Scrcpy', icon: 'Smartphone', category: 'tools', badge: 'LIVE', badgeColor: 'mint' },
  { id: 'files', label: 'Files', icon: 'FolderOpen', category: 'tools', badge: 'FS', badgeColor: 'yellow' },
  { id: 'logs', label: 'Logs', icon: 'FileText', category: 'tools', badge: 'LOGS', badgeColor: 'ice' },
  { id: 'qos', label: 'QoS', icon: 'Gauge', category: 'network', badge: 'QOS', badgeColor: 'lavender' },
  { id: 'vnstat', label: 'Vnstat', icon: 'Activity', category: 'network', badge: 'STATS', badgeColor: 'mint' },
  { id: 'nas', label: 'NAS', icon: 'HardDrive', category: 'tools', badge: 'NAS', badgeColor: 'ice' },
  { id: 'tunnel', label: 'Tunnel', icon: 'Globe', category: 'network', badge: 'WAN', badgeColor: 'yellow' },
  { id: 'speedtest', label: 'Speedtest', icon: 'Zap', category: 'tools', badge: 'TEST', badgeColor: 'mint' },
  { id: 'tools', label: 'Tools', icon: 'Wrench', category: 'tools', badge: 'SYS', badgeColor: 'lavender' },
  { id: 'telegram', label: 'Telegram', icon: 'Send', category: 'tools', badge: 'BOT', badgeColor: 'ice' },
  { id: 'about', label: 'About', icon: 'Info', category: 'core', badge: 'INFO', badgeColor: 'mint' },
]

class NavigationStore {
  activeTab = $state<string>('sysinfo') // Default to sysinfo for PoC
  sidebarOpen = $state<boolean>(false)
  layout = $state<NavLayoutId>('topbar')

  constructor() {
    if (typeof window !== 'undefined') {
      const hash = window.location.hash.replace('#', '')
      if (hash && AVAILABLE_TABS.some((t) => t.id === hash)) {
        this.activeTab = hash
      }

      window.addEventListener('hashchange', () => {
        const h = window.location.hash.replace('#', '')
        if (h && AVAILABLE_TABS.some((t) => t.id === h)) {
          this.activeTab = h
        }
      })
    }

    if (typeof localStorage !== 'undefined') {
      const savedLayout = localStorage.getItem('navLayout') as NavLayoutId
      if (savedLayout && (savedLayout === 'topbar' || savedLayout === 'sidebar')) {
        this.layout = savedLayout
      }
    }
  }

  setLayout(layout: NavLayoutId) {
    if (layout !== 'topbar' && layout !== 'sidebar') return
    this.layout = layout
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem('navLayout', layout)
    }
  }

  get currentCategory(): 'core' | 'network' | 'system' | 'tools' {
    const current = AVAILABLE_TABS.find((t) => t.id === this.activeTab)
    return current ? current.category : 'core'
  }

  getTabsForCategory(catId: 'core' | 'network' | 'system' | 'tools'): TabItem[] {
    return AVAILABLE_TABS.filter((t) => t.category === catId)
  }

  setTab(tabId: string) {
    this.activeTab = tabId
    this.sidebarOpen = false
    if (typeof window !== 'undefined') {
      window.location.hash = tabId
    }
  }

  setActiveTab(tabId: string) {
    this.setTab(tabId)
  }

  toggleSidebar() {
    this.sidebarOpen = !this.sidebarOpen
  }

  closeSidebar() {
    this.sidebarOpen = false
  }
}

export const navigationStore = new NavigationStore()
