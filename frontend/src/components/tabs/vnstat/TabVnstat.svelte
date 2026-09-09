<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Activity,
    RefreshCw,
    RotateCcw,
    ArrowDown,
    ArrowUp,
    Database,
    Calendar,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'

  interface InterfaceStat {
    rx_bytes: number
    tx_bytes: number
  }

  interface TotalStat {
    rx_bytes: number
    tx_bytes: number
    total: number
  }

  interface PeriodStats {
    total: TotalStat
    interfaces?: Record<string, InterfaceStat>
  }

  interface StatsResponse {
    daily?: PeriodStats
    monthly?: PeriodStats
    last_updated?: string
  }

  let stats = $state<StatsResponse | null>(null)
  let isLoading = $state(false)
  let isResetting = $state(false)

  onMount(async () => {
    await fetchVnstat()
  })

  async function fetchVnstat() {
    try {
      isLoading = true
      const res = await api.get<StatsResponse>('/api/vnstat/stats')
      stats = res
    } catch {
      // Keep defaults
    } finally {
      isLoading = false
    }
  }

  async function resetStats() {
    try {
      isResetting = true
      await api.post('/api/vnstat/reset')
      toastStore.success('Vnstat traffic database counters reset.')
      await fetchVnstat()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to reset vnstat')
    } finally {
      isResetting = false
    }
  }

  function formatBytes(bytes?: number): string {
    if (!bytes || bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
  }
</script>

<div class="space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Activity class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        Vnstat Bandwidth Accounting
      </h2>
    </div>

    <div class="flex items-center gap-2">
      <Button
        variant="outline"
        size="sm"
        disabled={isResetting}
        onclick={resetStats}
      >
        <RotateCcw class="w-3.5 h-3.5 mr-1 text-red-400" />
        <span>Reset Database</span>
      </Button>

      <Button
        variant="secondary"
        size="sm"
        disabled={isLoading}
        onclick={fetchVnstat}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoading ? 'animate-spin' : ''}" />
        <span>Refresh</span>
      </Button>
    </div>
  </div>

  <!-- Traffic Summary Grid -->
  <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 font-mono text-xs">
    <!-- Today / Daily -->
    <div class="neo-card bg-card p-5 rounded-lg flex flex-col justify-between space-y-3">
      <div class="flex items-center justify-between text-muted">
        <span class="font-bold uppercase tracking-wider flex items-center gap-1.5 text-foreground">
          <Calendar class="w-4 h-4 text-accent" /> Daily Traffic (24h)
        </span>
        <Badge variant="info">Today</Badge>
      </div>

      <div class="space-y-1.5">
        <div class="flex items-center justify-between">
          <span class="text-muted flex items-center gap-1">
            <ArrowDown class="w-3.5 h-3.5 text-emerald-400" /> Download (RX):
          </span>
          <span class="font-bold text-foreground">
            {formatBytes(stats?.daily?.total?.rx_bytes)}
          </span>
        </div>
        <div class="flex items-center justify-between">
          <span class="text-muted flex items-center gap-1">
            <ArrowUp class="w-3.5 h-3.5 text-purple-400" /> Upload (TX):
          </span>
          <span class="font-bold text-foreground">
            {formatBytes(stats?.daily?.total?.tx_bytes)}
          </span>
        </div>
      </div>

      <div class="pt-3 border-t border-border flex items-center justify-between font-bold">
        <span class="text-muted uppercase text-[10px]">Total Transferred:</span>
        <span class="text-base text-accent">
          {formatBytes(stats?.daily?.total?.total || ((stats?.daily?.total?.rx_bytes || 0) + (stats?.daily?.total?.tx_bytes || 0)))}
        </span>
      </div>
    </div>

    <!-- Monthly -->
    <div class="neo-card bg-card p-5 rounded-lg flex flex-col justify-between space-y-3">
      <div class="flex items-center justify-between text-muted">
        <span class="font-bold uppercase tracking-wider flex items-center gap-1.5 text-foreground">
          <Database class="w-4 h-4 text-purple-400" /> Monthly Quota Accounting
        </span>
        <Badge variant="warning">Billing Cycle</Badge>
      </div>

      <div class="space-y-1.5">
        <div class="flex items-center justify-between">
          <span class="text-muted flex items-center gap-1">
            <ArrowDown class="w-3.5 h-3.5 text-emerald-400" /> Download (RX):
          </span>
          <span class="font-bold text-foreground">
            {formatBytes(stats?.monthly?.total?.rx_bytes)}
          </span>
        </div>
        <div class="flex items-center justify-between">
          <span class="text-muted flex items-center gap-1">
            <ArrowUp class="w-3.5 h-3.5 text-purple-400" /> Upload (TX):
          </span>
          <span class="font-bold text-foreground">
            {formatBytes(stats?.monthly?.total?.tx_bytes)}
          </span>
        </div>
      </div>

      <div class="pt-3 border-t border-border flex items-center justify-between font-bold">
        <span class="text-muted uppercase text-[10px]">Monthly Total:</span>
        <span class="text-base text-amber-400">
          {formatBytes(stats?.monthly?.total?.total || ((stats?.monthly?.total?.rx_bytes || 0) + (stats?.monthly?.total?.tx_bytes || 0)))}
        </span>
      </div>
    </div>
  </div>

  <!-- Per-Interface Traffic Breakdown -->
  {#if stats?.daily?.interfaces && Object.keys(stats.daily.interfaces).length > 0}
    <Card title="Interface Telemetry Breakdown" subtitle="Traffic split per physical / cellular / softap interface">
      <div class="overflow-x-auto">
        <table class="w-full text-left font-mono text-xs">
          <thead>
            <tr class="border-b border-border bg-card-sub text-muted uppercase text-[10px]">
              <th class="py-2.5 px-3">Interface</th>
              <th class="py-2.5 px-3">RX (Down)</th>
              <th class="py-2.5 px-3">TX (Up)</th>
              <th class="py-2.5 px-3 text-right">Total</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            {#each Object.entries(stats.daily.interfaces) as [iface, data]}
              <tr class="hover:bg-card-sub/60 transition-colors">
                <td class="py-2.5 px-3 font-bold text-accent">{iface}</td>
                <td class="py-2.5 px-3 text-emerald-400">{formatBytes(data.rx_bytes)}</td>
                <td class="py-2.5 px-3 text-purple-400">{formatBytes(data.tx_bytes)}</td>
                <td class="py-2.5 px-3 text-right font-bold text-foreground">
                  {formatBytes(data.rx_bytes + data.tx_bytes)}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </Card>
  {/if}
</div>
