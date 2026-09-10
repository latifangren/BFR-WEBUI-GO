<script lang="ts">
  import {
    RefreshCw,
    Save,
    Plus,
    Check,
    Trash2,
    FileArchive,
    Copy,
    FolderInput,
  } from '@lucide/svelte'
  import Modal from '../../ui/Modal.svelte'
  import Button from '../../ui/Button.svelte'
  import Input from '../../ui/Input.svelte'
  import type { FileModalState } from './types'

  interface Props {
    modalState: FileModalState
    selectedPaths?: string[]
    isBatchProcessing?: boolean
    onSaveEditor?: () => Promise<void> | void
    onCreate?: () => Promise<void> | void
    onRename?: () => Promise<void> | void
    onDelete?: () => Promise<void> | void
    onChmod?: () => Promise<void> | void
    onCompress?: () => Promise<void> | void
    onExtract?: () => Promise<void> | void
    onCopyMove?: () => Promise<void> | void
    onBatchDelete?: () => Promise<void> | void
    onBatchCopy?: () => Promise<void> | void
    onBatchMove?: () => Promise<void> | void
    onBatchCompress?: () => Promise<void> | void
  }

  let {
    modalState,
    selectedPaths = [],
    isBatchProcessing = false,
    onSaveEditor,
    onCreate,
    onRename,
    onDelete,
    onChmod,
    onCompress,
    onExtract,
    onCopyMove,
    onBatchDelete,
    onBatchCopy,
    onBatchMove,
    onBatchCompress,
  }: Props = $props()
</script>

<!-- 1. Text File Editor Modal -->
{#if modalState.editor.isOpen}
  <Modal
    open={true}
    title={`Editing: ${modalState.editor.path}`}
    class="max-w-4xl"
    onclose={() => (modalState.editor.isOpen = false)}
  >
    <div class="space-y-4 font-mono text-xs">
      {#if modalState.editor.isLoading}
        <div class="p-12 text-center text-muted">
          <RefreshCw class="w-6 h-6 animate-spin mx-auto mb-2 text-accent" />
          <span>Reading file content...</span>
        </div>
      {:else}
        <textarea
          bind:value={modalState.editor.content}
          rows="18"
          class="w-full bg-card-sub border-2 border-border rounded p-3 font-mono text-xs text-foreground focus:outline-none focus:border-accent resize-y"
          placeholder="File is empty"
        ></textarea>
      {/if}

      <div class="flex items-center justify-between pt-2 border-t border-border">
        <span class="text-muted text-[11px] truncate max-w-sm">
          {modalState.editor.path}
        </span>
        <div class="flex items-center gap-2">
          <Button variant="ghost" size="md" onclick={() => (modalState.editor.isOpen = false)}>
            Cancel
          </Button>
          <Button
            variant="primary"
            size="md"
            disabled={modalState.editor.isSaving || modalState.editor.isLoading}
            onclick={() => onSaveEditor?.()}
          >
            {#if modalState.editor.isSaving}
              <RefreshCw class="w-3.5 h-3.5 mr-1.5 animate-spin" />
              <span>Saving...</span>
            {:else}
              <Save class="w-3.5 h-3.5 mr-1.5" />
              <span>Save File</span>
            {/if}
          </Button>
        </div>
      </div>
    </div>
  </Modal>
{/if}

<!-- 2. Create Item Modal -->
{#if modalState.create.isOpen}
  <Modal
    open={true}
    title={`Create New ${modalState.create.type === 'file' ? 'File' : 'Folder'}`}
    onclose={() => (modalState.create.isOpen = false)}
  >
    <form
      onsubmit={(e) => {
        e.preventDefault()
        onCreate?.()
      }}
      class="space-y-4 font-mono text-xs"
    >
      <div class="flex items-center gap-2 p-1 bg-card-sub border border-border rounded">
        <button
          type="button"
          class="flex-1 py-1.5 text-center font-bold rounded transition-colors {modalState.create.type === 'file' ? 'bg-accent text-accent-text' : 'text-muted hover:text-foreground'}"
          onclick={() => (modalState.create.type = 'file')}
        >
          File
        </button>
        <button
          type="button"
          class="flex-1 py-1.5 text-center font-bold rounded transition-colors {modalState.create.type === 'folder' ? 'bg-accent text-accent-text' : 'text-muted hover:text-foreground'}"
          onclick={() => (modalState.create.type = 'folder')}
        >
          Folder / Directory
        </button>
      </div>

      <Input
        label="Name"
        placeholder={modalState.create.type === 'file' ? 'example.conf' : 'new_folder'}
        bind:value={modalState.create.name}
      />
      <div class="flex justify-end gap-2 pt-2 border-t border-border">
        <Button variant="ghost" size="md" onclick={() => (modalState.create.isOpen = false)}>
          Cancel
        </Button>
        <Button type="submit" variant="primary" size="md" disabled={!modalState.create.name.trim()}>
          <Plus class="w-3.5 h-3.5 mr-1" />
          <span>Create</span>
        </Button>
      </div>
    </form>
  </Modal>
{/if}

<!-- 3. Rename Modal -->
{#if modalState.rename.isOpen && modalState.rename.target}
  <Modal
    open={true}
    title={`Rename: ${modalState.rename.target.name}`}
    onclose={() => (modalState.rename.isOpen = false)}
  >
    <form
      onsubmit={(e) => {
        e.preventDefault()
        onRename?.()
      }}
      class="space-y-4 font-mono text-xs"
    >
      <Input
        label="New Name"
        bind:value={modalState.rename.newName}
      />
      <div class="flex justify-end gap-2 pt-2 border-t border-border">
        <Button variant="ghost" size="md" onclick={() => (modalState.rename.isOpen = false)}>
          Cancel
        </Button>
        <Button
          type="submit"
          variant="primary"
          size="md"
          disabled={!modalState.rename.newName.trim() || modalState.rename.newName === modalState.rename.target.name}
        >
          <Check class="w-3.5 h-3.5 mr-1" />
          <span>Rename</span>
        </Button>
      </div>
    </form>
  </Modal>
{/if}

<!-- 4. Single Delete Modal -->
{#if modalState.delete.isOpen && modalState.delete.target}
  <Modal
    open={true}
    title="Confirm Deletion"
    onclose={() => (modalState.delete.isOpen = false)}
  >
    <div class="space-y-4 font-mono text-xs">
      <div class="p-3 bg-red-950/30 border border-red-800 rounded text-red-200 space-y-1">
        <p class="font-bold text-red-400">Permanently delete this item?</p>
        <p class="text-[11px] text-red-300/80">
          This operation cannot be undone. Target filesystem entry:
        </p>
      </div>

      <code class="block p-2 rounded bg-card-sub border border-border text-red-400 font-bold truncate text-[11px]">
        {modalState.delete.target.path}
      </code>

      <div class="flex justify-end gap-2 pt-2 border-t border-border">
        <Button variant="ghost" size="md" onclick={() => (modalState.delete.isOpen = false)}>
          Cancel
        </Button>
        <Button variant="danger" size="md" onclick={() => onDelete?.()}>
          <Trash2 class="w-3.5 h-3.5 mr-1" />
          <span>Delete Permanently</span>
        </Button>
      </div>
    </div>
  </Modal>
{/if}

<!-- 5. Permissions (Chmod) Modal -->
{#if modalState.chmod.isOpen && modalState.chmod.target}
  <Modal
    open={true}
    title={`Permissions (Chmod): ${modalState.chmod.target.name}`}
    onclose={() => (modalState.chmod.isOpen = false)}
  >
    <form
      onsubmit={(e) => {
        e.preventDefault()
        onChmod?.()
      }}
      class="space-y-4 font-mono text-xs"
    >
      <Input
        label="Octal Mode (e.g. 0755, 0644, 0777)"
        placeholder="0755"
        bind:value={modalState.chmod.mode}
      />
      <div class="flex gap-2">
        {#each ['0644', '0755', '0777'] as presetMode}
          <button
            type="button"
            class="px-2 py-1 bg-card-sub border border-border rounded text-[10px] hover:border-accent font-bold cursor-pointer"
            onclick={() => (modalState.chmod.mode = presetMode)}
          >
            {presetMode}
          </button>
        {/each}
      </div>
      <p class="text-[11px] text-muted">
        Standard Linux octal permission mode applied directly to filesystem inode.
      </p>
      <div class="flex justify-end gap-2 pt-2 border-t border-border">
        <Button variant="ghost" size="md" onclick={() => (modalState.chmod.isOpen = false)}>
          Cancel
        </Button>
        <Button type="submit" variant="primary" size="md" disabled={!modalState.chmod.mode}>
          <Check class="w-3.5 h-3.5 mr-1" />
          <span>Apply Mode</span>
        </Button>
      </div>
    </form>
  </Modal>
{/if}

<!-- 6. Compress Modal -->
{#if modalState.compress.isOpen && modalState.compress.target}
  <Modal
    open={true}
    title={`Compress: ${modalState.compress.target.name}`}
    onclose={() => (modalState.compress.isOpen = false)}
  >
    <form
      onsubmit={(e) => {
        e.preventDefault()
        onCompress?.()
      }}
      class="space-y-4 font-mono text-xs"
    >
      <Input
        label="ZIP Archive Output Name"
        placeholder="archive.zip"
        bind:value={modalState.compress.destName}
      />
      <p class="text-[11px] text-muted">
        Creates a ZIP archive at the current directory via system gzip/zip utility.
      </p>
      <div class="flex justify-end gap-2 pt-2 border-t border-border">
        <Button variant="ghost" size="md" onclick={() => (modalState.compress.isOpen = false)}>
          Cancel
        </Button>
        <Button type="submit" variant="primary" size="md" disabled={!modalState.compress.destName.trim()}>
          <FileArchive class="w-3.5 h-3.5 mr-1" />
          <span>Create ZIP</span>
        </Button>
      </div>
    </form>
  </Modal>
{/if}

<!-- 7. Extract Modal -->
{#if modalState.extract.isOpen && modalState.extract.target}
  <Modal
    open={true}
    title={`Extract: ${modalState.extract.target.name}`}
    onclose={() => (modalState.extract.isOpen = false)}
  >
    <form
      onsubmit={(e) => {
        e.preventDefault()
        onExtract?.()
      }}
      class="space-y-4 font-mono text-xs"
    >
      <Input
        label="Extract Destination Directory"
        bind:value={modalState.extract.destDir}
      />
      <p class="text-[11px] text-muted">
        Extracts all files from archive into target destination path.
      </p>
      <div class="flex justify-end gap-2 pt-2 border-t border-border">
        <Button variant="ghost" size="md" onclick={() => (modalState.extract.isOpen = false)}>
          Cancel
        </Button>
        <Button type="submit" variant="primary" size="md" disabled={!modalState.extract.destDir.trim()}>
          <FileArchive class="w-3.5 h-3.5 mr-1" />
          <span>Extract Here</span>
        </Button>
      </div>
    </form>
  </Modal>
{/if}

<!-- 8. Single Copy / Move Modal -->
{#if modalState.copyMove.isOpen && modalState.copyMove.target}
  <Modal
    open={true}
    title={`${modalState.copyMove.action === 'copy' ? 'Copy' : 'Move'}: ${modalState.copyMove.target.name}`}
    onclose={() => (modalState.copyMove.isOpen = false)}
  >
    <form
      onsubmit={(e) => {
        e.preventDefault()
        onCopyMove?.()
      }}
      class="space-y-4 font-mono text-xs"
    >
      <Input
        label="Destination Directory"
        bind:value={modalState.copyMove.destPath}
      />
      <div class="flex justify-end gap-2 pt-2 border-t border-border">
        <Button variant="ghost" size="md" onclick={() => (modalState.copyMove.isOpen = false)}>
          Cancel
        </Button>
        <Button type="submit" variant="primary" size="md" disabled={!modalState.copyMove.destPath.trim()}>
          {#if modalState.copyMove.action === 'copy'}
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

<!-- 9. Batch Delete Modal -->
{#if modalState.batch.deleteOpen}
  <Modal
    open={true}
    title={`Batch Delete (${selectedPaths.length} items)`}
    onclose={() => (modalState.batch.deleteOpen = false)}
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
        <Button variant="ghost" size="md" onclick={() => (modalState.batch.deleteOpen = false)}>
          Cancel
        </Button>
        <Button
          variant="danger"
          size="md"
          disabled={isBatchProcessing}
          onclick={() => onBatchDelete?.()}
        >
          <Trash2 class="w-3.5 h-3.5 mr-1" />
          <span>{isBatchProcessing ? 'Deleting...' : 'Delete All Selected'}</span>
        </Button>
      </div>
    </div>
  </Modal>
{/if}

<!-- 10. Batch Copy Modal -->
{#if modalState.batch.copyOpen}
  <Modal
    open={true}
    title={`Batch Copy (${selectedPaths.length} items)`}
    onclose={() => (modalState.batch.copyOpen = false)}
  >
    <form
      onsubmit={(e) => {
        e.preventDefault()
        onBatchCopy?.()
      }}
      class="space-y-4 font-mono text-xs"
    >
      <Input
        label="Destination Directory"
        bind:value={modalState.batch.destDir}
      />
      <div>
        <span class="text-[10px] uppercase font-bold text-muted block mb-1">Selected Items:</span>
        <div class="max-h-32 overflow-y-auto bg-card-sub border border-border rounded p-2 space-y-1">
          {#each selectedPaths.slice(0, 8) as p}
            <div class="text-muted truncate text-[11px]">• {p}</div>
          {/each}
          {#if selectedPaths.length > 8}
            <div class="text-accent font-bold text-[11px]">...and {selectedPaths.length - 8} more</div>
          {/if}
        </div>
      </div>
      <div class="flex justify-end gap-2 pt-2 border-t border-border">
        <Button variant="ghost" size="md" onclick={() => (modalState.batch.copyOpen = false)}>
          Cancel
        </Button>
        <Button type="submit" variant="primary" size="md" disabled={isBatchProcessing || !modalState.batch.destDir.trim()}>
          <Copy class="w-3.5 h-3.5 mr-1" />
          <span>{isBatchProcessing ? 'Copying...' : 'Copy Items'}</span>
        </Button>
      </div>
    </form>
  </Modal>
{/if}

<!-- 11. Batch Move Modal -->
{#if modalState.batch.moveOpen}
  <Modal
    open={true}
    title={`Batch Move (${selectedPaths.length} items)`}
    onclose={() => (modalState.batch.moveOpen = false)}
  >
    <form
      onsubmit={(e) => {
        e.preventDefault()
        onBatchMove?.()
      }}
      class="space-y-4 font-mono text-xs"
    >
      <Input
        label="Destination Directory"
        bind:value={modalState.batch.destDir}
      />
      <div>
        <span class="text-[10px] uppercase font-bold text-muted block mb-1">Selected Items:</span>
        <div class="max-h-32 overflow-y-auto bg-card-sub border border-border rounded p-2 space-y-1">
          {#each selectedPaths.slice(0, 8) as p}
            <div class="text-muted truncate text-[11px]">• {p}</div>
          {/each}
          {#if selectedPaths.length > 8}
            <div class="text-accent font-bold text-[11px]">...and {selectedPaths.length - 8} more</div>
          {/if}
        </div>
      </div>
      <div class="flex justify-end gap-2 pt-2 border-t border-border">
        <Button variant="ghost" size="md" onclick={() => (modalState.batch.moveOpen = false)}>
          Cancel
        </Button>
        <Button type="submit" variant="primary" size="md" disabled={isBatchProcessing || !modalState.batch.destDir.trim()}>
          <FolderInput class="w-3.5 h-3.5 mr-1" />
          <span>{isBatchProcessing ? 'Moving...' : 'Move Items'}</span>
        </Button>
      </div>
    </form>
  </Modal>
{/if}

<!-- 12. Batch Compress Modal -->
{#if modalState.batch.compressOpen}
  <Modal
    open={true}
    title={`Batch Compress (${selectedPaths.length} items)`}
    onclose={() => (modalState.batch.compressOpen = false)}
  >
    <form
      onsubmit={(e) => {
        e.preventDefault()
        onBatchCompress?.()
      }}
      class="space-y-4 font-mono text-xs"
    >
      <Input
        label="ZIP Archive Output Name"
        placeholder="archive.zip"
        bind:value={modalState.batch.zipName}
      />
      <div>
        <span class="text-[10px] uppercase font-bold text-muted block mb-1">Items Included:</span>
        <div class="max-h-32 overflow-y-auto bg-card-sub border border-border rounded p-2 space-y-1">
          {#each selectedPaths.slice(0, 8) as p}
            <div class="text-muted truncate text-[11px]">• {p}</div>
          {/each}
          {#if selectedPaths.length > 8}
            <div class="text-accent font-bold text-[11px]">...and {selectedPaths.length - 8} more</div>
          {/if}
        </div>
      </div>
      <div class="flex justify-end gap-2 pt-2 border-t border-border">
        <Button variant="ghost" size="md" onclick={() => (modalState.batch.compressOpen = false)}>
          Cancel
        </Button>
        <Button type="submit" variant="primary" size="md" disabled={isBatchProcessing || !modalState.batch.zipName.trim()}>
          <FileArchive class="w-3.5 h-3.5 mr-1" />
          <span>{isBatchProcessing ? 'Compressing...' : 'Create ZIP Archive'}</span>
        </Button>
      </div>
    </form>
  </Modal>
{/if}
