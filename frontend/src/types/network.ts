export interface DNSConfig {
  primary: string
  secondary: string
  mode?: string
}

export interface TTLConfig {
  ttl: number | string
  status?: string
}

export interface PingResult {
  host: string
  output?: string
  latency_ms?: number
  success?: boolean
}

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

export interface ChargerConfig {
  enabled: boolean
  start_percent: number
  stop_percent: number
  custom_path?: string
}

export type PowerAction = 'reboot' | 'recovery' | 'bootloader' | 'poweroff'

export interface SMSMessage {
  id: number
  address: string
  body: string
  date: number
  date_sent?: number
  read: number
  type: number
}

export interface SMSResponse {
  messages: SMSMessage[]
  total: number
  offset: number
  limit: number
}

export interface ATCommandResult {
  command: string
  response: string
  success?: boolean
}

export interface TweaksConfig {
  lte_carrier_aggregation: boolean
  tcp_buffer_optimization: boolean
  bbr2_congestion_control: boolean
  sysctl_buffers_opt: boolean
  dalvik_responsiveness: boolean
  settings_global_tweaks: boolean
  ttl_spoofing: boolean
  packet_steering_rps: boolean
  mtu_tuning: boolean
}

export interface NetworkTweaksStatus {
  enabled: boolean
  tweaks?: Record<string, string>
  tweaks_json?: TweaksConfig
  active_dns1?: string
  active_dns2?: string
}

export interface RPSConfig {
  interface: string
  bitmask: string
}
