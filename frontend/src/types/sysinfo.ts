export interface CPUCoreStat {
  core: number
  freq_mhz: number
  usage: number
}

export interface ThermalZone {
  name: string
  temp: number
}

export interface DiskPartition {
  path: string
  total: number
  used: number
  free: number
  used_pct: number
}

export interface DetailedBattery {
  capacity: number
  status: string
  temp: number
  voltage_mv: number
  health: string
  technology: string
  charge_full: number
  charge_now: number
  current_now: number
}

export interface LoadAverage {
  one: number
  five: number
  fifteen: number
}

export interface ServiceStatus {
  name: string
  running: boolean
  detail: string
}

export interface NetworkDetail {
  interface?: string
  ip?: string
  mac?: string
  rx_bytes?: number
  tx_bytes?: number
  rx_rate?: number
  tx_rate?: number
  ip_addresses?: string[]
  gateway?: string
  dns?: string[]
  dns1?: string
  dns2?: string
  wifi_ssid?: string
  wifi_signal?: string
  wifi_signal_dbm?: string
  wifi_rssi?: number
  wifi_full_info?: string
  mcc_mnc?: string[]
  roaming?: string
  hotspot_clients?: number
  sim_slots?: SIMSlot[]
}

export interface SIMSlot {
  slot: number
  operator: string
  network_type: string
  signal_strength: number
  imei?: string
  rsrp?: string
  rsrq?: string
  sinr?: string
  signal_status?: string
}

export interface SysinfoStats {
  cpu_usage: number
  cpu_cores: CPUCoreStat[]
  cpu_temp: number
  mem_total: number
  mem_free: number
  mem_available: number
  mem_used: number
  mem_used_pct: number
  swap_total: number
  swap_free: number
  swap_used: number
  swap_used_pct: number
  load_avg: LoadAverage
  active_services: ServiceStatus[]
  services?: Record<string, { status?: string; running?: boolean; name?: string }>
  uptime: number
  battery_level: number
  battery_status: string
  battery_temp: number
  battery_detail?: DetailedBattery
  thermals: ThermalZone[]
  disk_total: number
  disk_free: number
  disk_used: number
  disk_used_pct: number
  disks: DiskPartition[]
  model: string
  android_ver: string
  android_version?: string
  selinux: string
  security_patch: string
  sdk_ver: string
  resolution: string
  density: string
  mtu: string
  default_ttl: string
  kernel: string
  soc: string
  governor: string
  net_rx?: number
  net_tx?: number
  network_detail?: NetworkDetail
  network?: NetworkDetail
}
