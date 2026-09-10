<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import {
    Zap,
    Play,
    Square,
    ArrowDown,
    ArrowUp,
    Activity,
    Server,
    RefreshCw,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'

  interface SpeedtestData {
    ping_ms: number
    jitter_ms: number
    download_mbps: number
    upload_mbps: number
    progress_pct: number
    phase: string
    running: boolean
    server_name?: string
    server_host?: string
    error?: string
  }

  interface SpeedtestHistoryItem {
    timestamp: string
    ping: number
    jitter?: number
    download: number
    upload: number
    client_ip?: string
    isp?: string
    server_name?: string
  }

  let result = $state<SpeedtestData>({
    ping_ms: 0,
    jitter_ms: 0,
    download_mbps: 0,
    upload_mbps: 0,
    progress_pct: 0,
    phase: 'idle',
    running: false,
  })

  let history = $state<SpeedtestHistoryItem[]>([])
  let isLoadingHistory = $state(false)
  let pollTimer: ReturnType<typeof setInterval> | null = null

  onMount(() => {
    fetchStatus()
    fetchHistory()
    return () => {
      stopPolling()
    }
  })

  async function fetchHistory() {
    try {
      isLoadingHistory = true
      const res = await api.get<SpeedtestHistoryItem[]>('/api/speedtest/history')
      if (Array.isArray(res)) {
        history = res
      }
    } catch {
      // Ignored
    } finally {
      isLoadingHistory = false
    }
  }

  onDestroy(() => {
    stopPolling()
  })

  function startPolling() {
    stopPolling()
    pollTimer = setInterval(async () => {
      await fetchStatus()
      if (!result.running && result.phase !== 'idle' && result.phase !== 'download' && result.phase !== 'upload' && result.phase !== 'ping') {
        stopPolling()
      }
    }, 500)
  }

  function stopPolling() {
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  }

  async function fetchStatus() {
    try {
      const prevRunning = result.running
      const res = await api.get<SpeedtestData>('/api/speedtest/status')
      if (res) {
        result = res
        if (prevRunning && !res.running) {
          fetchHistory()
        }
      }
    } catch {
      // Ignored
    }
  }

  async function startTest() {
    try {
      result.running = true
      result.phase = 'ping'
      result.progress_pct = 5
      await api.post('/api/speedtest/start')
      startPolling()
      toastStore.info('Speedtest started...')
    } catch (err: unknown) {
      result.running = false
      toastStore.error(err instanceof Error ? err.message : 'Failed to start speedtest')
    }
  }

  async function stopTest() {
    try {
      await api.post('/api/speedtest/stop')
      result.running = false
      result.phase = 'idle'
      stopPolling()
      toastStore.info('Speedtest aborted.')
    } catch {
      // Ignored
    }
  }
</script>

<div class="space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Zap class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        Network Speedtest
      </h2>
      <Badge variant={result.running ? 'warning' : 'default'}>
        {result.running ? result.phase.toUpperCase() : 'Ready'}
      </Badge>
    </div>

    <div class="flex items-center gap-2">
      {#if result.running}
        <Button variant="danger" size="sm" onclick={stopTest}>
          <Square class="w-3.5 h-3.5 mr-1" />
          <span>Stop Test</span>
        </Button>
      {:else}
        <Button variant="primary" size="sm" onclick={startTest}>
          <Play class="w-3.5 h-3.5 mr-1" />
          <span>Start Test</span>
        </Button>
      {/if}
    </div>
  </div>

  <!-- Overall Progress Bar -->
  {#if result.running}
    <div class="space-y-1.5 font-mono text-xs">
      <div class="flex items-center justify-between text-muted">
        <span class="uppercase font-bold text-[10px]">Testing Phase: {result.phase}</span>
        <span>{result.progress_pct}%</span>
      </div>
      <div class="w-full bg-card-sub border border-border h-2 rounded overflow-hidden">
        <div
          class="h-full bg-accent transition-all duration-300"
          style="width: {result.progress_pct}%"
        ></div>
      </div>
    </div>
  {/if}

  <!-- Gauges Grid -->
  <div class="grid grid-cols-1 sm:grid-cols-3 gap-6 font-mono">
    <!-- Download Speed -->
    <div class="neo-card bg-card border-2 border-border shadow-neobrutal p-6 rounded flex flex-col items-center justify-center space-y-2 text-center">
      <div class="p-3 rounded-full bg-accent/10 border border-accent text-accent">
        <ArrowDown class="w-6 h-6" />
      </div>
      <span class="text-xs text-muted uppercase font-bold tracking-wider">Download Speed</span>
      <div class="text-3xl sm:text-4xl font-black text-foreground">
        {result.download_mbps.toFixed(1)}
      </div>
      <span class="text-xs text-muted">Mbps</span>
    </div>

    <!-- Upload Speed -->
    <div class="neo-card bg-card border-2 border-border shadow-neobrutal p-6 rounded flex flex-col items-center justify-center space-y-2 text-center">
      <div class="p-3 rounded-full bg-purple-500/10 border border-purple-500 text-purple-400">
        <ArrowUp class="w-6 h-6" />
      </div>
      <span class="text-xs text-muted uppercase font-bold tracking-wider">Upload Speed</span>
      <div class="text-3xl sm:text-4xl font-black text-foreground">
        {result.upload_mbps.toFixed(1)}
      </div>
      <span class="text-xs text-muted">Mbps</span>
    </div>

    <!-- Ping & Jitter -->
    <div class="neo-card bg-card border-2 border-border shadow-neobrutal p-6 rounded flex flex-col items-center justify-center space-y-2 text-center">
      <div class="p-3 rounded-full bg-emerald-500/10 border border-emerald-500 text-emerald-400">
        <Activity class="w-6 h-6" />
      </div>
      <span class="text-xs text-muted uppercase font-bold tracking-wider">Latency / Ping</span>
      <div class="text-3xl sm:text-4xl font-black text-foreground">
        {result.ping_ms.toFixed(0)}
      </div>
      <span class="text-xs text-muted">ms (Jitter: {result.jitter_ms.toFixed(0)} ms)</span>
    </div>
  </div>

  <!-- Server Info Card -->
  {#if result.server_name || result.server_host}
    <Card title="Speedtest Server" tone="mint">
      <div class="flex items-center gap-3 font-mono text-xs">
        <Server class="w-5 h-5 text-accent shrink-0" />
        <div>
          <span class="font-bold text-foreground">{result.server_name || 'Nearest Edge Node'}</span>
          <p class="text-muted text-[11px]">{result.server_host}</p>
        </div>
      </div>
    </Card>
  {/if}

  <!-- Speedtest History Section -->
  <Card title="Speedtest History & Diagnostics" tone="mint">
    {#snippet action()}
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoadingHistory}
        onclick={fetchHistory}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1 {isLoadingHistory ? 'animate-spin' : ''}" />
        <span>Refresh</span>
      </Button>
    {/snippet}

    {#if isLoadingHistory && history.length === 0}
      <div class="p-8 text-center font-mono text-xs text-muted">
        <Activity class="w-5 h-5 animate-spin mx-auto mb-2 text-accent" />
        Loading historical speedtest records...
      </div>
    {:else if history.length === 0}
      <div class="p-8 text-center font-mono text-xs text-muted border-2 border-dashed border-border rounded-lg">
        <Zap class="w-6 h-6 mx-auto mb-2 text-accent opacity-50" />
        No speedtest history records found. Run your first speedtest above!
      </div>
    {:else}
      <!-- Responsive Table for sm+ -->
      <div class="hidden sm:block overflow-x-auto">
        <table class="w-full text-left font-mono text-xs border-collapse">
          <thead>
            <tr class="border-b border-border text-muted uppercase text-[10px]">
              <th class="pb-2.5 font-bold">Timestamp</th>
              <th class="pb-2.5 font-bold">Server / Node</th>
              <th class="pb-2.5 font-bold text-right">Ping</th>
              <th class="pb-2.5 font-bold text-right">Download</th>
              <th class="pb-2.5 font-bold text-right">Upload</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border/60">
            {#each history as item}
              <tr class="hover:bg-card-sub transition-colors">
                <td class="py-2.5 text-foreground">{item.timestamp}</td>
                <td class="py-2.5 text-muted">
                  <span class="text-foreground font-bold">{item.server_name || 'Automatic'}</span>
                  {#if item.isp}
                    <span class="text-[10px] block opacity-75">{item.isp}</span>
                  {/if}
                </td>
                <td class="py-2.5 text-right text-foreground font-bold">
                  {item.ping ? item.ping.toFixed(1) : '0'} ms
                </td>
                <td class="py-2.5 text-right font-black text-accent">
                  {item.download ? item.download.toFixed(1) : '0'} Mbps
                </td>
                <td class="py-2.5 text-right font-black text-purple-400">
                  {item.upload ? item.upload.toFixed(1) : '0'} Mbps
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

      <!-- Card View for mobile screens -->
      <div class="sm:hidden space-y-2.5 font-mono text-xs">
        {#each history as item}
          <div class="p-3 rounded bg-card-sub border border-border space-y-2">
            <div class="flex items-center justify-between">
              <span class="text-[10px] text-muted">{item.timestamp}</span>
              <span class="text-[10px] font-bold text-foreground">{item.server_name || 'Speedtest'}</span>
            </div>
            <div class="grid grid-cols-3 gap-1 pt-1 border-t border-border/60 text-center">
              <div>
                <span class="text-[9px] text-muted block uppercase font-bold">Ping</span>
                <span class="font-bold text-foreground">{item.ping ? item.ping.toFixed(0) : '0'} ms</span>
              </div>
              <div>
                <span class="text-[9px] text-muted block uppercase font-bold">Down</span>
                <span class="font-black text-accent">{item.download ? item.download.toFixed(1) : '0'} M</span>
              </div>
              <div>
                <span class="text-[9px] text-muted block uppercase font-bold">Up</span>
                <span class="font-black text-purple-400">{item.upload ? item.upload.toFixed(1) : '0'} M</span>
              </div>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </Card>
</div>
