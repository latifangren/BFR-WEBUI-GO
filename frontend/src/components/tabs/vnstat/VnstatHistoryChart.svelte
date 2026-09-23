<script lang="ts">
  import {
    BarChart3,
    ArrowDown,
    ArrowUp,
  } from '@lucide/svelte'
  import type { DayRecord } from '../../../types/vnstat'
  import Card from '../../ui/Card.svelte'

  interface Props {
    history: DayRecord[] | undefined
  }

  let { history = [] }: Props = $props()

  function formatBytes(bytes?: number): string {
    if (!bytes || bytes <= 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
  }

  const records = $derived(history || [])

  const maxTotal = $derived(
    records.length > 0
      ? Math.max(...records.map((r) => r.total || r.rx_bytes + r.tx_bytes), 1)
      : 1
  )

  function formatDayLabel(dateStr: string): string {
    try {
      const parts = dateStr.split('-')
      if (parts.length === 3) {
        const d = new Date(Number(parts[0]), Number(parts[1]) - 1, Number(parts[2]))
        return d.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' })
      }
      return dateStr
    } catch {
      return dateStr
    }
  }
</script>

<Card
  title="7-DAY BANDWIDTH HISTORY"
  subtitle="Daily cumulative consumption trends and upload/download volume distributions"
  tone="ice"
>
  {#snippet action()}
    <div class="flex items-center gap-3 text-[11px] font-mono text-muted">
      <div class="flex items-center gap-1.5">
        <span class="w-2.5 h-2.5 rounded-sm bg-emerald-500 inline-block"></span>
        <span>Download (RX)</span>
      </div>
      <div class="flex items-center gap-1.5">
        <span class="w-2.5 h-2.5 rounded-sm bg-purple-500 inline-block"></span>
        <span>Upload (TX)</span>
      </div>
    </div>
  {/snippet}

  {#if records.length === 0}
    <div class="p-8 text-center border-2 border-dashed border-border rounded bg-card-sub font-mono">
      <BarChart3 class="w-8 h-8 mx-auto text-muted mb-2 opacity-50" />
      <p class="text-sm font-bold text-foreground">No Historical Traffic Data</p>
      <p class="text-xs text-muted mt-1">
        Vnstat daemon logs daily consumption over time. History will populate as traffic is recorded.
      </p>
    </div>
  {:else}
    <div class="space-y-3 font-mono text-xs">
      {#each records as record (record.date)}
        {@const rx = record.rx_bytes || 0}
        {@const tx = record.tx_bytes || 0}
        {@const total = record.total || rx + tx}
        {@const rxPct = Math.min(100, Math.max(0, (rx / maxTotal) * 100))}
        {@const txPct = Math.min(100 - rxPct, Math.max(0, (tx / maxTotal) * 100))}
        {@const totalPct = Math.min(100, (total / maxTotal) * 100)}

        <div class="p-3 bg-card-sub border border-border/80 rounded-lg hover:border-accent/40 transition-colors space-y-2">
          <!-- Row Header: Date & Total -->
          <div class="flex items-center justify-between text-xs">
            <div class="flex items-center gap-2">
              <span class="font-bold text-foreground tracking-wide">
                {formatDayLabel(record.date)}
              </span>
              <span class="text-[10px] text-muted hidden sm:inline">({record.date})</span>
            </div>
            <div class="flex items-center gap-2">
              <span class="text-muted text-[10px]">Peak Share:</span>
              <span class="font-bold text-foreground">{totalPct.toFixed(1)}%</span>
              <span class="text-accent font-bold text-sm ml-1">{formatBytes(total)}</span>
            </div>
          </div>

          <!-- Stacked Volume Bar -->
          <div class="w-full bg-card h-3 rounded overflow-hidden flex border border-border/60">
            {#if rxPct > 0}
              <div
                class="bg-emerald-500 h-full transition-all duration-500"
                style="width: {rxPct}%"
                title="Download: {formatBytes(rx)}"
              ></div>
            {/if}
            {#if txPct > 0}
              <div
                class="bg-purple-500 h-full transition-all duration-500"
                style="width: {txPct}%"
                title="Upload: {formatBytes(tx)}"
              ></div>
            {/if}
          </div>

          <!-- Bottom Breakdown Metrics -->
          <div class="flex items-center justify-between text-[11px] text-muted pt-0.5">
            <span class="flex items-center gap-1 text-emerald-400">
              <ArrowDown class="w-3 h-3" />
              <span>RX: {formatBytes(rx)}</span>
            </span>
            <span class="flex items-center gap-1 text-purple-400">
              <ArrowUp class="w-3 h-3" />
              <span>TX: {formatBytes(tx)}</span>
            </span>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</Card>
