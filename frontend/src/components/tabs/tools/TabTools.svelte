<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Wrench,
    RefreshCw,
    Download,
    Upload,
    Check,
    Layers,
    Sliders,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'

  interface ModuleInfo {
    id: string
    name: string
    version: string
    author: string
    description: string
    enabled: boolean
  }

  let modulesList = $state<ModuleInfo[]>([])
  let isLoading = $state(false)
  let isToggling = $state<string | null>(null)

  onMount(async () => {
    await fetchModules()
  })

  async function fetchModules() {
    try {
      isLoading = true
      const res = await api.get<{ modules: ModuleInfo[] }>('/api/modules')
      if (res?.modules && Array.isArray(res.modules)) {
        modulesList = res.modules
      }
    } catch {
      // Ignored
    } finally {
      isLoading = false
    }
  }

  async function toggleModule(mod: ModuleInfo) {
    try {
      isToggling = mod.id
      const nextEnable = !mod.enabled
      await api.post('/api/modules/toggle', { id: mod.id, enable: nextEnable })
      mod.enabled = nextEnable
      toastStore.success(`Module ${mod.name} ${nextEnable ? 'enabled' : 'disabled'}. Reboot to take effect.`)
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to toggle module')
    } finally {
      isToggling = null
    }
  }

  function exportBackup() {
    window.open('/api/backup/export', '_blank')
    toastStore.success('Configuration backup download initiated.')
  }
</script>

<div class="space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Wrench class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        System Tools & Magisk Modules
      </h2>
    </div>

    <div class="flex items-center gap-2">
      <Button
        variant="outline"
        size="sm"
        onclick={exportBackup}
      >
        <Download class="w-3.5 h-3.5 mr-1" />
        <span>Export Backup</span>
      </Button>

      <Button
        variant="secondary"
        size="sm"
        disabled={isLoading}
        onclick={fetchModules}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoading ? 'animate-spin' : ''}" />
        <span>Refresh</span>
      </Button>
    </div>
  </div>

  <!-- Root Modules List -->
  <Card
    title="Installed Magisk / KernelSU Modules"
    subtitle="Manage root extensions located in /data/adb/modules"
  >
    {#if isLoading && modulesList.length === 0}
      <div class="p-8 text-center font-mono text-xs text-muted">
        <RefreshCw class="w-5 h-5 animate-spin mx-auto mb-2 text-accent" />
        Scanning /data/adb/modules...
      </div>
    {:else if modulesList.length === 0}
      <div class="p-8 text-center font-mono text-xs text-muted">
        <Layers class="w-6 h-6 mx-auto mb-2 opacity-40 text-accent" />
        No active root modules detected.
      </div>
    {:else}
      <div class="space-y-3 font-mono text-xs">
        {#each modulesList as mod}
          <div class="p-3.5 bg-card-sub border border-border rounded flex flex-col sm:flex-row sm:items-center justify-between gap-3 hover:border-accent transition-colors">
            <div class="space-y-1">
              <div class="flex items-center gap-2">
                <span class="font-bold text-sm text-foreground">{mod.name || mod.id}</span>
                <span class="text-[10px] text-muted">v{mod.version}</span>
                <Badge variant={mod.enabled ? 'success' : 'default'}>
                  {mod.enabled ? 'Enabled' : 'Disabled'}
                </Badge>
              </div>
              <p class="text-muted text-[11px] leading-relaxed">{mod.description}</p>
              <div class="text-[10px] text-gray-500">Author: {mod.author}</div>
            </div>

            <div class="shrink-0">
              <Button
                variant={mod.enabled ? 'outline' : 'primary'}
                size="sm"
                disabled={isToggling === mod.id}
                onclick={() => toggleModule(mod)}
              >
                {mod.enabled ? 'Disable Module' : 'Enable Module'}
              </Button>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </Card>
</div>
