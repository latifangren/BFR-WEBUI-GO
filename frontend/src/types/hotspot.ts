export interface HotspotStatus {
  enabled: boolean
  ssid: string
  security_type?: string
  wifi_enabled?: boolean
  usb_enabled?: boolean
  ethernet_enabled?: boolean
  active_interfaces?: string[]
  gateway_ip?: string
  subnet?: string
}

export interface ConnectedClient {
  ip: string
  mac: string
  device: string
  state: string
  interface?: string
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

export interface RouterConfig {
  gateway_mode: 'auto' | 'static'
  gateway_ip: string
  subnet_mask: string
  dhcp_start: string
  dhcp_end: string
  lease_time: string
  dns_primary: string
  dns_secondary: string
  band: 'any' | '2' | '5'
  upstream_interface: string
  auto_start_wifi?: boolean
  auto_start_usb?: boolean
  auto_start_ethernet?: boolean
  ssid?: string
  password?: string
  security_type?: string
}

export interface RouterStatus {
  // Desired (persisted) state
  desired_mode: string
  desired_gateway: string
  desired_subnet: string
  desired_band: string
  desired_upstream: string
  dhcp_managed_by_os: boolean
  auto_start_wifi?: boolean
  auto_start_usb?: boolean
  auto_start_ethernet?: boolean

  // Apply state: "inactive" | "auto" | "applied" | "pending" | "failed"
  apply_state?: string

  // The gateway Android's DHCP server actually hands out to clients.
  client_gateway?: string
  client_subnet?: string
  // True when the client-facing gateway matches the desired static gateway.
  gateway_effective?: boolean

  // Observed (kernel-verified) state
  applied: boolean
  observed_gateway?: string
  observed_subnet?: string
  observed_iface: string
  upstream_table: string
  address_present: boolean
  rule_to_subnet_present: boolean
  rule_from_ap_present: boolean
  last_apply_error?: string
  last_apply_time?: string
}

export interface RouterSettingsResponse {
  config: RouterConfig
  status?: RouterStatus
}
