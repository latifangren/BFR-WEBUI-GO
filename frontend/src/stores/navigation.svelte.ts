import { HAS_VENDOR_MODEM } from '../config/features'

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
  { id: 'core', label: 'Core', icon: 'LayoutDashboard', tabs: ['overview', 'sysinfo', 'kernel', 'about'] },
  {
    id: 'network',
    label: 'Network',
    icon: 'Network',
    tabs: [
      'network',
      'hotspot',
      'proxy',
      'modem',
      ...(HAS_VENDOR_MODEM ? ['qualcomm', 'samsung', 'shannon'] : []),
      'sms',
      'vnstat',
      'tunnel',
    ],
  },
  { id: 'system', label: 'System', icon: 'Cpu', tabs: ['power', 'charger', 'modules', 'partitions'] },
  { id: 'tools', label: 'Tools', icon: 'Wrench', tabs: ['terminal', 'ssh', 'scrcpy', 'files', 'logs', 'nas', 'speedtest', 'tools', 'telegram'] },
]

const RAW_TABS: TabItem[] = [
  { id: 'overview', label: 'Overview', icon: 'LayoutDashboard', category: 'core', badge: 'PRO', badgeColor: 'amber' },
  { id: 'sysinfo', label: 'Sysinfo', icon: 'Cpu', category: 'core', badge: 'CORE', badgeColor: 'mint' },
  { id: 'kernel', label: 'Kernel', icon: 'Sliders', category: 'core', badge: 'NEW', badgeColor: 'mint' },
  { id: 'network', label: 'Network', icon: 'Network', category: 'network', badge: 'NET', badgeColor: 'ice' },
  { id: 'hotspot', label: 'Tethering & QoS', icon: 'Wifi', category: 'network', badge: 'ROUTER', badgeColor: 'mint' },
  { id: 'proxy', label: 'Proxy', icon: 'Shield', category: 'network', badge: 'CLASH', badgeColor: 'lavender' },
  { id: 'modem', label: 'Cellular Dashboard', icon: 'Radio', category: 'network', badge: 'CELL', badgeColor: 'ice' },
  { id: 'qualcomm', label: 'Snapdragon QMI', icon: 'Zap', category: 'network', badge: 'QCOM', badgeColor: 'mint' },
  { id: 'samsung', label: 'Samsung RIL', icon: 'Smartphone', category: 'network', badge: 'SEC', badgeColor: 'ice' },
  { id: 'shannon', label: 'Google Tensor', icon: 'Layers', category: 'network', badge: 'PIXEL', badgeColor: 'peach' },
  { id: 'sms', label: 'SMS', icon: 'MessageSquare', category: 'network', badge: 'SMS', badgeColor: 'peach' },
  { id: 'power', label: 'Power', icon: 'Power', category: 'system', badge: 'PWR', badgeColor: 'peach' },
  { id: 'charger', label: 'Charger', icon: 'BatteryCharging', category: 'system', badge: 'BATT', badgeColor: 'mint' },
  { id: 'modules', label: 'Modules', icon: 'Box', category: 'system', badge: 'ROOT', badgeColor: 'yellow' },
  { id: 'partitions', label: 'Partitions', icon: 'HardDriveDownload', category: 'system', badge: 'BLOCK', badgeColor: 'ice' },
  { id: 'terminal', label: 'Terminal', icon: 'Terminal', category: 'tools', badge: 'PTY', badgeColor: 'peach' },
  { id: 'ssh', label: 'SSH', icon: 'Terminal', category: 'tools', badge: 'SSH', badgeColor: 'peach' },
  { id: 'scrcpy', label: 'Scrcpy', icon: 'Smartphone', category: 'tools', badge: 'LIVE', badgeColor: 'mint' },
  { id: 'files', label: 'Files', icon: 'FolderOpen', category: 'tools', badge: 'FS', badgeColor: 'yellow' },
  { id: 'logs', label: 'Logs', icon: 'FileText', category: 'tools', badge: 'LOGS', badgeColor: 'ice' },
  { id: 'vnstat', label: 'Vnstat', icon: 'Activity', category: 'network', badge: 'STATS', badgeColor: 'mint' },
  { id: 'nas', label: 'NAS', icon: 'HardDrive', category: 'tools', badge: 'NAS', badgeColor: 'ice' },
  { id: 'tunnel', label: 'Tunnel', icon: 'Globe', category: 'network', badge: 'WAN', badgeColor: 'yellow' },
  { id: 'speedtest', label: 'Speedtest', icon: 'Zap', category: 'tools', badge: 'TEST', badgeColor: 'mint' },
  { id: 'tools', label: 'Tools', icon: 'Wrench', category: 'tools', badge: 'SYS', badgeColor: 'lavender' },
  { id: 'telegram', label: 'Telegram', icon: 'Send', category: 'tools', badge: 'BOT', badgeColor: 'ice' },
  { id: 'about', label: 'About', icon: 'Info', category: 'core', badge: 'INFO', badgeColor: 'mint' },
]

export const AVAILABLE_TABS: TabItem[] = RAW_TABS.filter((tab) => {
  if (!HAS_VENDOR_MODEM && (tab.id === 'qualcomm' || tab.id === 'samsung' || tab.id === 'shannon')) {
    return false
  }
  return true
})

class NavigationStore {
  activeTab = $state<string>('sysinfo') // Default to sysinfo for PoC
  sidebarOpen = $state<boolean>(false)
  layout = $state<NavLayoutId>('topbar')

  constructor() {
    if (typeof window !== 'undefined') {
      let hash = window.location.hash.replace('#', '')
      if (hash === 'qos') hash = 'hotspot'
      if (hash && AVAILABLE_TABS.some((t) => t.id === hash)) {
        this.activeTab = hash
      }

      window.addEventListener('hashchange', () => {
        let h = window.location.hash.replace('#', '')
        if (h === 'qos') h = 'hotspot'
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
    const target = tabId === 'qos' ? 'hotspot' : tabId
    this.activeTab = target
    this.sidebarOpen = false
    if (typeof window !== 'undefined') {
      window.location.hash = target
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
