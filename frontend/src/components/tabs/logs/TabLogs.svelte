<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import {
    FileText,
    RefreshCw,
    Play,
    Pause,
    Trash2,
    Search,
    Filter,
    Activity,
    Shield,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import { WebSocketClient } from '../../../ws/socket'
  import type { LogEntry, LogResponse } from '../../../types/logs'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Input from '../../ui/Input.svelte'

  // Log Mode: 'daemon' | 'logcat'
  let logMode = $state<'daemon' | 'logcat'>('daemon')

  // Daemon Logs State
  let daemonLogs = $state<LogEntry[]>([])
  let isLoadingDaemon = $state(false)
  let selectedCategory = $state('')
  let selectedLevel = $state('')
  let daemonSearch = $state('')

  // Logcat Live Stream State
  let logcatLines = $state<string[]>([])
  let isLogcatStreaming = $state(false)
  let logcatSearch = $state('')
  let autoScroll = $state(true)
  let logcatContainer: HTMLDivElement | null = $state(null)
  let logcatClient: WebSocketClient | null = null

  onMount(() => {
    fetchDaemonLogs()
    return () => {
      stopLogcatStream()
    }
  })

  onDestroy(() => {
    stopLogcatStream()
  })

  // Fetch Daemon Logs
  async function fetchDaemonLogs() {
    try {
      isLoadingDaemon = true
      const catQuery = selectedCategory ? `&category=${encodeURIComponent(selectedCategory)}` : ''
      const res = await api.get<LogResponse | { entries: LogEntry[] }>(`/api/logs?limit=150${catQuery}`)
      if (res && Array.isArray(res.entries)) {
        daemonLogs = res.entries
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to fetch logs')
    } finally {
      isLoadingDaemon = false
    }
  }

  async function clearDaemonLogs() {
    try {
      await api.post('/api/logs/clear')
      daemonLogs = []
      toastStore.success('Daemon log buffer cleared.')
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to clear logs')
    }
  }

  // Live Logcat Stream Handling
  function startLogcatStream() {
    if (logcatClient && logcatClient.readyState === WebSocket.OPEN) {
      return
    }

    logcatClient = new WebSocketClient({
      path: '/api/logs/logcat/stream',
      autoReconnect: true,
      onOpen: () => {
        isLogcatStreaming = true
        logcatLines = [...logcatLines, '[+] Connected to Android Logcat Kernel Pipe']
      },
      onMessage: (data) => {
        if (typeof data === 'string') {
          const split = data.split('\n').filter((l) => l.trim() !== '')
          // Cap buffer at 500 lines to prevent Android memory exhaustion
          const next = [...logcatLines, ...split]
          if (next.length > 500) {
            logcatLines = next.slice(next.length - 500)
          } else {
            logcatLines = next
          }

          if (autoScroll && logcatContainer) {
            setTimeout(() => {
              if (logcatContainer) {
                logcatContainer.scrollTop = logcatContainer.scrollHeight
              }
            }, 50)
          }
        }
      },
      onClose: () => {
        isLogcatStreaming = false
      },
      onError: () => {
        isLogcatStreaming = false
      },
    })

    logcatClient.connect()
  }

  function stopLogcatStream() {
    if (logcatClient) {
      logcatClient.disconnect()
      logcatClient = null
    }
    isLogcatStreaming = false
  }

  function clearLogcatBuffer() {
    logcatLines = []
  }

  const filteredDaemonLogs = $derived(
    daemonLogs.filter((entry) => {
      const matchLevel = !selectedLevel || entry.level.toUpperCase() === selectedLevel.toUpperCase()
      const matchSearch =
        !daemonSearch ||
        entry.message.toLowerCase().includes(daemonSearch.toLowerCase()) ||
        entry.category.toLowerCase().includes(daemonSearch.toLowerCase())
      return matchLevel && matchSearch
    })
  )

  const filteredLogcatLines = $derived(
    logcatSearch
      ? logcatLines.filter((l) => l.toLowerCase().includes(logcatSearch.toLowerCase()))
      : logcatLines
  )

  function getLevelBadgeVariant(level: string): 'default' | 'success' | 'warning' | 'danger' | 'info' {
    switch (level.toUpperCase()) {
      case 'ERROR':
        return 'danger'
      case 'WARN':
      case 'WARNING':
        return 'warning'
      case 'INFO':
        return 'info'
      case 'DEBUG':
        return 'default'
      default:
        return 'default'
    }
  }
</script>

<div class="space-y-4">
  <!-- Header Toolbar -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
    <div class="flex items-center gap-2">
      <FileText class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        System Logs & Logcat Stream
      </h2>
    </div>

    <!-- Mode Switcher -->
    <div class="flex items-center gap-1 font-mono text-xs bg-card p-1 rounded border border-border">
      <button
        type="button"
        class="px-3 py-1.5 rounded cursor-pointer transition-colors {logMode === 'daemon' ? 'bg-accent text-accent-text font-bold' : 'text-muted hover:text-foreground'}"
        onclick={() => (logMode = 'daemon')}
      >
        Daemon Logs
      </button>
      <button
        type="button"
        class="px-3 py-1.5 rounded cursor-pointer transition-colors {logMode === 'logcat' ? 'bg-accent text-accent-text font-bold' : 'text-muted hover:text-foreground'}"
        onclick={() => {
          logMode = 'logcat'
          if (!isLogcatStreaming) startLogcatStream()
        }}
      >
        Live Logcat
      </button>
    </div>
  </div>

  <!-- DAEMON LOGS VIEW -->
  {#if logMode === 'daemon'}
    <div class="space-y-4">
      <!-- Filter Bar -->
      <div class="grid grid-cols-1 sm:grid-cols-4 gap-2 font-mono text-xs">
        <Input
          placeholder="Search message or category..."
          bind:value={daemonSearch}
        />

        <select
          bind:value={selectedLevel}
          class="bg-card-sub border border-border rounded px-3 py-2 text-xs font-mono text-foreground focus:outline-none focus:border-accent"
        >
          <option value="">All Log Levels</option>
          <option value="INFO">INFO</option>
          <option value="WARN">WARN</option>
          <option value="ERROR">ERROR</option>
          <option value="DEBUG">DEBUG</option>
        </select>

        <select
          bind:value={selectedCategory}
          onchange={fetchDaemonLogs}
          class="bg-card-sub border border-border rounded px-3 py-2 text-xs font-mono text-foreground focus:outline-none focus:border-accent"
        >
          <option value="">All Categories</option>
          <option value="auth">auth</option>
          <option value="network">network</option>
          <option value="hotspot">hotspot</option>
          <option value="charger">charger</option>
          <option value="power">power</option>
          <option value="terminal">terminal</option>
          <option value="modem">modem</option>
          <option value="proxy">proxy</option>
        </select>

        <div class="flex items-center gap-2">
          <Button
            variant="secondary"
            size="md"
            disabled={isLoadingDaemon}
            onclick={fetchDaemonLogs}
            class="flex-1"
          >
            <RefreshCw class="w-3.5 h-3.5 mr-1 {isLoadingDaemon ? 'animate-spin' : ''}" />
            <span>Refresh</span>
          </Button>
          <Button
            variant="outline"
            size="md"
            onclick={clearDaemonLogs}
            title="Clear buffer"
          >
            <Trash2 class="w-3.5 h-3.5 text-red-400" />
          </Button>
        </div>
      </div>

      <!-- Logs Output Container -->
      <Card class="p-0 overflow-hidden font-mono text-xs">
        {#if isLoadingDaemon && daemonLogs.length === 0}
          <div class="p-12 text-center text-muted">
            <RefreshCw class="w-6 h-6 animate-spin mx-auto mb-2 text-accent" />
            Reading log memory buffer...
          </div>
        {:else if filteredDaemonLogs.length === 0}
          <div class="p-12 text-center text-muted">
            No daemon logs matching filter criteria.
          </div>
        {:else}
          <div class="max-h-[65vh] overflow-y-auto divide-y divide-border">
            {#each filteredDaemonLogs as log}
              <div class="p-3 flex items-start gap-3 hover:bg-card-sub/60 transition-colors">
                <span class="text-[10px] text-muted shrink-0 w-28 truncate">
                  {log.timestamp ? new Date(log.timestamp).toLocaleTimeString() : '—'}
                </span>
                <Badge variant={getLevelBadgeVariant(log.level)} class="shrink-0">
                  {log.level}
                </Badge>
                <span class="text-[10px] font-bold text-accent shrink-0 px-1.5 py-0.5 rounded bg-card-sub border border-border">
                  {log.category}
                </span>
                <span class="text-foreground break-all flex-1">
                  {log.message}
                </span>
              </div>
            {/each}
          </div>
        {/if}
      </Card>
    </div>
  {/if}

  <!-- LIVE LOGCAT STREAM VIEW -->
  {#if logMode === 'logcat'}
    <div class="space-y-4">
      <!-- Toolbar -->
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 font-mono text-xs">
        <div class="flex items-center gap-2 flex-1">
          <Input
            placeholder="Filter logcat regex / text..."
            bind:value={logcatSearch}
            class="max-w-md"
          />
          <label class="flex items-center gap-1.5 text-muted select-none cursor-pointer">
            <input type="checkbox" bind:checked={autoScroll} class="accent-accent" />
            <span>Auto-scroll</span>
          </label>
        </div>

        <div class="flex items-center gap-2">
          {#if isLogcatStreaming}
            <Button variant="danger" size="sm" onclick={stopLogcatStream}>
              <Pause class="w-3.5 h-3.5 mr-1" />
              <span>Pause</span>
            </Button>
          {:else}
            <Button variant="primary" size="sm" onclick={startLogcatStream}>
              <Play class="w-3.5 h-3.5 mr-1" />
              <span>Resume</span>
            </Button>
          {/if}
          <Button variant="outline" size="sm" onclick={clearLogcatBuffer}>
            <Trash2 class="w-3.5 h-3.5 mr-1" />
            <span>Clear</span>
          </Button>
        </div>
      </div>

      <!-- Logcat Terminal Display -->
      <div class="neo-card bg-card border-2 border-border shadow-neobrutal rounded-lg overflow-hidden flex flex-col">
        <div class="bg-card-sub border-b border-border px-3 py-2 flex items-center justify-between font-mono text-xs text-muted">
          <div class="flex items-center gap-2">
            <span class="w-2.5 h-2.5 rounded-full {isLogcatStreaming ? 'bg-emerald-500 animate-pulse' : 'bg-red-500'} inline-block"></span>
            <span class="font-bold text-foreground">Android OS Logcat Stream</span>
          </div>
          <span>{filteredLogcatLines.length} lines</span>
        </div>

        <div
          bind:this={logcatContainer}
          class="w-full h-[65vh] p-3 bg-[#090d16] font-mono text-[11px] text-gray-300 overflow-y-auto space-y-0.5 whitespace-pre-wrap leading-relaxed select-text"
        >
          {#each filteredLogcatLines as line}
            <div class="hover:bg-white/5 px-1 rounded {line.includes(' E ') || line.includes('Fatal') ? 'text-red-400 font-bold' : line.includes(' W ') ? 'text-amber-300' : line.includes(' I ') ? 'text-emerald-300' : 'text-gray-400'}">
              {line}
            </div>
          {/each}
        </div>
      </div>
    </div>
  {/if}
</div>
