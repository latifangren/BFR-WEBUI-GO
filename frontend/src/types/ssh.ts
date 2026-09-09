export interface SSHConfig {
  port: number
  root_login?: boolean
  password_auth?: boolean
  authorized_keys?: string
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
