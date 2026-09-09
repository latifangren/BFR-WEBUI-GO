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
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import type { FileEntry, FileListResponse } from '../../../types/files'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Input from '../../ui/Input.svelte'
  import Modal from '../../ui/Modal.svelte'

  let currentPath = $state('/sdcard')
  let files = $state<FileEntry[]>([])
  let isLoading = $state(false)

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

  // File Upload
  let uploadInputEl: HTMLInputElement | null = $state(null)
  let isUploading = $state(false)

  onMount(async () => {
    await navigateTo(currentPath)
  })

  async function navigateTo(path: string) {
    try {
      isLoading = true
      const res = await api.get<FileListResponse>(`/api/files/list?path=${encodeURIComponent(path)}`)
      currentPath = res.current_path || path
      files = Array.isArray(res.files) ? res.files : []
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

  async function openEditor(file: FileEntry) {
    try {
      isLoading = true
      editorFilePath = file.path
      const text = await api.get<string>(`/api/files/download?path=${encodeURIComponent(file.path)}`)
      editorContent = typeof text === 'string' ? text : JSON.stringify(text, null, 2)
      editorOpen = true
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to load file content')
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

  async function handleCreate() {
    if (!createName) return
    const targetPath = `${currentPath.replace(/\/$/, '')}/${createName}`
    try {
      if (createType === 'file') {
        await api.post('/api/files/create', { path: targetPath })
      } else {
        await api.post('/api/files/mkdir', { path: targetPath })
      }
      toastStore.success(`${createType === 'file' ? 'File' : 'Folder'} created.`)
      createModalOpen = false
      createName = ''
      await navigateTo(currentPath)
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to create item')
    }
  }

  function startRename(file: FileEntry) {
    renameTarget = file
    newFileName = file.name
    renameModalOpen = true
  }

  async function handleRename() {
    if (!renameTarget || !newFileName) return
    const parent = renameTarget.path.substring(0, renameTarget.path.lastIndexOf('/'))
    const newPath = `${parent}/${newFileName}`
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
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to delete item')
    }
  }

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

  <!-- Breadcrumb Path Navigator -->
  <div class="p-2.5 bg-card border-2 border-border shadow-neobrutal-sm rounded flex items-center gap-1.5 font-mono text-xs overflow-x-auto">
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
        class="text-foreground hover:text-accent hover:underline shrink-0 cursor-pointer font-bold"
        onclick={() => navigateTo(bc.path)}
      >
        {bc.name}
      </button>
    {/each}
  </div>

  <!-- File Table -->
  <Card class="overflow-hidden p-0">
    {#if isLoading && files.length === 0}
      <div class="p-12 text-center font-mono text-xs text-muted">
        <RefreshCw class="w-6 h-6 animate-spin mx-auto mb-2 text-accent" />
        Reading filesystem directory...
      </div>
    {:else if files.length === 0}
      <div class="p-12 text-center font-mono text-xs text-muted">
        <FolderOpen class="w-8 h-8 mx-auto mb-2 opacity-40 text-accent" />
        Directory is empty.
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-left font-mono text-xs">
          <thead>
            <tr class="border-b border-border bg-card-sub text-muted uppercase text-[10px]">
              <th class="py-2.5 px-3">Name</th>
              <th class="py-2.5 px-3">Size</th>
              <th class="py-2.5 px-3 hidden sm:table-cell">Perms</th>
              <th class="py-2.5 px-3 hidden md:table-cell">Modified</th>
              <th class="py-2.5 px-3 text-right">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            {#each files as file}
              <tr class="hover:bg-card-sub/60 transition-colors">
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
                      <FileText class="w-4 h-4 text-muted shrink-0" />
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
                  {file.permissions || '0644'}
                </td>

                <!-- Modified Time -->
                <td class="py-2.5 px-3 text-muted hidden md:table-cell text-[11px]">
                  {file.mod_time ? new Date(file.mod_time).toLocaleDateString() : '—'}
                </td>

                <!-- Actions -->
                <td class="py-2.5 px-3 text-right">
                  <div class="flex items-center justify-end gap-1">
                    {#if !file.is_dir}
                      <button
                        type="button"
                        class="p-1 rounded hover:bg-card text-muted hover:text-accent cursor-pointer"
                        onclick={() => openEditor(file)}
                        title="Edit text"
                      >
                        <Edit2 class="w-3.5 h-3.5" />
                      </button>
                      <button
                        type="button"
                        class="p-1 rounded hover:bg-card text-muted hover:text-emerald-400 cursor-pointer"
                        onclick={() => downloadFile(file)}
                        title="Download"
                      >
                        <Download class="w-3.5 h-3.5" />
                      </button>
                    {/if}
                    <button
                      type="button"
                      class="p-1 rounded hover:bg-card text-muted hover:text-blue-400 cursor-pointer"
                      onclick={() => startRename(file)}
                      title="Rename"
                    >
                      <File class="w-3.5 h-3.5" />
                    </button>
                    <button
                      type="button"
                      class="p-1 rounded hover:bg-card text-muted hover:text-red-400 cursor-pointer"
                      onclick={() => startDelete(file)}
                      title="Delete"
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

  <!-- In-Browser Text Editor Modal -->
  {#if editorOpen}
    <Modal
      open={true}
      title={`Editing: ${editorFilePath}`}
      class="max-w-3xl"
      onclose={() => (editorOpen = false)}
    >
      <div class="space-y-4 font-mono text-xs">
        <textarea
          bind:value={editorContent}
          rows="18"
          class="w-full bg-[#090d16] text-foreground border border-border rounded p-3 text-xs font-mono focus:outline-none focus:border-accent resize-y"
        ></textarea>

        <div class="flex items-center justify-end gap-2 border-t border-border pt-3">
          <Button
            variant="ghost"
            size="md"
            onclick={() => (editorOpen = false)}
          >
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
          <Button type="submit" variant="primary" size="md" disabled={!newFileName}>
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
      title="Confirm Delete"
      onclose={() => (deleteModalOpen = false)}
    >
      <div class="space-y-4 font-mono text-xs">
        <p class="text-foreground">
          Are you sure you want to permanently delete
          <span class="font-bold text-red-400 font-mono">
            {deleteTarget.path}
          </span>
          ?
        </p>
        <div class="flex justify-end gap-2 pt-2 border-t border-border">
          <Button variant="ghost" size="md" onclick={() => (deleteModalOpen = false)}>
            Cancel
          </Button>
          <Button variant="danger" size="md" onclick={handleDelete}>
            Delete
          </Button>
        </div>
      </div>
    </Modal>
  {/if}
</div>
