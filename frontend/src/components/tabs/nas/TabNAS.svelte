<script module lang="ts">
  export interface NASConfig {
    enabled?: boolean
    share_path: string
    port: number
    read_only?: boolean
    auth_required?: boolean
    username?: string
    password?: string
    protocol?: 'webdav' | 'http' | string
  }

  export interface NASStatus {
    active?: boolean
    running?: boolean
    port?: number
    share_path?: string
    url?: string
    protocol?: string
    storage_used?: string
    storage_total?: string
    ips?: string[]
    primary_ip?: string
    mount_urls?: Record<string, string> | string[]
    auth_required?: boolean
    read_only?: boolean
  }
</script>

<script lang="ts">
  import { onMount } from 'svelte'
  import {
    HardDrive,
    Server,
    Share2,
    Folder,
    Link,
    ExternalLink,
    Copy,
    Check,
    RefreshCw,
    Play,
    Square,
    Lock,
    Unlock,
    ShieldCheck,
    AlertCircle,
    Eye,
    EyeOff,
    Terminal,
    Monitor,
    Smartphone,
    Laptop,
    Globe,
    Wifi,
    Database,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Input from '../../ui/Input.svelte'

  let config = $state<NASConfig>({
    enabled: false,
    share_path: '/sdcard',
    port: 85,
    read_only: true,
    auth_required: true,
    username: 'admin',
    password: 'bfr',
    protocol: 'webdav',
  })

  let status = $state<NASStatus>({
    active: false,
    running: false,
    port: 85,
    share_path: '/sdcard',
    url: '',
    protocol: 'webdav',
    storage_used: '',
    storage_total: '',
    ips: [],
    primary_ip: '127.0.0.1',
    mount_urls: [],
    auth_required: false,
    read_only: false,
  })

  let isLoading = $state(false)
  let isToggling = $state(false)
  let isSaving = $state(false)
  let showPassword = $state(false)
  let copiedUrl = $state<string | null>(null)
  let activeOsTab = $state<'windows' | 'android' | 'macos' | 'linux'>('windows')

  let isServerRunning = $derived(Boolean(status.active || status.running))

  let storagePercent = $derived(() => {
    if (!status.storage_used || !status.storage_total) return 0
    const parseSize = (str: string): number => {
      const num = parseFloat(str.replace(/[^\d.]/g, ''))
      if (isNaN(num)) return 0
      const lower = str.toLowerCase()
      if (lower.includes('tb')) return num * 1024 * 1024
      if (lower.includes('gb')) return num * 1024
      if (lower.includes('mb')) return num
      if (lower.includes('kb')) return num / 1024
      return num
    }
    const used = parseSize(status.storage_used)
    const total = parseSize(status.storage_total)
    if (total <= 0) return 0
    return Math.min(100, Math.round((used / total) * 100))
  })

  let networkEndpoints = $derived(() => {
    const port = status.port || config.port || 85
    const list: { label: string; url: string; ip: string; isPrimary: boolean }[] = []
    const seenIps = new Set<string>()

    if (status.primary_ip && status.primary_ip !== '127.0.0.1') {
      seenIps.add(status.primary_ip)
      const isHotspot =
        status.primary_ip.startsWith('192.168.43.') ||
        status.primary_ip.startsWith('192.168.50.')
      list.push({
        label: isHotspot ? 'Wi-Fi SoftAP / Hotspot' : 'Primary Network / LAN',
        url: `http://${status.primary_ip}:${port}/`,
        ip: status.primary_ip,
        isPrimary: true,
      })
    }

    if (status.ips && Array.isArray(status.ips)) {
      for (const ip of status.ips) {
        if (ip && !seenIps.has(ip) && ip !== '127.0.0.1') {
          seenIps.add(ip)
          const isHotspot =
            ip.startsWith('192.168.43.') || ip.startsWith('192.168.50.')
          list.push({
            label: isHotspot ? 'Wi-Fi Hotspot' : 'Network Interface',
            url: `http://${ip}:${port}/`,
            ip,
            isPrimary: false,
          })
        }
      }
    }

    if (list.length === 0) {
      const fallbackIp = status.primary_ip || '127.0.0.1'
      list.push({
        label: 'Localhost Loopback',
        url: `http://${fallbackIp}:${port}/`,
        ip: fallbackIp,
        isPrimary: true,
      })
    }

    return list
  })

  let activeTargetIp = $derived(() => {
    const endpoints = networkEndpoints()
    return endpoints.length > 0 ? endpoints[0].ip : '192.168.1.100'
  })

  let activeTargetPort = $derived(status.port || config.port || 85)

  onMount(async () => {
    await fetchNAS()
  })

  async function fetchNAS() {
    try {
      isLoading = true
      const res = await api.get<{ config?: NASConfig; status?: NASStatus }>(
        '/api/nas/status'
      )
      if (res?.config) {
        config = {
          enabled: res.config.enabled ?? false,
          share_path: res.config.share_path || '/sdcard',
          port: res.config.port || 85,
          read_only: res.config.read_only ?? true,
          auth_required: res.config.auth_required ?? true,
          username: res.config.username || '',
          password: res.config.password || '',
          protocol: res.config.protocol || 'webdav',
        }
      }
      if (res?.status) {
        status = {
          active: res.status.active ?? res.status.running ?? false,
          running: res.status.running ?? res.status.active ?? false,
          port: res.status.port || config.port,
          share_path: res.status.share_path || config.share_path,
          url: res.status.url || '',
          protocol: res.status.protocol || config.protocol,
          storage_used: res.status.storage_used || '',
          storage_total: res.status.storage_total || '',
          ips: res.status.ips || [],
          primary_ip: res.status.primary_ip || '127.0.0.1',
          mount_urls: res.status.mount_urls || [],
          auth_required: res.status.auth_required ?? config.auth_required,
          read_only: res.status.read_only ?? config.read_only,
        }
      }
    } catch {
      // Keep defaults
    } finally {
      isLoading = false
    }
  }

  async function startNAS() {
    if (config.auth_required && (!config.username?.trim() || !config.password)) {
      toastStore.error('Set a NAS username and password before starting the authenticated share.')
      return
    }
    try {
      isToggling = true
      const payload: NASConfig = {
        ...config,
        enabled: true,
        port: Number(config.port) || 85,
        share_path: config.share_path.trim() || '/sdcard',
        protocol: config.protocol || 'webdav',
      }
      const res = await api.post<{
        success: boolean
        error?: string
        status?: NASStatus
      }>('/api/nas/start', payload)

      if (res?.error) {
        toastStore.error(res.error)
      } else {
        status.active = true
        status.running = true
        if (res?.status) status = { ...status, ...res.status }
        config.enabled = true
        toastStore.success(
          `NAS ${payload.protocol?.toUpperCase()} server started on port ${payload.port}`
        )
      }
    } catch (err: unknown) {
      toastStore.error(
        err instanceof Error ? err.message : 'Failed to start NAS server'
      )
    } finally {
      isToggling = false
    }
  }

  async function stopNAS() {
    try {
      isToggling = true
      const res = await api.post<{
        success: boolean
        error?: string
        status?: NASStatus
      }>('/api/nas/stop')

      if (res?.error) {
        toastStore.error(res.error)
      } else {
        status.active = false
        status.running = false
        if (res?.status) status = { ...status, ...res.status }
        config.enabled = false
        toastStore.success('NAS File Server stopped.')
      }
    } catch (err: unknown) {
      toastStore.error(
        err instanceof Error ? err.message : 'Failed to stop NAS server'
      )
    } finally {
      isToggling = false
    }
  }

  async function saveConfig() {
    if (config.auth_required && (!config.username?.trim() || !config.password)) {
      toastStore.error('Set a NAS username and password before saving an authenticated share.')
      return
    }
    try {
      isSaving = true
      const payload: NASConfig = {
        ...config,
        enabled: isServerRunning,
        port: Number(config.port) || 85,
        share_path: config.share_path.trim() || '/sdcard',
        protocol: config.protocol || 'webdav',
      }
      const res = await api.post<{
        success: boolean
        error?: string
        status?: NASStatus
      }>('/api/nas/start', payload)

      if (res?.error && payload.enabled) {
        toastStore.error(res.error)
      } else {
        if (res?.status) status = { ...status, ...res.status }
        toastStore.success(
          isServerRunning
            ? 'NAS server restarted with updated configuration.'
            : 'NAS server configuration saved.'
        )
      }
    } catch (err: unknown) {
      toastStore.error(
        err instanceof Error ? err.message : 'Failed to save configuration'
      )
    } finally {
      isSaving = false
    }
  }

  async function copyToClipboard(text: string) {
    let success = false

    // 1. Try modern async Clipboard API if available in secure context
    if (navigator?.clipboard && window.isSecureContext) {
      try {
        await navigator.clipboard.writeText(text)
        success = true
      } catch {
        success = false
      }
    }

    // 2. Fallback for non-secure contexts (HTTP over LAN IP)
    if (!success) {
      try {
        const textarea = document.createElement('textarea')
        textarea.value = text
        textarea.setAttribute('readonly', '')
        textarea.style.position = 'fixed'
        textarea.style.left = '-9999px'
        textarea.style.top = '0'
        textarea.style.opacity = '0'
        document.body.appendChild(textarea)
        textarea.focus()
        textarea.select()
        textarea.setSelectionRange(0, 99999)
        success = document.execCommand('copy')
        document.body.removeChild(textarea)
      } catch {
        success = false
      }
    }

    if (success) {
      copiedUrl = text
      toastStore.success('Copied to clipboard!')
      setTimeout(() => {
        if (copiedUrl === text) copiedUrl = null
      }, 2000)
    } else {
      toastStore.error('Failed to copy to clipboard')
    }
  }
</script>

<div class="space-y-3 sm:space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
    <div class="flex items-center gap-2.5 flex-wrap">
      <div
        class="w-9 h-9 rounded-lg bg-accent/20 border-2 border-accent flex items-center justify-center text-accent shadow-[2px_2px_0px_#000] dark:shadow-[2px_2px_0px_rgba(255,255,255,0.05)]"
      >
        <HardDrive class="w-5 h-5" />
      </div>
      <div>
        <div class="flex items-center gap-2">
          <h2
            class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground"
          >
            NAS & WebDAV File Sharing
          </h2>
          <Badge variant={isServerRunning ? 'success' : 'default'}>
            {isServerRunning ? 'Active' : 'Stopped'}
          </Badge>
          <Badge variant="info">
            {(config.protocol || 'webdav').toUpperCase()}
          </Badge>
          {#if config.read_only}
            <Badge variant="warning">Read-Only</Badge>
          {/if}
          {#if config.auth_required}
            <Badge variant="default">Auth</Badge>
          {/if}
        </div>
        <p class="text-xs text-muted font-mono">
          Expose Android storage to LAN & Wi-Fi Hotspot via WebDAV or HTTP
        </p>
      </div>
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

      {#if isServerRunning}
        <Button
          variant="danger"
          size="sm"
          disabled={isToggling}
          onclick={stopNAS}
          class="shadow-[2px_2px_0px_#000] dark:shadow-[2px_2px_0px_rgba(255,255,255,0.05)]"
        >
          <Square class="w-3.5 h-3.5 mr-1.5 {isToggling ? 'animate-spin' : ''}" />
          <span>Stop NAS</span>
        </Button>
      {:else}
        <Button
          variant="primary"
          size="sm"
          disabled={isToggling}
          onclick={startNAS}
          class="shadow-[2px_2px_0px_#000] dark:shadow-[2px_2px_0px_rgba(255,255,255,0.05)]"
        >
          <Play class="w-3.5 h-3.5 mr-1.5 {isToggling ? 'animate-spin' : ''}" />
          <span>Start NAS</span>
        </Button>
      {/if}
    </div>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-12 gap-3 sm:gap-6">
    <!-- KARTU 1: Server Configuration -->
    <Card
      title="Server Configuration"
      subtitle="Configure file sharing path, network port, protocol, and security"
      tone="ice"
      class="lg:col-span-7"
    >
      {#snippet action()}
        <Badge variant={config.protocol === 'webdav' ? 'success' : 'default'}>
          {config.protocol === 'webdav' ? 'WEBDAV MODE' : 'HTTP MODE'}
        </Badge>
      {/snippet}

      <div class="space-y-4 font-mono text-xs">
        <!-- Share Path with Presets -->
        <div class="space-y-1.5">
          <Input
            label="Share Path (Root Directory)"
            placeholder="/sdcard"
            bind:value={config.share_path}
          />
          <div class="flex items-center gap-1.5 flex-wrap pt-0.5">
            <span class="text-[10px] text-muted uppercase font-bold">Presets:</span>
            {#each ['/sdcard', '/storage/emulated/0', '/data/media/0'] as preset}
              <button
                type="button"
                class="px-2 py-0.5 rounded text-[10px] font-mono border border-border bg-card-sub hover:bg-card hover:border-accent text-foreground transition-colors cursor-pointer {config.share_path === preset ? 'border-accent text-accent font-bold' : ''}"
                onclick={() => (config.share_path = preset)}
              >
                {preset}
              </button>
            {/each}
          </div>
        </div>

        <!-- Port Input -->
        <div class="space-y-1.5">
          <Input
            type="number"
            label="Server Network Port"
            placeholder="85"
            value={String(config.port)}
            oninput={(e) => {
              const v = parseInt((e.target as HTMLInputElement).value, 10)
              if (!isNaN(v)) config.port = v
            }}
          />
          <p class="text-[10px] text-muted">
            Port used by the file server daemon (default: 85). Fully configurable via WebUI.
          </p>
        </div>

        <!-- Protocol Selector: WebDAV vs HTTP -->
        <div class="space-y-1.5 pt-1">
          <span class="block text-muted uppercase font-bold tracking-wider text-[11px]">
            File Sharing Protocol
          </span>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
            <!-- WebDAV Option -->
            <button
              type="button"
              class="p-3 rounded-lg border-2 text-left font-mono transition-all cursor-pointer {config.protocol === 'webdav'
                ? 'border-accent bg-accent/10 text-foreground shadow-[2px_2px_0px_#000] dark:shadow-[2px_2px_0px_rgba(255,255,255,0.05)]'
                : 'border-border bg-card-sub text-muted hover:border-border/80'}"
              onclick={() => (config.protocol = 'webdav')}
            >
              <div class="flex items-center justify-between">
                <span class="font-bold text-xs uppercase flex items-center gap-1.5 text-foreground">
                  <Server class="w-3.5 h-3.5 text-accent" />
                  WebDAV Server
                </span>
                {#if config.protocol === 'webdav'}
                  <Badge variant="success" class="text-[9px]">Active</Badge>
                {/if}
              </div>
              <p class="text-[10px] text-muted mt-1.5 leading-relaxed">
                Map Network Drive (Windows), Solid Explorer, VLC media streaming, Finder (macOS), Nautilus (Linux).
              </p>
            </button>

            <!-- HTTP Option -->
            <button
              type="button"
              class="p-3 rounded-lg border-2 text-left font-mono transition-all cursor-pointer {config.protocol === 'http'
                ? 'border-accent bg-accent/10 text-foreground shadow-[2px_2px_0px_#000] dark:shadow-[2px_2px_0px_rgba(255,255,255,0.05)]'
                : 'border-border bg-card-sub text-muted hover:border-border/80'}"
              onclick={() => (config.protocol = 'http')}
            >
              <div class="flex items-center justify-between">
                <span class="font-bold text-xs uppercase flex items-center gap-1.5 text-foreground">
                  <Globe class="w-3.5 h-3.5 text-cyan-400" />
                  HTTP Web Browser
                </span>
                {#if config.protocol === 'http'}
                  <Badge variant="info" class="text-[9px]">Active</Badge>
                {/if}
              </div>
              <p class="text-[10px] text-muted mt-1.5 leading-relaxed">
                Lightweight web-based directory index for simple downloads directly in web browsers.
              </p>
            </button>
          </div>
        </div>

        <!-- Security & Access Toggles -->
        <div class="space-y-3 pt-2">
          <!-- Read-Only Mode Toggle -->
          <label
            class="flex items-center justify-between p-3 bg-card-sub border-2 border-border rounded-lg cursor-pointer hover:border-accent transition-colors shadow-[2px_2px_0px_#000] dark:shadow-[2px_2px_0px_rgba(255,255,255,0.05)]"
          >
            <div class="space-y-0.5">
              <span class="font-bold text-foreground block">Read-Only Mode</span>
              <span class="text-[10px] text-muted block">
                Prohibit clients from writing, deleting, or moving files (blocks PUT, DELETE, MKCOL).
              </span>
            </div>
            <input
              type="checkbox"
              bind:checked={config.read_only}
              class="w-4 h-4 accent-accent cursor-pointer"
            />
          </label>
          {#if !config.read_only}
            <div class="rounded-lg border border-amber-500/50 bg-amber-500/10 p-3 text-[11px] text-amber-200">
              Writable sharing lets network clients create, change, and delete files in the selected directory. Enable it only on a trusted network.
            </div>
          {/if}
          {#if !config.auth_required}
            <div class="rounded-lg border border-amber-500/50 bg-amber-500/10 p-3 text-[11px] text-amber-200">
              Authentication is disabled. Anyone who can reach this port can access the shared files.
            </div>
          {/if}

          <!-- Authentication Required Toggle -->
          <label
            class="flex items-center justify-between p-3 bg-card-sub border-2 border-border rounded-lg cursor-pointer hover:border-accent transition-colors shadow-[2px_2px_0px_#000] dark:shadow-[2px_2px_0px_rgba(255,255,255,0.05)]"
          >
            <div class="space-y-0.5">
              <span class="font-bold text-foreground block">
                Authentication Required (HTTP Basic Auth)
              </span>
              <span class="text-[10px] text-muted block">
                Require username and password credentials before granting access to shared files.
              </span>
            </div>
            <input
              type="checkbox"
              bind:checked={config.auth_required}
              class="w-4 h-4 accent-accent cursor-pointer"
            />
          </label>

          <!-- Credential Inputs (if auth enabled) -->
          {#if config.auth_required}
            <div
              class="p-3 bg-black/40 border-2 border-accent/40 rounded-lg space-y-3 shadow-[1px_1px_0px_#000]"
            >
              <div class="flex items-center gap-1.5 text-accent font-bold">
                <Lock class="w-4 h-4" />
                <span>NAS Access Credentials</span>
              </div>
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <Input
                  label="Username"
                  placeholder="admin"
                  bind:value={config.username}
                />
                <div class="space-y-1 w-full">
                  <span class="block text-xs font-mono font-bold uppercase text-muted tracking-wider">
                    Password
                  </span>
                  <div class="relative">
                    <input
                      type={showPassword ? 'text' : 'password'}
                      class="neo-input w-full bg-card-sub border border-border rounded px-3 py-2 pr-9 text-sm font-mono text-foreground focus:outline-none focus:border-accent"
                      placeholder="bfr"
                      bind:value={config.password}
                    />
                    <button
                      type="button"
                      class="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted hover:text-foreground cursor-pointer"
                      onclick={() => (showPassword = !showPassword)}
                      title={showPassword ? 'Hide password' : 'Show password'}
                    >
                      {#if showPassword}
                        <EyeOff class="w-4 h-4" />
                      {:else}
                        <Eye class="w-4 h-4" />
                      {/if}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          {/if}
        </div>

        <!-- Action Save Button -->
        <div class="pt-2">
          <Button
            variant="primary"
            fullWidth
            disabled={isSaving}
            onclick={saveConfig}
            class="shadow-[2px_2px_0px_#000] dark:shadow-[2px_2px_0px_rgba(255,255,255,0.05)]"
          >
            {#if isSaving}
              <RefreshCw class="w-4 h-4 mr-2 animate-spin" />
              <span>Applying Changes...</span>
            {:else}
              <Check class="w-4 h-4 mr-2" />
              <span>{isServerRunning ? 'Apply Changes & Restart NAS' : 'Save Configuration'}</span>
            {/if}
          </Button>
        </div>
      </div>
    </Card>

    <!-- KARTU 2: Direct Access & Live Network Endpoints -->
    <Card
      title="Live Network Endpoints"
      subtitle="Accessible URLs across Wi-Fi SoftAP and local network"
      tone="peach"
      class="lg:col-span-5"
    >
      {#snippet action()}
        <Badge variant={isServerRunning ? 'success' : 'default'}>
          {isServerRunning ? 'READY' : 'OFFLINE'}
        </Badge>
      {/snippet}

      <div class="space-y-4 font-mono text-xs">
        <!-- Storage Usage Bar -->
        {#if status.storage_used && status.storage_total}
          <div
            class="p-3 bg-card-sub border-2 border-border rounded-lg space-y-2 shadow-[2px_2px_0px_#000] dark:shadow-[2px_2px_0px_rgba(255,255,255,0.05)]"
          >
            <div class="flex items-center justify-between font-mono text-xs">
              <span class="text-muted uppercase font-bold flex items-center gap-1.5">
                <Database class="w-3.5 h-3.5 text-accent" />
                Storage Utilization
              </span>
              <span class="font-black text-foreground">
                {status.storage_used} / {status.storage_total} ({storagePercent()}%)
              </span>
            </div>

            <div
              class="h-3 w-full bg-black/60 border border-border rounded overflow-hidden"
            >
              <div
                class="h-full bg-accent transition-all duration-300"
                style="width: {storagePercent()}%;"
              ></div>
            </div>
            <div class="flex items-center justify-between text-[10px] text-muted">
              <span>Path: {status.share_path || config.share_path}</span>
              <span>Available on network</span>
            </div>
          </div>
        {/if}

        <!-- Active Endpoints List -->
        <div class="space-y-3">
          <span class="block text-muted uppercase font-bold tracking-wider text-[11px]">
            Network Access Links:
          </span>

          {#if !isServerRunning}
            <div
              class="p-4 bg-card-sub border-2 border-dashed border-border rounded-lg text-center space-y-2"
            >
              <HardDrive class="w-6 h-6 text-muted mx-auto" />
              <p class="text-xs text-muted">
                Server is currently stopped. Click "Start NAS" above to activate these network endpoints.
              </p>
            </div>
          {/if}

          {#each networkEndpoints() as ep}
            <div
              class="p-3 bg-card-sub border-2 border-border rounded-lg space-y-2 shadow-[2px_2px_0px_#000] dark:shadow-[2px_2px_0px_rgba(255,255,255,0.05)]"
            >
              <div class="flex items-center justify-between">
                <span
                  class="text-[10px] font-mono font-bold uppercase tracking-wider text-muted flex items-center gap-1.5"
                >
                  {#if ep.label.includes('Hotspot') || ep.label.includes('SoftAP')}
                    <Wifi class="w-3.5 h-3.5 text-amber-400" />
                  {:else}
                    <Share2 class="w-3.5 h-3.5 text-accent" />
                  {/if}
                  {ep.label}
                </span>
                {#if ep.isPrimary}
                  <Badge variant="info">Primary</Badge>
                {/if}
              </div>

              <div
                class="flex items-center justify-between gap-2 bg-black/50 border border-border rounded p-2 font-mono text-xs"
              >
                <span class="text-foreground font-bold truncate select-all">{ep.url}</span>
                <div class="flex items-center gap-1 shrink-0">
                  <button
                    type="button"
                    class="p-1 rounded hover:bg-white/10 text-muted hover:text-foreground transition-colors cursor-pointer"
                    title="Copy URL"
                    onclick={() => copyToClipboard(ep.url)}
                  >
                    {#if copiedUrl === ep.url}
                      <Check class="w-4 h-4 text-emerald-400" />
                    {:else}
                      <Copy class="w-4 h-4" />
                    {/if}
                  </button>
                  <a
                    href={ep.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    class="p-1 rounded hover:bg-white/10 text-muted hover:text-foreground transition-colors"
                    title="Open in new browser tab"
                  >
                    <ExternalLink class="w-4 h-4" />
                  </a>
                </div>
              </div>
            </div>
          {/each}
        </div>

        <!-- Connection Summary Callout -->
        <div
          class="p-3 bg-black/40 border border-border rounded font-mono text-[11px] space-y-1.5 text-muted"
        >
          <div class="flex items-center justify-between text-foreground font-bold">
            <span>Status Summary</span>
            <span class={isServerRunning ? 'text-emerald-400' : 'text-zinc-500'}>
              {isServerRunning ? '● Live' : '○ Inactive'}
            </span>
          </div>
          <div>Mode: <strong class="text-foreground">{(config.protocol || 'webdav').toUpperCase()}</strong></div>
          <div>Access: <strong class="text-foreground">{config.auth_required ? `Protected (${config.username})` : 'Anonymous (No password)'}</strong></div>
          <div>Permissions: <strong class="text-foreground">{config.read_only ? 'Read-Only' : 'Read & Write'}</strong></div>
        </div>
      </div>
    </Card>
  </div>

  <!-- KARTU 3: OS Mount & Connection Guide (Interactive Tabs) -->
  <Card
    title="OS Mount & Connection Guide"
    subtitle="Step-by-step instructions to map network drive across operating systems"
    tone="lavender"
  >
    <div class="space-y-5 font-mono text-xs">
      <!-- OS Selector Tabs -->
      <div
        class="flex items-center gap-2 border-b border-border pb-3 overflow-x-auto select-none"
      >
        <button
          type="button"
          class="px-3 py-1.5 rounded-lg border-2 font-mono font-bold text-xs flex items-center gap-1.5 transition-all cursor-pointer whitespace-nowrap {activeOsTab ===
          'windows'
            ? 'border-accent bg-accent text-accent-text shadow-[2px_2px_0px_#000] dark:shadow-[2px_2px_0px_rgba(255,255,255,0.05)]'
            : 'border-border bg-card-sub text-muted hover:text-foreground'}"
          onclick={() => (activeOsTab = 'windows')}
        >
          <Monitor class="w-3.5 h-3.5" />
          <span>Windows (Map Drive)</span>
        </button>

        <button
          type="button"
          class="px-3 py-1.5 rounded-lg border-2 font-mono font-bold text-xs flex items-center gap-1.5 transition-all cursor-pointer whitespace-nowrap {activeOsTab ===
          'android'
            ? 'border-accent bg-accent text-accent-text shadow-[2px_2px_0px_#000] dark:shadow-[2px_2px_0px_rgba(255,255,255,0.05)]'
            : 'border-border bg-card-sub text-muted hover:text-foreground'}"
          onclick={() => (activeOsTab = 'android')}
        >
          <Smartphone class="w-3.5 h-3.5" />
          <span>Android & iOS</span>
        </button>

        <button
          type="button"
          class="px-3 py-1.5 rounded-lg border-2 font-mono font-bold text-xs flex items-center gap-1.5 transition-all cursor-pointer whitespace-nowrap {activeOsTab ===
          'macos'
            ? 'border-accent bg-accent text-accent-text shadow-[2px_2px_0px_#000] dark:shadow-[2px_2px_0px_rgba(255,255,255,0.05)]'
            : 'border-border bg-card-sub text-muted hover:text-foreground'}"
          onclick={() => (activeOsTab = 'macos')}
        >
          <Laptop class="w-3.5 h-3.5" />
          <span>macOS (Finder)</span>
        </button>

        <button
          type="button"
          class="px-3 py-1.5 rounded-lg border-2 font-mono font-bold text-xs flex items-center gap-1.5 transition-all cursor-pointer whitespace-nowrap {activeOsTab ===
          'linux'
            ? 'border-accent bg-accent text-accent-text shadow-[2px_2px_0px_#000] dark:shadow-[2px_2px_0px_rgba(255,255,255,0.05)]'
            : 'border-border bg-card-sub text-muted hover:text-foreground'}"
          onclick={() => (activeOsTab = 'linux')}
        >
          <Terminal class="w-3.5 h-3.5" />
          <span>Linux (Nautilus/Dolphin)</span>
        </button>
      </div>

      <!-- Tab Content: Windows -->
      {#if activeOsTab === 'windows'}
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <span class="font-bold text-foreground uppercase tracking-wider">
              Map Network Drive in Windows File Explorer
            </span>
            <Badge variant="info">WebDAV</Badge>
          </div>

          <ol class="space-y-2.5 text-zinc-300">
            <li class="flex items-start gap-2">
              <span class="w-5 h-5 rounded bg-accent/20 text-accent font-black flex items-center justify-center shrink-0">1</span>
              <span>Open <strong>File Explorer</strong>, right-click on <strong>This PC</strong>, and choose <strong>"Map network drive..."</strong>.</span>
            </li>
            <li class="flex items-start gap-2">
              <span class="w-5 h-5 rounded bg-accent/20 text-accent font-black flex items-center justify-center shrink-0">2</span>
              <span>Assign an available drive letter (e.g., <code>Z:</code>).</span>
            </li>
            <li class="flex items-start gap-2">
              <span class="w-5 h-5 rounded bg-accent/20 text-accent font-black flex items-center justify-center shrink-0">3</span>
              <div class="space-y-1 w-full">
                <span>Enter the WebDAV folder URL:</span>
                <div class="flex items-center gap-2 p-2 bg-black/60 border border-border rounded font-mono text-xs">
                  <code class="text-accent font-bold select-all">http://{activeTargetIp()}:{activeTargetPort}/</code>
                  <button
                    type="button"
                    class="ml-auto text-muted hover:text-foreground cursor-pointer"
                    onclick={() => copyToClipboard(`http://${activeTargetIp()}:${activeTargetPort}/`)}
                  >
                    <Copy class="w-3.5 h-3.5" />
                  </button>
                </div>
                <span class="text-[10px] text-muted block">
                  Alternate UNC path syntax: <code>\\{activeTargetIp()}@{activeTargetPort}\DavWWWRoot</code>
                </span>
              </div>
            </li>
            <li class="flex items-start gap-2">
              <span class="w-5 h-5 rounded bg-accent/20 text-accent font-black flex items-center justify-center shrink-0">4</span>
              <span>
                {#if config.auth_required}
                  Check <strong>"Connect using different credentials"</strong> and enter username (<code>{config.username}</code>) and your password.
                {:else}
                  Click <strong>Finish</strong> to mount the drive immediately.
                {/if}
              </span>
            </li>
          </ol>

          <div class="p-3 bg-card-sub border border-border rounded text-[11px] text-muted space-y-1">
            <span class="font-bold text-amber-400 block">Windows WebClient Notice:</span>
            <p>
              Windows requires HTTP Basic Auth enabled in the registry for unencrypted WebDAV links (<code>BasicAuthLevel = 2</code> under <code>HKLM\SYSTEM\CurrentControlSet\Services\WebClient\Parameters</code>). Alternatively, use third-party tools like <strong>RaiDrive</strong> or <strong>Cyberduck</strong>.
            </p>
          </div>
        </div>
      {/if}

      <!-- Tab Content: Android & iOS -->
      {#if activeOsTab === 'android'}
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <span class="font-bold text-foreground uppercase tracking-wider">
              Connect from Mobile Devices (Android & iOS)
            </span>
            <Badge variant="success">Solid / MiX / VLC</Badge>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <!-- Solid Explorer / MiXplorer -->
            <div class="p-3 bg-card-sub border-2 border-border rounded-lg space-y-2">
              <span class="font-bold text-foreground text-xs uppercase flex items-center gap-1.5">
                <Folder class="w-3.5 h-3.5 text-accent" />
                Solid Explorer / MiXplorer
              </span>
              <ol class="space-y-1 text-[11px] text-zinc-300">
                <li>1. Tap the <strong>+</strong> button and choose <strong>Storage Manager</strong>.</li>
                <li>2. Select <strong>WebDAV</strong> protocol.</li>
                <li>3. Server Address: <code>{activeTargetIp()}</code>, Port: <code>{activeTargetPort}</code>.</li>
                <li>4. Path: <code>/</code>.</li>
                {#if config.auth_required}
                  <li>5. Enter Username: <code>{config.username}</code> & Password.</li>
                {/if}
              </ol>
            </div>

            <!-- VLC Media Player -->
            <div class="p-3 bg-card-sub border-2 border-border rounded-lg space-y-2">
              <span class="font-bold text-foreground text-xs uppercase flex items-center gap-1.5">
                <Play class="w-3.5 h-3.5 text-amber-400" />
                VLC Media Player Streaming
              </span>
              <ol class="space-y-1 text-[11px] text-zinc-300">
                <li>1. Open VLC on your phone or tablet.</li>
                <li>2. Go to <strong>Browse</strong> tab -&gt; <strong>Local Network</strong>.</li>
                <li>3. Add new network stream or WebDAV location.</li>
                <li>4. Stream movies and music directly without downloading!</li>
              </ol>
            </div>
          </div>
        </div>
      {/if}

      <!-- Tab Content: macOS -->
      {#if activeOsTab === 'macos'}
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <span class="font-bold text-foreground uppercase tracking-wider">
              Mount in macOS Finder
            </span>
            <Badge variant="info">Cmd + K</Badge>
          </div>

          <ol class="space-y-2.5 text-zinc-300">
            <li class="flex items-start gap-2">
              <span class="w-5 h-5 rounded bg-accent/20 text-accent font-black flex items-center justify-center shrink-0">1</span>
              <span>Open <strong>Finder</strong> on macOS.</span>
            </li>
            <li class="flex items-start gap-2">
              <span class="w-5 h-5 rounded bg-accent/20 text-accent font-black flex items-center justify-center shrink-0">2</span>
              <span>Press <kbd class="px-1.5 py-0.5 rounded bg-black/60 border border-border text-foreground font-mono">Cmd + K</kbd> (or go to menu <strong>Go -&gt; Connect to Server...</strong>).</span>
            </li>
            <li class="flex items-start gap-2">
              <span class="w-5 h-5 rounded bg-accent/20 text-accent font-black flex items-center justify-center shrink-0">3</span>
              <div class="space-y-1 w-full">
                <span>Enter the Server Address:</span>
                <div class="flex items-center gap-2 p-2 bg-black/60 border border-border rounded font-mono text-xs">
                  <code class="text-accent font-bold select-all">http://{activeTargetIp()}:{activeTargetPort}/</code>
                  <button
                    type="button"
                    class="ml-auto text-muted hover:text-foreground cursor-pointer"
                    onclick={() => copyToClipboard(`http://${activeTargetIp()}:${activeTargetPort}/`)}
                  >
                    <Copy class="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>
            </li>
            <li class="flex items-start gap-2">
              <span class="w-5 h-5 rounded bg-accent/20 text-accent font-black flex items-center justify-center shrink-0">4</span>
              <span>
                Click <strong>Connect</strong>.
                {#if config.auth_required}
                  Select <strong>Registered User</strong> and enter Username: <code>{config.username}</code>.
                {:else}
                  Select <strong>Guest</strong>.
                {/if}
              </span>
            </li>
          </ol>
        </div>
      {/if}

      <!-- Tab Content: Linux -->
      {#if activeOsTab === 'linux'}
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <span class="font-bold text-foreground uppercase tracking-wider">
              Mount in Linux File Managers (GNOME Nautilus, KDE Dolphin, gio)
            </span>
            <Badge variant="info">dav://</Badge>
          </div>

          <div class="space-y-3 text-zinc-300">
            <!-- Nautilus / Files -->
            <div class="p-3 bg-card-sub border border-border rounded-lg space-y-1.5">
              <span class="font-bold text-foreground text-xs block">GNOME Nautilus / Files:</span>
              <p class="text-[11px] text-muted">
                Press <kbd class="px-1 py-0.5 rounded bg-black/60 border border-border font-mono">Ctrl + L</kbd> or click <strong>Other Locations</strong> -&gt; <strong>Connect to Server</strong>:
              </p>
              <div class="flex items-center gap-2 p-2 bg-black/60 border border-border rounded font-mono text-xs">
                <code class="text-accent font-bold select-all">dav://{activeTargetIp()}:{activeTargetPort}/</code>
                <button
                  type="button"
                  class="ml-auto text-muted hover:text-foreground cursor-pointer"
                  onclick={() => copyToClipboard(`dav://${activeTargetIp()}:${activeTargetPort}/`)}
                >
                  <Copy class="w-3.5 h-3.5" />
                </button>
              </div>
            </div>

            <!-- Command line gio mount -->
            <div class="p-3 bg-card-sub border border-border rounded-lg space-y-1.5">
              <span class="font-bold text-foreground text-xs block">Command Line (gvfs / gio mount):</span>
              <div class="flex items-center gap-2 p-2 bg-black/60 border border-border rounded font-mono text-xs">
                <code class="text-emerald-400 select-all">gio mount dav://{activeTargetIp()}:{activeTargetPort}/</code>
                <button
                  type="button"
                  class="ml-auto text-muted hover:text-foreground cursor-pointer"
                  onclick={() => copyToClipboard(`gio mount dav://${activeTargetIp()}:${activeTargetPort}/`)}
                >
                  <Copy class="w-3.5 h-3.5" />
                </button>
              </div>
            </div>
          </div>
        </div>
      {/if}
    </div>
  </Card>
</div>
