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

export interface NetworkTweaksStatus {
  enabled: boolean
  tweaks?: Record<string, string>
}
