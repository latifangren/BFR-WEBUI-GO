<script lang="ts">
  import { onMount } from 'svelte'
  import { HardDriveDownload } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import type {
    PartitionInfo,
    SystemSlotInfo,
    BackupFile,
    PartitionListResponse,
    BackupListResponse,
    BackupGroupResponse,
    BackupResult,
    FlashResult,
  } from '../../../types/partition'
  import PartitionHeader from './PartitionHeader.svelte'
  import PartitionQuickActions from './PartitionQuickActions.svelte'
  import PartitionTable from './PartitionTable.svelte'
  import PartitionBackupList from './PartitionBackupList.svelte'
  import PartitionFlashModal from './PartitionFlashModal.svelte'

  // Primary State
  let partitions = $state<PartitionInfo[]>([])
  let slotInfo = $state<SystemSlotInfo | null>(null)
  let backups = $state<BackupFile[]>([])
  let isLoadingPartitions = $state(false)
  let isLoadingBackups = $state(false)

  // Filters & Actions
  let activeFilter = $state<string>('all')
  let searchQuery = $state<string>('')
  let autoDownload = $state<boolean>(true)
  let isBackingUp = $state<string | null>(null)

  // Flashing Modal
  let isFlashModalOpen = $state<boolean>(false)
  let flashTarget = $state<PartitionInfo | null>(null)
  let isFlashing = $state<boolean>(false)

  onMount(async () => {
    await Promise.all([fetchPartitions(), fetchBackups()])
  })

  async function fetchPartitions() {
    try {
      isLoadingPartitions = true
      const res = await api.get<PartitionListResponse>('/api/partitions')
      if (res) {
        partitions = res.partitions || []
        slotInfo = res.slot_info || null
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to load partitions')
    } finally {
      isLoadingPartitions = false
    }
  }

  async function fetchBackups() {
    try {
      isLoadingBackups = true
      const res = await api.get<BackupListResponse>('/api/partitions/backups')
      if (res && Array.isArray(res.backups)) {
        backups = res.backups
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to load backup archives')
    } finally {
      isLoadingBackups = false
    }
  }

  async function handleRefreshAll() {
    await Promise.all([fetchPartitions(), fetchBackups()])
    toastStore.info('Partitions and backup archives refreshed')
  }

  async function handleBackupGroup(group: 'radio_efs' | 'boot_core') {
    try {
      isBackingUp = group
      const res = await api.post<BackupGroupResponse>('/api/partitions/backup-group', {
        group,
        compress: true,
      })

      const groupLabel = group === 'radio_efs' ? 'Radio & EFS' : 'Boot & Kernel Core'

      if (res && res.success && res.filename) {
        toastStore.success(`${groupLabel} packaged into ${res.filename} (${res.size_human})`)

        if (autoDownload && typeof document !== 'undefined') {
          const downloadUrl = `/api/partitions/download?filename=${encodeURIComponent(res.filename)}`
          const link = document.createElement('a')
          link.href = downloadUrl
          link.download = res.filename
          document.body.appendChild(link)
          link.click()
          document.body.removeChild(link)
        }
      } else {
        toastStore.error(res?.error || `Failed to create ${groupLabel} backup archive`)
      }

      await fetchBackups()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Group backup operation failed')
    } finally {
      isBackingUp = null
    }
  }

  async function handleBackupPartition(part: PartitionInfo) {
    try {
      isBackingUp = part.name
      const res = await api.post<BackupResult>('/api/partitions/backup', {
        partition: part.name,
        compress: true,
      })

      if (res && res.success) {
        toastStore.success(`Partition ${part.name} backed up successfully (${res.filename})`)
      } else {
        toastStore.error(res?.error || `Failed to backup partition ${part.name}`)
      }
      await fetchBackups()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : `Failed to backup partition ${part.name}`)
    } finally {
      isBackingUp = null
    }
  }

  function handleOpenFlashModal(part: PartitionInfo) {
    flashTarget = part
    isFlashModalOpen = true
  }

  function handleCloseFlashModal() {
    isFlashModalOpen = false
    flashTarget = null
  }

  async function handleConfirmFlash(file: File, autoBackup: boolean) {
    if (!flashTarget) return
    try {
      isFlashing = true
      const formData = new FormData()
      formData.append('partition', flashTarget.name)
      formData.append('safety_ack', 'I_UNDERSTAND_THE_RISKS')
      formData.append('auto_backup', autoBackup ? 'true' : 'false')
      formData.append('file', file)

      const res = await api.post<FlashResult>('/api/partitions/flash', formData)
      if (res && res.success) {
        toastStore.success(res.message || `Partition ${res.partition_name} flashed successfully!`)
        if (res.backup_file) {
          toastStore.info(`Auto-backup saved: ${res.backup_file}`)
        }
        handleCloseFlashModal()
        await fetchBackups()
      } else {
        toastStore.error(res?.error || 'Flashing process failed')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Flashing operation encountered an error')
    } finally {
      isFlashing = false
    }
  }

  async function handleDeleteBackup(filename: string) {
    try {
      await api.post('/api/partitions/backup/delete', { filename })
      toastStore.success(`Backup archive ${filename} deleted successfully`)
      await fetchBackups()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to delete backup archive')
    }
  }

  async function handleDeleteMultiple(filenames: string[]) {
    try {
      await api.post('/api/partitions/backup/delete', { filenames })
      toastStore.success(`Successfully deleted ${filenames.length} backup archive(s)`)
      await fetchBackups()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to delete selected archives')
    }
  }

  async function handleDeleteAll() {
    try {
      await api.post('/api/partitions/backup/delete', { all: true })
      toastStore.success('All backup archives deleted successfully')
      await fetchBackups()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to delete all archives')
    }
  }
</script>

<div class="space-y-6">
  <PartitionHeader
    {slotInfo}
    totalPartitions={partitions.length}
    backupCount={backups.length}
    isLoading={isLoadingPartitions || isLoadingBackups}
    onRefresh={handleRefreshAll}
  />

  <PartitionQuickActions
    {isBackingUp}
    {autoDownload}
    onToggleAutoDownload={(val) => (autoDownload = val)}
    onBackupGroup={handleBackupGroup}
  />

  <!-- Category Navigation Bar (Always Visible) -->
  <div class="bg-card border-2 border-border rounded-lg shadow-neobrutal p-1.5 flex items-center overflow-x-auto custom-scrollbar gap-1">
    <button
      type="button"
      onclick={() => (activeFilter = 'all')}
      class="px-3.5 py-1.5 text-xs font-mono font-bold uppercase tracking-wider rounded select-none cursor-pointer transition-colors whitespace-nowrap {activeFilter === 'all' ? 'bg-foreground text-background shadow-neobrutal' : 'text-muted hover:text-foreground'}"
    >
      All ({partitions.length})
    </button>

    <button
      type="button"
      onclick={() => (activeFilter = 'radio_efs')}
      class="px-3.5 py-1.5 text-xs font-mono font-bold uppercase tracking-wider rounded select-none cursor-pointer transition-colors whitespace-nowrap {activeFilter === 'radio_efs' ? 'bg-foreground text-background shadow-neobrutal' : 'text-muted hover:text-foreground'}"
    >
      Radio & EFS
    </button>

    <button
      type="button"
      onclick={() => (activeFilter = 'boot_recovery')}
      class="px-3.5 py-1.5 text-xs font-mono font-bold uppercase tracking-wider rounded select-none cursor-pointer transition-colors whitespace-nowrap {activeFilter === 'boot_recovery' ? 'bg-foreground text-background shadow-neobrutal' : 'text-muted hover:text-foreground'}"
    >
      Boot & Recovery
    </button>

    <button
      type="button"
      onclick={() => (activeFilter = 'system_os')}
      class="px-3.5 py-1.5 text-xs font-mono font-bold uppercase tracking-wider rounded select-none cursor-pointer transition-colors whitespace-nowrap {activeFilter === 'system_os' ? 'bg-foreground text-background shadow-neobrutal' : 'text-muted hover:text-foreground'}"
    >
      System OS
    </button>

    <button
      type="button"
      onclick={() => (activeFilter = 'backups')}
      class="px-3.5 py-1.5 text-xs font-mono font-bold uppercase tracking-wider rounded select-none cursor-pointer transition-colors whitespace-nowrap flex items-center gap-1.5 {activeFilter === 'backups' ? 'bg-foreground text-background shadow-neobrutal' : 'text-muted hover:text-foreground'}"
    >
      <HardDriveDownload size={13} />
      <span>Saved Backups ({backups.length})</span>
    </button>
  </div>

  {#if activeFilter === 'backups'}
    <PartitionBackupList
      {backups}
      isLoading={isLoadingBackups}
      onRefresh={fetchBackups}
      onDeleteBackup={handleDeleteBackup}
      onDeleteMultiple={handleDeleteMultiple}
      onDeleteAll={handleDeleteAll}
    />
  {:else}
    <PartitionTable
      {partitions}
      {activeFilter}
      {searchQuery}
      {isBackingUp}
      onSearchChange={(q) => (searchQuery = q)}
      onBackupPartition={handleBackupPartition}
      onOpenFlashModal={handleOpenFlashModal}
    />
  {/if}
</div>

<PartitionFlashModal
  isOpen={isFlashModalOpen}
  partition={flashTarget}
  {isFlashing}
  onClose={handleCloseFlashModal}
  onConfirmFlash={handleConfirmFlash}
/>
