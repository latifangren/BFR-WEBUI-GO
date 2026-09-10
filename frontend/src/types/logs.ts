export interface LogEntry {
  time?: string
  timestamp?: string
  level: 'DEBUG' | 'INFO' | 'WARN' | 'ERROR' | string
  category: string
  message: string
}

export interface LogResponse {
  entries: LogEntry[]
  total: number
}
