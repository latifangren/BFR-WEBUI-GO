<script lang="ts">
  import { onMount } from 'svelte'
  import { Plus, X, Pin, Folder } from '@lucide/svelte'
  import Modal from '../../ui/Modal.svelte'
  import Input from '../../ui/Input.svelte'
  import Button from '../../ui/Button.svelte'
  import type { Bookmark as BookmarkItem } from './types'

  interface Props {
    currentPath: string
    onNavigate: (path: string) => void
  }

  let { currentPath, onNavigate }: Props = $props()

  const PRESETS: BookmarkItem[] = [
    { name: 'Root', path: '/', isPreset: true },
    { name: 'Storage', path: '/sdcard', isPreset: true },
    { name: 'Magisk', path: '/data/adb', isPreset: true },
    { name: 'Modules', path: '/data/adb/modules', isPreset: true },
    { name: 'Temp', path: '/data/local/tmp', isPreset: true },
  ]

  const STORAGE_KEY = 'bfr_fm_custom_bookmarks'

  let customBookmarks = $state<BookmarkItem[]>([])
  let isAddModalOpen = $state(false)
  let newBookmarkName = $state('')

  onMount(() => {
    loadBookmarks()
  })

  function loadBookmarks() {
    if (typeof window === 'undefined' || typeof localStorage === 'undefined') return
    try {
      const saved = localStorage.getItem(STORAGE_KEY)
      if (saved) {
        const parsed = JSON.parse(saved)
        if (Array.isArray(parsed)) {
          customBookmarks = parsed.map((item) => ({
            name: String(item.name || 'Folder'),
            path: String(item.path || '/'),
            isPreset: false,
          }))
        }
      }
    } catch {
      // Ignored
    }
  }

  function saveBookmarks() {
    if (typeof window === 'undefined' || typeof localStorage === 'undefined') return
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(customBookmarks))
    } catch {
      // Ignored
    }
  }

  function openAddModal() {
    const parts = currentPath.split('/').filter(Boolean)
    newBookmarkName = parts.length > 0 ? parts[parts.length - 1] : 'Root'
    isAddModalOpen = true
  }

  function addBookmark() {
    const trimmed = newBookmarkName.trim() || 'Folder'
    const exists = customBookmarks.some((b) => b.path === currentPath)
    if (!exists) {
      customBookmarks = [...customBookmarks, { name: trimmed, path: currentPath, isPreset: false }]
      saveBookmarks()
    }
    isAddModalOpen = false
  }

  function removeBookmark(path: string, e: MouseEvent) {
    e.stopPropagation()
    customBookmarks = customBookmarks.filter((b) => b.path !== path)
    saveBookmarks()
  }
</script>

<div class="flex items-center gap-1.5 overflow-x-auto py-1 font-mono text-xs no-scrollbar select-none">
  <!-- Presets -->
  {#each PRESETS as preset}
    {@const isActive = currentPath === preset.path}
    <button
      type="button"
      class="inline-flex items-center gap-1 px-2.5 py-1 rounded border-2 border-border transition-all cursor-pointer shrink-0 font-bold {isActive ? 'bg-accent text-accent-text border-border shadow-neobrutal-sm' : 'bg-card-sub text-foreground hover:bg-card hover:border-accent'}"
      onclick={() => onNavigate(preset.path)}
      title={`Go to ${preset.path}`}
    >
      <Folder class="w-3 h-3 opacity-70" />
      <span>{preset.name}</span>
    </button>
  {/each}

  <!-- Divider if custom bookmarks exist -->
  {#if customBookmarks.length > 0}
    <div class="h-4 w-px bg-border shrink-0 mx-0.5"></div>
  {/if}

  <!-- Custom Bookmarks -->
  {#each customBookmarks as bookmark}
    {@const isActive = currentPath === bookmark.path}
    <div
      class="inline-flex items-center gap-1 pl-2.5 pr-1.5 py-1 rounded border-2 border-border transition-all shrink-0 font-bold {isActive ? 'bg-accent text-accent-text border-border shadow-neobrutal-sm' : 'bg-card-sub text-foreground hover:bg-card hover:border-accent'}"
    >
      <button
        type="button"
        class="inline-flex items-center gap-1 cursor-pointer"
        onclick={() => onNavigate(bookmark.path)}
        title={bookmark.path}
      >
        <Pin class="w-2.5 h-2.5 opacity-70" />
        <span class="max-w-[120px] truncate">{bookmark.name}</span>
      </button>
      <button
        type="button"
        class="p-0.5 rounded-full hover:bg-red-500/20 hover:text-red-400 cursor-pointer ml-0.5 opacity-70 hover:opacity-100 transition-opacity"
        onclick={(e) => removeBookmark(bookmark.path, e)}
        title="Remove bookmark"
        aria-label={`Remove ${bookmark.name} bookmark`}
      >
        <X class="w-3 h-3" />
      </button>
    </div>
  {/each}

  <!-- Add Bookmark Button -->
  <button
    type="button"
    class="inline-flex items-center gap-1 px-2 py-1 rounded border-2 border-dashed border-border hover:border-accent text-muted hover:text-foreground transition-all cursor-pointer shrink-0"
    onclick={openAddModal}
    title="Bookmark current location"
  >
    <Plus class="w-3 h-3" />
    <span>Pin</span>
  </button>
</div>

{#if isAddModalOpen}
  <Modal
    open={true}
    title="Pin Directory Bookmark"
    onclose={() => (isAddModalOpen = false)}
  >
    <form onsubmit={(e) => { e.preventDefault(); addBookmark() }} class="space-y-4 font-mono text-xs">
      <div>
        <span class="text-[10px] text-muted uppercase font-bold block mb-1">Target Path</span>
        <code class="block p-2 rounded bg-card-sub border border-border text-accent font-bold truncate text-[11px]">
          {currentPath}
        </code>
      </div>
      <Input
        label="Bookmark Label"
        placeholder="e.g. Config, Downloads, Magisk Modules"
        bind:value={newBookmarkName}
      />
      <div class="flex justify-end gap-2 pt-2 border-t border-border">
        <Button variant="ghost" size="md" onclick={() => (isAddModalOpen = false)}>
          Cancel
        </Button>
        <Button type="submit" variant="primary" size="md" disabled={!newBookmarkName.trim()}>
          <Pin class="w-3.5 h-3.5 mr-1" />
          <span>Save Bookmark</span>
        </Button>
      </div>
    </form>
  </Modal>
{/if}
