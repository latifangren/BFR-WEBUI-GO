import { api } from '../api/client'
import type { SysinfoStats } from '../types/sysinfo'

class SysinfoStore {
  stats = $state<SysinfoStats | null>(null)
  isLoading = $state<boolean>(false)
  isPolling = $state<boolean>(false)
  error = $state<string | null>(null)

  private timer: ReturnType<typeof setInterval> | null = null
  private pollInterval = 2000
  private isVisible = true

  constructor() {
    if (typeof document !== 'undefined') {
      document.addEventListener('visibilitychange', () => {
        this.isVisible = !document.hidden
        if (this.isVisible && this.isPolling && !this.timer) {
          this.refresh()
          this.startTimer()
        } else if (!this.isVisible && this.timer) {
          this.stopTimer()
        }
      })
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

  startPolling(intervalMs = 2000): void {
    this.pollInterval = intervalMs
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
