<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Route,
    Wifi,
    Cpu,
    Shield,
    Check,
    RotateCcw,
    Layers,
  } from '@lucide/svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Input from '../../ui/Input.svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'

  interface BoxSettings {
    bin_name: string
    network_mode: string
    ipv6: boolean
    tproxy_port: string
    redir_port: string
    cgroup_memcg: boolean
    memcg_limit: string
    quic: string
  }

  interface HotspotConfig {
    proxy_hotspot: boolean
    ap_list: string[]
  }

  let settings = $state<BoxSettings>({
    bin_name: 'mihomo',
    network_mode: 'tproxy',
    ipv6: false,
    tproxy_port: '9898',
    redir_port: '9797',
    cgroup_memcg: false,
    memcg_limit: '150M',
    quic: 'enable',
  })

  let hotspot = $state<HotspotConfig>({
    proxy_hotspot: true,
    ap_list: ['ap0', 'wlan1', 'rndis0'],
  })

  let isLoading = $state(false)
  let isSaving = $state(false)

  const networkModes = [
    { id: 'tproxy', label: 'TPROXY', desc: 'TCP + UDP transparent' },
    { id: 'redirect', label: 'REDIRECT', desc: 'TCP redir, UDP direct' },
    { id: 'tun', label: 'TUN Mode', desc: 'Virtual meta/tun0 device' },
    { id: 'mixed', label: 'MIXED', desc: 'TCP redir + UDP tun' },
  ]

  onMount(async () => {
    await fetchSettings()
  })

  async function fetchSettings() {
    try {
      isLoading = true
      const [resSettings, resHotspot] = await Promise.allSettled([
        api.get<{ settings: BoxSettings }>('/api/proxy/settings'),
        api.get<{ config: HotspotConfig }>('/api/proxy/hotspot'),
      ])

      if (resSettings.status === 'fulfilled' && resSettings.value?.settings) {
        settings = { ...settings, ...resSettings.value.settings }
      }
      if (resHotspot.status === 'fulfilled' && resHotspot.value?.config) {
        hotspot = { ...hotspot, ...resHotspot.value.config }
      }
    } catch {
      // Ignored
    } finally {
      isLoading = false
    }
  }

  async function saveAllSettings() {
    try {
      isSaving = true
      await Promise.all([
        api.post('/api/proxy/settings', settings),
        api.post('/api/proxy/hotspot', hotspot),
      ])
      toastStore.success('Box for Magisk routing configuration saved successfully!')
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to save configuration')
    } finally {
      isSaving = false
    }
  }
</script>

<Card
  title="Transparent Proxy & Tethering Routing"
  subtitle="Kernel iptables redirection modes, IPv6 interception, Hotspot tethering bypass & Cgroup RAM control"
  tone="peach"
>
  {#snippet action()}
    <Button
      variant="primary"
      size="sm"
      disabled={isSaving || isLoading}
      onclick={saveAllSettings}
    >
      <Check class="w-3.5 h-3.5 mr-1" />
      <span>{isSaving ? 'Saving...' : 'Save Settings'}</span>
    </Button>
  {/snippet}

  <div class="space-y-4 font-mono text-xs">
    <!-- Network Mode Selector -->
    <div>
      <span class="text-[10px] text-muted uppercase font-bold block mb-2">BFM Network Mode</span>
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
        {#each networkModes as m}
          <button
            type="button"
            onclick={() => (settings.network_mode = m.id)}
            class="neo-button p-2 text-center rounded border-2 transition-all cursor-pointer {settings.network_mode === m.id ? 'border-accent bg-accent/15 text-accent font-bold shadow-neobrutal-sm' : 'border-border bg-card-sub text-muted hover:text-foreground'}"
          >
            <div class="font-bold text-xs">{m.label}</div>
            <div class="text-[10px] opacity-75">{m.desc}</div>
          </button>
        {/each}
      </div>
    </div>

    <!-- Toggles Grid: IPv6, Hotspot, Cgroup, QUIC -->
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-2 border-t border-border">
      <!-- Hotspot / Tethering Proxy Toggle -->
      <label class="p-3 bg-card-sub border border-border rounded-lg flex items-center justify-between cursor-pointer hover:border-accent transition-colors">
        <div class="space-y-0.5">
          <div class="flex items-center gap-1.5 font-bold text-foreground">
            <Wifi class="w-4 h-4 text-accent" />
            <span>Proxy Hotspot / Tethering</span>
          </div>
          <p class="text-[10px] text-muted">Route connected Wi-Fi / USB hotspot clients through proxy.</p>
        </div>
        <input
          type="checkbox"
          bind:checked={hotspot.proxy_hotspot}
          class="w-4 h-4 accent-accent cursor-pointer"
        />
      </label>

      <!-- IPv6 Proxy Toggle -->
      <label class="p-3 bg-card-sub border border-border rounded-lg flex items-center justify-between cursor-pointer hover:border-accent transition-colors">
        <div class="space-y-0.5">
          <div class="flex items-center gap-1.5 font-bold text-foreground">
            <Layers class="w-4 h-4 text-accent" />
            <span>IPv6 Transparent Proxy</span>
          </div>
          <p class="text-[10px] text-muted">Intercept and redirect IPv6 traffic to proxy core (ip6tables).</p>
        </div>
        <input
          type="checkbox"
          bind:checked={settings.ipv6}
          class="w-4 h-4 accent-accent cursor-pointer"
        />
      </label>

      <!-- Cgroup Memory Limit Toggle -->
      <div class="p-3 bg-card-sub border border-border rounded-lg space-y-2">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-1.5 font-bold text-foreground">
            <Cpu class="w-4 h-4 text-accent" />
            <span>Cgroup RAM Limiter</span>
          </div>
          <input
            type="checkbox"
            bind:checked={settings.cgroup_memcg}
            class="w-4 h-4 accent-accent cursor-pointer"
          />
        </div>
        {#if settings.cgroup_memcg}
          <div class="flex items-center gap-2 pt-1">
            <Input placeholder="150M" bind:value={settings.memcg_limit} />
            <span class="text-[10px] text-muted shrink-0">Max Core RAM</span>
          </div>
        {:else}
          <p class="text-[10px] text-muted">Constrain core memory consumption to prevent kernel OOM freezes.</p>
        {/if}
      </div>

      <!-- QUIC UDP Block Toggle -->
      <label class="p-3 bg-card-sub border border-border rounded-lg flex items-center justify-between cursor-pointer hover:border-accent transition-colors">
        <div class="space-y-0.5">
          <div class="flex items-center gap-1.5 font-bold text-foreground">
            <Shield class="w-4 h-4 text-accent" />
            <span>Block QUIC (UDP 443)</span>
          </div>
          <p class="text-[10px] text-muted">Force browsers to fallback to TCP for robust proxy tunneling.</p>
        </div>
        <input
          type="checkbox"
          checked={settings.quic === 'disable'}
          onchange={(e) => (settings.quic = e.currentTarget.checked ? 'disable' : 'enable')}
          class="w-4 h-4 accent-accent cursor-pointer"
        />
      </label>
    </div>
  </div>
</Card>
