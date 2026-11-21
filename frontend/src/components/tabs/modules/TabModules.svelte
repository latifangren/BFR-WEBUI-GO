<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Box,
    RefreshCw,
    Upload,
    Check,
    X,
    AlertCircle,
    FileArchive,
    Terminal,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { LL } from '../../../i18n/i18n-svelte'
  import { toastStore } from '../../../stores/toast.svelte'
  import type { ModuleInfo, ModulesResponse } from '../../../types/modules'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import { themeStore } from '../../../stores/theme.svelte'
  import SakugaCheckbox from '../../ui/SakugaCheckbox.svelte'

  let modules = $state<ModuleInfo[]>([])
  let isLoading = $state(false)
  let isUploading = $state(false)
  let togglingId = $state<string | null>(null)

  // File Upload State
  let selectedFile = $state<File | null>(null)
  let isDragOver = $state(false)
  let fileInputEl: HTMLInputElement | null = $state(null)
  let installOutput = $state<string | null>(null)

  onMount(async () => {
    await fetchModules()
  })

  async function fetchModules() {
    try {
      isLoading = true
      const res = await api.get<ModulesResponse | { modules: ModuleInfo[] }>('/api/modules')
      if (res && Array.isArray(res.modules)) {
        modules = res.modules
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.modules.toasts.fetchFailed())
    } finally {
      isLoading = false
    }
  }

  async function toggleModule(mod: ModuleInfo) {
    try {
      togglingId = mod.id
      const nextState = !mod.enabled
      await api.post('/api/modules/toggle', {
        id: mod.id,
        enable: nextState,
        enabled: nextState,
      })
      mod.enabled = nextState
      toastStore.success(
        nextState
          ? $LL.modules.toasts.toggleEnabled({ name: mod.name })
          : $LL.modules.toasts.toggleDisabled({ name: mod.name })
      )
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.modules.toasts.toggleFailed())
    } finally {
      togglingId = null
    }
  }

  function handleFileSelect(e: Event) {
    const target = e.target as HTMLInputElement
    if (target.files && target.files[0]) {
      selectedFile = target.files[0]
    }
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault()
    isDragOver = false
    if (e.dataTransfer && e.dataTransfer.files && e.dataTransfer.files[0]) {
      selectedFile = e.dataTransfer.files[0]
    }
  }

  async function installModule() {
    if (!selectedFile) return

    try {
      isUploading = true
      installOutput = null

      const formData = new FormData()
      formData.append('module', selectedFile)
      formData.append('file', selectedFile)

      const res = await api.post<{ success?: boolean; output?: string; message?: string }>('/api/modules/install', formData)
      if (res.output) {
        installOutput = res.output
      } else if (res.message) {
        installOutput = res.message
      } else {
        installOutput = $LL.modules.installer.complete()
      }

      toastStore.success($LL.modules.toasts.installed({ name: selectedFile.name }))
      selectedFile = null
      if (fileInputEl) fileInputEl.value = ''
      await fetchModules()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : $LL.modules.toasts.installFailed()
      installOutput = `Error: ${msg}`
      toastStore.error(msg)
    } finally {
      isUploading = false
    }
  }
</script>

<div class="space-y-3 sm:space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Box class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        {$LL.modules.title()}
      </h2>
      <Badge variant="default">
        {$LL.modules.installedCount({ count: modules.length })}
      </Badge>
    </div>

    <div class="flex items-center gap-2">
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoading}
        onclick={fetchModules}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoading ? 'animate-spin' : ''}" />
        <span>{$LL.common.refresh()}</span>
      </Button>
    </div>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-3 sm:gap-6">
    <!-- Module Installer Card -->
    <!-- Module Installer Card (order-2 on mobile, order-1 on desktop) -->
    <Card
      title={$LL.modules.installer.title()}
      subtitle={$LL.modules.installer.subtitle()}
      tone="lavender"
      class="order-2 lg:order-1 lg:col-span-1"
    >
      <div class="space-y-4 font-mono text-xs">
        <!-- Hidden file input -->
        <input
          type="file"
          accept=".zip"
          class="hidden"
          bind:this={fileInputEl}
          onchange={handleFileSelect}
        />

        <!-- Drop Zone -->
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div
          class="border-2 border-dashed rounded-lg p-3.5 sm:p-6 text-center transition-colors cursor-pointer {isDragOver ? 'border-accent bg-accent/10' : 'border-border bg-card-sub hover:border-accent'}"
          ondragover={(e) => { e.preventDefault(); isDragOver = true }}
          ondragleave={() => { isDragOver = false }}
          ondrop={handleDrop}
          onclick={() => fileInputEl?.click()}
        >
          <Upload class="w-8 h-8 mx-auto mb-2 text-muted" />
          {#if selectedFile}
            <div class="text-foreground font-bold break-all">
              {selectedFile.name}
            </div>
            <span class="text-[10px] text-accent block mt-1">
              {(selectedFile.size / (1024 * 1024)).toFixed(2)} MB
            </span>
          {:else}
            <span class="font-bold text-foreground block">{$LL.modules.installer.selectOrDrop()}</span>
            <span class="text-[10px] text-muted">{$LL.modules.installer.supports()}</span>
          {/if}
        </div>

        <Button
          variant="primary"
          size="md"
          class="w-full justify-center"
          disabled={!selectedFile || isUploading}
          onclick={installModule}
        >
          <FileArchive class="w-3.5 h-3.5 mr-1.5 {isUploading ? 'animate-spin' : ''}" />
          <span>{isUploading ? $LL.modules.installer.flashing() : $LL.modules.installer.install()}</span>
        </Button>

        {#if installOutput}
          <div class="space-y-1 pt-2 border-t border-border">
            <span class="text-[10px] text-muted uppercase font-bold flex items-center gap-1">
              <Terminal class="w-3 h-3 text-accent" />
              {$LL.modules.installer.flashLog()}
            </span>
            <pre class="bg-black/80 border border-border rounded p-2 text-[11px] text-emerald-400 font-mono whitespace-pre-wrap max-h-48 overflow-y-auto">{installOutput}</pre>
          </div>
        {/if}
      </div>
    </Card>

    <!-- Installed Modules List Card (order-1 on mobile, order-2 on desktop) -->
    <Card
      title={$LL.modules.list.title()}
      subtitle={$LL.modules.list.subtitle()}
      tone="lavender"
      class="order-1 lg:order-2 lg:col-span-2"
    >
      {#if isLoading && modules.length === 0}
        <div class="p-8 text-center text-muted font-mono text-xs">
          <RefreshCw class="w-6 h-6 animate-spin mx-auto mb-2 text-accent" />
          {$LL.modules.list.scanning()}
        </div>
      {:else if modules.length === 0}
        <div class="p-8 text-center text-muted font-mono text-xs">
          <Box class="w-6 h-6 mx-auto mb-2 opacity-50" />
          {$LL.modules.list.empty()}
        </div>
      {:else}
        <div class="space-y-3 font-mono text-xs">
          {#each modules as mod (mod.id)}
            <div class="p-3.5 bg-card-sub border border-border rounded-lg space-y-2 hover:border-border transition-colors">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <div class="flex items-center gap-2">
                    <span class="font-bold text-foreground text-sm truncate">{mod.name}</span>
                    <Badge variant={mod.enabled ? 'success' : 'default'}>
                      {mod.enabled ? $LL.modules.list.enabled() : $LL.modules.list.disabled()}
                    </Badge>
                  </div>
                  <div class="flex items-center gap-3 text-[11px] text-muted mt-0.5">
                    <span>v{mod.version}</span>
                    <span>by {mod.author || $LL.common.unknown()}</span>
                    <span class="text-accent">({mod.id})</span>
                  </div>
                </div>

                <!-- Enable / Disable Switch -->
                <label class="flex items-center gap-1.5 cursor-pointer select-none shrink-0">
                  {#if themeStore.useSakugaControls()}
                    <SakugaCheckbox checked={mod.enabled} disabled={togglingId === mod.id} onchange={() => toggleModule(mod)} />
                  {:else}
                  <input
                    type="checkbox"
                    checked={mod.enabled}
                    disabled={togglingId === mod.id}
                    onchange={() => toggleModule(mod)}
                    class="w-4 h-4 accent-accent cursor-pointer"
                  />
                  {/if}
                </label>
              </div>

              {#if mod.description}
                <p class="text-muted text-[11px] border-t border-border/50 pt-2 leading-relaxed">
                  {mod.description}
                </p>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    </Card>
  </div>
</div>
