<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import {
    Shield,
    RefreshCw,
    Play,
    Square,
    Activity,
    Sliders,
    Zap,
    Check,
    FileCode,
    Save,
    RotateCcw,
    Terminal,
    Pause,
    Trash2,
    ExternalLink,
    ChevronDown,
    ChevronUp,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Modal from '../../ui/Modal.svelte'
  import Input from '../../ui/Input.svelte'

  interface ProxyCoreInfo {
    name: string
    path: string
    exists: boolean
    running: boolean
    pid: number
    memory: string
  }

  interface ProxyStatusResponse {
    cores: ProxyCoreInfo[]
    mode: string
    watchdog: boolean
  }

  interface ProxyStatus {
    running: boolean
    core?: string
    version?: string
    mode?: string
    memory?: string
    watchdog?: boolean
  }

  let status = $state<ProxyStatus>({ running: false, core: 'Mihomo', mode: 'Rule', watchdog: false })
  let isLoading = $state(false)
  let isToggling = $state(false)

  // Watchdog state
  let watchdog = $state(false)
  let isTogglingWatchdog = $state(false)

  // YAML Config Editor state
  let isConfigModalOpen = $state(false)
  let configPath = $state('')
  let configContent = $state('')
  let isLoadingConfig = $state(false)
  let isSavingConfig = $state(false)

  // Real-time Proxy Logs state
  let proxyLogs = $state<string[]>([])
  let isStreamingLogs = $state(false)
  let isLogsExpanded = $state(true)
  let logAutoScroll = $state(true)
  let logSearch = $state('')
  let logsContainer: HTMLDivElement | null = $state(null)
  let eventSource: EventSource | null = null

  onMount(async () => {
    await fetchProxyStatus()
    startLogStream()
  })

  onDestroy(() => {
    stopLogStream()
  })

  async function fetchProxyStatus() {
    try {
      isLoading = true
      const [resStatus, resWd] = await Promise.allSettled([
        api.get<ProxyStatusResponse>('/api/proxy/status'),
        api.get<{ watchdog: boolean }>('/api/proxy/watchdog'),
      ])

      if (resStatus.status === 'fulfilled' && resStatus.value) {
        const val = resStatus.value
        const activeCore = val.cores?.find((c) => c.running)

        if (activeCore) {
          status.running = true
          status.core = activeCore.name
          status.memory = activeCore.memory || '-'
        } else {
          status.running = false
          status.core = val.cores?.[0]?.name || 'Mihomo'
          status.memory = '-'
        }

        if (val.mode) {
          status.mode = val.mode
        }

        if (typeof val.watchdog === 'boolean') {
          watchdog = val.watchdog
          status.watchdog = val.watchdog
        }
      }

      if (resWd.status === 'fulfilled' && resWd.value && typeof resWd.value.watchdog === 'boolean') {
        watchdog = resWd.value.watchdog
        status.watchdog = resWd.value.watchdog
      }
    } catch {
      // Ignored
    } finally {
      isLoading = false
    }
  }

  async function controlProxy(action: 'start' | 'stop' | 'restart') {
    try {
      isToggling = true
      await api.post('/api/proxy/control', { action })
      status.running = action !== 'stop'
      toastStore.success(`Proxy core ${action}ed successfully.`)
      await fetchProxyStatus()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Proxy control failed')
    } finally {
      isToggling = false
    }
  }

  async function setMode(mode: string) {
    try {
      await api.post('/api/proxy/control', { mode: mode.toLowerCase() })
      status.mode = mode
      toastStore.success(`Proxy mode switched to ${mode}.`)
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to switch mode')
    }
  }

  async function toggleWatchdog() {
    try {
      isTogglingWatchdog = true
      const nextState = !watchdog
      const res = await api.post<{ success: boolean; watchdog: boolean }>('/api/proxy/watchdog', {
        enable: nextState,
      })
      watchdog = res?.watchdog !== undefined ? res.watchdog : nextState
      status.watchdog = watchdog
      toastStore.success(`Proxy Watchdog ${watchdog ? 'enabled' : 'disabled'}.`)
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to update watchdog')
    } finally {
      isTogglingWatchdog = false
    }
  }

  async function openConfigModal() {
    isConfigModalOpen = true
    try {
      isLoadingConfig = true
      const res = await api.get<{ path: string; content?: string; error?: string }>('/api/proxy/config')
      if (res) {
        configPath = res.path || ''
        configContent = res.content || ''
        if (res.error) {
          toastStore.warning(res.error, 'Config Notice')
        }
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to load config')
    } finally {
      isLoadingConfig = false
    }
  }

  async function saveConfig() {
    try {
      isSavingConfig = true
      const res = await api.post<{ success: boolean; error?: string }>('/api/proxy/config', {
        content: configContent,
      })
      if (res && res.success) {
        toastStore.success('Proxy configuration saved successfully.')
        isConfigModalOpen = false
      } else {
        toastStore.error(res?.error || 'Failed to save configuration')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Save failed')
    } finally {
      isSavingConfig = false
    }
  }

  function startLogStream() {
    if (eventSource) return
    try {
      eventSource = new EventSource('/api/proxy/logs')
      isStreamingLogs = true

      eventSource.onmessage = (e) => {
        if (!e.data) return
        const lines = String(e.data).split('\n').filter(Boolean)
        proxyLogs = [...proxyLogs.slice(-(500 - lines.length)), ...lines]
        if (logAutoScroll && logsContainer) {
          requestAnimationFrame(() => {
            if (logsContainer) logsContainer.scrollTop = logsContainer.scrollHeight
          })
        }
      }

      eventSource.onerror = () => {
        // Handled silently; browser automatically retries EventSource
      }
    } catch {
      isStreamingLogs = false
    }
  }

  function stopLogStream() {
    if (eventSource) {
      eventSource.close()
      eventSource = null
    }
    isStreamingLogs = false
  }

  function toggleStream() {
    if (isStreamingLogs) {
      stopLogStream()
    } else {
      startLogStream()
    }
  }

  function clearLogs() {
    proxyLogs = []
  }

  let filteredLogs = $derived(
    logSearch.trim() === ''
      ? proxyLogs
      : proxyLogs.filter((l) => l.toLowerCase().includes(logSearch.toLowerCase()))
  )
</script>

<div class="space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
    <div class="flex items-center gap-2">
      <Shield class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        Clash / Mihomo Core Manager
      </h2>
      <Badge variant={status.running ? 'success' : 'default'}>
        {status.running ? 'Core Active' : 'Stopped'}
      </Badge>
    </div>

    <div class="flex flex-wrap items-center gap-2">
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoading}
        onclick={fetchProxyStatus}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoading ? 'animate-spin' : ''}" />
        <span>Refresh</span>
      </Button>

      <Button
        variant="secondary"
        size="sm"
        onclick={openConfigModal}
      >
        <FileCode class="w-3.5 h-3.5 mr-1.5 text-accent" />
        <span>Edit Config</span>
      </Button>

      {#if status.running}
        <Button
          variant="danger"
          size="sm"
          disabled={isToggling}
          onclick={() => controlProxy('stop')}
        >
          <Square class="w-3.5 h-3.5 mr-1.5" />
          <span>Stop Core</span>
        </Button>
      {:else}
        <Button
          variant="primary"
          size="sm"
          disabled={isToggling}
          onclick={() => controlProxy('start')}
        >
          <Play class="w-3.5 h-3.5 mr-1.5" />
          <span>Start Core</span>
        </Button>
      {/if}
    </div>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <!-- Status & Routing Mode -->
    <Card title="Proxy Core Status & Mode" class="lg:col-span-2">
      <div class="space-y-5 font-mono text-xs">
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
          <div class="bg-card-sub border border-border p-3 rounded space-y-1">
            <span class="text-[10px] text-muted uppercase font-bold">Core Binary</span>
            <p class="font-bold text-foreground">{status.core || 'Mihomo'}</p>
          </div>
          <div class="bg-card-sub border border-border p-3 rounded space-y-1">
            <span class="text-[10px] text-muted uppercase font-bold">Version</span>
            <p class="font-bold text-foreground">{status.version || 'v1.18.x'}</p>
          </div>
          <div class="bg-card-sub border border-border p-3 rounded space-y-1">
            <span class="text-[10px] text-muted uppercase font-bold">Memory</span>
            <p class="font-bold text-emerald-400">{status.memory || '28.4 MB'}</p>
          </div>
          <div class="bg-card-sub border border-border p-3 rounded space-y-1">
            <span class="text-[10px] text-muted uppercase font-bold">Watchdog</span>
            <p class="font-bold {watchdog ? 'text-emerald-400' : 'text-muted'}">{watchdog ? 'ENABLED' : 'DISABLED'}</p>
          </div>
        </div>

        <!-- Watchdog Auto-Restart Neo-Brutalist Switch -->
        <div class="flex items-center justify-between p-3.5 bg-card-sub border-2 border-border rounded-md">
          <div class="space-y-0.5">
            <div class="flex items-center gap-2">
              <span class="font-bold text-foreground font-mono text-sm">Watchdog Auto-Restart</span>
              <Badge variant={watchdog ? 'success' : 'default'}>
                {watchdog ? 'ACTIVE' : 'DISABLED'}
              </Badge>
            </div>
            <p class="text-[11px] text-muted font-mono">
              Monitors Mihomo/Clash core and automatically restarts the daemon every 10s if terminated.
            </p>
          </div>

          <button
            type="button"
            role="switch"
            aria-checked={watchdog}
            disabled={isTogglingWatchdog}
            onclick={toggleWatchdog}
            title={watchdog ? 'Disable Watchdog' : 'Enable Watchdog'}
            class="relative inline-flex h-7 w-14 shrink-0 cursor-pointer rounded-full border-2 border-border transition-colors duration-200 ease-in-out focus:outline-none disabled:opacity-50 {watchdog ? 'bg-accent shadow-neobrutal-sm' : 'bg-card'}"
          >
            <span
              class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white border border-black shadow transition duration-200 ease-in-out mt-[2px] {watchdog ? 'translate-x-7' : 'translate-x-1'}"
            ></span>
          </button>
        </div>

        <!-- Mode Select Buttons -->
        <div>
          <span class="text-[10px] text-muted uppercase font-bold block mb-2">
            Routing Mode
          </span>
          <div class="grid grid-cols-3 gap-3">
            {#each ['Rule', 'Global', 'Direct'] as m}
              <button
                type="button"
                class="neo-button p-3 rounded border border-border bg-card-sub hover:border-accent text-center cursor-pointer transition-all {status.mode?.toLowerCase() === m.toLowerCase() ? 'border-accent bg-accent/15 text-accent font-bold shadow-neobrutal-sm' : 'text-foreground'}"
                onclick={() => setMode(m)}
              >
                <div class="font-black text-sm uppercase">{m}</div>
                <div class="text-[10px] text-muted mt-0.5">
                  {m === 'Rule' ? 'Routing By GeoIP' : m === 'Global' ? 'Tunnel All Traffic' : 'Bypass All'}
                </div>
              </button>
            {/each}
          </div>
        </div>
      </div>
    </Card>

    <!-- Proxy Controller & WebUI Info -->
    <Card title="Proxy Controller & Dashboard" subtitle="Mihomo / Clash external controller">
      <div class="space-y-4 font-mono text-xs">
        <p class="text-muted leading-relaxed">
          Node latency checks, proxy selector groups, and rule routing matrices are managed directly through the external REST API and WebUI.
        </p>

        <div class="space-y-2 border-t border-border pt-3">
          <div class="flex items-center justify-between p-2 rounded bg-card-sub border border-border">
            <span class="text-muted">REST Controller</span>
            <span class="text-accent font-bold">127.0.0.1:9090</span>
          </div>
          <div class="flex items-center justify-between p-2 rounded bg-card-sub border border-border">
            <span class="text-muted">DNS Listen Port</span>
            <span class="text-emerald-400 font-bold">1053</span>
          </div>
          <div class="flex items-center justify-between p-2 rounded bg-card-sub border border-border">
            <span class="text-muted">TProxy / Redir</span>
            <span class="text-foreground font-bold">7892</span>
          </div>
        </div>

        <div class="pt-2">
          <Button
            variant="outline"
            size="md"
            fullWidth={true}
            onclick={openConfigModal}
          >
            <FileCode class="w-4 h-4 mr-2 text-accent" />
            <span>Open YAML Config</span>
          </Button>
        </div>
      </div>
    </Card>
  </div>

  <!-- Real-time Proxy Logs Section -->
  <Card class="overflow-hidden p-0">
    <div class="bg-card-sub border-b border-border p-3 sm:p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 font-mono text-xs">
      <div class="flex items-center gap-2.5">
        <Terminal class="w-4 h-4 text-accent" />
        <div>
          <div class="flex items-center gap-2">
            <span class="font-bold text-foreground uppercase tracking-wider">Proxy Core Live Logs</span>
            <span class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-[10px] font-bold border {isStreamingLogs ? 'bg-emerald-950/60 text-emerald-400 border-emerald-800' : 'bg-card text-muted border-border'}">
              <span class="w-2 h-2 rounded-full {isStreamingLogs ? 'bg-emerald-400 animate-pulse' : 'bg-muted'}"></span>
              <span>{isStreamingLogs ? 'LIVE STREAM' : 'PAUSED'}</span>
            </span>
          </div>
          <span class="text-[11px] text-muted">Streaming stdout & runs.log via Server-Sent Events</span>
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <Input
          placeholder="Filter log output..."
          bind:value={logSearch}
          class="w-36 sm:w-48 !py-1 text-xs"
        />

        <label class="flex items-center gap-1.5 text-muted select-none cursor-pointer text-[11px] mr-1">
          <input type="checkbox" bind:checked={logAutoScroll} class="accent-accent" />
          <span>Auto-scroll</span>
        </label>

        {#if isStreamingLogs}
          <Button variant="outline" size="sm" onclick={toggleStream}>
            <Pause class="w-3.5 h-3.5 mr-1 text-amber-400" />
            <span>Pause</span>
          </Button>
        {:else}
          <Button variant="primary" size="sm" onclick={toggleStream}>
            <Play class="w-3.5 h-3.5 mr-1" />
            <span>Resume</span>
          </Button>
        {/if}

        <Button variant="outline" size="sm" onclick={clearLogs} title="Clear log buffer">
          <Trash2 class="w-3.5 h-3.5 text-red-400" />
        </Button>

        <button
          type="button"
          class="p-1.5 text-muted hover:text-foreground border border-border rounded cursor-pointer transition-colors"
          onclick={() => (isLogsExpanded = !isLogsExpanded)}
          title={isLogsExpanded ? 'Collapse logs' : 'Expand logs'}
        >
          {#if isLogsExpanded}
            <ChevronUp class="w-4 h-4" />
          {:else}
            <ChevronDown class="w-4 h-4" />
          {/if}
        </button>
      </div>
    </div>

    {#if isLogsExpanded}
      <div
        bind:this={logsContainer}
        class="w-full h-80 p-3.5 bg-[#090d16] font-mono text-[11px] text-gray-300 overflow-y-auto space-y-0.5 whitespace-pre-wrap leading-relaxed select-text"
      >
        {#if filteredLogs.length === 0}
          <div class="h-full flex flex-col items-center justify-center text-muted font-mono text-xs">
            <Terminal class="w-8 h-8 mb-2 opacity-40 text-accent" />
            {#if proxyLogs.length === 0}
              <p>Waiting for proxy core logs...</p>
              <p class="text-[10px] mt-1 text-muted/70">Start Mihomo/Clash core to see live execution outputs.</p>
            {:else}
              <p>No log entries match your filter "{logSearch}".</p>
            {/if}
          </div>
        {:else}
          {#each filteredLogs as line}
            <div
              class="hover:bg-white/5 px-1 rounded break-all {line.includes('level=error') || line.includes('ERR') || line.includes('error') ? 'text-red-400 font-bold' : line.includes('level=warning') || line.includes('WARN') || line.includes('warning') ? 'text-amber-300' : line.includes('level=info') ? 'text-emerald-300' : 'text-gray-300'}"
            >
              {line}
            </div>
          {/each}
        {/if}
      </div>
    {/if}
  </Card>
</div>

<!-- YAML Configuration Modal -->
<Modal
  open={isConfigModalOpen}
  title="Clash / Mihomo YAML Configuration"
  class="max-w-4xl"
  onclose={() => (isConfigModalOpen = false)}
>
  <div class="space-y-3 font-mono text-xs">
    {#if isLoadingConfig}
      <div class="p-12 text-center text-muted">
        <RefreshCw class="w-6 h-6 animate-spin mx-auto mb-2 text-accent" />
        <p>Loading configuration file...</p>
      </div>
    {:else}
      <div class="flex items-center justify-between bg-card-sub border border-border p-2.5 rounded text-[11px]">
        <div class="truncate text-muted">
          <span class="font-bold text-foreground">File Path:</span>
          <code class="text-accent ml-1">{configPath || '/data/adb/box/clash/config.yaml'}</code>
        </div>
        <Badge variant="default">YAML</Badge>
      </div>

      <div class="relative border-2 border-border rounded overflow-hidden shadow-neobrutal-sm">
        <textarea
          bind:value={configContent}
          rows="18"
          spellcheck="false"
          class="w-full p-3.5 bg-[#090d16] font-mono text-xs text-emerald-400 placeholder:text-muted focus:outline-none leading-relaxed resize-y select-text"
          placeholder="# Enter YAML proxy configuration here..."
        ></textarea>
      </div>
    {/if}
  </div>

  {#snippet footer()}
    <Button
      variant="outline"
      size="sm"
      disabled={isSavingConfig}
      onclick={() => (isConfigModalOpen = false)}
    >
      Cancel
    </Button>
    <Button
      variant="primary"
      size="sm"
      disabled={isSavingConfig || isLoadingConfig}
      onclick={saveConfig}
    >
      <Save class="w-3.5 h-3.5 mr-1.5" />
      <span>{isSavingConfig ? 'Saving...' : 'Save Config'}</span>
    </Button>
  {/snippet}
</Modal>
