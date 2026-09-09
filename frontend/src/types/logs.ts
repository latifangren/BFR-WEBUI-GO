export interface LogEntry {
  timestamp: string
  level: 'DEBUG' | 'INFO' | 'WARN' | 'ERROR' | string
  category: string
  message: string
}

export interface LogResponse {
  entries: LogEntry[]
  total: number
}
