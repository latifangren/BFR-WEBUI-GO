import { api } from '../api/client'
import type { SysinfoStats } from '../types/sysinfo'

class SysinfoStore {
  stats = $state<SysinfoStats | null>(null)
  isLoading = $state<boolean>(false)
  isPolling = $state<boolean>(false)
  error = $state<string | null>(null)

  private timer: ReturnType<typeof setInterval> | null = null
  private pollInterval = 10000
  private isVisible = true
  private isFocused = false

  constructor() {
    if (typeof document !== 'undefined') {
      this.isVisible = !document.hidden
      document.addEventListener('visibilitychange', () => {
        this.isVisible = !document.hidden
        if (this.isVisible) {
          if (this.isPolling) {
            this.refresh()
            this.startTimer()
          }
        } else {
          this.stopTimer()
        }
      })
    }
  }

  init(): void {
    if (!this.isPolling) {
      this.startPolling(this.isFocused ? 2000 : 10000)
    }
  }

  setTelemetryFocus(isFocused: boolean): void {
    this.isFocused = isFocused
    const nextInterval = isFocused ? 2000 : 10000
    if (this.pollInterval !== nextInterval) {
      this.pollInterval = nextInterval
      if (this.isPolling && this.isVisible) {
        if (isFocused) {
          // Immediately refresh for instant feedback when entering overview/sysinfo
          this.refresh()
        }
        this.startTimer()
      }
    }
  }

  async refresh(): Promise<void> {
    try {
      this.isLoading = true
      const data = await api.get<SysinfoStats>('/api/sysinfo')
      this.stats = data
      this.error = null
    } catch (err: unknown) {
      this.error = err instanceof Error ? err.message : 'Failed to fetch telemetry'
    } finally {
      this.isLoading = false
    }
  }

  startPolling(intervalMs?: number): void {
    if (intervalMs !== undefined) {
      this.pollInterval = intervalMs
    } else {
      this.pollInterval = this.isFocused ? 2000 : 10000
    }
    this.isPolling = true
    this.refresh()
    this.startTimer()
  }

  stopPolling(): void {
    this.isPolling = false
    this.stopTimer()
  }

  private startTimer(): void {
    this.stopTimer()
    if (!this.isVisible) return
    this.timer = setInterval(() => {
      this.refresh()
    }, this.pollInterval)
  }

  private stopTimer(): void {
    if (this.timer) {
      clearInterval(this.timer)
      this.timer = null
    }
  }
}

export const sysinfoStore = new SysinfoStore()
