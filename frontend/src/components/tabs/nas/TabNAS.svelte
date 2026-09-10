<script lang="ts">
  import { onMount } from 'svelte'
  import {
    HardDrive,
    RefreshCw,
    Play,
    Square,
    Folder,
    Link,
    Check,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Input from '../../ui/Input.svelte'

  interface NASConfig {
    enabled?: boolean
    port: number
    share_path: string
    read_only?: boolean
  }

  interface NASStatus {
    active?: boolean
    running?: boolean
    port?: number
    share_path?: string
    url?: string
  }

  let config = $state<NASConfig>({
    port: 8088,
    share_path: '/sdcard',
    read_only: false,
  })
  let status = $state<NASStatus>({ running: false, active: false })
  let isLoading = $state(false)
  let isToggling = $state(false)

  onMount(async () => {
    await fetchNAS()
  })

  async function fetchNAS() {
    try {
      isLoading = true
      const res = await api.get<{ config?: NASConfig; status?: NASStatus }>('/api/nas/status')
      if (res?.config) config = { ...config, ...res.config }
      if (res?.status) status = { ...status, ...res.status }
    } catch {
      // Keep defaults
    } finally {
      isLoading = false
    }
  }

  async function startNAS() {
    try {
      isToggling = true
      await api.post('/api/nas/start', config)
      status.active = true
      status.running = true
      toastStore.success(`NAS HTTP File Server started on port ${config.port}`)
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to start NAS')
    } finally {
      isToggling = false
    }
  }

  async function stopNAS() {
    try {
      isToggling = true
      await api.post('/api/nas/stop')
      status.active = false
      status.running = false
      toastStore.success('NAS File Server stopped.')
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to stop NAS')
    } finally {
      isToggling = false
    }
  }
</script>

<div class="space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <HardDrive class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        NAS & Local File Sharing
      </h2>
      <Badge variant={(status.active || status.running) ? 'success' : 'default'}>
        {(status.active || status.running) ? 'Server Active' : 'Stopped'}
      </Badge>
    </div>

    <div class="flex items-center gap-2">
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoading}
        onclick={fetchNAS}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoading ? 'animate-spin' : ''}" />
        <span>Refresh</span>
      </Button>

      {#if (status.active || status.running)}
        <Button
          variant="danger"
          size="sm"
          disabled={isToggling}
          onclick={stopNAS}
        >
          <Square class="w-3.5 h-3.5 mr-1" />
          <span>Stop NAS</span>
        </Button>
      {:else}
        <Button
          variant="primary"
          size="sm"
          disabled={isToggling}
          onclick={startNAS}
        >
          <Play class="w-3.5 h-3.5 mr-1" />
          <span>Start NAS</span>
        </Button>
      {/if}
    </div>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <!-- Server Config Card -->
    <Card
      title="File Server Configuration"
      subtitle="Expose storage to devices on Wi-Fi SoftAP & LAN"
      tone="ice"
      class="lg:col-span-2"
    >
      <div class="space-y-4 font-mono text-xs">
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <Input
            label="Share Path"
            placeholder="/sdcard"
            bind:value={config.share_path}
            disabled={Boolean(status.active || status.running)}
          />
          <Input
            type="number"
            label="HTTP Server Port"
            placeholder="8088"
            value={String(config.port)}
            oninput={(e) => {
              const v = parseInt((e.target as HTMLInputElement).value, 10)
              if (!isNaN(v)) config.port = v
            }}
            disabled={Boolean(status.active || status.running)}
          />
        </div>

        <label class="flex items-center justify-between p-3 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
          <div>
            <span class="font-bold text-foreground block">Read-Only Mode</span>
            <span class="text-[10px] text-muted">Prevent clients from deleting or modifying files</span>
          </div>
          <input
            type="checkbox"
            bind:checked={config.read_only}
            disabled={Boolean(status.active || status.running)}
            class="w-4 h-4 accent-accent"
          />
        </label>
      </div>
    </Card>

    <!-- Network Access Info -->
    <Card title="Direct Access Link" subtitle="LAN HTTP URL" tone="ice">
      <div class="space-y-3 font-mono text-xs">
        {#if (status.active || status.running)}
          <div class="p-3 bg-card-sub border border-border rounded space-y-1">
            <span class="text-[10px] text-muted uppercase font-bold">LAN Web Share:</span>
            <p class="font-bold text-accent break-all">
              http://192.168.43.1:{config.port}
            </p>
          </div>
          <p class="text-muted text-[11px]">
            Any device connected to this phone's Wi-Fi hotspot can download files directly using any web browser.
          </p>
        {:else}
          <p class="text-muted text-center py-4">
            Server is stopped. Start the service to share files on your Wi-Fi network.
          </p>
        {/if}
      </div>
    </Card>
  </div>
</div>
