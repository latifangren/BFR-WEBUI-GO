export interface SSHConfig {
  port: number
  enabled?: boolean
  bind?: string
  key_auth_only?: boolean
}

export interface SSHStatus {
  running: boolean
  port: number
  pid?: number
  config?: SSHConfig
}
