export interface ClientLimit {
  ip: string
  mac: string
  hostname: string
  download_limit: number // Mbps
  upload_limit: number   // Mbps
  priority: 'gaming' | 'stream' | 'browsing' | 'strict' | 'custom'
  enabled: boolean
}

export interface QoSConfig {
  enabled: boolean
  engine: 'auto' | 'tc' | 'iptables'
  preset?: 'custom' | 'gaming' | 'fair' | 'saver'
  global_download: number
  global_upload: number
  prioritize_gaming: boolean
  prioritize_voip: boolean
  scheduled_limit_enabled?: boolean
  schedule_start?: string
  schedule_end?: string
  client_limits: ClientLimit[]
}

export interface QoSStatus {
  active: boolean
  engine_used: string
  active_rules_count: number
  detected_iface: string
}

export interface ConnectedHotspotClient {
  ip: string
  mac: string
  device: string
  state: string
}
