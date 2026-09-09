<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Gauge,
    RefreshCw,
    Check,
    RotateCcw,
    Gamepad2,
    PhoneCall,
    Sliders,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Input from '../../ui/Input.svelte'

  interface QoSConfig {
    enabled: boolean
    engine: string
    global_download: number
    global_upload: number
    prioritize_gaming: boolean
    prioritize_voip: boolean
  }

  let config = $state<QoSConfig>({
    enabled: false,
    engine: 'auto',
    global_download: 50,
    global_upload: 20,
    prioritize_gaming: true,
    prioritize_voip: true,
  })
  let isLoading = $state(false)
  let isSaving = $state(false)

  onMount(async () => {
    await fetchQoS()
  })

  async function fetchQoS() {
    try {
      isLoading = true
      const res = await api.get<{ config?: QoSConfig }>('/api/qos/status')
      if (res && res.config) {
        config = { ...config, ...res.config }
      }
    } catch {
      // Keep defaults if stubbed
    } finally {
      isLoading = false
    }
  }

  async function applyQoS() {
    try {
      isSaving = true
      await api.post('/api/qos/apply', config)
      toastStore.success('QoS bandwidth limits applied.')
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to apply QoS')
    } finally {
      isSaving = false
    }
  }

  async function clearQoS() {
    try {
      isSaving = true
      await api.post('/api/qos/clear')
      config.enabled = false
      toastStore.success('QoS bandwidth rules cleared.')
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to clear QoS')
    } finally {
      isSaving = false
    }
  }
</script>

<div class="space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Gauge class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        Quality of Service (QoS) & Bandwidth
      </h2>
      <Badge variant={config.enabled ? 'success' : 'default'}>
        {config.enabled ? 'QoS Active' : 'Disabled'}
      </Badge>
    </div>

    <div class="flex items-center gap-2">
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoading}
        onclick={fetchQoS}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoading ? 'animate-spin' : ''}" />
        <span>Refresh</span>
      </Button>
    </div>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <!-- Main QoS Configuration -->
    <Card
      title="Bandwidth Rate Limiting"
      subtitle="Traffic shaping & packet queuing via Linux tc / iptables"
      class="lg:col-span-2"
    >
      {#snippet action()}
        <div class="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={isSaving}
            onclick={clearQoS}
          >
            <RotateCcw class="w-3.5 h-3.5 mr-1 text-red-400" />
            <span>Clear Rules</span>
          </Button>
          <Button
            variant="primary"
            size="sm"
            disabled={isSaving}
            onclick={applyQoS}
          >
            <Check class="w-3.5 h-3.5 mr-1" />
            <span>Apply QoS</span>
          </Button>
        </div>
      {/snippet}

      <div class="space-y-5 font-mono text-xs">
        <!-- Master Enable Toggle -->
        <label class="flex items-center justify-between p-3 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
          <div>
            <span class="font-bold text-foreground block">Global Traffic Shaper</span>
            <span class="text-[10px] text-muted">Enforce speed limits on SoftAP & Tethering</span>
          </div>
          <input
            type="checkbox"
            bind:checked={config.enabled}
            class="w-4 h-4 accent-accent cursor-pointer"
          />
        </label>

        <!-- Bandwidth Sliders -->
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-2">
          <div class="bg-card-sub border border-border p-3 rounded space-y-2">
            <div class="flex items-center justify-between">
              <span class="text-muted uppercase font-bold">Download Limit</span>
              <span class="text-sm font-black text-accent">{config.global_download} Mbps</span>
            </div>
            <input
              type="range"
              min="1"
              max="200"
              step="1"
              bind:value={config.global_download}
              class="w-full accent-accent cursor-pointer"
            />
          </div>

          <div class="bg-card-sub border border-border p-3 rounded space-y-2">
            <div class="flex items-center justify-between">
              <span class="text-muted uppercase font-bold">Upload Limit</span>
              <span class="text-sm font-black text-purple-400">{config.global_upload} Mbps</span>
            </div>
            <input
              type="range"
              min="1"
              max="100"
              step="1"
              bind:value={config.global_upload}
              class="w-full accent-purple-500 cursor-pointer"
            />
          </div>
        </div>

        <!-- Priority Packets -->
        <div class="space-y-2 pt-2 border-t border-border">
          <span class="text-[10px] text-muted uppercase font-bold block">
            Low Latency Priority Classification (DSCP / TOS)
          </span>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
            <label class="flex items-center justify-between p-2.5 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
              <div class="flex items-center gap-2">
                <Gamepad2 class="w-4 h-4 text-emerald-400" />
                <span>Prioritize Gaming Packets</span>
              </div>
              <input
                type="checkbox"
                bind:checked={config.prioritize_gaming}
                class="w-4 h-4 accent-accent"
              />
            </label>

            <label class="flex items-center justify-between p-2.5 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
              <div class="flex items-center gap-2">
                <PhoneCall class="w-4 h-4 text-blue-400" />
                <span>Prioritize VoIP / Discord</span>
              </div>
              <input
                type="checkbox"
                bind:checked={config.prioritize_voip}
                class="w-4 h-4 accent-accent"
              />
            </label>
          </div>
        </div>
      </div>
    </Card>

    <!-- Engine Info -->
    <Card title="QoS Engine & Queue" subtitle="Underlying Linux kernel subsystem">
      <div class="space-y-4 font-mono text-xs">
        <div class="p-3 bg-card-sub border border-border rounded space-y-1">
          <span class="text-[10px] text-muted uppercase font-bold">QoS Engine</span>
          <p class="font-bold text-foreground">HTB + fq_codel (Bufferbloat Free)</p>
        </div>

        <p class="text-muted leading-relaxed">
          Traffic shaping limits bufferbloat during simultaneous downloads, keeping ping and game latency stable under load.
        </p>
      </div>
    </Card>
  </div>
</div>
