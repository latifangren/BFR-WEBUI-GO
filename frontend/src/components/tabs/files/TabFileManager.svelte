<script lang="ts">
  import { onMount } from 'svelte'
  import {
    FolderOpen,
    Folder,
    FileText,
    RefreshCw,
    ArrowUp,
    Upload,
    Plus,
    Download,
    Edit2,
    Trash2,
    Save,
    X,
    File,
    Search,
    HardDrive,
    Shield,
    FileArchive,
    Copy,
    FolderInput,
    Check,
    AlertTriangle,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import type {
    FileEntry,
    FileListResponse,
    FileReadResponse,
    FileStorageInfo,
  } from '../../../types/files'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Input from '../../ui/Input.svelte'
  import Modal from '../../ui/Modal.svelte'

  let currentPath = $state('/sdcard')
  let files = $state<FileEntry[]>([])
  let isLoading = $state(false)
  let storageInfo = $state<FileStorageInfo | null>(null)

  // Search State
  let searchQuery = $state('')
  let isSearching = $state(false)
  let searchResults = $state<FileEntry[] | null>(null)

  // Editor Modal
  let editorOpen = $state(false)
  let editorFilePath = $state('')
  let editorContent = $state('')
  let isSavingFile = $state(false)

  // Create Modal
  let createModalOpen = $state(false)
  let createType = $state<'file' | 'folder'>('file')
  let createName = $state('')

  // Rename Modal
  let renameModalOpen = $state(false)
  let renameTarget = $state<FileEntry | null>(null)
  let newFileName = $state('')

  // Delete Modal
  let deleteModalOpen = $state(false)
  let deleteTarget = $state<FileEntry | null>(null)

  // Permissions Modal (Chmod)
  let permModalOpen = $state(false)
  let permTarget = $state<FileEntry | null>(null)
  let permMode = $state('0755')

  // Compress Modal (ZIP)
  let compressModalOpen = $state(false)
  let compressTarget = $state<FileEntry | null>(null)
  let compressDestName = $state('')

  // Extract Modal (Unzip)
  let extractModalOpen = $state(false)
  let extractTarget = $state<FileEntry | null>(null)
  let extractDestDir = $state('')

  // Copy & Move Modal
  let copyMoveModalOpen = $state(false)
  let copyMoveTarget = $state<FileEntry | null>(null)
  let copyMoveAction = $state<'copy' | 'move'>('copy')
  let copyMoveDest = $state('')

  // File Upload
  let uploadInputEl: HTMLInputElement | null = $state(null)
  let isUploading = $state(false)

  // Multi-Select & Batch Toolbar State
  let selectedPaths = $state<string[]>([])
  let batchDeleteModalOpen = $state(false)
  let batchCopyModalOpen = $state(false)
  let copyDest = $state('')
  let batchMoveModalOpen = $state(false)
  let moveDest = $state('')
  let batchCompressModalOpen = $state(false)
  let batchCompressName = $state('')
  let isBatchProcessing = $state(false)

  let isCopyDestSameDir = $derived(
    copyDest.trim().replace(/\/+$/, '') === currentPath.replace(/\/+$/, '')
  )
  let isMoveDestSameDir = $derived(
    moveDest.trim().replace(/\/+$/, '') === currentPath.replace(/\/+$/, '')
  )

  let displayedFiles = $derived(
    searchResults !== null ? searchResults : files
  )

  let allSelected = $derived(
    displayedFiles.length > 0 && displayedFiles.every((f) => selectedPaths.includes(f.path))
  )

  function toggleSelectAll() {
    if (allSelected) {
      selectedPaths = []
    } else {
      selectedPaths = displayedFiles.map((f) => f.path)
    }
  }

  function toggleSelectPath(path: string) {
    if (selectedPaths.includes(path)) {
      selectedPaths = selectedPaths.filter((p) => p !== path)
    } else {
      selectedPaths = [...selectedPaths, path]
    }
  }

  function deselectAll() {
    selectedPaths = []
  }

  function openBatchCopy() {
    copyDest = currentPath
    batchCopyModalOpen = true
  }

  function openBatchMove() {
    moveDest = currentPath
    batchMoveModalOpen = true
  }

  function openBatchCompress() {
    const today = new Date().toISOString().slice(0, 10)
    batchCompressName = `archive_${today}.zip`
    batchCompressModalOpen = true
  }

  async function handleBatchDelete() {
    if (selectedPaths.length === 0) return
    try {
      isBatchProcessing = true
      const res = await api.post<{ success: boolean; error?: string }>('/api/files/batch', {
        action: 'delete',
        paths: selectedPaths,
      })
      if (res && res.success) {
        toastStore.success(`Deleted ${selectedPaths.length} items successfully.`)
        batchDeleteModalOpen = false
        selectedPaths = []
        await navigateTo(currentPath)
        await fetchStorageInfo()
      } else {
        toastStore.error(res?.error || 'Batch delete failed')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Batch delete error')
    } finally {
      isBatchProcessing = false
    }
  }

  async function handleBatchCopy() {
    if (selectedPaths.length === 0) return
    if (isCopyDestSameDir) {
      toastStore.error('Destination cannot be the current directory')
      return
    }
    try {
      isBatchProcessing = true
      const res = await api.post<{ success: boolean; error?: string }>('/api/files/batch', {
        action: 'copy',
        paths: selectedPaths,
        dest_dir: copyDest,
      })
      if (res && res.success) {
        toastStore.success(`Copied ${selectedPaths.length} items to ${copyDest}.`)
        batchCopyModalOpen = false
        selectedPaths = []
        await navigateTo(currentPath)
        await fetchStorageInfo()
      } else {
        toastStore.error(res?.error || 'Batch copy failed')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Batch copy error')
    } finally {
      isBatchProcessing = false
    }
  }

  async function handleBatchMove() {
    if (selectedPaths.length === 0) return
    if (isMoveDestSameDir) {
      toastStore.error('Items are already in this directory')
      return
    }
    try {
      isBatchProcessing = true
      const res = await api.post<{ success: boolean; error?: string }>('/api/files/batch', {
        action: 'move',
        paths: selectedPaths,
        dest_dir: moveDest,
      })
      if (res && res.success) {
        toastStore.success(`Moved ${selectedPaths.length} items to ${moveDest}.`)
        batchMoveModalOpen = false
        selectedPaths = []
        await navigateTo(currentPath)
        await fetchStorageInfo()
      } else {
        toastStore.error(res?.error || 'Batch move failed')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Batch move error')
    } finally {
      isBatchProcessing = false
    }
  }

  async function handleBatchCompress() {
    if (selectedPaths.length === 0) return
    let name = batchCompressName.trim()
    if (!name) name = 'archive.zip'
    if (!name.endsWith('.zip')) name += '.zip'

    try {
      isBatchProcessing = true
      const zipDest = `${currentPath.replace(/\/+$/, '')}/${name}`
      const res = await api.post<{ success: boolean; error?: string }>('/api/files/compress', {
        paths: selectedPaths,
        dest_zip: zipDest,
        destination: zipDest,
      })
      if (res && res.success) {
        toastStore.success(`Compressed ${selectedPaths.length} items to ${name}.`)
        batchCompressModalOpen = false
        selectedPaths = []
        await navigateTo(currentPath)
        await fetchStorageInfo()
      } else {
        toastStore.error(res?.error || 'Batch compress failed')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Batch compress error')
    } finally {
      isBatchProcessing = false
    }
  }

  onMount(async () => {
    await navigateTo(currentPath)
    await fetchStorageInfo()
  })

  async function fetchStorageInfo() {
    try {
      const res = await api.get<FileStorageInfo>('/api/files/storage')
      if (res) storageInfo = res
    } catch {
      // Ignored if storage query unsupported
    }
  }

  async function navigateTo(path: string) {
    try {
      isLoading = true
      const res = await api.get<FileListResponse>(`/api/files/list?path=${encodeURIComponent(path)}`)
      currentPath = res.path || res.current_path || path
      files = Array.isArray(res.files) ? res.files : []
      searchResults = null
      selectedPaths = []
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to list directory')
    } finally {
      isLoading = false
    }
  }

  function goUp() {
    if (currentPath === '/' || currentPath === '') return
    const parts = currentPath.split('/').filter(Boolean)
    parts.pop()
    const parent = '/' + parts.join('/')
    navigateTo(parent || '/')
  }

  function formatBytes(bytes: number): string {
    if (!bytes || bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`
  }

  function isArchive(file: FileEntry): boolean {
    if (file.is_dir) return false
    const n = file.name.toLowerCase()
    return n.endsWith('.zip') || n.endsWith('.tar.gz') || n.endsWith('.tgz') || n.endsWith('.tar')
  }

  // --- Search ---
  async function runSearch() {
    if (!searchQuery.trim()) {
      searchResults = null
      selectedPaths = []
      return
    }
    try {
      isSearching = true
      selectedPaths = []
      const res = await api.get<{ files?: FileEntry[] }>(
        `/api/files/search?path=${encodeURIComponent(currentPath)}&query=${encodeURIComponent(searchQuery.trim())}`
      )
      searchResults = Array.isArray(res.files) ? res.files : []
      toastStore.success(`Found ${searchResults.length} matches for "${searchQuery}"`)
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Search failed')
    } finally {
      isSearching = false
    }
  }

  function clearSearch() {
    searchQuery = ''
    searchResults = null
    selectedPaths = []
  }

  // --- Text File Reading & Editing ---
  async function openEditor(file: FileEntry) {
    try {
      isLoading = true
      editorFilePath = file.path
      const res = await api.get<FileReadResponse>(`/api/files/read?path=${encodeURIComponent(file.path)}`)
      editorContent = res.content || ''
      editorOpen = true
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to read file content')
    } finally {
      isLoading = false
    }
  }

  async function saveFileContent() {
    try {
      isSavingFile = true
      await api.post('/api/files/save', {
        path: editorFilePath,
        content: editorContent,
      })
      toastStore.success('File saved successfully.')
      editorOpen = false
      await navigateTo(currentPath)
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to save file')
    } finally {
      isSavingFile = false
    }
  }

  // --- Creation ---
  async function handleCreate() {
    if (!createName) return
    const newPath = `${currentPath.replace(/\/+$/, '')}/${createName}`

    try {
      if (createType === 'file') {
        await api.post('/api/files/create', { path: newPath })
      } else {
        await api.post('/api/files/mkdir', { path: newPath })
      }
      toastStore.success(`Created ${createType} successfully.`)
      createModalOpen = false
      createName = ''
      await navigateTo(currentPath)
      await fetchStorageInfo()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : `Failed to create ${createType}`)
    }
  }

  // --- Rename ---
  function startRename(file: FileEntry) {
    renameTarget = file
    newFileName = file.name
    renameModalOpen = true
  }

  async function handleRename() {
    if (!renameTarget || !newFileName) return
    const newPath = `${currentPath.replace(/\/+$/, '')}/${newFileName}`

    try {
      await api.post('/api/files/rename', {
        old_path: renameTarget.path,
        new_path: newPath,
      })
      toastStore.success('Renamed successfully.')
      renameModalOpen = false
      renameTarget = null
      await navigateTo(currentPath)
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to rename item')
    }
  }

  // --- Delete ---
  function startDelete(file: FileEntry) {
    deleteTarget = file
    deleteModalOpen = true
  }

  async function handleDelete() {
    if (!deleteTarget) return
    try {
      await api.post('/api/files/delete', { path: deleteTarget.path })
      toastStore.success('Deleted successfully.')
      deleteModalOpen = false
      deleteTarget = null
      await navigateTo(currentPath)
      await fetchStorageInfo()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to delete item')
    }
  }

  // --- Upload & Download ---
  async function handleUpload(e: Event) {
    const target = e.target as HTMLInputElement
    if (!target.files || target.files.length === 0) return
    const file = target.files[0]
    const formData = new FormData()
    formData.append('path', currentPath)
    formData.append('file', file)

    try {
      isUploading = true
      await api.post('/api/files/upload', formData)
      toastStore.success(`Uploaded ${file.name} successfully.`)
      if (uploadInputEl) uploadInputEl.value = ''
      await navigateTo(currentPath)
      await fetchStorageInfo()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Upload failed')
    } finally {
      isUploading = false
    }
  }

  function downloadFile(file: FileEntry) {
    const link = document.createElement('a')
    link.href = `/api/files/download?path=${encodeURIComponent(file.path)}`
    link.download = file.name
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  }

  // --- Permissions (Chmod) ---
  function startPermissions(file: FileEntry) {
    permTarget = file
    permMode = file.permissions ? file.permissions.slice(-4) : file.is_dir ? '0755' : '0644'
    permModalOpen = true
  }

  async function handlePermissions() {
    if (!permTarget) return
    try {
      await api.post('/api/files/permissions', {
        path: permTarget.path,
        mode: permMode,
      })
      toastStore.success(`Permissions for ${permTarget.name} set to ${permMode}.`)
      permModalOpen = false
      permTarget = null
      await navigateTo(currentPath)
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to update permissions')
    }
  }

  // --- Compress (ZIP) ---
  function startCompress(file: FileEntry) {
    compressTarget = file
    compressDestName = `${file.name}.zip`
    compressModalOpen = true
  }

  async function handleCompress() {
    if (!compressTarget) return
    try {
      const destZip = `${currentPath.replace(/\/+$/, '')}/${compressDestName}`
      await api.post('/api/files/compress', {
        paths: [compressTarget.path],
        dest_zip: destZip,
        destination: destZip,
      })
      toastStore.success(`Archive "${compressDestName}" created.`)
      compressModalOpen = false
      compressTarget = null
      await navigateTo(currentPath)
      await fetchStorageInfo()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to compress item')
    }
  }

  // --- Extract (Unzip) ---
  function startExtract(file: FileEntry) {
    extractTarget = file
    extractDestDir = currentPath
    extractModalOpen = true
  }

  async function handleExtract() {
    if (!extractTarget) return
    try {
      await api.post('/api/files/extract', {
        path: extractTarget.path,
        zip_path: extractTarget.path,
        destination: extractDestDir,
        dest_dir: extractDestDir,
      })
      toastStore.success(`Extracted "${extractTarget.name}" to ${extractDestDir}.`)
      extractModalOpen = false
      extractTarget = null
      await navigateTo(currentPath)
      await fetchStorageInfo()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to extract archive')
    }
  }

  // --- Copy & Move ---
  function startCopyMove(file: FileEntry, action: 'copy' | 'move') {
    copyMoveTarget = file
    copyMoveAction = action
    copyMoveDest = currentPath
    copyMoveModalOpen = true
  }

  async function handleCopyMove() {
    if (!copyMoveTarget) return
    try {
      const dst = `${copyMoveDest.replace(/\/+$/, '')}/${copyMoveTarget.name}`
      const endpoint = copyMoveAction === 'copy' ? '/api/files/copy' : '/api/files/move'
      await api.post(endpoint, {
        src: copyMoveTarget.path,
        dst,
      })
      toastStore.success(`${copyMoveAction === 'copy' ? 'Copied' : 'Moved'} to ${copyMoveDest} successfully.`)
      copyMoveModalOpen = false
      copyMoveTarget = null
      await navigateTo(currentPath)
      await fetchStorageInfo()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : `Failed to ${copyMoveAction} item`)
    }
  }

  const breadcrumbs = $derived(
    currentPath
      .split('/')
      .filter(Boolean)
      .map((seg, idx, arr) => ({
        name: seg,
        path: '/' + arr.slice(0, idx + 1).join('/'),
      }))
  )
</script>

<div class="space-y-4">
  <!-- Top Toolbar -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
    <div class="flex items-center gap-2">
      <FolderOpen class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        Root File Explorer
      </h2>
    </div>

    <!-- Actions Bar -->
    <div class="flex items-center gap-2 flex-wrap">
      <Button
        variant="outline"
        size="sm"
        disabled={currentPath === '/' || currentPath === ''}
        onclick={goUp}
      >
        <ArrowUp class="w-3.5 h-3.5 mr-1" />
        <span>Up</span>
      </Button>

      <Button
        variant="secondary"
        size="sm"
        disabled={isLoading}
        onclick={() => navigateTo(currentPath)}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1 {isLoading ? 'animate-spin' : ''}" />
        <span>Refresh</span>
      </Button>

      <Button
        variant="secondary"
        size="sm"
        onclick={() => {
          createType = 'file'
          createName = ''
          createModalOpen = true
        }}
      >
        <Plus class="w-3.5 h-3.5 mr-1 text-accent" />
        <span>New File</span>
      </Button>

      <Button
        variant="secondary"
        size="sm"
        onclick={() => {
          createType = 'folder'
          createName = ''
          createModalOpen = true
        }}
      >
        <Plus class="w-3.5 h-3.5 mr-1 text-emerald-400" />
        <span>New Folder</span>
      </Button>

      <label class="neo-button inline-flex items-center justify-center font-mono font-bold select-none cursor-pointer bg-accent text-accent-text px-3.5 py-1.5 text-xs rounded">
        <Upload class="w-3.5 h-3.5 mr-1" />
        <span>{isUploading ? 'Uploading...' : 'Upload'}</span>
        <input
          type="file"
          bind:this={uploadInputEl}
          onchange={handleUpload}
          class="hidden"
          disabled={isUploading}
        />
      </label>
    </div>
  </div>

  <!-- Storage Capacity Banner -->
  {#if storageInfo}
    <div class="p-3 bg-card-sub border-2 border-border shadow-neobrutal-sm rounded font-mono text-xs flex flex-col sm:flex-row sm:items-center justify-between gap-3">
      <div class="flex items-center gap-2">
        <HardDrive class="w-4 h-4 text-accent shrink-0" />
        <div>
          <span class="font-bold text-foreground">Storage Mount: {storageInfo.mount || currentPath}</span>
          <span class="text-muted text-[11px] block sm:inline sm:ml-2">
            ({formatBytes(storageInfo.used)} used of {formatBytes(storageInfo.total)})
          </span>
        </div>
      </div>
      <div class="flex items-center gap-3">
        <div class="w-32 sm:w-44 bg-card border border-border rounded-full h-2.5 overflow-hidden">
          <div
            class="bg-accent h-full rounded-full transition-all"
            style="width: {Math.min(100, Math.max(0, storageInfo.used_pct || 0))}%;"
          ></div>
        </div>
        <span class="font-bold text-accent text-[11px] whitespace-nowrap">
          {(storageInfo.used_pct || 0).toFixed(1)}% ({formatBytes(storageInfo.free)} free)
        </span>
      </div>
    </div>
  {/if}

  <!-- Breadcrumb Path Navigator & Search Row -->
  <div class="grid grid-cols-1 md:grid-cols-3 gap-2">
    <!-- Breadcrumbs -->
    <div class="md:col-span-2 p-2.5 bg-card border-2 border-border shadow-neobrutal-sm rounded flex items-center gap-1.5 font-mono text-xs overflow-x-auto">
      <button
        type="button"
        class="text-accent hover:underline font-bold shrink-0 cursor-pointer"
        onclick={() => navigateTo('/')}
      >
        /
      </button>
      {#each breadcrumbs as bc}
        <span class="text-muted shrink-0">/</span>
        <button
          type="button"
          class="text-foreground hover:text-accent truncate cursor-pointer {bc.path === currentPath ? 'font-bold text-accent' : ''}"
          onclick={() => navigateTo(bc.path)}
        >
          {bc.name}
        </button>
      {/each}
    </div>

    <!-- Search Input -->
    <div class="flex items-center gap-1.5">
      <div class="relative flex-1">
        <Input
          placeholder="Search current folder..."
          bind:value={searchQuery}
          onchange={runSearch}
        />
      </div>
      <Button
        variant="secondary"
        size="md"
        disabled={isSearching}
        onclick={runSearch}
        title="Search"
      >
        <Search class="w-3.5 h-3.5 {isSearching ? 'animate-spin' : ''}" />
      </Button>
      {#if searchResults !== null}
        <Button
          variant="outline"
          size="md"
          onclick={clearSearch}
          title="Clear search"
        >
          <X class="w-3.5 h-3.5 text-red-400" />
        </Button>
      {/if}
    </div>
  </div>

  <!-- Search Filter Active Banner -->
  {#if searchResults !== null}
    <div class="p-2.5 bg-card-sub border border-border rounded flex items-center justify-between font-mono text-xs text-foreground">
      <span>
        Showing search results for <strong class="text-accent">"{searchQuery}"</strong> ({searchResults.length} matches)
      </span>
      <button
        type="button"
        class="text-xs text-muted hover:text-red-400 cursor-pointer flex items-center gap-1"
        onclick={clearSearch}
      >
        <X class="w-3 h-3" />
        <span>Clear</span>
      </button>
    </div>
  {/if}

  <!-- Files Table Container -->
  <Card class="p-0 overflow-hidden font-mono text-xs">
    {#if isLoading && files.length === 0}
      <div class="p-12 text-center text-muted">
        <RefreshCw class="w-6 h-6 animate-spin mx-auto mb-2 text-accent" />
        Loading filesystem entries...
      </div>
    {:else if displayedFiles.length === 0}
      <div class="p-12 text-center text-muted">
        <FolderOpen class="w-6 h-6 mx-auto mb-2 opacity-50" />
        {searchResults !== null ? 'No files match your search query.' : 'This directory is empty.'}
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-left">
          <thead>
            <tr class="border-b-2 border-border text-muted uppercase text-[10px] bg-card-sub">
              <th class="py-2.5 px-3 w-10 text-center">
                <input
                  type="checkbox"
                  checked={allSelected}
                  onchange={toggleSelectAll}
                  class="w-4 h-4 accent-accent rounded cursor-pointer"
                  title={allSelected ? 'Deselect All' : 'Select All'}
                />
              </th>
              <th class="py-2.5 px-3">Name</th>
              <th class="py-2.5 px-3 w-24">Size</th>
              <th class="py-2.5 px-3 w-28 hidden sm:table-cell">Perms</th>
              <th class="py-2.5 px-3 w-36 hidden md:table-cell">Modified</th>
              <th class="py-2.5 px-3 text-right">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            {#each displayedFiles as file (file.path)}
              {@const isSelected = selectedPaths.includes(file.path)}
              <tr class="transition-colors {isSelected ? 'bg-accent/15' : 'hover:bg-card-sub/60'}">
                <td class="py-2.5 px-3 w-10 text-center">
                  <input
                    type="checkbox"
                    checked={isSelected}
                    onchange={() => toggleSelectPath(file.path)}
                    class="w-4 h-4 accent-accent rounded cursor-pointer"
                  />
                </td>
                <!-- Name & Icon -->
                <td class="py-2.5 px-3">
                  {#if file.is_dir}
                    <button
                      type="button"
                      class="flex items-center gap-2 text-foreground font-bold hover:text-accent cursor-pointer truncate max-w-xs sm:max-w-md text-left"
                      onclick={() => navigateTo(file.path)}
                    >
                      <Folder class="w-4 h-4 text-accent shrink-0" />
                      <span class="truncate">{file.name}</span>
                    </button>
                  {:else}
                    <button
                      type="button"
                      class="flex items-center gap-2 text-foreground hover:text-accent cursor-pointer truncate max-w-xs sm:max-w-md text-left"
                      onclick={() => openEditor(file)}
                    >
                      {#if isArchive(file)}
                        <FileArchive class="w-4 h-4 text-amber-400 shrink-0" />
                      {:else}
                        <FileText class="w-4 h-4 text-muted shrink-0" />
                      {/if}
                      <span class="truncate">{file.name}</span>
                    </button>
                  {/if}
                </td>

                <!-- Size -->
                <td class="py-2.5 px-3 text-muted">
                  {file.is_dir ? '—' : formatBytes(file.size)}
                </td>

                <!-- Perms -->
                <td class="py-2.5 px-3 text-muted hidden sm:table-cell text-[11px]">
                  <button
                    type="button"
                    class="hover:text-accent hover:underline cursor-pointer"
                    onclick={() => startPermissions(file)}
                    title="Change permissions"
                  >
                    {file.permissions || (file.is_dir ? '0755' : '0644')}
                  </button>
                </td>

                <!-- Modified Time -->
                <td class="py-2.5 px-3 text-muted hidden md:table-cell text-[11px]">
                  {file.mod_time ? new Date(file.mod_time).toLocaleDateString() : '—'}
                </td>

                <!-- Actions -->
                <td class="py-2.5 px-3 text-right">
                  <div class="flex items-center justify-end gap-1">
                    <!-- Extract Archive -->
                    {#if isArchive(file)}
                      <button
                        type="button"
                        class="p-1 rounded hover:bg-card text-muted hover:text-amber-400 transition-colors cursor-pointer"
                        onclick={() => startExtract(file)}
                        title="Extract Archive"
                        aria-label="Extract Archive"
                      >
                        <FileArchive class="w-3.5 h-3.5" />
                      </button>
                    {/if}

                    <!-- Compress Item -->
                    <button
                      type="button"
                      class="p-1 rounded hover:bg-card text-muted hover:text-blue-400 transition-colors cursor-pointer"
                      onclick={() => startCompress(file)}
                      title="Compress into ZIP"
                      aria-label="Compress"
                    >
                      <FolderInput class="w-3.5 h-3.5" />
                    </button>

                    <!-- Copy Item -->
                    <button
                      type="button"
                      class="p-1 rounded hover:bg-card text-muted hover:text-purple-400 transition-colors cursor-pointer"
                      onclick={() => startCopyMove(file, 'copy')}
                      title="Copy"
                      aria-label="Copy"
                    >
                      <Copy class="w-3.5 h-3.5" />
                    </button>

                    <!-- Download File -->
                    {#if !file.is_dir}
                      <button
                        type="button"
                        class="p-1 rounded hover:bg-card text-muted hover:text-accent transition-colors cursor-pointer"
                        onclick={() => downloadFile(file)}
                        title="Download"
                        aria-label="Download"
                      >
                        <Download class="w-3.5 h-3.5" />
                      </button>

                      <button
                        type="button"
                        class="p-1 rounded hover:bg-card text-muted hover:text-accent transition-colors cursor-pointer"
                        onclick={() => openEditor(file)}
                        title="Edit text"
                        aria-label="Edit"
                      >
                        <Edit2 class="w-3.5 h-3.5" />
                      </button>
                    {/if}

                    <!-- Chmod Permissions -->
                    <button
                      type="button"
                      class="p-1 rounded hover:bg-card text-muted hover:text-amber-400 transition-colors cursor-pointer"
                      onclick={() => startPermissions(file)}
                      title="Permissions (Chmod)"
                      aria-label="Permissions"
                    >
                      <Shield class="w-3.5 h-3.5" />
                    </button>

                    <!-- Rename Item -->
                    <button
                      type="button"
                      class="p-1 rounded hover:bg-card text-muted hover:text-foreground transition-colors cursor-pointer"
                      onclick={() => startRename(file)}
                      title="Rename"
                      aria-label="Rename"
                    >
                      <Edit2 class="w-3.5 h-3.5" />
                    </button>

                    <!-- Delete Item -->
                    <button
                      type="button"
                      class="p-1 rounded hover:bg-card text-muted hover:text-red-400 transition-colors cursor-pointer"
                      onclick={() => startDelete(file)}
                      title="Delete"
                      aria-label="Delete"
                    >
                      <Trash2 class="w-3.5 h-3.5" />
                    </button>
                  </div>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </Card>

  <!-- Text File Editor Modal -->
  {#if editorOpen}
    <Modal
      open={true}
      title={`Editing: ${editorFilePath}`}
      class="max-w-4xl"
      onclose={() => (editorOpen = false)}
    >
      <div class="space-y-4 font-mono text-xs">
        <textarea
          bind:value={editorContent}
          rows="18"
          class="w-full bg-card-sub border-2 border-border rounded p-3 font-mono text-xs text-foreground focus:outline-none focus:border-accent resize-y"
          placeholder="File is empty"
        ></textarea>

        <div class="flex items-center justify-between pt-2 border-t border-border">
          <span class="text-muted text-[11px] truncate max-w-sm">
            {editorFilePath}
          </span>
          <div class="flex items-center gap-2">
            <Button variant="ghost" size="md" onclick={() => (editorOpen = false)}>
              Cancel
            </Button>
            <Button
              variant="primary"
              size="md"
              disabled={isSavingFile}
              onclick={saveFileContent}
            >
              <Save class="w-3.5 h-3.5 mr-1.5" />
              <span>{isSavingFile ? 'Saving...' : 'Save File'}</span>
            </Button>
          </div>
        </div>
      </div>
    </Modal>
  {/if}

  <!-- Create Item Modal -->
  {#if createModalOpen}
    <Modal
      open={true}
      title={`Create New ${createType === 'file' ? 'File' : 'Folder'}`}
      onclose={() => (createModalOpen = false)}
    >
      <form onsubmit={(e) => { e.preventDefault(); handleCreate() }} class="space-y-4 font-mono text-xs">
        <Input
          label="Name"
          placeholder={createType === 'file' ? 'example.conf' : 'new_folder'}
          bind:value={createName}
        />
        <div class="flex justify-end gap-2 pt-2 border-t border-border">
          <Button variant="ghost" size="md" onclick={() => (createModalOpen = false)}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" size="md" disabled={!createName}>
            Create
          </Button>
        </div>
      </form>
    </Modal>
  {/if}

  <!-- Rename Modal -->
  {#if renameModalOpen && renameTarget}
    <Modal
      open={true}
      title={`Rename: ${renameTarget.name}`}
      onclose={() => (renameModalOpen = false)}
    >
      <form onsubmit={(e) => { e.preventDefault(); handleRename() }} class="space-y-4 font-mono text-xs">
        <Input
          label="New Name"
          bind:value={newFileName}
        />
        <div class="flex justify-end gap-2 pt-2 border-t border-border">
          <Button variant="ghost" size="md" onclick={() => (renameModalOpen = false)}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" size="md" disabled={!newFileName || newFileName === renameTarget.name}>
            Rename
          </Button>
        </div>
      </form>
    </Modal>
  {/if}

  <!-- Delete Modal -->
  {#if deleteModalOpen && deleteTarget}
    <Modal
      open={true}
      title="Confirm Deletion"
      onclose={() => (deleteModalOpen = false)}
    >
      <div class="space-y-4 font-mono text-xs">
        <p class="text-foreground leading-relaxed">
          Are you sure you want to permanently delete:
          <span class="font-bold text-red-400 block mt-1 break-all">
            {deleteTarget.path}
          </span>
        </p>

        <div class="flex justify-end gap-2 pt-2 border-t border-border">
          <Button variant="ghost" size="md" onclick={() => (deleteModalOpen = false)}>
            Cancel
          </Button>
          <Button variant="danger" size="md" onclick={handleDelete}>
            <Trash2 class="w-3.5 h-3.5 mr-1" />
            <span>Delete Permanently</span>
          </Button>
        </div>
      </div>
    </Modal>
  {/if}

  <!-- Permissions Modal (Chmod) -->
  {#if permModalOpen && permTarget}
    <Modal
      open={true}
      title={`Permissions (Chmod): ${permTarget.name}`}
      onclose={() => (permModalOpen = false)}
    >
      <form onsubmit={(e) => { e.preventDefault(); handlePermissions() }} class="space-y-4 font-mono text-xs">
        <Input
          label="Octal Mode (e.g. 0755, 0644, 0777)"
          placeholder="0755"
          bind:value={permMode}
        />
        <p class="text-[11px] text-muted">
          Standard Linux octal permission mode applied directly to filesystem inode.
        </p>
        <div class="flex justify-end gap-2 pt-2 border-t border-border">
          <Button variant="ghost" size="md" onclick={() => (permModalOpen = false)}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" size="md" disabled={!permMode}>
            <Check class="w-3.5 h-3.5 mr-1" />
            <span>Apply Chmod</span>
          </Button>
        </div>
      </form>
    </Modal>
  {/if}

  <!-- Compress Modal (ZIP) -->
  {#if compressModalOpen && compressTarget}
    <Modal
      open={true}
      title={`Compress: ${compressTarget.name}`}
      onclose={() => (compressModalOpen = false)}
    >
      <form onsubmit={(e) => { e.preventDefault(); handleCompress() }} class="space-y-4 font-mono text-xs">
        <Input
          label="Archive ZIP File Name"
          placeholder="archive.zip"
          bind:value={compressDestName}
        />
        <div class="flex justify-end gap-2 pt-2 border-t border-border">
          <Button variant="ghost" size="md" onclick={() => (compressModalOpen = false)}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" size="md" disabled={!compressDestName}>
            <FolderInput class="w-3.5 h-3.5 mr-1" />
            <span>Create ZIP</span>
          </Button>
        </div>
      </form>
    </Modal>
  {/if}

  <!-- Extract Modal (Unzip) -->
  {#if extractModalOpen && extractTarget}
    <Modal
      open={true}
      title={`Extract Archive: ${extractTarget.name}`}
      onclose={() => (extractModalOpen = false)}
    >
      <form onsubmit={(e) => { e.preventDefault(); handleExtract() }} class="space-y-4 font-mono text-xs">
        <Input
          label="Destination Folder"
          placeholder="/sdcard/extracted"
          bind:value={extractDestDir}
        />
        <div class="flex justify-end gap-2 pt-2 border-t border-border">
          <Button variant="ghost" size="md" onclick={() => (extractModalOpen = false)}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" size="md" disabled={!extractDestDir}>
            <FileArchive class="w-3.5 h-3.5 mr-1" />
            <span>Extract Files</span>
          </Button>
        </div>
      </form>
    </Modal>
  {/if}

  <!-- Copy & Move Modal -->
  {#if copyMoveModalOpen && copyMoveTarget}
    <Modal
      open={true}
      title={`${copyMoveAction === 'copy' ? 'Copy' : 'Move'}: ${copyMoveTarget.name}`}
      onclose={() => (copyMoveModalOpen = false)}
    >
      <form onsubmit={(e) => { e.preventDefault(); handleCopyMove() }} class="space-y-4 font-mono text-xs">
        <div class="p-2.5 bg-card-sub border border-border rounded flex items-center justify-between">
          <span class="text-muted">Target Item:</span>
          <span class="font-bold text-foreground">{copyMoveTarget.name}</span>
        </div>
        <Input
          label="Destination Directory"
          placeholder="/sdcard/download"
          bind:value={copyMoveDest}
        />
        <div class="flex justify-end gap-2 pt-2 border-t border-border">
          <Button variant="ghost" size="md" onclick={() => (copyMoveModalOpen = false)}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" size="md" disabled={!copyMoveDest}>
            {#if copyMoveAction === 'copy'}
              <Copy class="w-3.5 h-3.5 mr-1" />
              <span>Copy Here</span>
            {:else}
              <FolderInput class="w-3.5 h-3.5 mr-1" />
              <span>Move Here</span>
            {/if}
          </Button>
        </div>
      </form>
    </Modal>
  {/if}

  <!-- Batch Action Toolbar -->
  {#if selectedPaths.length > 0}
    <div
      class="fixed bottom-5 left-4 right-4 sm:left-auto sm:right-6 z-40 max-w-2xl bg-black/90 backdrop-blur-md border-2 border-border shadow-neobrutal p-3 rounded-lg flex flex-wrap items-center justify-between gap-3 font-mono text-xs animate-in fade-in slide-in-from-bottom-3"
    >
      <div class="flex items-center gap-2.5">
        <span class="px-2.5 py-1 bg-accent text-accent-text font-black rounded text-xs shadow-neobrutal-sm">
          {selectedPaths.length} selected
        </span>
        <Button variant="ghost" size="sm" onclick={deselectAll}>
          <X class="w-3.5 h-3.5 mr-1" />
          <span>Deselect All</span>
        </Button>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <Button
          variant="secondary"
          size="sm"
          disabled={isBatchProcessing}
          onclick={openBatchCopy}
        >
          <Copy class="w-3.5 h-3.5 mr-1" />
          <span>Batch Copy</span>
        </Button>

        <Button
          variant="secondary"
          size="sm"
          disabled={isBatchProcessing}
          onclick={openBatchMove}
        >
          <FolderInput class="w-3.5 h-3.5 mr-1" />
          <span>Batch Move</span>
        </Button>

        <Button
          variant="secondary"
          size="sm"
          disabled={isBatchProcessing}
          onclick={openBatchCompress}
        >
          <FileArchive class="w-3.5 h-3.5 mr-1 text-accent" />
          <span>Batch Compress</span>
        </Button>

        <Button
          variant="danger"
          size="sm"
          disabled={isBatchProcessing}
          onclick={() => (batchDeleteModalOpen = true)}
        >
          <Trash2 class="w-3.5 h-3.5 mr-1" />
          <span>Batch Delete</span>
        </Button>
      </div>
    </div>
  {/if}

  <!-- Batch Delete Modal -->
  {#if batchDeleteModalOpen}
    <Modal
      open={true}
      title={`Batch Delete (${selectedPaths.length} items)`}
      onclose={() => (batchDeleteModalOpen = false)}
    >
      <div class="space-y-4 font-mono text-xs">
        <div class="p-3 bg-red-950/30 border border-red-800 rounded text-red-200 space-y-1">
          <p class="font-bold text-red-400">Permanently delete selected files & directories?</p>
          <p class="text-[11px] text-red-300/80">
            This operation cannot be undone. All selected files and folder contents will be permanently deleted from the Android filesystem.
          </p>
        </div>

        <div class="max-h-40 overflow-y-auto bg-card-sub border border-border rounded p-2.5 space-y-1">
          {#each selectedPaths.slice(0, 10) as p}
            <div class="text-muted truncate text-[11px]">• {p}</div>
          {/each}
          {#if selectedPaths.length > 10}
            <div class="text-accent font-bold pt-1 text-[11px]">...and {selectedPaths.length - 10} more items</div>
          {/if}
        </div>

        <div class="flex justify-end gap-2 pt-2 border-t border-border">
          <Button variant="ghost" size="sm" onclick={() => (batchDeleteModalOpen = false)}>
            Cancel
          </Button>
          <Button
            variant="danger"
            size="sm"
            disabled={isBatchProcessing}
            onclick={handleBatchDelete}
          >
            <Trash2 class="w-3.5 h-3.5 mr-1" />
            <span>{isBatchProcessing ? 'Deleting...' : `Delete ${selectedPaths.length} Items`}</span>
          </Button>
        </div>
      </div>
    </Modal>
  {/if}

  <!-- Batch Copy Modal -->
  {#if batchCopyModalOpen}
    <Modal
      open={true}
      title={`Batch Copy (${selectedPaths.length} items)`}
      onclose={() => (batchCopyModalOpen = false)}
    >
      <form onsubmit={(e) => { e.preventDefault(); handleBatchCopy() }} class="space-y-4 font-mono text-xs">
        <Input
          label="Destination Folder (dest_dir)"
          placeholder="/sdcard/download"
          bind:value={copyDest}
        />
        {#if isCopyDestSameDir}
          <p class="text-[11px] text-amber-500 font-bold">
            Destination cannot be the current directory
          </p>
        {/if}
        <div class="max-h-36 overflow-y-auto bg-card-sub border border-border rounded p-2.5 space-y-1 text-muted">
          <span class="text-[10px] uppercase font-bold text-foreground block mb-1">Selected Items:</span>
          {#each selectedPaths.slice(0, 6) as p}
            <div class="truncate text-[11px]">• {p}</div>
          {/each}
          {#if selectedPaths.length > 6}
            <div class="text-accent text-[11px] font-bold">...and {selectedPaths.length - 6} more items</div>
          {/if}
        </div>

        <div class="flex justify-end gap-2 pt-2 border-t border-border">
          <Button variant="ghost" size="sm" onclick={() => (batchCopyModalOpen = false)}>
            Cancel
          </Button>
          <Button
            type="submit"
            variant="primary"
            size="sm"
            disabled={isBatchProcessing || !copyDest.trim() || isCopyDestSameDir}
          >
            <Copy class="w-3.5 h-3.5 mr-1" />
            <span>{isBatchProcessing ? 'Copying...' : 'Copy Items Here'}</span>
          </Button>
        </div>
      </form>
    </Modal>
  {/if}

  <!-- Batch Move Modal -->
  {#if batchMoveModalOpen}
    <Modal
      open={true}
      title={`Batch Move (${selectedPaths.length} items)`}
      onclose={() => (batchMoveModalOpen = false)}
    >
      <form onsubmit={(e) => { e.preventDefault(); handleBatchMove() }} class="space-y-4 font-mono text-xs">
        <Input
          label="Destination Folder (dest_dir)"
          placeholder="/sdcard/download"
          bind:value={moveDest}
        />
        {#if isMoveDestSameDir}
          <p class="text-[11px] text-amber-500 font-bold">
            Items are already in this directory
          </p>
        {/if}
        <div class="max-h-36 overflow-y-auto bg-card-sub border border-border rounded p-2.5 space-y-1 text-muted">
          <span class="text-[10px] uppercase font-bold text-foreground block mb-1">Selected Items:</span>
          {#each selectedPaths.slice(0, 6) as p}
            <div class="truncate text-[11px]">• {p}</div>
          {/each}
          {#if selectedPaths.length > 6}
            <div class="text-accent text-[11px] font-bold">...and {selectedPaths.length - 6} more items</div>
          {/if}
        </div>

        <div class="flex justify-end gap-2 pt-2 border-t border-border">
          <Button variant="ghost" size="sm" onclick={() => (batchMoveModalOpen = false)}>
            Cancel
          </Button>
          <Button
            type="submit"
            variant="primary"
            size="sm"
            disabled={isBatchProcessing || !moveDest.trim() || isMoveDestSameDir}
          >
            <FolderInput class="w-3.5 h-3.5 mr-1" />
            <span>{isBatchProcessing ? 'Moving...' : 'Move Items Here'}</span>
          </Button>
        </div>
      </form>
    </Modal>
  {/if}

  <!-- Batch Compress Modal -->
  {#if batchCompressModalOpen}
    <Modal
      open={true}
      title={`Batch Compress (${selectedPaths.length} items)`}
      onclose={() => (batchCompressModalOpen = false)}
    >
      <form onsubmit={(e) => { e.preventDefault(); handleBatchCompress() }} class="space-y-4 font-mono text-xs">
        <Input
          label="Archive Name (ZIP)"
          placeholder="archive.zip"
          bind:value={batchCompressName}
        />
        <div class="max-h-36 overflow-y-auto bg-card-sub border border-border rounded p-2.5 space-y-1 text-muted">
          <span class="text-[10px] uppercase font-bold text-foreground block mb-1">Items to Compress:</span>
          {#each selectedPaths.slice(0, 6) as p}
            <div class="truncate text-[11px]">• {p}</div>
          {/each}
          {#if selectedPaths.length > 6}
            <div class="text-accent text-[11px] font-bold">...and {selectedPaths.length - 6} more items</div>
          {/if}
        </div>

        <div class="flex justify-end gap-2 pt-2 border-t border-border">
          <Button variant="ghost" size="sm" onclick={() => (batchCompressModalOpen = false)}>
            Cancel
          </Button>
          <Button
            type="submit"
            variant="primary"
            size="sm"
            disabled={isBatchProcessing || !batchCompressName.trim()}
          >
            <FileArchive class="w-3.5 h-3.5 mr-1 text-accent" />
            <span>{isBatchProcessing ? 'Compressing...' : 'Compress Items'}</span>
          </Button>
        </div>
      </form>
    </Modal>
  {/if}
</div>
