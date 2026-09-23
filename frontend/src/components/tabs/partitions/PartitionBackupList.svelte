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
    AlertTriangle,
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
    onDeleteMultiple: (filenames: string[]) => void
    onDeleteAll: () => void
  }

  let {
    backups,
    isLoading,
    onRefresh,
    onDeleteBackup,
    onDeleteMultiple,
    onDeleteAll,
  }: Props = $props()

  let copiedHash = $state<string | null>(null)
  let searchQuery = $state('')

  // Multi-selection state
  let selectedFilenames = $state<Set<string>>(new Set())

  // Modal State
  let isConfirmModalOpen = $state(false)
  let deleteMode = $state<'single' | 'selected' | 'all'>('single')
  let targetSingleBackup = $state<BackupFile | null>(null)
  let isDeleting = $state(false)

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

  const selectedFiles = $derived(
    backups.filter((b) => selectedFilenames.has(b.filename))
  )

  const selectedTotalSizeBytes = $derived(
    selectedFiles.reduce((acc, curr) => acc + (curr.size_bytes || 0), 0)
  )

  const allTotalSizeBytes = $derived(
    backups.reduce((acc, curr) => acc + (curr.size_bytes || 0), 0)
  )

  const isAllFilteredSelected = $derived(
    filteredBackups.length > 0 &&
      filteredBackups.every((b) => selectedFilenames.has(b.filename))
  )

  function toggleSelect(filename: string) {
    const next = new Set(selectedFilenames)
    if (next.has(filename)) {
      next.delete(filename)
    } else {
      next.add(filename)
    }
    selectedFilenames = next
  }

  function toggleSelectAll() {
    if (isAllFilteredSelected) {
      selectedFilenames = new Set()
    } else {
      selectedFilenames = new Set(filteredBackups.map((b) => b.filename))
    }
  }

  function clearSelection() {
    selectedFilenames = new Set()
  }

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

  function formatBytes(bytes: number): string {
    if (bytes <= 0) return '0 B'
    const units = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(1024))
    return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
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

  // Modal Openers
  function openDeleteSingleModal(file: BackupFile) {
    targetSingleBackup = file
    deleteMode = 'single'
    isConfirmModalOpen = true
  }

  function openDeleteSelectedModal() {
    if (selectedFilenames.size === 0) return
    deleteMode = 'selected'
    isConfirmModalOpen = true
  }

  function openDeleteAllModal() {
    if (backups.length === 0) return
    deleteMode = 'all'
    isConfirmModalOpen = true
  }

  function closeConfirmModal() {
    if (isDeleting) return
    isConfirmModalOpen = false
    targetSingleBackup = null
  }

  async function confirmDeletion() {
    isDeleting = true
    try {
      if (deleteMode === 'single' && targetSingleBackup) {
        onDeleteBackup(targetSingleBackup.filename)
        if (selectedFilenames.has(targetSingleBackup.filename)) {
          const next = new Set(selectedFilenames)
          next.delete(targetSingleBackup.filename)
          selectedFilenames = next
        }
      } else if (deleteMode === 'selected') {
        const toDelete = Array.from(selectedFilenames)
        onDeleteMultiple(toDelete)
        selectedFilenames = new Set()
      } else if (deleteMode === 'all') {
        onDeleteAll()
        selectedFilenames = new Set()
      }
      closeConfirmModal()
    } finally {
      isDeleting = false
    }
  }
</script>

<div class="space-y-4">
  <Card
    title="SAVED BACKUP ARCHIVES"
    subtitle="Partition images stored on module internal storage (/data/adb/bfr/backups)"
    tone="ice"
  >
    {#snippet action()}
      <div class="flex items-center gap-2 flex-wrap justify-end">
        <!-- Search Input -->
        <div class="relative w-40 sm:w-52">
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

        <!-- Refresh Button -->
        <Button
          variant="outline"
          size="sm"
          disabled={isLoading}
          onclick={onRefresh}
          title="Refresh backup archives"
        >
          <RefreshCw size={13} class={isLoading ? 'animate-spin' : ''} />
        </Button>

        <!-- Delete All Button -->
        <Button
          variant="outline"
          size="sm"
          disabled={isLoading || backups.length === 0}
          onclick={openDeleteAllModal}
          class="border-red-900/60 hover:bg-red-950/30 text-red-400 font-mono text-xs"
          title="Delete all backup archives stored on device"
        >
          <Trash2 size={13} class="mr-1" />
          <span>Delete All</span>
        </Button>
      </div>
    {/snippet}

    <!-- Multi-Selection Action Bar -->
    {#if selectedFilenames.size > 0}
      <div class="mb-4 bg-card-sub border-2 border-border p-2.5 rounded-lg flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-2 shadow-neobrutal">
        <div class="flex items-center gap-2 text-xs font-mono">
          <span class="px-2 py-0.5 rounded bg-accent text-accent-text font-bold">
            {selectedFilenames.size}
          </span>
          <span class="text-foreground font-bold">archive(s) selected</span>
          <span class="text-muted">({formatBytes(selectedTotalSizeBytes)})</span>
        </div>

        <div class="flex items-center gap-2 justify-end">
          <Button
            variant="ghost"
            size="sm"
            onclick={clearSelection}
            class="text-xs"
          >
            Clear Selection
          </Button>

          <Button
            variant="danger"
            size="sm"
            onclick={openDeleteSelectedModal}
            class="text-xs"
          >
            <Trash2 size={13} class="mr-1.5" />
            <span>Delete Selected ({selectedFilenames.size})</span>
          </Button>
        </div>
      </div>
    {/if}

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
              <!-- Checkbox Master Column -->
              <th class="p-3 w-10 text-center">
                <input
                  type="checkbox"
                  checked={isAllFilteredSelected}
                  onchange={toggleSelectAll}
                  class="rounded bg-card border-border text-accent focus:ring-accent w-4 h-4 cursor-pointer"
                  title="Select / deselect all filtered backups"
                />
              </th>
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
              <tr class="hover:bg-card-sub/50 transition-colors {selectedFilenames.has(file.filename) ? 'bg-card-sub/80' : ''}">
                <!-- Row Selection Checkbox -->
                <td class="p-3 text-center">
                  <input
                    type="checkbox"
                    checked={selectedFilenames.has(file.filename)}
                    onchange={() => toggleSelect(file.filename)}
                    class="rounded bg-card border-border text-accent focus:ring-accent w-4 h-4 cursor-pointer"
                    title="Select {file.filename}"
                  />
                </td>

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
                  {#if file.filename.toLowerCase().endsWith('.zip')}
                    <span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-mono font-bold uppercase bg-amber-950/60 text-amber-400 border border-amber-800">
                      ZIP (.ZIP)
                    </span>
                  {:else if file.is_compressed}
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
                      onclick={() => openDeleteSingleModal(file)}
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
</div>

<!-- Reusable Safety Deletion Confirmation Modal -->
<Modal
  open={isConfirmModalOpen}
  title={deleteMode === 'all'
    ? 'DELETE ALL BACKUP ARCHIVES'
    : deleteMode === 'selected'
      ? 'DELETE SELECTED ARCHIVES'
      : 'DELETE BACKUP ARCHIVE'}
  onclose={closeConfirmModal}
>
  <div class="space-y-3.5 text-xs font-mono">
    <!-- Danger Warning Box -->
    <div class="p-3.5 bg-red-950/40 border-2 border-red-600 rounded text-red-200 space-y-2">
      <div class="flex items-center gap-2 text-red-400 font-bold text-sm uppercase">
        <AlertTriangle size={18} class="shrink-0 text-red-500 animate-pulse" />
        <span>Permanent Deletion Warning</span>
      </div>

      {#if deleteMode === 'all'}
        <p class="leading-relaxed">
          Permanently delete <strong class="text-white underline">ALL {backups.length}</strong> backup archives (<span class="font-bold text-white">{formatBytes(allTotalSizeBytes)}</span>)? This will erase all cellular/IMEI and boot backups stored on this device. This operation <strong class="text-white underline">cannot be undone</strong>!
        </p>
      {:else if deleteMode === 'selected'}
        <p class="leading-relaxed">
          Permanently delete <strong class="text-white">{selectedFilenames.size}</strong> selected backup archive(s) (<span class="font-bold text-white">{formatBytes(selectedTotalSizeBytes)}</span>)? The raw images and sidecar checksums will be removed permanently from disk.
        </p>
      {:else}
        <p class="leading-relaxed">
          Are you sure you want to permanently delete backup archive <strong class="text-white">{targetSingleBackup?.filename}</strong>? This action cannot be undone.
        </p>
      {/if}
    </div>

    <!-- Targeted Files Information -->
    {#if deleteMode === 'single' && targetSingleBackup}
      <div class="p-3 bg-card-sub border border-border rounded space-y-1 text-foreground">
        <div>
          <span class="text-muted">Target Partition:</span>
          <span class="font-bold ml-1">{targetSingleBackup.partition_name}</span>
        </div>
        <div>
          <span class="text-muted">Archive Size:</span>
          <span class="font-bold ml-1">{targetSingleBackup.size_human}</span>
        </div>
        <div class="truncate">
          <span class="text-muted">File:</span>
          <span class="font-bold ml-1">{targetSingleBackup.filename}</span>
        </div>
      </div>
    {:else if deleteMode === 'selected'}
      <div class="space-y-1">
        <div class="text-[10px] text-muted uppercase font-bold tracking-wider">
          Archives to be deleted ({selectedFilenames.size}):
        </div>
        <div class="max-h-36 overflow-y-auto custom-scrollbar p-2 bg-card-sub border border-border rounded text-[11px] text-muted space-y-1">
          {#each selectedFiles as f}
            <div class="flex items-center justify-between gap-2 truncate">
              <span class="text-foreground truncate">• {f.filename}</span>
              <span class="shrink-0 font-bold">{f.size_human}</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  </div>

  {#snippet footer()}
    <div class="flex items-center justify-end gap-2 w-full">
      <Button
        variant="ghost"
        disabled={isDeleting}
        onclick={closeConfirmModal}
      >
        Cancel
      </Button>

      <Button
        variant="danger"
        disabled={isDeleting}
        onclick={confirmDeletion}
      >
        {#if isDeleting}
          <RefreshCw size={14} class="animate-spin mr-1.5" />
          <span>Deleting...</span>
        {:else}
          <Trash2 size={14} class="mr-1.5" />
          <span>
            {deleteMode === 'all'
              ? 'Delete All Permanently'
              : deleteMode === 'selected'
                ? `Delete ${selectedFilenames.size} Archives`
                : 'Delete Permanently'}
          </span>
        {/if}
      </Button>
    </div>
  {/snippet}
</Modal>
