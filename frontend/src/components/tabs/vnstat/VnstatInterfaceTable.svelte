<script lang="ts">
  import {
    Network,
    ArrowDown,
    ArrowUp,
  } from '@lucide/svelte'
  import type { StatsResponse } from '../../../types/vnstat'
  import Card from '../../ui/Card.svelte'

  interface Props {
    stats: StatsResponse | null
  }

  let { stats }: Props = $props()

  let selectedPeriod = $state<'daily' | 'yesterday' | 'weekly' | 'monthly'>('daily')
  let selectedCategory = $state<'all' | 'cellular' | 'wifi' | 'softap' | 'other'>('all')

  function formatBytes(bytes?: number): string {
    if (!bytes || bytes <= 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
  }

  function getCategory(name: string): 'cellular' | 'wifi' | 'softap' | 'other' {
    const lower = name.toLowerCase()
    if (lower.startsWith('rmnet') || lower.startsWith('ccmni') || lower.startsWith('wwan')) return 'cellular'
    if (lower.startsWith('wlan') || lower.startsWith('wifi') || lower.startsWith('ra')) return 'wifi'
    if (lower.startsWith('ap') || lower.startsWith('softap')) return 'softap'
    return 'other'
  }

  const activePeriodStats = $derived(stats?.[selectedPeriod])
  const activeInterfaces = $derived(activePeriodStats?.interfaces || {})
  const periodTotalBytes = $derived(
    activePeriodStats?.total?.total ||
      (activePeriodStats?.total?.rx_bytes || 0) + (activePeriodStats?.total?.tx_bytes || 0)
  )

  const rows = $derived(
    Object.entries(activeInterfaces)
      .map(([name, stat]) => {
        const rx = stat.rx_bytes || 0
        const tx = stat.tx_bytes || 0
        const total = rx + tx
        const cat = getCategory(name)
        const sharePct = periodTotalBytes > 0 ? Math.min(100, (total / periodTotalBytes) * 100) : 0
        return {
          name,
          category: cat,
          rx_bytes: rx,
          tx_bytes: tx,
          total,
          sharePct,
        }
      })
      .filter((row) => (selectedCategory === 'all' ? true : row.category === selectedCategory))
      .sort((a, b) => b.total - a.total)
  )

  const categoryBadges = {
    cellular: { label: 'CELLULAR', class: 'bg-cyan-950/60 text-cyan-300 border-cyan-800' },
    wifi: { label: 'WI-FI', class: 'bg-emerald-950/60 text-emerald-300 border-emerald-800' },
    softap: { label: 'SOFTAP', class: 'bg-amber-950/60 text-amber-300 border-amber-800' },
    other: { label: 'OTHER', class: 'bg-card-sub text-muted border-border' },
  }
</script>

<Card
  title="INTERFACE BANDWIDTH BREAKDOWN"
  subtitle="Per-interface data accounting and relative traffic share across selected periods"
  tone="none"
>
  <!-- Controls: Period Selector & Category Filter Pills -->
  <div class="space-y-3 mb-4 font-mono text-xs">
    <!-- Period Pills -->
    <div class="flex items-center gap-1.5 flex-wrap">
      <span class="text-muted text-[11px] font-bold uppercase tracking-wider mr-1">Period:</span>
      {#each [
        { id: 'daily', label: 'Today' },
        { id: 'yesterday', label: 'Yesterday' },
        { id: 'weekly', label: 'This Week' },
        { id: 'monthly', label: 'This Month' }
      ] as p}
        <button
          type="button"
          onclick={() => (selectedPeriod = p.id as typeof selectedPeriod)}
          class="px-2.5 py-1 rounded text-xs font-bold uppercase tracking-wider select-none cursor-pointer transition-colors {selectedPeriod === p.id ? 'bg-foreground text-background shadow-neobrutal' : 'bg-card border border-border text-muted hover:text-foreground'}"
        >
          {p.label}
        </button>
      {/each}
    </div>

    <!-- Category Filter Pills -->
    <div class="flex items-center gap-1.5 flex-wrap">
      <span class="text-muted text-[11px] font-bold uppercase tracking-wider mr-1">Category:</span>
      {#each [
        { id: 'all', label: 'All' },
        { id: 'cellular', label: 'Cellular (rmnet)' },
        { id: 'wifi', label: 'Wi-Fi (wlan)' },
        { id: 'softap', label: 'SoftAP (ap)' },
        { id: 'other', label: 'Other' }
      ] as c}
        <button
          type="button"
          onclick={() => (selectedCategory = c.id as typeof selectedCategory)}
          class="px-2.5 py-1 rounded text-[11px] font-bold uppercase tracking-wider select-none cursor-pointer transition-colors {selectedCategory === c.id ? 'bg-accent text-accent-text border border-accent shadow-sm' : 'bg-card border border-border text-muted hover:text-foreground'}"
        >
          {c.label}
        </button>
      {/each}
    </div>
  </div>

  {#if rows.length === 0}
    <div class="p-8 text-center border-2 border-dashed border-border rounded bg-card-sub font-mono">
      <Network class="w-8 h-8 mx-auto text-muted mb-2 opacity-50" />
      <p class="text-sm font-bold text-foreground">No Interface Traffic Detected</p>
      <p class="text-xs text-muted mt-1">
        No active traffic recorded for the selected period and category filter.
      </p>
    </div>
  {:else}
    <div class="overflow-x-auto border border-border rounded custom-scrollbar">
      <table class="w-full text-left border-collapse text-xs font-mono">
        <thead>
          <tr class="bg-card-sub border-b border-border text-muted uppercase text-[10px] tracking-wider">
            <th class="p-3">Interface</th>
            <th class="p-3">Category</th>
            <th class="p-3">Download (RX)</th>
            <th class="p-3">Upload (TX)</th>
            <th class="p-3">Total Volume</th>
            <th class="p-3 text-right">Traffic Share</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          {#each rows as row (row.name)}
            {@const badge = categoryBadges[row.category]}
            <tr class="hover:bg-card-sub/50 transition-colors">
              <!-- Interface Name -->
              <td class="p-3 font-bold text-foreground text-xs whitespace-nowrap">
                {row.name}
              </td>

              <!-- Category Badge -->
              <td class="p-3 whitespace-nowrap">
                <span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-mono font-bold uppercase border {badge.class}">
                  {badge.label}
                </span>
              </td>

              <!-- RX -->
              <td class="p-3 whitespace-nowrap text-emerald-400">
                <span class="inline-flex items-center gap-1">
                  <ArrowDown class="w-3 h-3" />
                  <span>{formatBytes(row.rx_bytes)}</span>
                </span>
              </td>

              <!-- TX -->
              <td class="p-3 whitespace-nowrap text-purple-400">
                <span class="inline-flex items-center gap-1">
                  <ArrowUp class="w-3 h-3" />
                  <span>{formatBytes(row.tx_bytes)}</span>
                </span>
              </td>

              <!-- Total -->
              <td class="p-3 whitespace-nowrap font-bold text-foreground">
                {formatBytes(row.total)}
              </td>

              <!-- Traffic Share Progress Bar -->
              <td class="p-3 whitespace-nowrap text-right">
                <div class="flex items-center justify-end gap-2">
                  <span class="text-[11px] font-bold text-foreground">{row.sharePct.toFixed(1)}%</span>
                  <div class="w-20 bg-card h-2 rounded overflow-hidden border border-border/80">
                    <div
                      class="bg-accent h-full transition-all duration-300"
                      style="width: {row.sharePct}%"
                    ></div>
                  </div>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</Card>
