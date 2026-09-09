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
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Modal from '../../ui/Modal.svelte'
  import Input from '../../ui/Input.svelte'

  interface ModuleInfo {
    id: string
    name: string
    version: string
    author: string
    description: string
    enabled: boolean
  }

  interface CloudConfig {
    enabled: boolean
    provider: string
    url: string
    username: string
    password?: string
    interval_hours: number
    last_sync?: string
  }

  // Modules State
  let modulesList = $state<ModuleInfo[]>([])
  let isLoading = $state(false)
  let isToggling = $state<string | null>(null)

  // Backup & Cloud Sync State
  let showImportModal = $state(false)
  let showCloudModal = $state(false)
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
    await Promise.all([fetchModules(), fetchCloudConfig()])
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
        toastStore.success('Cloud backup settings saved successfully.')
        showCloudModal = false
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to save cloud config')
    } finally {
      isSavingCloud = false
    }
  }

  async function syncCloudBackup() {
    try {
      isSyncingCloud = true
      const res = await api.post<{ success: boolean; error?: string }>('/api/backup/cloud/sync')
      if (res?.success) {
        toastStore.success('Cloud synchronization succeeded!')
        await fetchCloudConfig()
      } else {
        toastStore.error(res?.error || 'Cloud sync failed')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Error triggering cloud sync')
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
      toastStore.success('Configuration backup exported.')
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
      toastStore.error('Please select a valid JSON backup file')
      return
    }
    try {
      isImporting = true
      const parsed = JSON.parse(importJsonText)
      const res = await api.post<{ success: boolean; error?: string }>('/api/backup/import', parsed)
      if (res?.success) {
        toastStore.success('Backup configuration imported successfully!')
        showImportModal = false
        importJsonText = ''
      } else {
        toastStore.error(res?.error || 'Failed to import backup payload')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Invalid JSON backup format')
    } finally {
      isImporting = false
    }
  }
</script>

<div class="space-y-6 font-mono">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between flex-wrap gap-3">
    <div class="flex items-center gap-2">
      <Wrench class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-bold uppercase tracking-wider text-foreground">
        System Tools & Backup
      </h2>
    </div>

    <div class="flex items-center gap-2 flex-wrap">
      <Button
        variant="outline"
        size="sm"
        onclick={exportBackup}
        title="Download full JSON configuration backup"
      >
        <Download class="w-3.5 h-3.5 mr-1" />
        <span>Export Backup</span>
      </Button>

      <Button
        variant="secondary"
        size="sm"
        onclick={() => (showImportModal = true)}
        title="Restore system configuration from JSON"
      >
        <Upload class="w-3.5 h-3.5 mr-1" />
        <span>Import</span>
      </Button>

      <Button
        variant="primary"
        size="sm"
        onclick={() => (showCloudModal = true)}
        title="Configure WebDAV Cloud Backup"
      >
        <Cloud class="w-3.5 h-3.5 mr-1" />
        <span>Cloud Sync</span>
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

  <!-- Backup & Cloud Synchronization Status Card -->
  <Card title="Configuration Backup & Cloud Synchronization">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 font-mono text-xs">
      <!-- Cloud State -->
      <div class="p-3.5 rounded bg-card-sub border border-border space-y-2 flex flex-col justify-between">
        <div>
          <div class="flex items-center justify-between mb-1">
            <span class="text-[10px] text-muted uppercase font-bold">Cloud Provider</span>
            <Badge variant={cloudConfig.enabled ? 'success' : 'default'}>
              {cloudConfig.enabled ? 'Active' : 'Disabled'}
            </Badge>
          </div>
          <div class="text-sm font-bold text-foreground capitalize">
            {cloudConfig.provider || 'WebDAV'}
          </div>
          <p class="text-[11px] text-muted truncate mt-0.5">
            {cloudConfig.url || 'No cloud endpoint configured'}
          </p>
        </div>

        <div class="pt-2 border-t border-border/60 flex items-center justify-between text-[10px]">
          <span class="text-muted">Last Synced:</span>
          <span class="font-bold text-foreground">
            {cloudConfig.last_sync ? new Date(cloudConfig.last_sync).toLocaleString() : 'Never'}
          </span>
        </div>
      </div>

      <!-- Quick Actions Strip -->
      <div class="p-3.5 rounded bg-card-sub border border-border space-y-2.5 flex flex-col justify-between">
        <div>
          <span class="text-[10px] text-muted uppercase font-bold block mb-1">Instant Cloud Operations</span>
          <p class="text-[11px] text-muted leading-relaxed">
            Push local configuration settings (network tweaks, DNS, TTL, shortcuts, charging limits) to remote cloud repository.
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
            <span>{isSyncingCloud ? 'Syncing...' : 'Sync Cloud Now'}</span>
          </Button>
        </div>
      </div>

      <!-- Local Backup Management -->
      <div class="p-3.5 rounded bg-card-sub border border-border space-y-2.5 flex flex-col justify-between">
        <div>
          <span class="text-[10px] text-muted uppercase font-bold block mb-1">Local Configuration Snapshot</span>
          <p class="text-[11px] text-muted leading-relaxed">
            Create or restore offline snapshots. Encodes full BFR daemon configuration into a single portable JSON file.
          </p>
        </div>

        <div class="flex items-center gap-2 pt-2 border-t border-border/60">
          <Button variant="outline" size="sm" class="flex-1" onclick={exportBackup}>
            <Download class="w-3.5 h-3.5 mr-1" />
            <span>Export</span>
          </Button>
          <Button variant="secondary" size="sm" class="flex-1" onclick={() => (showImportModal = true)}>
            <Upload class="w-3.5 h-3.5 mr-1" />
            <span>Restore</span>
          </Button>
        </div>
      </div>
    </div>
  </Card>

  <!-- Root Modules List -->
  <Card
    title="Installed Magisk / KernelSU Modules"
    subtitle="Manage systemless root extensions located in /data/adb/modules"
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
      <div class="space-y-3 font-mono">
        {#each modulesList as mod}
          <div class="bg-card-sub border border-border p-4 rounded flex flex-col sm:flex-row sm:items-center justify-between gap-4 hover:border-accent transition-colors">
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

<!-- Cloud Sync Configuration Modal -->
<Modal
  open={showCloudModal}
  title="Cloud Backup & WebDAV Synchronization"
  onclose={() => (showCloudModal = false)}
  class="!max-w-md"
>
  <div class="space-y-3 font-mono text-xs">
    <div class="flex items-center justify-between p-3 rounded bg-card-sub border border-border">
      <div>
        <span class="font-bold text-foreground block text-xs">Enable Automatic Cloud Sync</span>
        <span class="text-[10px] text-muted">Scheduled remote backup sync</span>
      </div>
      <input
        type="checkbox"
        bind:checked={cloudConfig.enabled}
        class="w-4 h-4 accent-accent cursor-pointer"
      />
    </div>

    <div class="space-y-1">
      <label for="cloud-provider" class="text-[10px] uppercase font-bold text-muted block">Provider Protocol</label>
      <select
        id="cloud-provider"
        bind:value={cloudConfig.provider}
        class="neo-input w-full bg-card-sub border border-border rounded px-3 py-2 text-xs font-mono text-foreground focus:outline-none focus:border-accent"
      >
        <option value="webdav">WebDAV (Nextcloud, ownCloud, Fastmail, Box)</option>
      </select>
    </div>

    <Input
      label="WebDAV Server Endpoint URL"
      placeholder="https://dav.example.com/remote.php/webdav/backups/"
      bind:value={cloudConfig.url}
    />

    <Input
      label="Username"
      placeholder="e.g. admin or service account"
      bind:value={cloudConfig.username}
    />

    <Input
      type="password"
      label="Password / App Token"
      placeholder="Enter WebDAV token or password"
      bind:value={cloudConfig.password}
    />

    <div class="space-y-1">
      <label for="interval-select" class="text-[10px] uppercase font-bold text-muted block">Sync Interval</label>
      <select
        id="interval-select"
        bind:value={cloudConfig.interval_hours}
        class="neo-input w-full bg-card-sub border border-border rounded px-3 py-2 text-xs font-mono text-foreground focus:outline-none focus:border-accent"
      >
        <option value={6}>Every 6 Hours</option>
        <option value={12}>Every 12 Hours</option>
        <option value={24}>Every 24 Hours (Daily)</option>
        <option value={48}>Every 48 Hours (2 Days)</option>
      </select>
    </div>
  </div>

  {#snippet footer()}
    <Button
      variant="secondary"
      size="sm"
      onclick={() => (showCloudModal = false)}
    >
      Cancel
    </Button>
    <Button
      variant="primary"
      size="sm"
      disabled={isSavingCloud}
      onclick={saveCloudConfig}
    >
      {isSavingCloud ? 'Saving...' : 'Save Settings'}
    </Button>
  {/snippet}
</Modal>

<!-- Import Configuration Backup Modal -->
<Modal
  open={showImportModal}
  title="Restore Configuration from Backup"
  onclose={() => (showImportModal = false)}
  class="!max-w-lg"
>
  <div class="space-y-4 font-mono text-xs">
    <div class="p-3 rounded bg-amber-950/30 border border-amber-800 text-amber-300 flex items-start gap-2.5">
      <AlertCircle class="w-4 h-4 text-amber-400 shrink-0 mt-0.5" />
      <div class="space-y-1 leading-relaxed">
        <span class="font-bold block text-amber-300">Warning: Configuration Overwrite</span>
        <p class="text-[11px]">
          Importing a backup will replace current system parameters, network tweaks, DNS, TTL, and shortcuts.
        </p>
      </div>
    </div>

    <!-- File Picker Input -->
    <div class="space-y-1.5">
      <label for="backup-file" class="text-[10px] uppercase font-bold text-muted block">Select Backup JSON File</label>
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
      <label for="backup-json-preview" class="text-[10px] uppercase font-bold text-muted block">Or Paste JSON Payload</label>
      <textarea
        id="backup-json-preview"
        bind:value={importJsonText}
        placeholder="Paste JSON configuration payload here..."
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
      Cancel
    </Button>
    <Button
      variant="danger"
      size="sm"
      disabled={isImporting || !importJsonText.trim()}
      onclick={doImportBackup}
    >
      {isImporting ? 'Importing...' : 'Restore Configuration'}
    </Button>
  {/snippet}
</Modal>
