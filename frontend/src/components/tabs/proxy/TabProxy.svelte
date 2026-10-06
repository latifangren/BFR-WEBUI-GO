<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Shield,
    RefreshCw,
    ExternalLink,
    FileCode,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import ProxyHero from './ProxyHero.svelte'
  import ProxyNodes from './ProxyNodes.svelte'
  import ProxyRouting from './ProxyRouting.svelte'
  import ProxyApps from './ProxyApps.svelte'
  import ProxySubscription from './ProxySubscription.svelte'
  import ProxyLogs from './ProxyLogs.svelte'

  interface CoreInfo {
    name: string
    path: string
    exists: boolean
    running: boolean
    pid: number
    memory: string
  }

  interface BoxSettings {
    bin_name: string
    network_mode: string
    ipv6: boolean
  }

  interface StatusResponse {
    cores: CoreInfo[]
    mode: string
    watchdog: boolean
    box_settings?: BoxSettings
    version?: string
    config_path?: string
  }

  let isRunning = $state(false)
  let activeCore = $state('Mihomo')
  let activePid = $state(0)
  let memoryUsage = $state('0 MB')
  let coreVersion = $state('')
  let currentMode = $state('Rule')
  let watchdog = $state(false)
  let isLoading = $state(false)

  onMount(async () => {
    await fetchStatus()
  })

  async function fetchStatus() {
    try {
      isLoading = true
      const res = await api.get<StatusResponse>('/api/proxy/status')
      if (res) {
        currentMode = res.mode || 'Rule'
        watchdog = !!res.watchdog
        coreVersion = res.version || ''

        if (res.box_settings?.bin_name) {
          activeCore = res.box_settings.bin_name
        }

        const runningCore = res.cores?.find((c) => c.running)
        if (runningCore) {
          isRunning = true
          activeCore = runningCore.name
          activePid = runningCore.pid
          memoryUsage = runningCore.memory || 'N/A'
        } else {
          isRunning = false
          activePid = 0
          memoryUsage = '0 MB'
        }
      }
    } catch {
      // Ignored
    } finally {
      isLoading = false
    }
  }

  async function handleServiceAction(action: string) {
    try {
      isLoading = true
      const res = await api.post<{ success: boolean; error?: string }>('/api/proxy/control', {
        action,
      })
      if (res && res.success) {
        toastStore.success(`Proxy service action '${action}' dispatched successfully.`)
        await fetchStatus()
      } else {
        toastStore.error(res?.error || `Failed to execute ${action}`)
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : `Failed to execute ${action}`)
    } finally {
      isLoading = false
    }
  }

  async function handleSwitchCore(core: string) {
    try {
      isLoading = true
      const res = await api.post<{ success: boolean; error?: string }>('/api/proxy/control', {
        action: 'switch_core',
        core,
      })
      if (res && res.success) {
        toastStore.success(`Active core switched to ${core}. Restarting service...`)
        activeCore = core
        await fetchStatus()
      } else {
        toastStore.error(res?.error || 'Failed to switch core')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Core switch error')
    } finally {
      isLoading = false
    }
  }

  async function handleModeChange(mode: string) {
    try {
      isLoading = true
      const res = await api.post<{ success: boolean; error?: string }>('/api/proxy/control', {
        mode,
      })
      if (res && res.success) {
        toastStore.success(`Routing mode updated to ${mode}`)
        currentMode = mode
      } else {
        toastStore.error(res?.error || 'Failed to change mode')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Mode update error')
    } finally {
      isLoading = false
    }
  }

  async function handleToggleWatchdog() {
    try {
      const nextState = !watchdog
      const res = await api.post<{ success: boolean; enabled: boolean }>('/api/proxy/watchdog', {
        enable: nextState,
      })
      if (res && res.success) {
        watchdog = res.enabled
        toastStore.info(`Watchdog supervisor: ${watchdog ? 'Enabled' : 'Disabled'}`)
      }
    } catch {
      toastStore.error('Failed to update watchdog state')
    }
  }
</script>

<div class="space-y-4 sm:space-y-6 font-mono">
  <!-- Top Bar Header -->
  <div class="flex items-center justify-between pb-2 border-b border-border">
    <div class="flex items-center gap-2">
      <Shield class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-bold uppercase tracking-wider text-foreground">
        Clash / Mihomo Core Manager
      </h2>
      <Badge variant="default" class="text-[10px]">Box for Magisk Suite</Badge>
    </div>

    <div class="flex items-center gap-2">
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoading}
        onclick={fetchStatus}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoading ? 'animate-spin' : ''}" />
        <span>Refresh</span>
      </Button>
    </div>
  </div>

  <!-- 1. Proxy Hero & Lifecycle Manager -->
  <ProxyHero
    {isRunning}
    {activeCore}
    {activePid}
    {memoryUsage}
    {coreVersion}
    {watchdog}
    {isLoading}
    onAction={handleServiceAction}
    onSwitchCore={handleSwitchCore}
    onToggleWatchdog={handleToggleWatchdog}
  />

  <!-- 2. Nodes, Groups & Latency -->
  <ProxyNodes
    {currentMode}
    onModeChange={handleModeChange}
  />

  <!-- 3. Transparent Routing & Hotspot Tethering -->
  <ProxyRouting />

  <!-- 4. Per-App Proxy Rules (Blacklist vs Whitelist) -->
  <ProxyApps />

  <!-- 5. Subscription & Assets -->
  <ProxySubscription />

  <!-- 6. Real-Time Console Logs -->
  <ProxyLogs />
</div>
