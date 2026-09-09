<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Globe,
    RefreshCw,
    Play,
    Square,
    ExternalLink,
    Shield,
    Check,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Input from '../../ui/Input.svelte'

  interface TunnelConfig {
    engine: 'cloudflare' | 'frp' | 'zerotier'
    token?: string
    server?: string
    local_port?: number
  }

  interface TunnelStatus {
    running: boolean
    engine?: string
    public_url?: string
    status_detail?: string
  }

  let config = $state<TunnelConfig>({
    engine: 'cloudflare',
    token: '',
    local_port: 80,
  })
  let status = $state<TunnelStatus>({ running: false })
  let isLoading = $state(false)
  let isToggling = $state(false)

  onMount(async () => {
    await fetchTunnel()
  })

  async function fetchTunnel() {
    try {
      isLoading = true
      const res = await api.get<{ config?: TunnelConfig; status?: TunnelStatus }>('/api/tunnel/status')
      if (res?.config) config = { ...config, ...res.config }
      if (res?.status) status = { ...status, ...res.status }
    } catch {
      // Ignored
    } finally {
      isLoading = false
    }
  }

  async function startTunnel() {
    try {
      isToggling = true
      await api.post('/api/tunnel/start', config)
      status.running = true
      toastStore.success(`Tunnel (${config.engine}) started successfully.`)
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to start tunnel')
    } finally {
      isToggling = false
    }
  }

  async function stopTunnel() {
    try {
      isToggling = true
      await api.post('/api/tunnel/stop')
      status.running = false
      toastStore.success('Tunnel stopped.')
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to stop tunnel')
    } finally {
      isToggling = false
    }
  }
</script>

<div class="space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Globe class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        Remote Access Tunnels
      </h2>
      <Badge variant={status.running ? 'success' : 'default'}>
        {status.running ? `${config.engine.toUpperCase()} Active` : 'Inactive'}
      </Badge>
    </div>

    <div class="flex items-center gap-2">
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoading}
        onclick={fetchTunnel}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoading ? 'animate-spin' : ''}" />
        <span>Refresh</span>
      </Button>

      {#if status.running}
        <Button
          variant="danger"
          size="sm"
          disabled={isToggling}
          onclick={stopTunnel}
        >
          <Square class="w-3.5 h-3.5 mr-1" />
          <span>Stop Tunnel</span>
        </Button>
      {:else}
        <Button
          variant="primary"
          size="sm"
          disabled={isToggling}
          onclick={startTunnel}
        >
          <Play class="w-3.5 h-3.5 mr-1" />
          <span>Start Tunnel</span>
        </Button>
      {/if}
    </div>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <!-- Tunnel Configuration -->
    <Card
      title="Tunnel Configuration"
      subtitle="Expose WebUI remotely without port forwarding"
      class="lg:col-span-2"
    >
      <div class="space-y-5 font-mono text-xs">
        <!-- Engine Selector -->
        <div>
          <span class="text-[10px] text-muted uppercase font-bold block mb-2">
            Tunnel Provider Engine
          </span>
          <div class="grid grid-cols-3 gap-3">
            {#each ['cloudflare', 'frp', 'zerotier'] as eng}
              <button
                type="button"
                class="neo-button p-3 rounded border border-border bg-card-sub hover:border-accent text-center cursor-pointer transition-all {config.engine === eng ? 'border-accent bg-accent/15 text-accent font-bold shadow-neobrutal-sm' : 'text-foreground'}"
                onclick={() => (config.engine = eng as TunnelConfig['engine'])}
                disabled={status.running}
              >
                <div class="font-black text-sm uppercase">{eng}</div>
                <div class="text-[10px] text-muted mt-0.5">
                  {eng === 'cloudflare' ? 'Cloudflare Tunnel' : eng === 'frp' ? 'Fast Reverse Proxy' : 'P2P Virtual LAN'}
                </div>
              </button>
            {/each}
          </div>
        </div>

        <!-- Token / Credential Input -->
        {#if config.engine === 'cloudflare'}
          <Input
            type="password"
            label="Cloudflare Tunnel Token (cloudflared tunnel run --token ...)"
            placeholder="eyJhIjoi..."
            bind:value={config.token}
            disabled={status.running}
          />
        {:else if config.engine === 'frp'}
          <div class="space-y-3">
            <Input
              label="FRP Server Address"
              placeholder="vps.example.com:7000"
              bind:value={config.server}
              disabled={status.running}
            />
            <Input
              type="password"
              label="FRP Auth Token"
              placeholder="secret_token"
              bind:value={config.token}
              disabled={status.running}
            />
          </div>
        {:else if config.engine === 'zerotier'}
          <Input
            label="ZeroTier Network ID (16 Hex Digits)"
            placeholder="8056c2e21c000001"
            bind:value={config.token}
            disabled={status.running}
          />
        {/if}
      </div>
    </Card>

    <!-- Public URL & Connectivity -->
    <Card title="Public Remote Endpoint" subtitle="Worldwide access URL">
      <div class="space-y-3 font-mono text-xs">
        {#if status.running && status.public_url}
          <div class="p-3 bg-card-sub border border-border rounded space-y-1">
            <span class="text-[10px] text-muted uppercase font-bold">Public Domain:</span>
            <a
              href={status.public_url}
              target="_blank"
              rel="noreferrer"
              class="font-bold text-accent break-all hover:underline flex items-center gap-1"
            >
              <span>{status.public_url}</span>
              <ExternalLink class="w-3.5 h-3.5 shrink-0" />
            </a>
          </div>
        {:else if status.running}
          <div class="p-3 bg-card-sub border border-border rounded text-emerald-400 font-bold">
            Tunnel daemon is connected.
          </div>
        {:else}
          <p class="text-muted text-center py-4">
            Tunnel is stopped. Configure your provider credentials and click Start.
          </p>
        {/if}
      </div>
    </Card>
  </div>
</div>
