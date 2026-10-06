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

export interface SwitchNode {
  path: string
  on_value: string
  off_value: string
}

export interface SwitchProfile {
  id: string
  name: string
  description: string
  nodes: SwitchNode[]
  is_bypass: boolean
}

export interface ChargerConfig {
  enabled: boolean
  start_percent: number
  stop_percent: number
  selected_switch?: string
  custom_path?: string
  custom_on?: string
  custom_off?: string
}

export interface BatteryInfo {
  capacity: number
  status: string
  current_now: number
  voltage_now: number
  temperature: number
  power_supply: string
  is_online: boolean
  is_idle: boolean
}

export interface ProbeResult {
  profile_id: string
  name: string
  success: boolean
  current_before: number
  current_after: number
  message: string
  tested_at: string
}

export interface ChargerStatusResponse {
  config: ChargerConfig
  active_profile?: SwitchProfile
  detected_path: string
  detected_type: string
  charging_disabled: boolean
  current_level: number
  battery: BatteryInfo
  available_switches: SwitchProfile[]
  probe_results?: ProbeResult[]
  logs: string[]
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


export type ScheduleMode = 'disabled' | 'interval' | 'clock'

export interface ScheduleConfig {
  enabled: boolean
  mode: ScheduleMode
  interval_hours: number
  clock_time: string
}

export interface ScheduleStatus {
  config: ScheduleConfig
  uptime_seconds: number
  next_reboot_in_sec: number
  next_reboot_time: string
  description: string
}
