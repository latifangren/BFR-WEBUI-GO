<script lang="ts">
  import {
    Folder,
    ArrowUp,
    RefreshCw,
    Plus,
    Upload,
    HardDrive,
    Search,
    X,
    ArrowRightLeft,
    Copy,
    FolderInput,
    FileArchive,
    Trash2,
    Edit2,
    ChevronRight,
  } from '@lucide/svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Input from '../../ui/Input.svelte'
  import Badge from '../../ui/Badge.svelte'
  import BookmarksBar from './BookmarksBar.svelte'
  import FileTable from './FileTable.svelte'
  import { PaneState } from './paneState.svelte'
  import type { FileEntry } from './types'

  interface Props {
    pane: PaneState
    paneLabel?: string
    isActive?: boolean
    hasOtherPane?: boolean
    otherPanePath?: string
    onActivate?: () => void
    onOpenModal: (action: string, target?: FileEntry, extra?: any) => void
    onTransferToOtherPane?: (paths: string[], action: 'copy' | 'move') => void
    onUpload: (files: FileList | File[]) => Promise<void>
  }

  let {
    pane,
    paneLabel,
    isActive = false,
    hasOtherPane = false,
    otherPanePath = '',
    onActivate,
    onOpenModal,
    onTransferToOtherPane,
    onUpload,
  }: Props = $props()

  let fileInputEl = $state<HTMLInputElement | null>(null)
  let isEditingPath = $state(false)
  let inputPath = $state(pane.currentPath)

  $effect(() => {
    inputPath = pane.currentPath
  })

  function submitPath() {
    isEditingPath = false
    const trimmed = inputPath.trim()
    if (trimmed && trimmed !== pane.currentPath) {
      pane.navigateTo(trimmed)
    }
  }

  function handleFileSelect(e: Event) {
    const input = e.target as HTMLInputElement
    if (input.files && input.files.length > 0) {
      onUpload(input.files)
      input.value = ''
    }
  }
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="flex flex-col h-full space-y-3 transition-all {isActive && hasOtherPane ? 'ring-2 ring-accent rounded-lg' : ''}"
  onclick={() => onActivate?.()}
>
  <Card tone="ice" class="p-3 sm:p-4 space-y-3">
    <!-- Header: Label, Breadcrumbs/Path, and Quick Navigation Controls -->
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-2.5 pb-2 border-b border-border">
      <!-- Breadcrumbs / Inline Path Editor -->
      <div class="flex items-center gap-1.5 flex-1 min-w-0 font-mono text-xs">
        {#if paneLabel}
          <span class="px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider shrink-0 {isActive ? 'bg-accent text-accent-text shadow-neobrutal-sm' : 'bg-card-sub text-muted border border-border'}">
            {paneLabel}
          </span>
        {/if}

        {#if isEditingPath}
          <form onsubmit={(e) => { e.preventDefault(); submitPath() }} class="flex-1 flex items-center gap-1">
            <input
              type="text"
              bind:value={inputPath}
              onblur={submitPath}
              onkeydown={(e) => { if (e.key === 'Escape') isEditingPath = false }}
              class="neo-input w-full bg-card-sub border-2 border-accent rounded px-2 py-1 text-xs font-mono text-foreground focus:outline-none"
              placeholder="/sdcard"
              autofocus
            />
          </form>
        {:else}
          <div class="flex items-center gap-1 overflow-x-auto no-scrollbar py-0.5 max-w-full">
            <button
              type="button"
              class="p-1 rounded hover:bg-card text-foreground font-bold hover:text-accent cursor-pointer shrink-0"
              onclick={() => pane.navigateTo('/')}
              title="Root directory (/)"
            >
              /
            </button>

            {#each pane.breadcrumbs as crumb, idx}
              <ChevronRight class="w-3 h-3 text-muted shrink-0 opacity-60" />
              <button
                type="button"
                class="p-1 rounded hover:bg-card text-foreground hover:text-accent font-bold cursor-pointer shrink-0 max-w-[140px] truncate"
                onclick={() => pane.navigateTo(crumb.path)}
                title={crumb.path}
              >
                {crumb.name}
              </button>
            {/each}

            <button
              type="button"
              class="p-1 text-muted hover:text-foreground cursor-pointer shrink-0 opacity-60 hover:opacity-100 ml-1"
              onclick={() => { isEditingPath = true }}
              title="Edit path directly"
            >
              <Edit2 class="w-3 h-3" />
            </button>
          </div>
        {/if}
      </div>

      <!-- Action Buttons Toolbar -->
      <div class="flex items-center gap-1.5 shrink-0 self-end md:self-auto font-mono text-xs">
        <!-- Hidden file input for native upload -->
        <input
          type="file"
          multiple
          class="hidden"
          bind:this={fileInputEl}
          onchange={handleFileSelect}
        />

        <Button
          variant="outline"
          size="sm"
          disabled={pane.currentPath === '/' || pane.isLoading}
          onclick={() => pane.goUp()}
          title="Parent Directory (Up)"
        >
          <ArrowUp class="w-3.5 h-3.5 mr-1" />
          <span class="hidden sm:inline">Up</span>
        </Button>

        <Button
          variant="outline"
          size="sm"
          disabled={pane.isLoading}
          onclick={() => pane.refresh()}
          title="Refresh Directory"
        >
          <RefreshCw class="w-3.5 h-3.5 {pane.isLoading ? 'animate-spin' : ''}" />
        </Button>

        <Button
          variant="secondary"
          size="sm"
          onclick={() => onOpenModal('create', undefined, { type: 'file' })}
          title="Create New File"
        >
          <Plus class="w-3.5 h-3.5 mr-1 text-accent" />
          <span class="hidden sm:inline">File</span>
        </Button>

        <Button
          variant="secondary"
          size="sm"
          onclick={() => onOpenModal('create', undefined, { type: 'folder' })}
          title="Create New Folder"
        >
          <Plus class="w-3.5 h-3.5 mr-1 text-accent" />
          <span class="hidden sm:inline">Folder</span>
        </Button>

        <Button
          variant="primary"
          size="sm"
          disabled={pane.isLoading}
          onclick={() => fileInputEl?.click()}
          title="Upload Files"
        >
          <Upload class="w-3.5 h-3.5 mr-1" />
          <span>Upload</span>
        </Button>
      </div>
    </div>

    <!-- Quick Bookmarks Bar -->
    <BookmarksBar
      currentPath={pane.currentPath}
      onNavigate={(path) => pane.navigateTo(path)}
    />

    <!-- Storage Volume Capacity Bar -->
    {#if pane.storageInfo}
      <div class="flex items-center justify-between gap-3 px-3 py-1.5 bg-card-sub border border-border rounded font-mono text-[11px] text-muted">
        <div class="flex items-center gap-1.5 truncate">
          <HardDrive class="w-3.5 h-3.5 text-accent shrink-0" />
          <span class="truncate font-bold text-foreground">{pane.storageInfo.mount || '/sdcard'}</span>
          <span>({pane.storageInfo.used_pct || 0}% used)</span>
        </div>
        <div class="flex items-center gap-2 shrink-0">
          <div class="w-20 sm:w-28 bg-card border border-border h-2 rounded overflow-hidden">
            <div
              class="h-full bg-accent transition-all duration-300"
              style="width: {pane.storageInfo.used_pct || 0}%"
            ></div>
          </div>
          <span class="text-[10px] hidden sm:inline">
            {PaneState.formatBytes(pane.storageInfo.used || 0)} / {PaneState.formatBytes(pane.storageInfo.total || 0)}
          </span>
        </div>
      </div>
    {/if}

    <!-- Search Input Bar -->
    <div class="flex items-center gap-2 font-mono text-xs">
      <div class="relative flex-1">
        <Input
          placeholder="Search directory..."
          bind:value={pane.searchQuery}
          onkeydown={(e: KeyboardEvent) => { if (e.key === 'Enter') pane.runSearch() }}
        />
        {#if pane.searchQuery}
          <button
            type="button"
            class="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted hover:text-foreground cursor-pointer"
            onclick={() => pane.clearSearch()}
          >
            <X class="w-3.5 h-3.5" />
          </button>
        {/if}
      </div>
      <Button
        variant="secondary"
        size="sm"
        disabled={pane.isSearching || !pane.searchQuery.trim()}
        onclick={() => pane.runSearch()}
      >
        {#if pane.isSearching}
          <RefreshCw class="w-3 h-3 mr-1 animate-spin" />
          <span>Searching</span>
        {:else}
          <Search class="w-3 h-3 mr-1" />
          <span>Search</span>
        {/if}
      </Button>
    </div>

    <!-- Active Search Filter Banner -->
    {#if pane.searchResults !== null}
      <div class="flex items-center justify-between p-2 bg-accent/10 border border-accent/30 rounded font-mono text-xs text-accent">
        <span>Found {pane.searchResults.length} entries matching query "{pane.searchQuery}"</span>
        <button
          type="button"
          class="text-xs font-bold hover:underline cursor-pointer"
          onclick={() => pane.clearSearch()}
        >
          Clear Search
        </button>
      </div>
    {/if}

    <!-- File Table Component -->
    <FileTable
      pane={pane}
      hasOtherPane={hasOtherPane}
      otherPanePath={otherPanePath}
      onOpenEditor={(file) => onOpenModal('editor', file)}
      onRename={(file) => onOpenModal('rename', file)}
      onDelete={(file) => onOpenModal('delete', file)}
      onPermissions={(file) => onOpenModal('chmod', file)}
      onCompress={(file) => onOpenModal('compress', file)}
      onExtract={(file) => onOpenModal('extract', file)}
      onCopyMove={(file, action) => onOpenModal('copyMove', file, { action })}
      onTransferToOtherPane={onTransferToOtherPane}
    />

    <!-- Sticky Batch Action Bar (Appears when items are selected) -->
    {#if pane.selectedPaths.length > 0}
      <div
        class="sticky bottom-2 z-30 bg-card border-2 border-border shadow-neobrutal p-2.5 rounded-lg flex flex-wrap items-center justify-between gap-2 font-mono text-xs animate-in fade-in"
      >
        <div class="flex items-center gap-2">
          <span class="px-2 py-0.5 bg-accent text-accent-text font-black rounded text-[11px] shadow-neobrutal-sm">
            {pane.selectedPaths.length} selected
          </span>
          <Button variant="ghost" size="sm" onclick={() => pane.clearSelection()}>
            <X class="w-3 h-3 mr-1" />
            <span>Clear</span>
          </Button>
        </div>

        <div class="flex flex-wrap items-center gap-1.5">
          {#if hasOtherPane && onTransferToOtherPane}
            <Button
              variant="primary"
              size="sm"
              onclick={() => onTransferToOtherPane?.(pane.selectedPaths, 'copy')}
              title={`Copy to other pane (${otherPanePath || 'secondary'})`}
            >
              <ArrowRightLeft class="w-3 h-3 mr-1" />
              <span>Copy to {otherPanePath ? otherPanePath.split('/').filter(Boolean).pop() || 'Other' : 'Other'}</span>
            </Button>
            <Button
              variant="outline"
              size="sm"
              onclick={() => onTransferToOtherPane?.(pane.selectedPaths, 'move')}
              title={`Move to other pane (${otherPanePath || 'secondary'})`}
            >
              <FolderInput class="w-3 h-3 mr-1 text-emerald-400" />
              <span>Move to {otherPanePath ? otherPanePath.split('/').filter(Boolean).pop() || 'Other' : 'Other'}</span>
            </Button>
          {/if}

          <Button
            variant="secondary"
            size="sm"
            onclick={() => onOpenModal('batchCopy')}
          >
            <Copy class="w-3 h-3 mr-1" />
            <span>Copy</span>
          </Button>

          <Button
            variant="secondary"
            size="sm"
            onclick={() => onOpenModal('batchMove')}
          >
            <FolderInput class="w-3 h-3 mr-1" />
            <span>Move</span>
          </Button>

          <Button
            variant="secondary"
            size="sm"
            onclick={() => onOpenModal('batchCompress')}
          >
            <FileArchive class="w-3 h-3 mr-1 text-accent" />
            <span>Compress</span>
          </Button>

          <Button
            variant="danger"
            size="sm"
            onclick={() => onOpenModal('batchDelete')}
          >
            <Trash2 class="w-3 h-3 mr-1" />
            <span>Delete</span>
          </Button>
        </div>
      </div>
    {/if}
  </Card>
</div>
