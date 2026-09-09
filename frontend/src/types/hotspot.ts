export interface HotspotStatus {
  enabled: boolean
  ssid: string
}

export interface ConnectedClient {
  ip: string
  mac: string
  device: string
  state: string
}

export interface MACFilterConfig {
  mode: 'disabled' | 'blacklist' | 'whitelist'
  blocked_macs: string[]
  allowed_macs: string[]
}

export interface MACFilterStatus {
  active_mode: string
  blocked_count: number
  allowed_count: number
  rules_count: number
}

export interface MACFilterResponse {
  config: MACFilterConfig
  status?: MACFilterStatus
}
