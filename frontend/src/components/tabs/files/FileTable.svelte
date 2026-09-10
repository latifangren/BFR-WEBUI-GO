<script lang="ts">
  import {
    Folder,
    FolderOpen,
    FileText,
    FileArchive,
    FileCode,
    FileImage,
    File,
    Download,
    Edit2,
    Trash2,
    Copy,
    FolderInput,
    Shield,
    RefreshCw,
    ArrowUp,
    ArrowDown,
    ArrowUpDown,
    ArrowRightLeft,
  } from '@lucide/svelte'
  import type { FileEntry } from './types'
  import type { PaneState } from './paneState.svelte'

  interface Props {
    pane: PaneState
    onOpenEditor: (file: FileEntry) => void
    onRename: (file: FileEntry) => void
    onDelete: (file: FileEntry) => void
    onPermissions: (file: FileEntry) => void
    onCompress: (file: FileEntry) => void
    onExtract: (file: FileEntry) => void
    onCopyMove: (file: FileEntry, action: 'copy' | 'move') => void
    onTransferToOtherPane?: (paths: string[], action: 'copy' | 'move') => void
    hasOtherPane?: boolean
    otherPanePath?: string
  }

  let {
    pane,
    onOpenEditor,
    onRename,
    onDelete,
    onPermissions,
    onCompress,
    onExtract,
    onCopyMove,
    onTransferToOtherPane,
    hasOtherPane = false,
    otherPanePath = '',
  }: Props = $props()

  function isArchive(file: FileEntry): boolean {
    if (file.is_dir) return false
    const ext = file.name.split('.').pop()?.toLowerCase() || ''
    return ['zip', 'tar', 'gz', 'tgz', 'bz2', 'xz', '7z', 'apk'].includes(ext)
  }

  function isCodeOrText(file: FileEntry): boolean {
    if (file.is_dir) return false
    const ext = file.name.split('.').pop()?.toLowerCase() || ''
    return [
      'txt', 'md', 'json', 'yaml', 'yml', 'xml', 'html', 'css', 'js', 'ts',
      'sh', 'bash', 'conf', 'ini', 'prop', 'log', 'env', 'rc', 'toml', 'go',
    ].includes(ext)
  }

  function isImage(file: FileEntry): boolean {
    if (file.is_dir) return false
    const ext = file.name.split('.').pop()?.toLowerCase() || ''
    return ['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg', 'ico'].includes(ext)
  }

  function formatBytes(bytes: number): string {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`
  }

  function downloadFile(file: FileEntry) {
    window.open(`/api/files/download?path=${encodeURIComponent(file.path)}`, '_blank')
  }
</script>

<div class="overflow-x-auto select-none">
  {#if pane.isLoading && pane.files.length === 0}
    <div class="p-12 text-center text-muted font-mono text-xs">
      <RefreshCw class="w-6 h-6 animate-spin mx-auto mb-2 text-accent" />
      <span>Loading filesystem entries...</span>
    </div>
  {:else if pane.displayedFiles.length === 0}
    <div class="p-12 text-center text-muted font-mono text-xs">
      <FolderOpen class="w-6 h-6 mx-auto mb-2 opacity-50" />
      <span>
        {pane.searchResults !== null
          ? 'No files match your search query.'
          : 'This directory is empty.'}
      </span>
    </div>
  {:else}
    <table class="w-full text-left font-mono text-xs">
      <thead>
        <tr class="border-b-2 border-border text-muted uppercase text-[10px] bg-card-sub">
          <!-- Selection Column -->
          <th class="py-2.5 px-3 w-10 text-center">
            <input
              type="checkbox"
              checked={pane.allSelected}
              onchange={() => pane.toggleSelectAll()}
              class="w-4 h-4 accent-accent rounded cursor-pointer"
              title={pane.allSelected ? 'Deselect All' : 'Select All'}
            />
          </th>

          <!-- Name Column (Sortable) -->
          <th
            class="py-2.5 px-3 cursor-pointer select-none hover:text-foreground transition-colors"
            onclick={() => pane.setSort('name')}
            title="Sort by Name"
          >
            <div class="flex items-center gap-1">
              <span>Name</span>
              {#if pane.sortField === 'name'}
                {#if pane.sortOrder === 'asc'}
                  <ArrowUp class="w-3 h-3 text-accent" />
                {:else}
                  <ArrowDown class="w-3 h-3 text-accent" />
                {/if}
              {:else}
                <ArrowUpDown class="w-3 h-3 opacity-30" />
              {/if}
            </div>
          </th>

          <!-- Size Column (Sortable) -->
          <th
            class="py-2.5 px-3 w-24 cursor-pointer select-none hover:text-foreground transition-colors"
            onclick={() => pane.setSort('size')}
            title="Sort by Size"
          >
            <div class="flex items-center gap-1">
              <span>Size</span>
              {#if pane.sortField === 'size'}
                {#if pane.sortOrder === 'asc'}
                  <ArrowUp class="w-3 h-3 text-accent" />
                {:else}
                  <ArrowDown class="w-3 h-3 text-accent" />
                {/if}
              {:else}
                <ArrowUpDown class="w-3 h-3 opacity-30" />
              {/if}
            </div>
          </th>

          <!-- Perms Column (Desktop/Tablet) -->
          <th class="py-2.5 px-3 w-28 hidden sm:table-cell">Perms</th>

          <!-- Modified Column (Desktop, Sortable) -->
          <th
            class="py-2.5 px-3 w-36 hidden md:table-cell cursor-pointer select-none hover:text-foreground transition-colors"
            onclick={() => pane.setSort('modified')}
            title="Sort by Date Modified"
          >
            <div class="flex items-center gap-1">
              <span>Modified</span>
              {#if pane.sortField === 'modified'}
                {#if pane.sortOrder === 'asc'}
                  <ArrowUp class="w-3 h-3 text-accent" />
                {:else}
                  <ArrowDown class="w-3 h-3 text-accent" />
                {/if}
              {:else}
                <ArrowUpDown class="w-3 h-3 opacity-30" />
              {/if}
            </div>
          </th>

          <!-- Actions Column -->
          <th class="py-2.5 px-3 text-right">Actions</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-border">
        {#each pane.displayedFiles as file (file.path)}
          {@const isSelected = pane.selectedPaths.includes(file.path)}
          <tr class="transition-colors {isSelected ? 'bg-accent/15' : 'hover:bg-card-sub/60'}">
            <!-- Row Checkbox -->
            <td class="py-2.5 px-3 w-10 text-center">
              <input
                type="checkbox"
                checked={isSelected}
                onchange={() => pane.toggleSelect(file.path)}
                class="w-4 h-4 accent-accent rounded cursor-pointer"
              />
            </td>

            <!-- Name & Icon -->
            <td class="py-2.5 px-3">
              {#if file.is_dir}
                <button
                  type="button"
                  class="flex items-center gap-2 text-foreground font-bold hover:text-accent cursor-pointer truncate max-w-xs sm:max-w-md text-left"
                  onclick={() => pane.navigateTo(file.path)}
                  title={file.name}
                >
                  <Folder class="w-4 h-4 text-accent shrink-0" />
                  <span class="truncate">{file.name}</span>
                </button>
              {:else}
                <button
                  type="button"
                  class="flex items-center gap-2 text-foreground hover:text-accent cursor-pointer truncate max-w-xs sm:max-w-md text-left"
                  onclick={() => onOpenEditor(file)}
                  title={file.name}
                >
                  {#if isArchive(file)}
                    <FileArchive class="w-4 h-4 text-amber-400 shrink-0" />
                  {:else if isImage(file)}
                    <FileImage class="w-4 h-4 text-purple-400 shrink-0" />
                  {:else if isCodeOrText(file)}
                    <FileCode class="w-4 h-4 text-cyan-400 shrink-0" />
                  {:else}
                    <FileText class="w-4 h-4 text-muted shrink-0" />
                  {/if}
                  <span class="truncate">{file.name}</span>
                </button>
              {/if}
            </td>

            <!-- File Size -->
            <td class="py-2.5 px-3 text-muted text-[11px]">
              {file.is_dir ? '—' : formatBytes(file.size)}
            </td>

            <!-- Permissions -->
            <td class="py-2.5 px-3 text-muted hidden sm:table-cell text-[11px]">
              <button
                type="button"
                class="hover:text-accent hover:underline cursor-pointer"
                onclick={() => onPermissions(file)}
                title="Change permissions"
              >
                {file.permissions || (file.is_dir ? '0755' : '0644')}
              </button>
            </td>

            <!-- Modified Time -->
            <td class="py-2.5 px-3 text-muted hidden md:table-cell text-[11px]">
              {file.mod_time ? new Date(file.mod_time).toLocaleDateString() : '—'}
            </td>

            <!-- Row Actions -->
            <td class="py-2.5 px-3 text-right">
              <div class="flex items-center justify-end gap-1">
                <!-- Dual Pane Quick Transfer -->
                {#if hasOtherPane && onTransferToOtherPane}
                  <button
                    type="button"
                    class="p-1 rounded hover:bg-card text-muted hover:text-cyan-400 transition-colors cursor-pointer"
                    onclick={() => onTransferToOtherPane([file.path], 'copy')}
                    title={`Copy to other pane (${otherPanePath || 'secondary'})`}
                    aria-label="Copy to other pane"
                  >
                    <ArrowRightLeft class="w-3.5 h-3.5" />
                  </button>
                {/if}

                <!-- Extract Archive -->
                {#if isArchive(file)}
                  <button
                    type="button"
                    class="p-1 rounded hover:bg-card text-muted hover:text-amber-400 transition-colors cursor-pointer"
                    onclick={() => onExtract(file)}
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
                  onclick={() => onCompress(file)}
                  title="Compress into ZIP"
                  aria-label="Compress"
                >
                  <FolderInput class="w-3.5 h-3.5" />
                </button>

                <!-- Copy Item -->
                <button
                  type="button"
                  class="p-1 rounded hover:bg-card text-muted hover:text-purple-400 transition-colors cursor-pointer"
                  onclick={() => onCopyMove(file, 'copy')}
                  title="Copy"
                  aria-label="Copy"
                >
                  <Copy class="w-3.5 h-3.5" />
                </button>

                <!-- Move Item -->
                <button
                  type="button"
                  class="p-1 rounded hover:bg-card text-muted hover:text-emerald-400 transition-colors cursor-pointer"
                  onclick={() => onCopyMove(file, 'move')}
                  title="Move"
                  aria-label="Move"
                >
                  <FolderInput class="w-3.5 h-3.5" />
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
                    onclick={() => onOpenEditor(file)}
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
                  onclick={() => onPermissions(file)}
                  title="Permissions (Chmod)"
                  aria-label="Permissions"
                >
                  <Shield class="w-3.5 h-3.5" />
                </button>

                <!-- Rename Item -->
                <button
                  type="button"
                  class="p-1 rounded hover:bg-card text-muted hover:text-foreground transition-colors cursor-pointer"
                  onclick={() => onRename(file)}
                  title="Rename"
                  aria-label="Rename"
                >
                  <Edit2 class="w-3.5 h-3.5" />
                </button>

                <!-- Delete Item -->
                <button
                  type="button"
                  class="p-1 rounded hover:bg-card text-muted hover:text-red-400 transition-colors cursor-pointer"
                  onclick={() => onDelete(file)}
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
  {/if}
</div>
