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
    Cpu,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import type { TweaksConfig, RPSConfig } from '../../../types/network'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Input from '../../ui/Input.svelte'

  // Tweaks state
  let tweaks = $state<TweaksConfig>({
    lte_carrier_aggregation: false,
    tcp_buffer_optimization: true,
    bbr2_congestion_control: true,
    sysctl_buffers_opt: true,
    dalvik_responsiveness: true,
    settings_global_tweaks: false,
    ttl_spoofing: false,
    packet_steering_rps: false,
    mtu_tuning: false,
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

  // RPS (Receive Packet Steering) state
  let rpsConfigs = $state<RPSConfig[]>([])
  let isLoadingRPS = $state(false)
  let isApplyingRPS = $state<Record<string, boolean>>({})
  let editedBitmasks = $state<Record<string, string>>({})

  onMount(async () => {
    await fetchNetworkData()
  })

  async function fetchNetworkData() {
    try {
      isLoadingTweaks = true
      const tweaksRes = await api.get<{
        tweaks_json?: TweaksConfig
        active_dns1?: string
        active_dns2?: string
        preset_dns?: Record<string, { primary: string; secondary: string }>
      }>('/api/network/tweaks')
      if (tweaksRes?.tweaks_json && typeof tweaksRes.tweaks_json === 'object') {
        tweaks = { ...tweaks, ...tweaksRes.tweaks_json }
      }
      if (tweaksRes?.active_dns1) activeDNS1 = tweaksRes.active_dns1
      if (tweaksRes?.active_dns2) activeDNS2 = tweaksRes.active_dns2
    } catch {
      // Keep defaults if endpoint is stubbed
    } finally {
      isLoadingTweaks = false
    }

    try {
      const ttlRes = await api.get<{ ttl?: string | number; current_ttl?: string | number; ttl_spoof?: boolean }>('/api/network/ttl')
      if (ttlRes?.current_ttl) {
        currentTTL = String(ttlRes.current_ttl)
        selectedTTL = String(ttlRes.current_ttl)
      } else if (ttlRes?.ttl) {
        currentTTL = String(ttlRes.ttl)
        selectedTTL = String(ttlRes.ttl)
      }
      if (ttlRes?.ttl_spoof !== undefined) {
        tweaks.ttl_spoofing = Boolean(ttlRes.ttl_spoof)
      }
    } catch {
      // Ignored
    }

    try {
      const dnsRes = await api.get<{ primary?: string; secondary?: string; dns1?: string; dns2?: string; presets?: any[] }>('/api/network/dns')
      if (dnsRes?.primary) activeDNS1 = dnsRes.primary
      else if (dnsRes?.dns1) activeDNS1 = dnsRes.dns1

      if (dnsRes?.secondary) activeDNS2 = dnsRes.secondary
      else if (dnsRes?.dns2) activeDNS2 = dnsRes.dns2
    } catch {
      // Ignored
    }

    await fetchRPSConfigs()
  }

  async function fetchRPSConfigs() {
    try {
      isLoadingRPS = true
      const res = await api.get<{ configs: RPSConfig[]; error?: string }>('/api/network/rps')
      if (res && Array.isArray(res.configs)) {
        rpsConfigs = res.configs
        const masks: Record<string, string> = {}
        for (const c of res.configs) {
          masks[c.interface] = c.bitmask || 'f'
        }
        editedBitmasks = masks
      }
    } catch {
      // Ignored
    } finally {
      isLoadingRPS = false
    }
  }

  async function applyRPS(iface: string) {
    try {
      isApplyingRPS[iface] = true
      const bitmask = editedBitmasks[iface] || 'f'
      const res = await api.post<{ success: boolean; error?: string }>('/api/network/rps', {
        interface: iface,
        bitmask,
      })
      if (res && res.success) {
        toastStore.success(`RPS applied to ${iface} with bitmask ${bitmask}`)
        const idx = rpsConfigs.findIndex((c) => c.interface === iface)
        if (idx !== -1) {
          rpsConfigs[idx].bitmask = bitmask
        }
      } else {
        toastStore.error(res?.error || `Failed to apply RPS for ${iface}`)
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to apply RPS')
    } finally {
      isApplyingRPS[iface] = false
    }
  }

  function setBitmaskPreset(iface: string, val: string) {
    editedBitmasks[iface] = val
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
      toastStore.success('Network settings restored to module defaults.')
      await fetchNetworkData()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to restore defaults')
    } finally {
      isLoadingTweaks = false
    }
  }

  async function applyTTL(ttl: string) {
    try {
      isApplyingTTL = true
      selectedTTL = ttl
      await api.post('/api/network/ttl', { enable: true, ttl: parseInt(ttl, 10) })
      currentTTL = ttl
      toastStore.success(`TTL set to ${ttl}. Hotspot traffic spoofed.`)
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to apply TTL')
    } finally {
      isApplyingTTL = false
    }
  }

  async function setDNS(primary: string, secondary: string) {
    try {
      isApplyingDNS = true
      await api.post('/api/network/dns', { primary, secondary })
      activeDNS1 = primary
      activeDNS2 = secondary
      toastStore.success(`DNS updated to ${primary} / ${secondary}`)
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to apply DNS')
    } finally {
      isApplyingDNS = false
    }
  }

  async function runPing() {
    if (!pingHost) return
    try {
      isPinging = true
      pingResult = 'Sending ICMP ping requests...'
      const res = await api.post<{ output?: string; latency_ms?: number }>('/api/network/ping', {
        host: pingHost,
        count: 4,
      })
      if (res && res.output) {
        pingResult = res.output
      } else if (res && res.latency_ms !== undefined) {
        pingResult = `Host: ${pingHost}\nLatency: ${res.latency_ms} ms\nStatus: Reachable`
      } else {
        pingResult = `Host: ${pingHost}\nStatus: Responded successfully.`
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
        disabled={isLoadingTweaks || isLoadingRPS}
        onclick={fetchNetworkData}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {(isLoadingTweaks || isLoadingRPS) ? 'animate-spin' : ''}" />
        <span>Refresh</span>
      </Button>
    </div>
  </div>

  <!-- Dynamic Optimizations Card -->
  <Card title="Dynamic Optimizations & Sysctl Tweaks" subtitle="Hardware and kernel tuning (saved to tweaks.json)" tone="butter">
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
          <Check class="w-3.5 h-3.5 mr-1.5" />
          <span>Save Changes</span>
        </Button>
      </div>
    {/snippet}

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 font-mono text-xs">
      <label class="flex items-center justify-between p-3 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
        <div>
          <span class="font-bold text-foreground block">Carrier Aggregation</span>
          <span class="text-[10px] text-muted">LTE-A CA Band Lock</span>
        </div>
        <input type="checkbox" bind:checked={tweaks.lte_carrier_aggregation} class="w-4 h-4 accent-accent" />
      </label>

      <label class="flex items-center justify-between p-3 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
        <div>
          <span class="font-bold text-foreground block">TCP Buffer Optimization</span>
          <span class="text-[10px] text-muted">Auto-scale tcp wmem/rmem</span>
        </div>
        <input type="checkbox" bind:checked={tweaks.tcp_buffer_optimization} class="w-4 h-4 accent-accent" />
      </label>

      <label class="flex items-center justify-between p-3 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
        <div>
          <span class="font-bold text-foreground block">BBR / BBR2 Congestion</span>
          <span class="text-[10px] text-muted">High throughput, low bufferbloat</span>
        </div>
        <input type="checkbox" bind:checked={tweaks.bbr2_congestion_control} class="w-4 h-4 accent-accent" />
      </label>

      <label class="flex items-center justify-between p-3 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
        <div>
          <span class="font-bold text-foreground block">Core Kernel Buffers</span>
          <span class="text-[10px] text-muted">rmem_max & wmem_max tuning</span>
        </div>
        <input type="checkbox" bind:checked={tweaks.sysctl_buffers_opt} class="w-4 h-4 accent-accent" />
      </label>

      <label class="flex items-center justify-between p-3 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
        <div>
          <span class="font-bold text-foreground block">Dalvik Responsiveness</span>
          <span class="text-[10px] text-muted">VM dirty ratios & latency reduction</span>
        </div>
        <input type="checkbox" bind:checked={tweaks.dalvik_responsiveness} class="w-4 h-4 accent-accent" />
      </label>

      <label class="flex items-center justify-between p-3 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
        <div>
          <span class="font-bold text-foreground block">Android Settings Tweaks</span>
          <span class="text-[10px] text-muted">Disable mobile data throttling</span>
        </div>
        <input type="checkbox" bind:checked={tweaks.settings_global_tweaks} class="w-4 h-4 accent-accent" />
      </label>

      <label class="flex items-center justify-between p-3 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
        <div>
          <span class="font-bold text-foreground block">TTL Spoofing</span>
          <span class="text-[10px] text-muted">Bypass operator tether restrictions</span>
        </div>
        <input type="checkbox" bind:checked={tweaks.ttl_spoofing} class="w-4 h-4 accent-accent" />
      </label>

      <label class="flex items-center justify-between p-3 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
        <div>
          <span class="font-bold text-foreground block">RPS Packet Steering</span>
          <span class="text-[10px] text-muted">Distribute packet processing across CPU cores</span>
        </div>
        <input type="checkbox" bind:checked={tweaks.packet_steering_rps} class="w-4 h-4 accent-accent" />
      </label>

      <label class="flex items-center justify-between p-3 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
        <div>
          <span class="font-bold text-foreground block">Interface MTU Tuning</span>
          <span class="text-[10px] text-muted">Optimal MSS & MTU sizing</span>
        </div>
        <input type="checkbox" bind:checked={tweaks.mtu_tuning} class="w-4 h-4 accent-accent" />
      </label>
    </div>
  </Card>

  <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
    <!-- TTL Modifier Card -->
    <Card title="Dynamic TTL Modifier" subtitle="Bypass operator hotspot quota restriction" tone="mint">
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
              <div class="text-[10px] text-muted">Windows PC</div>
            </button>
          </div>
        </div>

        <div class="pt-2 border-t border-border flex items-center gap-2">
          <Input placeholder="Custom TTL (e.g. 65)" bind:value={selectedTTL} />
          <Button
            variant="secondary"
            size="md"
            disabled={isApplyingTTL || !selectedTTL}
            onclick={() => applyTTL(selectedTTL)}
          >
            <Zap class="w-3.5 h-3.5 mr-1 text-amber-400" />
            <span>Apply</span>
          </Button>
        </div>
      </div>
    </Card>

    <!-- DNS Presets Card -->
    <Card title="DNS Presets & Configuration" subtitle="Override system resolver to prevent ISP DNS hijacking" tone="mint">
      <div class="space-y-4 font-mono text-xs">
        <div class="flex items-center justify-between p-3 bg-card-sub border border-border rounded">
          <span class="text-muted uppercase font-bold">Active DNS:</span>
          <span class="text-sm font-black text-accent">{activeDNS1} / {activeDNS2}</span>
        </div>

        <div>
          <span class="text-[10px] text-muted uppercase font-bold block mb-2">Popular Secure Resolvers</span>
          <div class="grid grid-cols-2 gap-2">
            {#each dnsPresets as preset}
              <button
                type="button"
                class="neo-button p-2 text-left rounded border border-border bg-card-sub hover:border-accent cursor-pointer {activeDNS1 === preset.p ? 'border-accent bg-accent/10 text-accent font-bold' : 'text-foreground'}"
                onclick={() => setDNS(preset.p, preset.s)}
              >
                <div class="font-bold">{preset.name}</div>
                <div class="text-[10px] text-muted mt-0.5">{preset.p}</div>
              </button>
            {/each}
          </div>
        </div>

        <div class="pt-2 border-t border-border space-y-2">
          <span class="text-[10px] text-muted uppercase font-bold block">Custom DNS Addresses</span>
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

  <!-- Receive Packet Steering (RPS) Card -->
  <Card title="Receive Packet Steering (RPS)" subtitle="Distribute network packet interrupts across CPU cores to maximize throughput" tone="mint">
    {#snippet action()}
      <Button
        variant="outline"
        size="sm"
        disabled={isLoadingRPS}
        onclick={fetchRPSConfigs}
      >
        <RefreshCw class="w-3 h-3 mr-1.5 {isLoadingRPS ? 'animate-spin' : ''}" />
        <span>Scan RPS</span>
      </Button>
    {/snippet}

    <div class="space-y-4 font-mono text-xs">
      <div class="flex items-start gap-3 p-3 bg-card-sub border border-border rounded text-muted">
        <Cpu class="w-4 h-4 text-accent shrink-0 mt-0.5" />
        <div class="space-y-1">
          <p class="text-foreground font-bold">Multi-Core Network Interrupt Steering</p>
          <p class="text-[11px] leading-relaxed">
            RPS routes incoming network traffic packet queues (<code class="text-accent">rx-*</code>) to specific CPU cores via hexadecimal bitmask. Distributing packet processing prevents single-core thermal saturation and boosts high-throughput cellular/Wi-Fi transfers.
          </p>
        </div>
      </div>

      {#if isLoadingRPS && rpsConfigs.length === 0}
        <div class="p-8 text-center text-muted">
          <RefreshCw class="w-5 h-5 animate-spin mx-auto mb-2 text-accent" />
          <span>Scanning network interfaces for RPS queues...</span>
        </div>
      {:else if rpsConfigs.length === 0}
        <div class="p-6 text-center text-muted bg-card-sub border border-border rounded">
          <Cpu class="w-6 h-6 mx-auto mb-2 text-muted opacity-40" />
          <p class="font-bold text-foreground">No RPS-Capable Queues Detected</p>
          <p class="text-[11px] mt-0.5">Active network interfaces on this device do not expose multi-core rps_cpus queues.</p>
        </div>
      {:else}
        <div class="space-y-3">
          {#each rpsConfigs as item}
            <div class="p-3.5 bg-card-sub border-2 border-border rounded-md space-y-3 font-mono text-xs">
              <div class="flex flex-wrap items-center justify-between gap-2 border-b border-border pb-2.5">
                <div class="flex items-center gap-2">
                  <span class="font-bold text-sm text-foreground uppercase tracking-wide">{item.interface}</span>
                  <Badge variant="default">Current Mask: {item.bitmask || '0'}</Badge>
                </div>
                <div class="text-[11px] text-muted">
                  Target Mask: <span class="font-bold text-accent">{editedBitmasks[item.interface] || item.bitmask || 'f'}</span>
                </div>
              </div>

              <div class="grid grid-cols-1 md:grid-cols-2 gap-4 items-center">
                <!-- CPU Core Bitmask Presets -->
                <div>
                  <span class="text-[10px] text-muted uppercase font-bold block mb-1.5">CPU Core Presets</span>
                  <div class="flex flex-wrap gap-1.5">
                    {#each [
                      { label: 'Quad (f)', val: 'f', desc: 'Cores 0-3' },
                      { label: 'Octa (ff)', val: 'ff', desc: 'Cores 0-7' },
                      { label: 'Dual (3)', val: '3', desc: 'Cores 0-1' },
                      { label: 'Single (1)', val: '1', desc: 'Core 0' },
                      { label: 'Disabled (0)', val: '0', desc: 'Disable RPS' }
                    ] as p}
                      <button
                        type="button"
                        class="px-2 py-1 text-[11px] font-bold rounded border transition-all cursor-pointer {editedBitmasks[item.interface] === p.val ? 'bg-accent text-accent-text border-border shadow-neobrutal-sm font-black' : 'bg-card text-muted hover:text-foreground border-border'}"
                        onclick={() => setBitmaskPreset(item.interface, p.val)}
                        title={p.desc}
                      >
                        {p.label}
                      </button>
                    {/each}
                  </div>
                </div>

                <!-- Custom Bitmask & Apply Button -->
                <div class="flex items-end gap-2">
                  <div class="flex-1">
                    <label for={`rps-${item.interface}`} class="text-[10px] text-muted uppercase font-bold block mb-1.5">
                      Hexadecimal Mask
                    </label>
                    <input
                      id={`rps-${item.interface}`}
                      type="text"
                      bind:value={editedBitmasks[item.interface]}
                      placeholder="e.g. f, ff, 3"
                      class="neo-input w-full bg-card border border-border rounded px-3 py-1.5 text-xs font-mono text-foreground focus:outline-none focus:border-accent"
                    />
                  </div>

                  <Button
                    variant="primary"
                    size="sm"
                    disabled={isApplyingRPS[item.interface] || !editedBitmasks[item.interface]}
                    onclick={() => applyRPS(item.interface)}
                  >
                    <Zap class="w-3.5 h-3.5 mr-1 text-amber-400 {isApplyingRPS[item.interface] ? 'animate-bounce' : ''}" />
                    <span>{isApplyingRPS[item.interface] ? 'Applying...' : 'Apply RPS'}</span>
                  </Button>
                </div>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </Card>

  <!-- Network Ping Diagnostic -->
  <Card title="Ping Diagnostics" subtitle="Test network latency and reachability" tone="mint">
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
