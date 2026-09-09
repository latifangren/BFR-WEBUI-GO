<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Network,
    RefreshCw,
    Sliders,
    Globe,
    Zap,
    RotateCcw,
    Check,
    Radio,
    Search,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Input from '../../ui/Input.svelte'

  // Tweaks state
  let tweaks = $state<Record<string, boolean>>({
    tcp_bbr_enabled: true,
    low_latency_mode: true,
    fastopen_enabled: true,
    packet_steering_rps: false,
    mtu_tuning: false,
    ipv6_disable: false,
    buffer_tuning: true,
  })
  let isLoadingTweaks = $state(false)

  // TTL state
  let currentTTL = $state('64')
  let selectedTTL = $state('64')
  let isApplyingTTL = $state(false)

  // DNS state
  let activeDNS1 = $state('1.1.1.1')
  let activeDNS2 = $state('1.0.0.1')
  let customDNS1 = $state('')
  let customDNS2 = $state('')
  let isApplyingDNS = $state(false)

  const dnsPresets = [
    { name: 'Cloudflare', p: '1.1.1.1', s: '1.0.0.1' },
    { name: 'Google', p: '8.8.8.8', s: '8.8.4.4' },
    { name: 'AdGuard', p: '94.140.14.14', s: '94.140.15.15' },
    { name: 'Quad9', p: '9.9.9.9', s: '149.112.112.112' },
  ]

  // Ping state
  let pingHost = $state('1.1.1.1')
  let pingResult = $state<string | null>(null)
  let isPinging = $state(false)

  onMount(async () => {
    await fetchNetworkData()
  })

  async function fetchNetworkData() {
    try {
      isLoadingTweaks = true
      const tweaksRes = await api.get<Record<string, unknown>>('/api/network/tweaks')
      if (typeof tweaksRes === 'object' && tweaksRes !== null) {
        Object.keys(tweaksRes).forEach((key) => {
          if (typeof tweaksRes[key] === 'boolean') {
            tweaks[key] = tweaksRes[key] as boolean
          }
        })
      }
    } catch {
      // Keep defaults if endpoint is stubbed
    } finally {
      isLoadingTweaks = false
    }

    try {
      const ttlRes = await api.get<{ ttl?: string | number; current_ttl?: string | number }>('/api/network/ttl')
      const val = ttlRes.current_ttl || ttlRes.ttl
      if (val) {
        currentTTL = String(val)
        selectedTTL = String(val)
      }
    } catch {
      // Ignored
    }

    try {
      const dnsRes = await api.get<{ primary?: string; secondary?: string; dns1?: string; dns2?: string }>('/api/network/dns')
      if (dnsRes.primary || dnsRes.dns1) activeDNS1 = dnsRes.primary || dnsRes.dns1 || '1.1.1.1'
      if (dnsRes.secondary || dnsRes.dns2) activeDNS2 = dnsRes.secondary || dnsRes.dns2 || '1.0.0.1'
    } catch {
      // Ignored
    }
  }

  async function saveTweaks() {
    try {
      isLoadingTweaks = true
      await api.post('/api/network/tweaks?action=save_tweaks', tweaks)
      toastStore.success('Network optimizations saved and applied successfully.')
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to save tweaks')
    } finally {
      isLoadingTweaks = false
    }
  }

  async function restoreDefaults() {
    try {
      isLoadingTweaks = true
      await api.post('/api/network/tweaks/restore')
      await fetchNetworkData()
      toastStore.success('Restored sysctl parameters to original defaults.')
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to restore defaults')
    } finally {
      isLoadingTweaks = false
    }
  }

  async function applyTTL(val: string) {
    try {
      isApplyingTTL = true
      selectedTTL = val
      const ttlNum = parseInt(val, 10) || 64
      await api.post('/api/network/ttl', { enable: true, ttl: ttlNum })
      currentTTL = val
      toastStore.success(`TTL set to ${val}`)
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to set TTL')
    } finally {
      isApplyingTTL = false
    }
  }

  async function setDNS(p: string, s: string) {
    try {
      isApplyingDNS = true
      await api.post('/api/network/dns', { primary: p, secondary: s })
      activeDNS1 = p
      activeDNS2 = s
      toastStore.success(`DNS set to ${p} / ${s}`)
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to set DNS')
    } finally {
      isApplyingDNS = false
    }
  }

  async function runPing() {
    if (!pingHost) return
    try {
      isPinging = true
      pingResult = null
      const res = await api.post<{ output?: string; latency_ms?: number }>('/api/network/ping', { host: pingHost })
      if (res.output) {
        pingResult = res.output
      } else if (res.latency_ms !== undefined) {
        pingResult = `Host: ${pingHost}\nLatency: ${res.latency_ms} ms`
      } else {
        pingResult = JSON.stringify(res, null, 2)
      }
    } catch (err: unknown) {
      pingResult = err instanceof Error ? `Ping error: ${err.message}` : 'Ping failed'
    } finally {
      isPinging = false
    }
  }
</script>

<div class="space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Network class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        Network Tuning & Tweaks
      </h2>
    </div>
    <div class="flex items-center gap-2">
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoadingTweaks}
        onclick={fetchNetworkData}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoadingTweaks ? 'animate-spin' : ''}" />
        <span>Refresh</span>
      </Button>
    </div>
  </div>

  <!-- Dynamic Optimizations Card -->
  <Card title="Dynamic Optimizations & Sysctl Tweaks" subtitle="Hardware and kernel tuning (saved to tweaks.json)">
    {#snippet action()}
      <div class="flex items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          disabled={isLoadingTweaks}
          onclick={restoreDefaults}
        >
          <RotateCcw class="w-3 h-3 mr-1.5" />
          <span>Restore</span>
        </Button>
        <Button
          variant="primary"
          size="sm"
          disabled={isLoadingTweaks}
          onclick={saveTweaks}
        >
          <Check class="w-3 h-3 mr-1.5" />
          <span>Apply</span>
        </Button>
      </div>
    {/snippet}

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 font-mono text-xs">
      <label class="flex items-center justify-between p-3 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
        <div>
          <span class="font-bold text-foreground block">TCP BBR Congestion</span>
          <span class="text-[10px] text-muted">Bottleneck Bandwidth and RTT</span>
        </div>
        <input type="checkbox" bind:checked={tweaks.tcp_bbr_enabled} class="w-4 h-4 accent-accent" />
      </label>

      <label class="flex items-center justify-between p-3 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
        <div>
          <span class="font-bold text-foreground block">Low Latency Mode</span>
          <span class="text-[10px] text-muted">Aggressive ACK & TCP nodelay</span>
        </div>
        <input type="checkbox" bind:checked={tweaks.low_latency_mode} class="w-4 h-4 accent-accent" />
      </label>

      <label class="flex items-center justify-between p-3 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
        <div>
          <span class="font-bold text-foreground block">TCP Fast Open (TFO)</span>
          <span class="text-[10px] text-muted">Reduce SYN roundtrip latency</span>
        </div>
        <input type="checkbox" bind:checked={tweaks.fastopen_enabled} class="w-4 h-4 accent-accent" />
      </label>

      <label class="flex items-center justify-between p-3 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
        <div>
          <span class="font-bold text-foreground block">RPS Packet Steering</span>
          <span class="text-[10px] text-muted">Distribute packet processing</span>
        </div>
        <input type="checkbox" bind:checked={tweaks.packet_steering_rps} class="w-4 h-4 accent-accent" />
      </label>

      <label class="flex items-center justify-between p-3 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
        <div>
          <span class="font-bold text-foreground block">Interface MTU Tuning</span>
          <span class="text-[10px] text-muted">Optimize MSS and MTU</span>
        </div>
        <input type="checkbox" bind:checked={tweaks.mtu_tuning} class="w-4 h-4 accent-accent" />
      </label>

      <label class="flex items-center justify-between p-3 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
        <div>
          <span class="font-bold text-foreground block">Disable IPv6</span>
          <span class="text-[10px] text-muted">Prevent IPv6 leak / routing loops</span>
        </div>
        <input type="checkbox" bind:checked={tweaks.ipv6_disable} class="w-4 h-4 accent-accent" />
      </label>

      <label class="flex items-center justify-between p-3 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
        <div>
          <span class="font-bold text-foreground block">Kernel Buffer Tuning</span>
          <span class="text-[10px] text-muted">Increase wmem and rmem caps</span>
        </div>
        <input type="checkbox" bind:checked={tweaks.buffer_tuning} class="w-4 h-4 accent-accent" />
      </label>
    </div>
  </Card>

  <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
    <!-- TTL Modifier Card -->
    <Card title="Dynamic TTL Modifier" subtitle="Bypass operator hotspot quota restriction">
      <div class="space-y-4 font-mono text-xs">
        <div class="flex items-center justify-between p-3 bg-card-sub border border-border rounded">
          <span class="text-muted uppercase font-bold">Active System TTL:</span>
          <span class="text-sm font-black text-accent">{currentTTL}</span>
        </div>

        <div>
          <span class="text-[10px] text-muted uppercase font-bold block mb-2">Preset Profiles</span>
          <div class="grid grid-cols-3 gap-2">
            <button
              type="button"
              class="neo-button p-2 text-center rounded border border-border bg-card-sub hover:border-accent cursor-pointer {selectedTTL === '64' ? 'border-accent bg-accent/10 text-accent font-bold' : 'text-foreground'}"
              onclick={() => applyTTL('64')}
            >
              <div class="text-sm font-black">64</div>
              <div class="text-[10px] text-muted">Default Android</div>
            </button>

            <button
              type="button"
              class="neo-button p-2 text-center rounded border border-border bg-card-sub hover:border-accent cursor-pointer {selectedTTL === '65' ? 'border-accent bg-accent/10 text-accent font-bold' : 'text-foreground'}"
              onclick={() => applyTTL('65')}
            >
              <div class="text-sm font-black">65</div>
              <div class="text-[10px] text-muted">Hotspot Bypass</div>
            </button>

            <button
              type="button"
              class="neo-button p-2 text-center rounded border border-border bg-card-sub hover:border-accent cursor-pointer {selectedTTL === '128' ? 'border-accent bg-accent/10 text-accent font-bold' : 'text-foreground'}"
              onclick={() => applyTTL('128')}
            >
              <div class="text-sm font-black">128</div>
              <div class="text-[10px] text-muted">Windows Target</div>
            </button>
          </div>
        </div>

        <div class="flex items-center gap-2 pt-2 border-t border-border">
          <Input
            type="number"
            placeholder="Custom TTL (e.g. 65)"
            bind:value={selectedTTL}
          />
          <Button
            variant="primary"
            size="md"
            disabled={isApplyingTTL || !selectedTTL}
            onclick={() => applyTTL(selectedTTL)}
          >
            Apply
          </Button>
        </div>
      </div>
    </Card>

    <!-- DNS Resolver Card -->
    <Card title="DNS Resolver Switcher" subtitle="System-wide upstream DNS routing">
      <div class="space-y-4 font-mono text-xs">
        <div class="flex items-center justify-between p-3 bg-card-sub border border-border rounded">
          <span class="text-muted uppercase font-bold">Active DNS:</span>
          <span class="text-xs font-bold text-accent truncate max-w-[200px]">
            {activeDNS1} / {activeDNS2}
          </span>
        </div>

        <div>
          <span class="text-[10px] text-muted uppercase font-bold block mb-2">Upstream Presets</span>
          <div class="grid grid-cols-2 gap-2">
            {#each dnsPresets as preset}
              <button
                type="button"
                class="neo-button p-2.5 rounded border border-border bg-card-sub hover:border-accent text-left cursor-pointer transition-colors {activeDNS1 === preset.p ? 'border-accent bg-accent/10' : ''}"
                onclick={() => setDNS(preset.p, preset.s)}
              >
                <div class="font-bold text-foreground text-xs">{preset.name}</div>
                <div class="text-[10px] text-muted">{preset.p} / {preset.s}</div>
              </button>
            {/each}
          </div>
        </div>

        <div class="space-y-2 pt-2 border-t border-border">
          <div class="grid grid-cols-2 gap-2">
            <Input placeholder="Primary (e.g. 1.1.1.1)" bind:value={customDNS1} />
            <Input placeholder="Secondary (e.g. 1.0.0.1)" bind:value={customDNS2} />
          </div>
          <Button
            variant="secondary"
            size="sm"
            fullWidth={true}
            disabled={isApplyingDNS || !customDNS1}
            onclick={() => setDNS(customDNS1, customDNS2 || customDNS1)}
          >
            Apply Custom DNS
          </Button>
        </div>
      </div>
    </Card>
  </div>

  <!-- Network Ping Diagnostic -->
  <Card title="Ping Diagnostics" subtitle="Test network latency and reachability">
    <div class="space-y-4 font-mono text-xs">
      <div class="flex items-center gap-2">
        <Input
          placeholder="Target host (e.g. 1.1.1.1, google.com)"
          bind:value={pingHost}
        />
        <Button
          variant="primary"
          size="md"
          disabled={isPinging || !pingHost}
          onclick={runPing}
        >
          <Search class="w-3.5 h-3.5 mr-1.5" />
          <span>{isPinging ? 'Pinging...' : 'Ping'}</span>
        </Button>
      </div>

      {#if pingResult}
        <pre class="bg-card-sub border border-border p-3 rounded text-[11px] font-mono text-muted overflow-x-auto whitespace-pre-wrap">{pingResult}</pre>
      {/if}
    </div>
  </Card>
</div>
