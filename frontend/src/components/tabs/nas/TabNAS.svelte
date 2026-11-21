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
  import { themeStore } from '../../../stores/theme.svelte'
  import SakugaCheckbox from '../../ui/SakugaCheckbox.svelte'
  import { LL } from '../../../i18n/i18n-svelte'

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
        label: isHotspot ? $LL.nas.endpoints.hotspot() : $LL.nas.endpoints.primaryLan(),
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
            label: isHotspot ? $LL.nas.endpoints.wifiHotspot() : $LL.nas.endpoints.networkInterface(),
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
        label: $LL.nas.endpoints.localhost(),
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
      toastStore.error($LL.nas.toasts.setCredentialsStart())
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
          $LL.nas.toasts.started({ protocol: (payload.protocol || 'webdav').toUpperCase(), port: payload.port })
        )
      }
    } catch (err: unknown) {
      toastStore.error(
        err instanceof Error ? err.message : $LL.nas.toasts.startFailed()
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
        toastStore.success($LL.nas.toasts.stopped())
      }
    } catch (err: unknown) {
      toastStore.error(
        err instanceof Error ? err.message : $LL.nas.toasts.stopFailed()
      )
    } finally {
      isToggling = false
    }
  }

  async function saveConfig() {
    if (config.auth_required && (!config.username?.trim() || !config.password)) {
      toastStore.error($LL.nas.toasts.setCredentialsSave())
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
            ? $LL.nas.toasts.restarted()
            : $LL.nas.toasts.saved()
        )
      }
    } catch (err: unknown) {
      toastStore.error(
        err instanceof Error ? err.message : $LL.nas.toasts.saveFailed()
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
      toastStore.success($LL.nas.toasts.copied())
      setTimeout(() => {
        if (copiedUrl === text) copiedUrl = null
      }, 2000)
    } else {
      toastStore.error($LL.nas.toasts.copyFailed())
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
            {$LL.nas.title()}
          </h2>
          <Badge variant={isServerRunning ? 'success' : 'default'}>
            {isServerRunning ? $LL.nas.status.active() : $LL.nas.status.stopped()}
          </Badge>
          <Badge variant="info">
            {(config.protocol || 'webdav').toUpperCase()}
          </Badge>
          {#if config.read_only}
            <Badge variant="warning">{$LL.nas.config.readOnly()}</Badge>
          {/if}
          {#if config.auth_required}
            <Badge variant="default">{$LL.nas.status.auth()}</Badge>
          {/if}
        </div>
        <p class="text-xs text-muted font-mono">
          {$LL.nas.subtitle()}
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
        <span>{$LL.common.refresh()}</span>
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
          <span>{$LL.nas.stopNas()}</span>
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
          <span>{$LL.nas.startNas()}</span>
        </Button>
      {/if}
    </div>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-12 gap-3 sm:gap-6">
    <!-- KARTU 1: Server Configuration -->
    <Card
      title={$LL.nas.config.title()}
      subtitle={$LL.nas.config.subtitle()}
      tone="ice"
      class="lg:col-span-7"
    >
      {#snippet action()}
        <Badge variant={config.protocol === 'webdav' ? 'success' : 'default'}>
          {config.protocol === 'webdav' ? $LL.nas.config.webdavMode() : $LL.nas.config.httpMode()}
        </Badge>
      {/snippet}

      <div class="space-y-4 font-mono text-xs">
        <!-- Share Path with Presets -->
        <div class="space-y-1.5">
          <Input
            label={$LL.nas.config.sharePath()}
            placeholder="/sdcard"
            bind:value={config.share_path}
          />
          <div class="flex items-center gap-1.5 flex-wrap pt-0.5">
            <span class="text-[10px] text-muted uppercase font-bold">{$LL.nas.config.presets()}</span>
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
            label={$LL.nas.config.port()}
            placeholder="85"
            value={String(config.port)}
            oninput={(e) => {
              const v = parseInt((e.target as HTMLInputElement).value, 10)
              if (!isNaN(v)) config.port = v
            }}
          />
          <p class="text-[10px] text-muted">
            {$LL.nas.config.portHint()}
          </p>
        </div>

        <!-- Protocol Selector: WebDAV vs HTTP -->
        <div class="space-y-1.5 pt-1">
          <span class="block text-muted uppercase font-bold tracking-wider text-[11px]">
            {$LL.nas.config.protocol()}
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
                  {$LL.nas.config.webdavServer()}
                </span>
                {#if config.protocol === 'webdav'}
                  <Badge variant="success" class="text-[9px]">{$LL.nas.status.active()}</Badge>
                {/if}
              </div>
              <p class="text-[10px] text-muted mt-1.5 leading-relaxed">
                {$LL.nas.config.webdavDesc()}
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
                  {$LL.nas.config.httpBrowser()}
                </span>
                {#if config.protocol === 'http'}
                  <Badge variant="info" class="text-[9px]">{$LL.nas.status.active()}</Badge>
                {/if}
              </div>
              <p class="text-[10px] text-muted mt-1.5 leading-relaxed">
                {$LL.nas.config.httpDesc()}
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
              <span class="font-bold text-foreground block">{$LL.nas.config.readOnlyMode()}</span>
              <span class="text-[10px] text-muted block">
                {$LL.nas.config.readOnlyDesc()}
              </span>
            </div>
            {#if themeStore.useSakugaControls()}
              <SakugaCheckbox bind:checked={config.read_only} />
            {:else}
            <input
              type="checkbox"
              bind:checked={config.read_only}
              class="w-4 h-4 accent-accent cursor-pointer"
            />
            {/if}
          </label>
          {#if !config.read_only}
            <div class="rounded-lg border border-amber-500/50 bg-amber-500/10 p-3 text-[11px] text-amber-200">
              {$LL.nas.config.writableWarning()}
            </div>
          {/if}
          {#if !config.auth_required}
            <div class="rounded-lg border border-amber-500/50 bg-amber-500/10 p-3 text-[11px] text-amber-200">
              {$LL.nas.config.authDisabledWarning()}
            </div>
          {/if}

          <!-- Authentication Required Toggle -->
          <label
            class="flex items-center justify-between p-3 bg-card-sub border-2 border-border rounded-lg cursor-pointer hover:border-accent transition-colors shadow-[2px_2px_0px_#000] dark:shadow-[2px_2px_0px_rgba(255,255,255,0.05)]"
          >
            <div class="space-y-0.5">
              <span class="font-bold text-foreground block">
                {$LL.nas.config.authRequired()}
              </span>
              <span class="text-[10px] text-muted block">
                {$LL.nas.config.authRequiredDesc()}
              </span>
            </div>
            {#if themeStore.useSakugaControls()}
              <SakugaCheckbox bind:checked={config.auth_required} />
            {:else}
            <input
              type="checkbox"
              bind:checked={config.auth_required}
              class="w-4 h-4 accent-accent cursor-pointer"
            />
            {/if}
          </label>

          <!-- Credential Inputs (if auth enabled) -->
          {#if config.auth_required}
            <div
              class="p-3 bg-black/40 border-2 border-accent/40 rounded-lg space-y-3 shadow-[1px_1px_0px_#000]"
            >
              <div class="flex items-center gap-1.5 text-accent font-bold">
                <Lock class="w-4 h-4" />
                <span>{$LL.nas.config.credentials()}</span>
              </div>
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <Input
                  label={$LL.nas.config.username()}
                  placeholder="admin"
                  bind:value={config.username}
                />
                <div class="space-y-1 w-full">
                  <span class="block text-xs font-mono font-bold uppercase text-muted tracking-wider">
                    {$LL.nas.config.password()}
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
                      title={showPassword ? $LL.nas.config.hidePassword() : $LL.nas.config.showPassword()}
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
              <span>{$LL.nas.config.applying()}</span>
            {:else}
              <Check class="w-4 h-4 mr-2" />
              <span>{isServerRunning ? $LL.nas.config.applyRestart() : $LL.nas.config.saveConfig()}</span>
            {/if}
          </Button>
        </div>
      </div>
    </Card>

    <!-- KARTU 2: Direct Access & Live Network Endpoints -->
    <Card
      title={$LL.nas.endpoints.title()}
      subtitle={$LL.nas.endpoints.subtitle()}
      tone="peach"
      class="lg:col-span-5"
    >
      {#snippet action()}
        <Badge variant={isServerRunning ? 'success' : 'default'}>
          {isServerRunning ? $LL.nas.status.ready() : $LL.nas.status.offline()}
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
                {$LL.nas.endpoints.storageTitle()}
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
              <span>{$LL.nas.endpoints.path()} {status.share_path || config.share_path}</span>
              <span>{$LL.nas.endpoints.available()}</span>
            </div>
          </div>
        {/if}

        <!-- Active Endpoints List -->
        <div class="space-y-3">
          <span class="block text-muted uppercase font-bold tracking-wider text-[11px]">
            {$LL.nas.endpoints.links()}
          </span>

          {#if !isServerRunning}
            <div
              class="p-4 bg-card-sub border-2 border-dashed border-border rounded-lg text-center space-y-2"
            >
              <HardDrive class="w-6 h-6 text-muted mx-auto" />
              <p class="text-xs text-muted">
                {$LL.nas.endpoints.stoppedA()}{$LL.nas.startNas()}{$LL.nas.endpoints.stoppedB()}
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
                  <Badge variant="info">{$LL.nas.endpoints.primary()}</Badge>
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
                    title={$LL.nas.endpoints.copyUrl()}
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
                    title={$LL.nas.endpoints.openTab()}
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
            <span>{$LL.nas.summary.title()}</span>
            <span class={isServerRunning ? 'text-emerald-400' : 'text-zinc-500'}>
              {isServerRunning ? $LL.nas.summary.live() : $LL.nas.summary.inactive()}
            </span>
          </div>
          <div>{$LL.nas.summary.mode()} <strong class="text-foreground">{(config.protocol || 'webdav').toUpperCase()}</strong></div>
          <div>{$LL.nas.summary.access()} <strong class="text-foreground">{config.auth_required ? $LL.nas.summary.protected({ username: config.username }) : $LL.nas.summary.anonymous()}</strong></div>
          <div>{$LL.nas.summary.permissions()} <strong class="text-foreground">{config.read_only ? $LL.nas.config.readOnly() : $LL.nas.summary.readWrite()}</strong></div>
        </div>
      </div>
    </Card>
  </div>

  <!-- KARTU 3: OS Mount & Connection Guide (Interactive Tabs) -->
  <Card
    title={$LL.nas.guide.title()}
    subtitle={$LL.nas.guide.subtitle()}
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
          <span>{$LL.nas.guide.windows()}</span>
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
          <span>{$LL.nas.guide.android()}</span>
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
          <span>{$LL.nas.guide.macos()}</span>
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
          <span>{$LL.nas.guide.linux()}</span>
        </button>
      </div>

      <!-- Tab Content: Windows -->
      {#if activeOsTab === 'windows'}
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <span class="font-bold text-foreground uppercase tracking-wider">
              {$LL.nas.guide.windowsTitle()}
            </span>
            <Badge variant="info">WebDAV</Badge>
          </div>

          <ol class="space-y-2.5 text-zinc-300">
            <li class="flex items-start gap-2">
              <span class="w-5 h-5 rounded bg-accent/20 text-accent font-black flex items-center justify-center shrink-0">1</span>
              <span>{$LL.nas.guide.winStep1A()}<strong>File Explorer</strong>{$LL.nas.guide.winStep1B()}<strong>This PC</strong>{$LL.nas.guide.winStep1C()}<strong>"Map network drive..."</strong>{$LL.nas.guide.winStep1D()}</span>
            </li>
            <li class="flex items-start gap-2">
              <span class="w-5 h-5 rounded bg-accent/20 text-accent font-black flex items-center justify-center shrink-0">2</span>
              <span>{$LL.nas.guide.winStep2A()}<code>Z:</code>{$LL.nas.guide.winStep2B()}</span>
            </li>
            <li class="flex items-start gap-2">
              <span class="w-5 h-5 rounded bg-accent/20 text-accent font-black flex items-center justify-center shrink-0">3</span>
              <div class="space-y-1 w-full">
                <span>{$LL.nas.guide.winStep3()}</span>
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
                  {$LL.nas.guide.winAltA()}<code>\\{activeTargetIp()}@{activeTargetPort}\DavWWWRoot</code>
                </span>
              </div>
            </li>
            <li class="flex items-start gap-2">
              <span class="w-5 h-5 rounded bg-accent/20 text-accent font-black flex items-center justify-center shrink-0">4</span>
              <span>
                {#if config.auth_required}
                  {$LL.nas.guide.winStep4AuthA()}<strong>"Connect using different credentials"</strong>{$LL.nas.guide.winStep4AuthB()}<code>{config.username}</code>{$LL.nas.guide.winStep4AuthC()}
                {:else}
                  {$LL.nas.guide.winStep4GuestA()}<strong>Finish</strong>{$LL.nas.guide.winStep4GuestB()}
                {/if}
              </span>
            </li>
          </ol>

          <div class="p-3 bg-card-sub border border-border rounded text-[11px] text-muted space-y-1">
            <span class="font-bold text-amber-400 block">{$LL.nas.guide.winNoticeTitle()}</span>
            <p>
              {$LL.nas.guide.winNoticeA()}<code>BasicAuthLevel = 2</code>{$LL.nas.guide.winNoticeB()}<code>HKLM\SYSTEM\CurrentControlSet\Services\WebClient\Parameters</code>{$LL.nas.guide.winNoticeC()}<strong>RaiDrive</strong>{$LL.nas.guide.winNoticeD()}<strong>Cyberduck</strong>{$LL.nas.guide.winNoticeE()}
            </p>
          </div>
        </div>
      {/if}

      <!-- Tab Content: Android & iOS -->
      {#if activeOsTab === 'android'}
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <span class="font-bold text-foreground uppercase tracking-wider">
              {$LL.nas.guide.androidTitle()}
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
                <li>1. {$LL.nas.guide.andStep1A()}<strong>+</strong>{$LL.nas.guide.andStep1B()}<strong>Storage Manager</strong>{$LL.nas.guide.andStep1C()}</li>
                <li>2. {$LL.nas.guide.andStep2A()}<strong>WebDAV</strong>{$LL.nas.guide.andStep2B()}</li>
                <li>3. {$LL.nas.guide.andStep3A()}<code>{activeTargetIp()}</code>{$LL.nas.guide.andStep3B()}<code>{activeTargetPort}</code>{$LL.nas.guide.andStep3C()}</li>
                <li>4. {$LL.nas.guide.andStep4A()}<code>/</code>{$LL.nas.guide.andStep4B()}</li>
                {#if config.auth_required}
                  <li>5. {$LL.nas.guide.andStep5A()}<code>{config.username}</code>{$LL.nas.guide.andStep5B()}</li>
                {/if}
              </ol>
            </div>

            <!-- VLC Media Player -->
            <div class="p-3 bg-card-sub border-2 border-border rounded-lg space-y-2">
              <span class="font-bold text-foreground text-xs uppercase flex items-center gap-1.5">
                <Play class="w-3.5 h-3.5 text-amber-400" />
                {$LL.nas.guide.vlcTitle()}
              </span>
              <ol class="space-y-1 text-[11px] text-zinc-300">
                <li>1. {$LL.nas.guide.vlcStep1()}</li>
                <li>2. {$LL.nas.guide.vlcStep2A()}<strong>Browse</strong>{$LL.nas.guide.vlcStep2B()}<strong>Local Network</strong>{$LL.nas.guide.vlcStep2C()}</li>
                <li>3. {$LL.nas.guide.vlcStep3()}</li>
                <li>4. {$LL.nas.guide.vlcStep4()}</li>
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
              {$LL.nas.guide.macosTitle()}
            </span>
            <Badge variant="info">Cmd + K</Badge>
          </div>

          <ol class="space-y-2.5 text-zinc-300">
            <li class="flex items-start gap-2">
              <span class="w-5 h-5 rounded bg-accent/20 text-accent font-black flex items-center justify-center shrink-0">1</span>
              <span>{$LL.nas.guide.macStep1A()}<strong>Finder</strong>{$LL.nas.guide.macStep1B()}</span>
            </li>
            <li class="flex items-start gap-2">
              <span class="w-5 h-5 rounded bg-accent/20 text-accent font-black flex items-center justify-center shrink-0">2</span>
              <span>{$LL.nas.guide.macStep2A()}<kbd class="px-1.5 py-0.5 rounded bg-black/60 border border-border text-foreground font-mono">Cmd + K</kbd>{$LL.nas.guide.macStep2B()}<strong>Go -&gt; Connect to Server...</strong>{$LL.nas.guide.macStep2C()}</span>
            </li>
            <li class="flex items-start gap-2">
              <span class="w-5 h-5 rounded bg-accent/20 text-accent font-black flex items-center justify-center shrink-0">3</span>
              <div class="space-y-1 w-full">
                <span>{$LL.nas.guide.macStep3()}</span>
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
                {$LL.nas.guide.macStep4A()}<strong>Connect</strong>{$LL.nas.guide.macStep4B()}
                {#if config.auth_required}
                  {$LL.nas.guide.macStep4AuthA()}<strong>Registered User</strong>{$LL.nas.guide.macStep4AuthB()}<code>{config.username}</code>{$LL.nas.guide.macStep4AuthC()}
                {:else}
                  {$LL.nas.guide.macStep4GuestA()}<strong>Guest</strong>{$LL.nas.guide.macStep4GuestB()}
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
              {$LL.nas.guide.linuxTitle()}
            </span>
            <Badge variant="info">dav://</Badge>
          </div>

          <div class="space-y-3 text-zinc-300">
            <!-- Nautilus / Files -->
            <div class="p-3 bg-card-sub border border-border rounded-lg space-y-1.5">
              <span class="font-bold text-foreground text-xs block">{$LL.nas.guide.nautilusTitle()}</span>
              <p class="text-[11px] text-muted">
                {$LL.nas.guide.linuxNautilusA()}<kbd class="px-1 py-0.5 rounded bg-black/60 border border-border font-mono">Ctrl + L</kbd>{$LL.nas.guide.linuxNautilusB()}<strong>Other Locations</strong>{$LL.nas.guide.linuxNautilusC()}<strong>Connect to Server</strong>{$LL.nas.guide.linuxNautilusD()}
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
              <span class="font-bold text-foreground text-xs block">{$LL.nas.guide.cliTitle()}</span>
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
