<script lang="ts">
  import { onMount } from 'svelte'
  import {
    FolderOpen,
    LayoutGrid,
    Square,
    Columns2,
    HardDrive,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import FilePane from './FilePane.svelte'
  import FileDropzone from './FileDropzone.svelte'
  import FileModals from './FileModals.svelte'
  import { createPaneState } from './paneState.svelte'
  import type {
    FileEntry,
    FileModalState,
    FileReadResponse,
    FileSavePayload,
    FileRenamePayload,
    FilePermissionsPayload,
    FileCompressPayload,
    FileExtractPayload,
    FileBatchPayload,
  } from './types'

  // Initialize Dual-Pane States
  const leftPane = createPaneState('left', '/sdcard')
  const rightPane = createPaneState('right', '/data/adb')

  let viewMode = $state<'single' | 'dual'>(
    (typeof window !== 'undefined' &&
      typeof localStorage !== 'undefined' &&
      (localStorage.getItem('bfr_fm_view_mode') as 'single' | 'dual')) ||
      'single'
  )

  let activePaneId = $state<'left' | 'right'>('left')
  let activePane = $derived(activePaneId === 'left' ? leftPane : rightPane)
  let otherPane = $derived(activePaneId === 'left' ? rightPane : leftPane)

  let isUploading = $state(false)
  let isBatchProcessing = $state(false)

  // Centralized Modal State
  let modalState = $state<FileModalState>({
    editor: { isOpen: false, path: '', content: '', isLoading: false, isSaving: false },
    create: { isOpen: false, type: 'file', name: '' },
    rename: { isOpen: false, target: null, newName: '' },
    delete: { isOpen: false, target: null },
    chmod: { isOpen: false, target: null, mode: '0755', owner: 'root' },
    compress: { isOpen: false, target: null, destName: '' },
    extract: { isOpen: false, target: null, destDir: '' },
    copyMove: { isOpen: false, target: null, action: 'copy', destPath: '' },
    batch: {
      deleteOpen: false,
      copyOpen: false,
      moveOpen: false,
      compressOpen: false,
      destDir: '',
      zipName: 'archive.zip',
    },
  })

  onMount(() => {
    leftPane.navigateTo(leftPane.currentPath)
    leftPane.fetchStorage()

    if (viewMode === 'dual') {
      rightPane.navigateTo(rightPane.currentPath)
      rightPane.fetchStorage()
    }
  })

  function setViewMode(mode: 'single' | 'dual') {
    viewMode = mode
    if (typeof window !== 'undefined' && typeof localStorage !== 'undefined') {
      try {
        localStorage.setItem('bfr_fm_view_mode', mode)
      } catch {
        // Ignored
      }
    }
    if (mode === 'dual' && rightPane.files.length === 0) {
      rightPane.navigateTo(rightPane.currentPath)
      rightPane.fetchStorage()
    }
  }

  // Handle Modal Open Requests from Panes
  async function handleOpenModal(action: string, target?: FileEntry, extra?: any) {
    switch (action) {
      case 'editor':
        if (!target) return
        modalState.editor.isOpen = true
        modalState.editor.path = target.path
        modalState.editor.content = ''
        modalState.editor.isLoading = true
        try {
          const res = await api.get<FileReadResponse>(
            `/api/files/read?path=${encodeURIComponent(target.path)}`
          )
          modalState.editor.content = res?.content || ''
        } catch (err: unknown) {
          toastStore.error(err instanceof Error ? err.message : 'Failed to read file')
          modalState.editor.isOpen = false
        } finally {
          modalState.editor.isLoading = false
        }
        break

      case 'create':
        modalState.create.isOpen = true
        modalState.create.type = extra?.type || 'file'
        modalState.create.name = ''
        break

      case 'rename':
        if (!target) return
        modalState.rename.isOpen = true
        modalState.rename.target = target
        modalState.rename.newName = target.name
        break

      case 'delete':
        if (!target) return
        modalState.delete.isOpen = true
        modalState.delete.target = target
        break

      case 'chmod':
        if (!target) return
        modalState.chmod.isOpen = true
        modalState.chmod.target = target
        modalState.chmod.mode = target.permissions || (target.is_dir ? '0755' : '0644')
        modalState.chmod.owner = 'root'
        break

      case 'compress':
        if (!target) return
        modalState.compress.isOpen = true
        modalState.compress.target = target
        modalState.compress.destName = `${target.name}.zip`
        break

      case 'extract':
        if (!target) return
        modalState.extract.isOpen = true
        modalState.extract.target = target
        modalState.extract.destDir = activePane.currentPath
        break

      case 'copyMove':
        if (!target) return
        modalState.copyMove.isOpen = true
        modalState.copyMove.target = target
        modalState.copyMove.action = extra?.action || 'copy'
        modalState.copyMove.destPath =
          viewMode === 'dual' ? otherPane.currentPath : activePane.currentPath
        break

      case 'batchDelete':
        modalState.batch.deleteOpen = true
        break

      case 'batchCopy':
        modalState.batch.destDir =
          viewMode === 'dual' ? otherPane.currentPath : activePane.currentPath
        modalState.batch.copyOpen = true
        break

      case 'batchMove':
        modalState.batch.destDir =
          viewMode === 'dual' ? otherPane.currentPath : activePane.currentPath
        modalState.batch.moveOpen = true
        break

      case 'batchCompress':
        modalState.batch.zipName = 'archive.zip'
        modalState.batch.compressOpen = true
        break
    }
  }

  // Cross-Pane Transfer (Copy / Move between left and right)
  async function handleCrossPaneTransfer(paths: string[], action: 'copy' | 'move') {
    if (!paths || paths.length === 0) return
    try {
      await api.post<FileBatchPayload>('/api/files/batch', {
        action,
        paths,
        destination: otherPane.currentPath,
        dest_dir: otherPane.currentPath,
      })
      toastStore.success(
        `${action === 'copy' ? 'Copied' : 'Moved'} ${paths.length} item(s) to ${otherPane.currentPath}`
      )
      activePane.clearSelection()
      await Promise.all([leftPane.refresh(), rightPane.refresh()])
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Cross-pane transfer failed')
    }
  }

  // File Upload Handler
  async function handleUpload(files: FileList | File[]) {
    if (!files || files.length === 0) return
    isUploading = true
    const formData = new FormData()
    for (let i = 0; i < files.length; i++) {
      formData.append('files', files[i])
    }
    formData.append('path', activePane.currentPath)

    try {
      const res = await fetch(
        `/api/files/upload?path=${encodeURIComponent(activePane.currentPath)}`,
        {
          method: 'POST',
          body: formData,
          credentials: 'include',
        }
      )
      if (!res.ok) throw new Error((await res.text()) || 'Upload failed')
      toastStore.success(`Uploaded ${files.length} file(s) successfully`)
      await activePane.refresh()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Upload failed')
    } finally {
      isUploading = false
    }
  }

  // Modal Actions Handlers
  async function handleSaveEditor() {
    modalState.editor.isSaving = true
    try {
      await api.post<FileSavePayload>('/api/files/save', {
        path: modalState.editor.path,
        content: modalState.editor.content,
      })
      toastStore.success('File saved successfully')
      modalState.editor.isOpen = false
      await activePane.refresh()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to save file')
    } finally {
      modalState.editor.isSaving = false
    }
  }

  async function handleCreate() {
    const fullPath = `${activePane.currentPath.replace(/\/$/, '')}/${modalState.create.name.trim()}`
    try {
      await api.post('/api/files/create', {
        path: fullPath,
        is_dir: modalState.create.type === 'folder',
      })
      toastStore.success(`Created ${modalState.create.type}: ${modalState.create.name}`)
      modalState.create.isOpen = false
      await activePane.refresh()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to create item')
    }
  }

  async function handleRename() {
    if (!modalState.rename.target) return
    try {
      await api.post<FileRenamePayload>('/api/files/rename', {
        old_path: modalState.rename.target.path,
        new_path: `${activePane.currentPath.replace(/\/$/, '')}/${modalState.rename.newName.trim()}`,
      })
      toastStore.success('Renamed successfully')
      modalState.rename.isOpen = false
      await activePane.refresh()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to rename item')
    }
  }

  async function handleDelete() {
    if (!modalState.delete.target) return
    try {
      await api.delete(
        `/api/files/delete?path=${encodeURIComponent(modalState.delete.target.path)}`
      )
      toastStore.success('Deleted successfully')
      modalState.delete.isOpen = false
      await activePane.refresh()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to delete item')
    }
  }

  async function handleChmod() {
    if (!modalState.chmod.target) return
    try {
      await api.post<FilePermissionsPayload>('/api/files/permissions', {
        path: modalState.chmod.target.path,
        mode: modalState.chmod.mode,
        owner: modalState.chmod.owner,
      })
      toastStore.success('Permissions updated')
      modalState.chmod.isOpen = false
      await activePane.refresh()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to update permissions')
    }
  }

  async function handleCompress() {
    if (!modalState.compress.target) return
    try {
      const dest = `${activePane.currentPath.replace(/\/$/, '')}/${modalState.compress.destName.trim()}`
      await api.post<FileCompressPayload>('/api/files/compress', {
        paths: [modalState.compress.target.path],
        destination: dest,
        dest_zip: dest,
      })
      toastStore.success('Compressed successfully')
      modalState.compress.isOpen = false
      await activePane.refresh()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Compression failed')
    }
  }

  async function handleExtract() {
    if (!modalState.extract.target) return
    try {
      await api.post<FileExtractPayload>('/api/files/extract', {
        path: modalState.extract.target.path,
        zip_path: modalState.extract.target.path,
        destination: modalState.extract.destDir.trim(),
        dest_dir: modalState.extract.destDir.trim(),
      })
      toastStore.success('Archive extracted successfully')
      modalState.extract.isOpen = false
      await activePane.refresh()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Extraction failed')
    }
  }

  async function handleCopyMove() {
    if (!modalState.copyMove.target) return
    const action = modalState.copyMove.action
    try {
      await api.post<FileBatchPayload>('/api/files/batch', {
        action,
        paths: [modalState.copyMove.target.path],
        destination: modalState.copyMove.destPath.trim(),
        dest_dir: modalState.copyMove.destPath.trim(),
      })
      toastStore.success(`${action === 'copy' ? 'Copied' : 'Moved'} successfully`)
      modalState.copyMove.isOpen = false
      await Promise.all([leftPane.refresh(), rightPane.refresh()])
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : `${action} failed`)
    }
  }

  async function handleBatchDelete() {
    isBatchProcessing = true
    try {
      await api.post<FileBatchPayload>('/api/files/batch', {
        action: 'delete',
        paths: activePane.selectedPaths,
      })
      toastStore.success(`Deleted ${activePane.selectedPaths.length} items`)
      activePane.clearSelection()
      modalState.batch.deleteOpen = false
      await activePane.refresh()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Batch delete failed')
    } finally {
      isBatchProcessing = false
    }
  }

  async function handleBatchCopyMove(action: 'copy' | 'move') {
    isBatchProcessing = true
    try {
      const dest = modalState.batch.destDir.trim()
      await api.post<FileBatchPayload>('/api/files/batch', {
        action,
        paths: activePane.selectedPaths,
        destination: dest,
        dest_dir: dest,
      })
      toastStore.success(
        `${action === 'copy' ? 'Copied' : 'Moved'} ${activePane.selectedPaths.length} items`
      )
      activePane.clearSelection()
      modalState.batch.copyOpen = false
      modalState.batch.moveOpen = false
      await Promise.all([leftPane.refresh(), rightPane.refresh()])
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : `Batch ${action} failed`)
    } finally {
      isBatchProcessing = false
    }
  }

  async function handleBatchCompress() {
    isBatchProcessing = true
    try {
      const dest = `${activePane.currentPath.replace(/\/$/, '')}/${modalState.batch.zipName.trim()}`
      await api.post<FileCompressPayload>('/api/files/compress', {
        paths: activePane.selectedPaths,
        destination: dest,
        dest_zip: dest,
      })
      toastStore.success(
        `Compressed ${activePane.selectedPaths.length} items into ${modalState.batch.zipName}`
      )
      activePane.clearSelection()
      modalState.batch.compressOpen = false
      await activePane.refresh()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Batch compress failed')
    } finally {
      isBatchProcessing = false
    }
  }
</script>

<div class="space-y-4 font-mono">
  <!-- Top Navigation & View Mode Header -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-1 border-b border-border">
    <div class="flex items-center gap-2">
      <FolderOpen class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-bold uppercase tracking-wider text-foreground">
        Root File Explorer
      </h2>
      <Badge variant="default" class="text-[10px]">
        {viewMode === 'dual' ? 'Dual Pane' : 'Single Pane'}
      </Badge>
    </div>

    <!-- Mode Switcher Buttons: [ Single View ] [ Dual Pane ] -->
    <div class="flex items-center gap-1 bg-card-sub border-2 border-border p-0.5 rounded">
      <button
        type="button"
        class="inline-flex items-center gap-1.5 px-3 py-1 rounded text-xs font-bold transition-all cursor-pointer {viewMode === 'single' ? 'bg-accent text-accent-text shadow-neobrutal-sm' : 'text-muted hover:text-foreground'}"
        onclick={() => setViewMode('single')}
        title="Single Pane Mode"
      >
        <Square class="w-3.5 h-3.5" />
        <span>Single View</span>
      </button>

      <button
        type="button"
        class="inline-flex items-center gap-1.5 px-3 py-1 rounded text-xs font-bold transition-all cursor-pointer {viewMode === 'dual' ? 'bg-accent text-accent-text shadow-neobrutal-sm' : 'text-muted hover:text-foreground'}"
        onclick={() => setViewMode('dual')}
        title="Dual-Pane Split View"
      >
        <Columns2 class="w-3.5 h-3.5" />
        <span>Dual Pane</span>
      </button>
    </div>
  </div>

  <!-- Mobile Dual-Pane Switcher Pills (shown only on small screens when Dual Mode is active) -->
  {#if viewMode === 'dual'}
    <div class="flex lg:hidden items-center gap-2 p-1 bg-card-sub border border-border rounded font-mono text-xs">
      <button
        type="button"
        class="flex-1 py-1.5 px-2 text-center rounded font-bold transition-all cursor-pointer truncate {activePaneId === 'left' ? 'bg-accent text-accent-text shadow-neobrutal-sm' : 'text-muted hover:text-foreground'}"
        onclick={() => (activePaneId = 'left')}
      >
        Panel A: {leftPane.currentPath.split('/').filter(Boolean).pop() || '/'}
      </button>
      <button
        type="button"
        class="flex-1 py-1.5 px-2 text-center rounded font-bold transition-all cursor-pointer truncate {activePaneId === 'right' ? 'bg-accent text-accent-text shadow-neobrutal-sm' : 'text-muted hover:text-foreground'}"
        onclick={() => (activePaneId = 'right')}
      >
        Panel B: {rightPane.currentPath.split('/').filter(Boolean).pop() || '/'}
      </button>
    </div>
  {/if}

  <!-- Panes Grid Container -->
  {#if viewMode === 'single'}
    <FilePane
      pane={leftPane}
      paneLabel="Primary"
      isActive={true}
      hasOtherPane={false}
      onOpenModal={handleOpenModal}
      onUpload={handleUpload}
    />
  {:else}
    <!-- Desktop Dual-Pane Grid (Side by Side) -->
    <div class="hidden lg:grid lg:grid-cols-2 gap-4 items-start">
      <FilePane
        pane={leftPane}
        paneLabel="Panel A"
        isActive={activePaneId === 'left'}
        hasOtherPane={true}
        otherPanePath={rightPane.currentPath}
        onActivate={() => (activePaneId = 'left')}
        onOpenModal={(action, target, extra) => {
          activePaneId = 'left'
          handleOpenModal(action, target, extra)
        }}
        onTransferToOtherPane={handleCrossPaneTransfer}
        onUpload={handleUpload}
      />

      <FilePane
        pane={rightPane}
        paneLabel="Panel B"
        isActive={activePaneId === 'right'}
        hasOtherPane={true}
        otherPanePath={leftPane.currentPath}
        onActivate={() => (activePaneId = 'right')}
        onOpenModal={(action, target, extra) => {
          activePaneId = 'right'
          handleOpenModal(action, target, extra)
        }}
        onTransferToOtherPane={handleCrossPaneTransfer}
        onUpload={handleUpload}
      />
    </div>

    <!-- Mobile Dual-Pane View (Tabs between Panel A & B) -->
    <div class="block lg:hidden">
      <FilePane
        pane={activePane}
        paneLabel={activePaneId === 'left' ? 'Panel A' : 'Panel B'}
        isActive={true}
        hasOtherPane={true}
        otherPanePath={otherPane.currentPath}
        onOpenModal={handleOpenModal}
        onTransferToOtherPane={handleCrossPaneTransfer}
        onUpload={handleUpload}
      />
    </div>
  {/if}
</div>

<!-- Global Dropzone Overlay (Uploads to Active Pane Target) -->
<FileDropzone
  currentPath={activePane.currentPath}
  onUpload={handleUpload}
  isUploading={isUploading}
/>

<!-- Consolidated Modal Dialogs -->
<FileModals
  modalState={modalState}
  selectedPaths={activePane.selectedPaths}
  isBatchProcessing={isBatchProcessing}
  onSaveEditor={handleSaveEditor}
  onCreate={handleCreate}
  onRename={handleRename}
  onDelete={handleDelete}
  onChmod={handleChmod}
  onCompress={handleCompress}
  onExtract={handleExtract}
  onCopyMove={handleCopyMove}
  onBatchDelete={handleBatchDelete}
  onBatchCopy={() => handleBatchCopyMove('copy')}
  onBatchMove={() => handleBatchCopyMove('move')}
  onBatchCompress={handleBatchCompress}
/>
