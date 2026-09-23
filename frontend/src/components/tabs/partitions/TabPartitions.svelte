<script lang="ts">
  import { onMount } from 'svelte'
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
  let compress = $state<boolean>(true)
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
        compress,
      })

      if (res && Array.isArray(res.results)) {
        const successes = res.results.filter((r) => r.success)
        const fails = res.results.filter((r) => !r.success)
        const groupLabel = group === 'radio_efs' ? 'Radio & EFS' : 'Boot & Kernel Core'

        if (fails.length === 0) {
          toastStore.success(`All ${successes.length} ${groupLabel} partitions backed up successfully.`)
        } else {
          toastStore.warning(`Partial backup: ${successes.length} succeeded, ${fails.length} failed.`)
        }
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
        compress,
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
    {compress}
    onToggleCompress={(val) => (compress = val)}
    onBackupGroup={handleBackupGroup}
  />

  {#if activeFilter === 'backups'}
    <PartitionBackupList
      {backups}
      isLoading={isLoadingBackups}
      onRefresh={fetchBackups}
      onDeleteBackup={handleDeleteBackup}
    />
  {:else}
    <PartitionTable
      {partitions}
      {activeFilter}
      {searchQuery}
      {isBackingUp}
      backupCount={backups.length}
      onSelectFilter={(f) => (activeFilter = f)}
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
