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
    Upload,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { LL } from '../../../i18n/i18n-svelte'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Input from '../../ui/Input.svelte'

  interface TunnelConfig {
    engine: 'cloudflare' | 'frp' | 'zerotier'
    token?: string
    cloudflare_token?: string
    zerotier_network_id?: string
    server?: string
    local_port?: number
  }

  interface TunnelStatus {
    active?: boolean
    running?: boolean
    engine?: string
    pid?: number
    public_url?: string
    logs?: string[]
  }

  let config = $state<TunnelConfig>({
    engine: 'cloudflare',
    token: '',
    local_port: 80,
  })
  let status = $state<TunnelStatus>({ running: false, active: false })
  let isLoading = $state(false)
  let isToggling = $state(false)

  onMount(async () => {
    await fetchTunnel()
  })

  async function fetchTunnel() {
    try {
      isLoading = true
      const res = await api.get<{ config?: TunnelConfig & { cloudflare_token?: string; zerotier_network_id?: string }; status?: TunnelStatus }>('/api/tunnel/status')
      if (res?.config) {
        config = { ...config, ...res.config }
        if (res.config.cloudflare_token && !config.token) {
          config.token = res.config.cloudflare_token
        }
        if (res.config.zerotier_network_id && config.engine === 'zerotier' && !config.token) {
          config.token = res.config.zerotier_network_id
        }
      }
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
      const payload: Record<string, unknown> = {
        ...config,
        cloudflare_token: config.engine === 'cloudflare' ? (config.cloudflare_token || config.token) : undefined,
        zerotier_network_id: config.engine === 'zerotier' ? config.token : undefined,
      }
      await api.post('/api/tunnel/start', payload)
      status.active = true
      status.running = true
      toastStore.success($LL.tunnel.toasts.started({ engine: config.engine }))
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.tunnel.toasts.startFailed())
    } finally {
      isToggling = false
    }
  }

  async function stopTunnel() {
    try {
      isToggling = true
      await api.post('/api/tunnel/stop')
      status.active = false
      status.running = false
      toastStore.success($LL.tunnel.toasts.stopped())
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.tunnel.toasts.stopFailed())
    } finally {
      isToggling = false
    }
  }

  // Binary Upload State
  let uploadEngine = $state<'cloudflare' | 'zerotier'>('cloudflare')
  let selectedBinaryFile = $state<File | null>(null)
  let isUploadingBinary = $state(false)
  let binaryFileInputEl: HTMLInputElement | null = $state(null)

  function handleFileSelect(e: Event) {
    const input = e.target as HTMLInputElement
    if (input.files && input.files[0]) {
      selectedBinaryFile = input.files[0]
    }
  }

  async function uploadBinary() {
    if (!selectedBinaryFile) {
      toastStore.warning($LL.tunnel.toasts.selectBinaryFirst())
      return
    }
    try {
      isUploadingBinary = true
      const formData = new FormData()
      formData.append('engine', uploadEngine)
      formData.append('binary', selectedBinaryFile)

      const res = await api.post<{ success: boolean; path?: string; error?: string }>(
        '/api/tunnel/upload',
        formData
      )

      if (res && res.success) {
        toastStore.success($LL.tunnel.toasts.binaryInstalled({ engine: uploadEngine, path: res.path || 'system' }))
        selectedBinaryFile = null
        if (binaryFileInputEl) binaryFileInputEl.value = ''
      } else {
        toastStore.error(res?.error || $LL.tunnel.toasts.uploadFailed())
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.tunnel.toasts.uploadFailed())
    } finally {
      isUploadingBinary = false
    }
  }
</script>

<div class="space-y-3 sm:space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Globe class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        {$LL.tunnel.title()}
      </h2>
      <Badge variant={(status.active || status.running) ? 'success' : 'default'}>
        {(status.active || status.running) ? $LL.tunnel.badgeActive({ engine: config.engine.toUpperCase() }) : $LL.tunnel.badgeInactive()}
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
        <span>{$LL.common.refresh()}</span>
      </Button>

      {#if (status.active || status.running)}
        <Button
          variant="danger"
          size="sm"
          disabled={isToggling}
          onclick={stopTunnel}
        >
          <Square class="w-3.5 h-3.5 mr-1" />
          <span>{$LL.tunnel.stopTunnel()}</span>
        </Button>
      {:else}
        <Button
          variant="primary"
          size="sm"
          disabled={isToggling}
          onclick={startTunnel}
        >
          <Play class="w-3.5 h-3.5 mr-1" />
          <span>{$LL.tunnel.startTunnel()}</span>
        </Button>
      {/if}
    </div>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-3 sm:gap-6">
    <!-- Tunnel Configuration -->
    <Card
      title={$LL.tunnel.config.title()}
      subtitle={$LL.tunnel.config.subtitle()}
      tone="ice"
      class="lg:col-span-2"
    >
      <div class="space-y-5 font-mono text-xs">
        <!-- Engine Selector -->
        <div>
          <span class="text-[10px] text-muted uppercase font-bold block mb-2">
            {$LL.tunnel.config.engineLabel()}
          </span>
          <div class="grid grid-cols-3 gap-3">
            {#each ['cloudflare', 'frp', 'zerotier'] as eng}
              <button
                type="button"
                class="neo-button p-3 rounded border border-border bg-card-sub hover:border-accent text-center cursor-pointer transition-all {config.engine === eng ? 'border-accent bg-accent/15 text-accent font-bold shadow-neobrutal-sm' : 'text-foreground'}"
                onclick={() => (config.engine = eng as TunnelConfig['engine'])}
                disabled={Boolean(status.active || status.running)}
              >
                <div class="font-black text-sm uppercase">{eng}</div>
                <div class="text-[10px] text-muted mt-0.5">
                  {eng === 'cloudflare' ? $LL.tunnel.config.engineDesc.cloudflare() : eng === 'frp' ? $LL.tunnel.config.engineDesc.frp() : $LL.tunnel.config.engineDesc.zerotier()}
                </div>
              </button>
            {/each}
          </div>
        </div>

        <!-- Token / Credential Input -->
        {#if config.engine === 'cloudflare'}
          <Input
            type="password"
            label={$LL.tunnel.config.cloudflareTokenLabel()}
            placeholder="eyJhIjoi..."
            bind:value={config.token}
            disabled={Boolean(status.active || status.running)}
          />
        {:else if config.engine === 'frp'}
          <div class="space-y-3">
            <Input
              label={$LL.tunnel.config.frpServerLabel()}
              placeholder="vps.example.com:7000"
              bind:value={config.server}
              disabled={Boolean(status.active || status.running)}
            />
            <Input
              type="password"
              label={$LL.tunnel.config.frpTokenLabel()}
              placeholder="secret_token"
              bind:value={config.token}
              disabled={Boolean(status.active || status.running)}
            />
          </div>
        {:else if config.engine === 'zerotier'}
          <Input
            label={$LL.tunnel.config.zerotierNetworkLabel()}
            placeholder="8056c2e21c000001"
            bind:value={config.token}
            disabled={Boolean(status.active || status.running)}
          />
        {/if}
      </div>
    </Card>

    <!-- Public URL & Connectivity -->
    <Card title={$LL.tunnel.endpoint.title()} subtitle={$LL.tunnel.endpoint.subtitle()} tone="ice">
      <div class="space-y-3 font-mono text-xs">
        {#if (status.active || status.running) && status.public_url}
          <div class="p-3 bg-card-sub border border-border rounded space-y-1">
            <span class="text-[10px] text-muted uppercase font-bold">{$LL.tunnel.endpoint.publicDomain()}</span>
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
        {:else if (status.active || status.running)}
          <div class="p-3 bg-card-sub border border-border rounded text-emerald-400 font-bold">
            {$LL.tunnel.endpoint.daemonConnected()}
          </div>
        {:else}
          <p class="text-muted text-center py-4">
            {$LL.tunnel.endpoint.stopped()}
          </p>
        {/if}
      </div>
    </Card>
  </div>

  <!-- Engine Binary Management Card -->
  <Card
    title={$LL.tunnel.binary.title()}
    subtitle={$LL.tunnel.binary.subtitle()}
    tone="ice"
  >
    <div class="space-y-4 font-mono text-xs">
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <!-- Engine selection -->
        <div class="space-y-2">
          <span class="text-[10px] text-muted uppercase font-bold block">{$LL.tunnel.binary.targetEngine()}</span>
          <div class="grid grid-cols-2 gap-2">
            <button
              type="button"
              class="neo-button p-2.5 rounded border transition-all text-center cursor-pointer {uploadEngine === 'cloudflare' ? 'border-accent bg-accent/15 text-accent font-bold shadow-neobrutal-sm' : 'border-border bg-card-sub text-muted hover:text-foreground'}"
              onclick={() => (uploadEngine = 'cloudflare')}
            >
              <div class="font-bold text-xs">Cloudflare</div>
              <div class="text-[10px] text-muted">cloudflared</div>
            </button>

            <button
              type="button"
              class="neo-button p-2.5 rounded border transition-all text-center cursor-pointer {uploadEngine === 'zerotier' ? 'border-accent bg-accent/15 text-accent font-bold shadow-neobrutal-sm' : 'border-border bg-card-sub text-muted hover:text-foreground'}"
              onclick={() => (uploadEngine = 'zerotier')}
            >
              <div class="font-bold text-xs">ZeroTier</div>
              <div class="text-[10px] text-muted">zerotier-one</div>
            </button>
          </div>
        </div>

        <!-- File selection & Upload -->
        <div class="space-y-2">
          <span class="text-[10px] text-muted uppercase font-bold block">{$LL.tunnel.binary.selectLabel()}</span>
          <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-2">
            <label class="flex-1 neo-button bg-card-sub border border-dashed border-border hover:border-accent p-2.5 rounded cursor-pointer text-center truncate select-none">
              <span class="text-xs text-foreground truncate">
                {selectedBinaryFile ? selectedBinaryFile.name : $LL.tunnel.binary.chooseFile()}
              </span>
              <input
                type="file"
                bind:this={binaryFileInputEl}
                onchange={handleFileSelect}
                class="hidden"
              />
            </label>

            <Button
              variant="primary"
              size="md"
              disabled={isUploadingBinary || !selectedBinaryFile}
              onclick={uploadBinary}
            >
              <Upload class="w-3.5 h-3.5 mr-1 {isUploadingBinary ? 'animate-bounce' : ''}" />
              <span>{isUploadingBinary ? $LL.common.uploading() : $LL.tunnel.binary.upload()}</span>
            </Button>
          </div>
        </div>
      </div>

      <div class="p-3 bg-card-sub border border-border rounded flex flex-col sm:flex-row sm:items-center justify-between gap-2 text-muted text-[11px]">
        <span>{$LL.tunnel.binary.savedNoteA()}<code class="text-accent font-bold">{$LL.tunnel.binary.savedNoteB()}</code>{$LL.tunnel.binary.savedNoteC()}</span>
        <Badge variant="default">ARM64 / ARMv7</Badge>
      </div>
    </div>
  </Card>
</div>
