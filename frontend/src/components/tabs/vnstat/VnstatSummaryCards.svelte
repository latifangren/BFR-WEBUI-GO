<script lang="ts">
  import {
    ArrowDown,
    ArrowUp,
    Calendar,
    History,
    CalendarDays,
    Database,
  } from '@lucide/svelte'
  import type { StatsResponse, PeriodStats } from '../../../types/vnstat'
  import { LL } from '../../../i18n/i18n-svelte'
  import Badge from '../../ui/Badge.svelte'

  interface Props {
    stats: StatsResponse | null
  }

  let { stats }: Props = $props()

  function formatBytes(bytes?: number): string {
    if (!bytes || bytes <= 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
  }

  interface CardConfig {
    id: string
    title: string
    subtitle: string
    badge: string
    badgeVariant: 'info' | 'default' | 'success' | 'warning'
    icon: typeof Calendar
    iconColor: string
    accentColor: string
    period?: PeriodStats
  }

  const cards = $derived<CardConfig[]>([
    {
      id: 'today',
      title: $LL.vnstat.summary.today.title(),
      subtitle: $LL.vnstat.summary.today.subtitle(),
      badge: $LL.vnstat.summary.today.badge(),
      badgeVariant: 'info',
      icon: Calendar,
      iconColor: 'text-cyan-400',
      accentColor: 'text-cyan-400',
      period: stats?.daily,
    },
    {
      id: 'yesterday',
      title: $LL.vnstat.summary.yesterday.title(),
      subtitle: $LL.vnstat.summary.yesterday.subtitle(),
      badge: $LL.vnstat.summary.yesterday.badge(),
      badgeVariant: 'default',
      icon: History,
      iconColor: 'text-indigo-400',
      accentColor: 'text-indigo-300',
      period: stats?.yesterday,
    },
    {
      id: 'weekly',
      title: $LL.vnstat.summary.weekly.title(),
      subtitle: $LL.vnstat.summary.weekly.subtitle(),
      badge: $LL.vnstat.summary.weekly.badge(),
      badgeVariant: 'success',
      icon: CalendarDays,
      iconColor: 'text-emerald-400',
      accentColor: 'text-emerald-400',
      period: stats?.weekly,
    },
    {
      id: 'monthly',
      title: $LL.vnstat.summary.monthly.title(),
      subtitle: $LL.vnstat.summary.monthly.subtitle(),
      badge: $LL.vnstat.summary.monthly.badge(),
      badgeVariant: 'warning',
      icon: Database,
      iconColor: 'text-amber-400',
      accentColor: 'text-amber-400',
      period: stats?.monthly,
    },
  ])
</script>

<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 font-mono text-xs">
  {#each cards as card (card.id)}
    {@const rx = card.period?.total?.rx_bytes || 0}
    {@const tx = card.period?.total?.tx_bytes || 0}
    {@const total = card.period?.total?.total || rx + tx}
    <div
      class="neo-card bg-card border-2 border-border p-4 rounded-lg flex flex-col justify-between space-y-3.5 shadow-neobrutal"
    >
      <!-- Card Header -->
      <div class="flex items-center justify-between border-b border-border/80 pb-2.5">
        <div class="flex items-center gap-1.5 truncate">
          <card.icon class="w-4 h-4 shrink-0 {card.iconColor}" />
          <div>
            <div class="font-bold text-foreground tracking-wide text-xs">{card.title}</div>
            <div class="text-[10px] text-muted">{card.subtitle}</div>
          </div>
        </div>
        <Badge variant={card.badgeVariant} class="shrink-0">{card.badge}</Badge>
      </div>

      <!-- RX / TX Metrics -->
      <div class="space-y-2 py-1">
        <div class="flex items-center justify-between bg-card-sub px-2.5 py-1.5 rounded border border-border/60">
          <span class="text-muted flex items-center gap-1.5 text-[11px]">
            <ArrowDown class="w-3.5 h-3.5 text-emerald-400 shrink-0" />
            <span>{$LL.vnstat.summary.downloadRx()}</span>
          </span>
          <span class="font-bold text-foreground">{formatBytes(rx)}</span>
        </div>

        <div class="flex items-center justify-between bg-card-sub px-2.5 py-1.5 rounded border border-border/60">
          <span class="text-muted flex items-center gap-1.5 text-[11px]">
            <ArrowUp class="w-3.5 h-3.5 text-purple-400 shrink-0" />
            <span>{$LL.vnstat.summary.uploadTx()}</span>
          </span>
          <span class="font-bold text-foreground">{formatBytes(tx)}</span>
        </div>
      </div>

      <!-- Total Transferred -->
      <div class="pt-2 border-t border-border flex items-baseline justify-between">
        <span class="text-muted uppercase text-[10px] tracking-wider font-bold">{$LL.vnstat.summary.totalVolume()}</span>
        <span class="text-base font-bold {card.accentColor}">
          {formatBytes(total)}
        </span>
      </div>
    </div>
  {/each}
</div>
