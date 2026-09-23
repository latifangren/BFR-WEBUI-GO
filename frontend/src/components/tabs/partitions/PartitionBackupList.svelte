<script lang="ts">
  import {
    FileArchive,
    Download,
    Trash2,
    Copy,
    Check,
    RefreshCw,
    Search,
    X,
    HardDriveDownload,
  } from '@lucide/svelte'
  import type { BackupFile } from '../../../types/partition'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Modal from '../../ui/Modal.svelte'
  import { toastStore } from '../../../stores/toast.svelte'

  interface Props {
    backups: BackupFile[]
    isLoading: boolean
    onRefresh: () => void
    onDeleteBackup: (filename: string) => void
  }

  let {
    backups,
    isLoading,
    onRefresh,
    onDeleteBackup,
  }: Props = $props()

  let copiedHash = $state<string | null>(null)
  let searchQuery = $state('')
  let backupToDelete = $state<BackupFile | null>(null)
  let isDeleteModalOpen = $state(false)

  const filteredBackups = $derived(
    backups.filter((b) => {
      if (!searchQuery.trim()) return true
      const q = searchQuery.toLowerCase().trim()
      return (
        b.filename.toLowerCase().includes(q) ||
        b.partition_name.toLowerCase().includes(q) ||
        b.sha256.toLowerCase().includes(q)
      )
    })
  )

  async function copyToClipboard(text: string, id: string) {
    try {
      await navigator.clipboard.writeText(text)
      copiedHash = id
      toastStore.info('SHA256 checksum copied to clipboard')
      setTimeout(() => {
        if (copiedHash === id) copiedHash = null
      }, 2500)
    } catch {
      toastStore.error('Failed to copy hash to clipboard')
    }
  }

  function formatDate(dateStr: string): string {
    try {
      const d = new Date(dateStr)
      return d.toLocaleString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      })
    } catch {
      return dateStr
    }
  }

  function openDeleteModal(file: BackupFile) {
    backupToDelete = file
    isDeleteModalOpen = true
  }

  function closeDeleteModal() {
    isDeleteModalOpen = false
    backupToDelete = null
  }

  function confirmDelete() {
    if (backupToDelete) {
      onDeleteBackup(backupToDelete.filename)
      closeDeleteModal()
    }
  }
</script>

<Card
  title="SAVED BACKUP ARCHIVES"
  subtitle="Partition images stored on module internal storage (/data/adb/bfr/backups)"
  tone="ice"
>
  {#snippet action()}
    <div class="flex items-center gap-2">
      <!-- Search Input -->
      <div class="relative w-48 sm:w-56">
        <div class="absolute inset-y-0 left-0 pl-2.5 flex items-center pointer-events-none text-muted">
          <Search size={13} />
        </div>
        <input
          type="text"
          bind:value={searchQuery}
          placeholder="Filter archives..."
          class="neo-input w-full bg-card-sub border border-border rounded pl-8 pr-7 py-1 text-xs font-mono text-foreground placeholder:text-muted focus:outline-none focus:border-accent"
        />
        {#if searchQuery}
          <button
            type="button"
            onclick={() => (searchQuery = '')}
            class="absolute inset-y-0 right-0 pr-2 flex items-center text-muted hover:text-foreground cursor-pointer"
          >
            <X size={12} />
          </button>
        {/if}
      </div>

      <Button
        variant="outline"
        size="sm"
        disabled={isLoading}
        onclick={onRefresh}
        title="Refresh backup archives"
      >
        <RefreshCw size={13} class={isLoading ? 'animate-spin' : ''} />
      </Button>
    </div>
  {/snippet}

  {#if isLoading && backups.length === 0}
    <div class="p-8 text-center text-muted font-mono text-sm flex items-center justify-center gap-2">
      <RefreshCw size={16} class="animate-spin" />
      <span>Loading saved backups...</span>
    </div>
  {:else if filteredBackups.length === 0}
    <div class="p-8 text-center border-2 border-dashed border-border rounded bg-card-sub">
      <HardDriveDownload size={32} class="mx-auto text-muted mb-2 opacity-50" />
      <p class="font-mono text-sm text-foreground font-bold">No Backup Archives Found</p>
      <p class="text-xs text-muted mt-1">
        Create partition backups using the Quick Actions above or the Partition Table backup buttons.
      </p>
    </div>
  {:else}
    <div class="overflow-x-auto border border-border rounded custom-scrollbar">
      <table class="w-full text-left border-collapse text-xs font-mono">
        <thead>
          <tr class="bg-card-sub border-b border-border text-muted uppercase text-[10px] tracking-wider">
            <th class="p-3">File / Target Partition</th>
            <th class="p-3">Size</th>
            <th class="p-3">Format</th>
            <th class="p-3">Created At</th>
            <th class="p-3">SHA256 Checksum</th>
            <th class="p-3 text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          {#each filteredBackups as file (file.filename)}
            <tr class="hover:bg-card-sub/50 transition-colors">
              <!-- File Name & Target -->
              <td class="p-3">
                <div class="flex items-center gap-2">
                  <FileArchive size={16} class="text-accent shrink-0" />
                  <div>
                    <div class="font-bold text-foreground text-xs break-all">
                      {file.filename}
                    </div>
                    <div class="text-[10px] text-muted flex items-center gap-1">
                      <span>Target:</span>
                      <span class="font-bold text-foreground">{file.partition_name}</span>
                    </div>
                  </div>
                </div>
              </td>

              <!-- Size -->
              <td class="p-3 whitespace-nowrap text-foreground font-bold">
                {file.size_human}
              </td>

              <!-- Format Badge -->
              <td class="p-3 whitespace-nowrap">
                {#if file.is_compressed}
                  <span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-mono font-bold uppercase bg-emerald-950/60 text-emerald-400 border border-emerald-800">
                    GZIP (.GZ)
                  </span>
                {:else}
                  <span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-mono font-bold uppercase bg-card-sub text-muted border border-border">
                    RAW (.IMG)
                  </span>
                {/if}
              </td>

              <!-- Date -->
              <td class="p-3 whitespace-nowrap text-muted text-[11px]">
                {formatDate(file.created_at)}
              </td>

              <!-- Checksum -->
              <td class="p-3 whitespace-nowrap">
                {#if file.sha256}
                  <button
                    type="button"
                    onclick={() => copyToClipboard(file.sha256, file.filename)}
                    title="Click to copy full SHA256 checksum"
                    class="inline-flex items-center gap-1.5 px-2 py-1 bg-card-sub hover:bg-card border border-border rounded text-[10px] text-muted hover:text-foreground font-mono transition-colors cursor-pointer"
                  >
                    {#if copiedHash === file.filename}
                      <Check size={11} class="text-emerald-400" />
                      <span class="text-emerald-400 font-bold">Copied!</span>
                    {:else}
                      <Copy size={11} />
                      <span>{file.sha256.substring(0, 10)}...</span>
                    {/if}
                  </button>
                {:else}
                  <span class="text-muted text-[10px]">-</span>
                {/if}
              </td>

              <!-- Actions -->
              <td class="p-3 whitespace-nowrap text-right">
                <div class="inline-flex items-center gap-1.5 justify-end">
                  <a
                    href="/api/partitions/download?filename={encodeURIComponent(file.filename)}"
                    download={file.filename}
                    class="neo-button inline-flex items-center justify-center font-mono font-bold px-2.5 py-1 text-xs bg-card text-foreground hover:bg-card-sub border-border border rounded cursor-pointer"
                    title="Download archive to your device"
                  >
                    <Download size={13} class="mr-1" />
                    <span>Download</span>
                  </a>

                  <Button
                    variant="danger"
                    size="sm"
                    onclick={() => openDeleteModal(file)}
                    title="Delete backup archive"
                  >
                    <Trash2 size={13} />
                  </Button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</Card>

<!-- Delete Confirmation Modal -->
<Modal
  open={isDeleteModalOpen}
  title="DELETE BACKUP ARCHIVE"
  onclose={closeDeleteModal}
>
  <div class="space-y-3 text-xs font-mono">
    <div class="p-3 bg-red-950/30 border border-red-700/60 rounded text-red-200">
      <p>
        Are you sure you want to permanently delete <strong class="text-white font-bold">{backupToDelete?.filename}</strong>?
      </p>
      <p class="text-[11px] text-muted mt-1">
        This action cannot be undone. The raw image file and its sidecar SHA256 checksum will be removed from disk.
      </p>
    </div>
  </div>

  {#snippet footer()}
    <div class="flex items-center justify-end gap-2 w-full">
      <Button variant="ghost" onclick={closeDeleteModal}>
        Cancel
      </Button>
      <Button variant="danger" onclick={confirmDelete}>
        <Trash2 size={14} class="mr-1.5" />
        <span>Delete Permanently</span>
      </Button>
    </div>
  {/snippet}
</Modal>
