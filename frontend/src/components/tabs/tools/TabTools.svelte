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
    Cloud,
    CloudUpload,
    Settings,
    FileText,
    ShieldCheck,
    AlertCircle,
    KeyRound,
    ArrowRight,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { LL } from '../../../i18n/i18n-svelte'
  import { toastStore } from '../../../stores/toast.svelte'
  import { navigationStore } from '../../../stores/navigation.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Modal from '../../ui/Modal.svelte'
  import Input from '../../ui/Input.svelte'
  import ChangePasswordModal from '../../modals/ChangePasswordModal.svelte'
  import { themeStore } from '../../../stores/theme.svelte'
  import SakugaCheckbox from '../../ui/SakugaCheckbox.svelte'

  interface CloudConfig {
    enabled: boolean
    provider: string
    url: string
    username: string
    password?: string
    interval_hours: number
    last_sync?: string
  }

  let isLoading = $state(false)

  // Backup & Cloud Sync State
  let showImportModal = $state(false)
  let showCloudModal = $state(false)
  let showPasswordModal = $state(false)
  let importJsonText = $state('')
  let isImporting = $state(false)
  let isSavingCloud = $state(false)
  let isSyncingCloud = $state(false)

  let cloudConfig = $state<CloudConfig>({
    enabled: false,
    provider: 'webdav',
    url: '',
    username: '',
    password: '',
    interval_hours: 24,
  })

  onMount(async () => {
    await fetchCloudConfig()
  })

  async function fetchCloudConfig() {
    try {
      const res = await api.get<CloudConfig>('/api/backup/cloud/config')
      if (res) {
        cloudConfig = {
          ...res,
          provider: res.provider || 'webdav',
          interval_hours: res.interval_hours || 24,
        }
      }
    } catch {
      // Ignored
    }
  }

  async function saveCloudConfig() {
    try {
      isSavingCloud = true
      const res = await api.post<CloudConfig>('/api/backup/cloud/config', cloudConfig)
      if (res) {
        cloudConfig = res
        toastStore.success($LL.tools.toasts.cloudSaved())
        showCloudModal = false
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.tools.toasts.saveCloudFailed())
    } finally {
      isSavingCloud = false
    }
  }

  async function syncCloudBackup() {
    try {
      isSyncingCloud = true
      const res = await api.post<{ success: boolean; error?: string }>('/api/backup/cloud/sync')
      if (res?.success) {
        toastStore.success($LL.tools.toasts.syncSucceeded())
        await fetchCloudConfig()
      } else {
        toastStore.error(res?.error || $LL.tools.toasts.syncFailed())
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.tools.toasts.syncError())
    } finally {
      isSyncingCloud = false
    }
  }

  async function exportBackup() {
    try {
      const res = await fetch('/api/backup/export', { credentials: 'include' })
      if (!res.ok) throw new Error(`HTTP error ${res.status}`)
      const blob = await res.blob()
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `bfr_backup_${new Date().toISOString().slice(0, 10)}.json`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      window.URL.revokeObjectURL(url)
      toastStore.success($LL.tools.toasts.exported())
    } catch {
      window.open('/api/backup/export', '_blank')
    }
  }

  async function handleFileSelected(e: Event) {
    const input = e.target as HTMLInputElement
    if (input.files && input.files[0]) {
      const file = input.files[0]
      importJsonText = await file.text()
    }
  }

  async function doImportBackup() {
    if (!importJsonText.trim()) {
      toastStore.error($LL.tools.toasts.selectValidJson())
      return
    }
    try {
      isImporting = true
      const parsed = JSON.parse(importJsonText)
      const res = await api.post<{ success: boolean; error?: string }>('/api/backup/import', parsed)
      if (res?.success) {
        toastStore.success($LL.tools.toasts.imported())
        showImportModal = false
        importJsonText = ''
      } else {
        toastStore.error(res?.error || $LL.tools.toasts.importFailed())
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.tools.toasts.invalidJson())
    } finally {
      isImporting = false
    }
  }
</script>

<div class="space-y-3 sm:space-y-6 font-mono">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between flex-wrap gap-3">
    <div class="flex items-center gap-2">
      <Wrench class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-bold uppercase tracking-wider text-foreground">
        {$LL.tools.title()}
      </h2>
    </div>

    <div class="flex items-center gap-2 flex-wrap">
      <Button
        variant="outline"
        size="sm"
        onclick={exportBackup}
        title={$LL.tools.exportTitle()}
      >
        <Download class="w-3.5 h-3.5 mr-1" />
        <span>{$LL.tools.exportBackup()}</span>
      </Button>

      <Button
        variant="secondary"
        size="sm"
        onclick={() => (showImportModal = true)}
        title={$LL.tools.importTitle()}
      >
        <Upload class="w-3.5 h-3.5 mr-1" />
        <span>{$LL.tools.import()}</span>
      </Button>

      <Button
        variant="primary"
        size="sm"
        onclick={() => (showCloudModal = true)}
        title={$LL.tools.cloudTitle()}
      >
        <Cloud class="w-3.5 h-3.5 mr-1" />
        <span>{$LL.tools.cloudSync()}</span>
      </Button>

      <Button
        variant="secondary"
        size="sm"
        disabled={isLoading}
        onclick={fetchCloudConfig}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoading ? 'animate-spin' : ''}" />
        <span>{$LL.common.refresh()}</span>
      </Button>
    </div>
  </div>

  <!-- Security & Authentication Card -->
  <Card title={$LL.tools.security.title()} subtitle={$LL.tools.security.subtitle()} tone="butter">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-3.5 bg-card-sub border border-border rounded font-mono text-xs">
      <div class="flex items-center gap-3">
        <div class="p-2.5 rounded bg-card border border-border text-accent">
          <KeyRound class="w-5 h-5" />
        </div>
        <div>
          <span class="font-bold text-foreground text-sm block">{$LL.tools.security.adminPassword()}</span>
          <span class="text-[11px] text-muted">{$LL.tools.security.adminPasswordDesc()}</span>
        </div>
      </div>
      <Button
        variant="primary"
        size="sm"
        onclick={() => (showPasswordModal = true)}
      >
        <KeyRound class="w-3.5 h-3.5 mr-1.5" />
        <span>{$LL.common.changePassword()}</span>
      </Button>
    </div>
  </Card>

  <!-- Backup & Cloud Synchronization Status Card -->
  <Card title={$LL.tools.backup.title()} tone="butter">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 font-mono text-xs">
      <!-- Cloud State -->
      <div class="p-3.5 rounded bg-card-sub border border-border space-y-2 flex flex-col justify-between">
        <div>
          <div class="flex items-center justify-between mb-1">
            <span class="text-[10px] text-muted uppercase font-bold">{$LL.tools.cloud.provider()}</span>
            <Badge variant={cloudConfig.enabled ? 'success' : 'default'}>
              {cloudConfig.enabled ? $LL.common.active() : $LL.tools.cloud.disabled()}
            </Badge>
          </div>
          <div class="text-sm font-bold text-foreground capitalize">
            {cloudConfig.provider || 'WebDAV'}
          </div>
          <p class="text-[11px] text-muted truncate mt-0.5">
            {cloudConfig.url || $LL.tools.cloud.noEndpoint()}
          </p>
        </div>

        <div class="pt-2 border-t border-border/60 flex items-center justify-between text-[10px]">
          <span class="text-muted">{$LL.tools.cloud.lastSynced()}</span>
          <span class="font-bold text-foreground">
            {cloudConfig.last_sync ? new Date(cloudConfig.last_sync).toLocaleString() : $LL.tools.cloud.never()}
          </span>
        </div>
      </div>

      <!-- Quick Actions Strip -->
      <div class="p-3.5 rounded bg-card-sub border border-border space-y-2.5 flex flex-col justify-between">
        <div>
          <span class="text-[10px] text-muted uppercase font-bold block mb-1">{$LL.tools.cloud.instantOps()}</span>
          <p class="text-[11px] text-muted leading-relaxed">
            {$LL.tools.cloud.instantOpsDesc()}
          </p>
        </div>

        <div class="flex items-center gap-2 pt-2 border-t border-border/60">
          <Button
            variant="primary"
            size="sm"
            fullWidth={true}
            disabled={isSyncingCloud || !cloudConfig.enabled}
            onclick={syncCloudBackup}
          >
            <CloudUpload class="w-3.5 h-3.5 mr-1 {isSyncingCloud ? 'animate-spin' : ''}" />
            <span>{isSyncingCloud ? $LL.tools.cloud.syncing() : $LL.tools.cloud.syncNow()}</span>
          </Button>
        </div>
      </div>

      <!-- Local Backup Management -->
      <div class="p-3.5 rounded bg-card-sub border border-border space-y-2.5 flex flex-col justify-between">
        <div>
          <span class="text-[10px] text-muted uppercase font-bold block mb-1">{$LL.tools.local.title()}</span>
          <p class="text-[11px] text-muted leading-relaxed">
            {$LL.tools.local.desc()}
          </p>
        </div>

        <div class="flex items-center gap-2 pt-2 border-t border-border/60">
          <Button variant="outline" size="sm" class="flex-1" onclick={exportBackup}>
            <Download class="w-3.5 h-3.5 mr-1" />
            <span>{$LL.tools.export()}</span>
          </Button>
          <Button variant="secondary" size="sm" class="flex-1" onclick={() => (showImportModal = true)}>
            <Upload class="w-3.5 h-3.5 mr-1" />
            <span>{$LL.tools.restore()}</span>
          </Button>
        </div>
      </div>
    </div>
  </Card>

  <!-- Root Modules Quick Navigation Card -->
  <Card
    title={$LL.tools.modules.title()}
    subtitle={$LL.tools.modules.subtitle()}
    tone="butter"
  >
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-3.5 bg-card-sub border border-border rounded font-mono text-xs">
      <div class="flex items-center gap-3">
        <div class="p-2.5 rounded bg-card border border-border text-accent">
          <Layers class="w-5 h-5" />
        </div>
        <div>
          <span class="font-bold text-foreground text-sm block">{$LL.tools.modules.overlays()}</span>
          <span class="text-[11px] text-muted">
            {$LL.tools.modules.overlaysDesc()}
          </span>
        </div>
      </div>
      <Button
        variant="primary"
        size="sm"
        onclick={() => navigationStore.setTab('modules')}
      >
        <span>{$LL.tools.modules.openModules()}</span>
        <ArrowRight class="w-3.5 h-3.5 ml-1.5" />
      </Button>
    </div>
  </Card>
</div>

<!-- Cloud Sync Configuration Modal -->
<Modal
  open={showCloudModal}
  title={$LL.tools.cloudModal.title()}
  onclose={() => (showCloudModal = false)}
  class="!max-w-md"
>
  <div class="space-y-3 font-mono text-xs">
    <div class="flex items-center justify-between p-3 rounded bg-card-sub border border-border">
      <div>
        <span class="font-bold text-foreground block text-xs">{$LL.tools.cloudModal.enableAuto()}</span>
        <span class="text-[10px] text-muted">{$LL.tools.cloudModal.enableAutoDesc()}</span>
      </div>
      {#if themeStore.useSakugaControls()}
        <SakugaCheckbox bind:checked={cloudConfig.enabled} />
      {:else}
      <input
        type="checkbox"
        bind:checked={cloudConfig.enabled}
        class="w-4 h-4 accent-accent cursor-pointer"
      />
      {/if}
    </div>

    <div class="space-y-1">
      <label for="cloud-provider" class="text-[10px] uppercase font-bold text-muted block">{$LL.tools.cloudModal.providerLabel()}</label>
      <select
        id="cloud-provider"
        bind:value={cloudConfig.provider}
        class="neo-input w-full bg-card-sub border border-border rounded px-3 py-2 text-xs font-mono text-foreground focus:outline-none focus:border-accent"
      >
        <option value="webdav">{$LL.tools.cloudModal.providerWebdav()}</option>
      </select>
    </div>

    <Input
      label={$LL.tools.cloudModal.endpointLabel()}
      placeholder="https://dav.example.com/remote.php/webdav/backups/"
      bind:value={cloudConfig.url}
    />

    <Input
      label={$LL.tools.cloudModal.username()}
      placeholder={$LL.tools.cloudModal.usernamePlaceholder()}
      bind:value={cloudConfig.username}
    />

    <Input
      type="password"
      label={$LL.tools.cloudModal.passwordLabel()}
      placeholder={$LL.tools.cloudModal.passwordPlaceholder()}
      bind:value={cloudConfig.password}
    />

    <div class="space-y-1">
      <label for="interval-select" class="text-[10px] uppercase font-bold text-muted block">{$LL.tools.cloudModal.intervalLabel()}</label>
      <select
        id="interval-select"
        bind:value={cloudConfig.interval_hours}
        class="neo-input w-full bg-card-sub border border-border rounded px-3 py-2 text-xs font-mono text-foreground focus:outline-none focus:border-accent"
      >
        <option value={6}>{$LL.tools.cloudModal.interval6()}</option>
        <option value={12}>{$LL.tools.cloudModal.interval12()}</option>
        <option value={24}>{$LL.tools.cloudModal.interval24()}</option>
        <option value={48}>{$LL.tools.cloudModal.interval48()}</option>
      </select>
    </div>
  </div>

  {#snippet footer()}
    <Button
      variant="secondary"
      size="sm"
      onclick={() => (showCloudModal = false)}
    >
      {$LL.common.cancel()}
    </Button>
    <Button
      variant="primary"
      size="sm"
      disabled={isSavingCloud}
      onclick={saveCloudConfig}
    >
      {isSavingCloud ? $LL.tools.cloudModal.saving() : $LL.tools.cloudModal.saveSettings()}
    </Button>
  {/snippet}
</Modal>

<!-- Import Configuration Backup Modal -->
<Modal
  open={showImportModal}
  title={$LL.tools.importModal.title()}
  onclose={() => (showImportModal = false)}
  class="!max-w-lg"
>
  <div class="space-y-4 font-mono text-xs">
    <div class="p-3 rounded bg-amber-950/30 border border-amber-800 text-amber-300 flex items-start gap-2.5">
      <AlertCircle class="w-4 h-4 text-amber-400 shrink-0 mt-0.5" />
      <div class="space-y-1 leading-relaxed">
        <span class="font-bold block text-amber-300">{$LL.tools.importModal.warningTitle()}</span>
        <p class="text-[11px]">
          {$LL.tools.importModal.warningBody()}
        </p>
      </div>
    </div>

    <!-- File Picker Input -->
    <div class="space-y-1.5">
      <label for="backup-file" class="text-[10px] uppercase font-bold text-muted block">{$LL.tools.importModal.selectFile()}</label>
      <input
        id="backup-file"
        type="file"
        accept=".json"
        onchange={handleFileSelected}
        class="w-full bg-card-sub border border-border rounded p-2 text-xs text-foreground file:mr-3 file:py-1 file:px-2.5 file:rounded file:border file:border-border file:bg-card file:text-xs file:font-mono file:text-foreground hover:file:border-accent cursor-pointer"
      />
    </div>

    <!-- Raw JSON Preview / Paste Area -->
    <div class="space-y-1.5">
      <label for="backup-json-preview" class="text-[10px] uppercase font-bold text-muted block">{$LL.tools.importModal.pasteJson()}</label>
      <textarea
        id="backup-json-preview"
        bind:value={importJsonText}
        placeholder={$LL.tools.importModal.pastePlaceholder()}
        rows={6}
        class="neo-input w-full bg-card-sub border border-border rounded p-2.5 text-xs font-mono text-foreground placeholder:text-muted focus:outline-none focus:border-accent resize-none"
      ></textarea>
    </div>
  </div>

  {#snippet footer()}
    <Button
      variant="secondary"
      size="sm"
      onclick={() => (showImportModal = false)}
    >
      {$LL.common.cancel()}
    </Button>
    <Button
      variant="danger"
      size="sm"
      disabled={isImporting || !importJsonText.trim()}
      onclick={doImportBackup}
    >
      {isImporting ? $LL.tools.importModal.importing() : $LL.tools.importModal.restore()}
    </Button>
  {/snippet}
</Modal>

<ChangePasswordModal bind:open={showPasswordModal} />
