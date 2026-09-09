export interface TabItem {
  id: string
  label: string
  icon: string
  category: 'core' | 'system' | 'network' | 'tools'
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
  { id: 'network', label: 'Network', icon: 'Network', tabs: ['network', 'hotspot', 'proxy', 'modem', 'sms', 'qos', 'vnstat', 'tunnel'] },
  { id: 'system', label: 'System', icon: 'Cpu', tabs: ['power', 'charger', 'modules'] },
  { id: 'tools', label: 'Tools', icon: 'Wrench', tabs: ['terminal', 'ssh', 'scrcpy', 'files', 'logs', 'nas', 'speedtest', 'tools', 'telegram'] },
]

export const AVAILABLE_TABS: TabItem[] = [
  { id: 'overview', label: 'Overview', icon: 'LayoutDashboard', category: 'core' },
  { id: 'sysinfo', label: 'Sysinfo', icon: 'Cpu', category: 'core' },
  { id: 'network', label: 'Network', icon: 'Network', category: 'network' },
  { id: 'hotspot', label: 'Hotspot', icon: 'Wifi', category: 'network' },
  { id: 'proxy', label: 'Proxy', icon: 'Shield', category: 'network' },
  { id: 'modem', label: 'Modem', icon: 'Radio', category: 'network' },
  { id: 'sms', label: 'SMS', icon: 'MessageSquare', category: 'network' },
  { id: 'power', label: 'Power', icon: 'Power', category: 'system' },
  { id: 'charger', label: 'Charger', icon: 'BatteryCharging', category: 'system' },
  { id: 'modules', label: 'Modules', icon: 'Box', category: 'system' },
  { id: 'terminal', label: 'Terminal', icon: 'Terminal', category: 'tools' },
  { id: 'ssh', label: 'SSH', icon: 'Terminal', category: 'tools' },
  { id: 'scrcpy', label: 'Scrcpy', icon: 'Smartphone', category: 'tools' },
  { id: 'files', label: 'Files', icon: 'FolderOpen', category: 'tools' },
  { id: 'logs', label: 'Logs', icon: 'FileText', category: 'tools' },
  { id: 'qos', label: 'QoS', icon: 'Gauge', category: 'network' },
  { id: 'vnstat', label: 'Vnstat', icon: 'Activity', category: 'network' },
  { id: 'nas', label: 'NAS', icon: 'HardDrive', category: 'tools' },
  { id: 'tunnel', label: 'Tunnel', icon: 'Globe', category: 'network' },
  { id: 'speedtest', label: 'Speedtest', icon: 'Zap', category: 'tools' },
  { id: 'tools', label: 'Tools', icon: 'Wrench', category: 'tools' },
  { id: 'telegram', label: 'Telegram', icon: 'Send', category: 'tools' },
  { id: 'about', label: 'About', icon: 'Info', category: 'core' },
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
