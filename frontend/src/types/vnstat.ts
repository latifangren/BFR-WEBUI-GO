export interface InterfaceStat {
  rx_bytes: number
  tx_bytes: number
}

export interface TotalStat {
  rx_bytes: number
  tx_bytes: number
  total: number
}

export interface PeriodStats {
  total: TotalStat
  interfaces?: Record<string, InterfaceStat>
}

export interface DayRecord {
  date: string
  rx_bytes: number
  tx_bytes: number
  total: number
}

export interface StatsResponse {
  daily?: PeriodStats
  yesterday?: PeriodStats
  weekly?: PeriodStats
  monthly?: PeriodStats
  daily_history?: DayRecord[]
  last_updated?: string
}
