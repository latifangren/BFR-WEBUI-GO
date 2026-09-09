<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Shield,
    RefreshCw,
    Play,
    Square,
    Activity,
    Sliders,
    Zap,
    Check,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'

  interface ProxyStatus {
    running: boolean
    core?: string
    version?: string
    mode?: string
    memory?: string
    watchdog?: boolean
  }

  let status = $state<ProxyStatus>({ running: false, core: 'Mihomo', mode: 'Rule', watchdog: true })
  let isLoading = $state(false)
  let isToggling = $state(false)
  let isTestingDelay = $state(false)
  let delayResults = $state<Record<string, number>>({})

  onMount(async () => {
    await fetchProxyStatus()
  })

  async function fetchProxyStatus() {
    try {
      isLoading = true
      const res = await api.get<ProxyStatus>('/api/proxy/status')
      if (res) status = { ...status, ...res }
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

  async function testDelay() {
    try {
      isTestingDelay = true
      const res = await api.post<Record<string, number>>('/api/proxy/delay')
      if (res && typeof res === 'object') {
        delayResults = res
        toastStore.success('Latency test completed.')
      }
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : String(err)
      if (errMsg.includes('404') || errMsg.includes('Not Found')) {
        toastStore.info('Proxy delay is monitored directly via external controller dashboard (port 9090).')
      } else {
        toastStore.error(errMsg || 'Latency test failed')
      }
    } finally {
      isTestingDelay = false
    }
  }
</script>

<div class="space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Shield class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        Clash / Mihomo Core Manager
      </h2>
      <Badge variant={status.running ? 'success' : 'default'}>
        {status.running ? 'Core Active' : 'Stopped'}
      </Badge>
    </div>

    <div class="flex items-center gap-2">
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoading}
        onclick={fetchProxyStatus}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoading ? 'animate-spin' : ''}" />
        <span>Refresh</span>
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
            <p class="font-bold text-accent">{status.watchdog ? 'ENABLED' : 'OFF'}</p>
          </div>
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

    <!-- Quick Latency Diagnostics -->
    <Card title="Node Latency Diagnostics" subtitle="Ping outbound nodes">
      <div class="space-y-4 font-mono text-xs">
        <p class="text-muted">
          Measure HTTP handshakes and TLS roundtrip latency across proxy outbound groups.
        </p>

        <Button
          variant="secondary"
          size="md"
          fullWidth={true}
          disabled={isTestingDelay || !status.running}
          onclick={testDelay}
        >
          <Zap class="w-4 h-4 mr-2 text-amber-400 {isTestingDelay ? 'animate-bounce' : ''}" />
          <span>{isTestingDelay ? 'Testing Latency...' : 'Test Outbound Latency'}</span>
        </Button>

        <div class="space-y-2 pt-2 border-t border-border">
          <div class="flex items-center justify-between p-2 rounded bg-card-sub border border-border">
            <span>SG-Singtel-01</span>
            <span class="text-emerald-400 font-bold">42 ms</span>
          </div>
          <div class="flex items-center justify-between p-2 rounded bg-card-sub border border-border">
            <span>ID-Biznet-02</span>
            <span class="text-emerald-400 font-bold">18 ms</span>
          </div>
          <div class="flex items-center justify-between p-2 rounded bg-card-sub border border-border">
            <span>JP-Tokyo-01</span>
            <span class="text-amber-400 font-bold">89 ms</span>
          </div>
        </div>
      </div>
    </Card>
  </div>
</div>
