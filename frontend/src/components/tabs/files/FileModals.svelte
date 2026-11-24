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
  import { LL } from '../../../i18n/i18n-svelte'

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
    title={$LL.files.modals.editor.title({ path: modalState.editor.path })}
    class="max-w-4xl"
    onclose={() => (modalState.editor.isOpen = false)}
  >
    <div class="space-y-4 font-mono text-xs">
      {#if modalState.editor.isLoading}
        <div class="p-12 text-center text-muted">
          <RefreshCw class="w-6 h-6 animate-spin mx-auto mb-2 text-accent" />
          <span>{$LL.files.modals.editor.reading()}</span>
        </div>
      {:else}
        <textarea
          bind:value={modalState.editor.content}
          rows="18"
          class="w-full bg-card-sub border-2 border-border rounded p-3 font-mono text-xs text-foreground focus:outline-none focus:border-accent resize-y"
          placeholder={$LL.files.modals.editor.emptyPlaceholder()}
        ></textarea>
      {/if}

      <div class="flex items-center justify-between pt-2 border-t border-border">
        <span class="text-muted text-[11px] truncate max-w-sm">
          {modalState.editor.path}
        </span>
        <div class="flex items-center gap-2">
          <Button variant="ghost" size="md" onclick={() => (modalState.editor.isOpen = false)}>
            {$LL.common.cancel()}
          </Button>
          <Button
            variant="primary"
            size="md"
            disabled={modalState.editor.isSaving || modalState.editor.isLoading}
            onclick={() => onSaveEditor?.()}
          >
            {#if modalState.editor.isSaving}
              <RefreshCw class="w-3.5 h-3.5 mr-1.5 animate-spin" />
              <span>{$LL.files.modals.editor.saving()}</span>
            {:else}
              <Save class="w-3.5 h-3.5 mr-1.5" />
              <span>{$LL.files.modals.editor.saveFile()}</span>
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
    title={modalState.create.type === 'file' ? $LL.files.modals.create.titleFile() : $LL.files.modals.create.titleFolder()}
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
          {$LL.files.pane.file()}
        </button>
        <button
          type="button"
          class="flex-1 py-1.5 text-center font-bold rounded transition-colors {modalState.create.type === 'folder' ? 'bg-accent text-accent-text' : 'text-muted hover:text-foreground'}"
          onclick={() => (modalState.create.type = 'folder')}
        >
          {$LL.files.modals.create.folderDirectory()}
        </button>
      </div>

      <Input
        label={$LL.files.modals.create.nameLabel()}
        placeholder={modalState.create.type === 'file' ? 'example.conf' : 'new_folder'}
        bind:value={modalState.create.name}
      />
      <div class="flex justify-end gap-2 pt-2 border-t border-border">
        <Button variant="ghost" size="md" onclick={() => (modalState.create.isOpen = false)}>
          {$LL.common.cancel()}
        </Button>
        <Button type="submit" variant="primary" size="md" disabled={!modalState.create.name.trim()}>
          <Plus class="w-3.5 h-3.5 mr-1" />
          <span>{$LL.files.modals.create.create()}</span>
        </Button>
      </div>
    </form>
  </Modal>
{/if}

<!-- 3. Rename Modal -->
{#if modalState.rename.isOpen && modalState.rename.target}
  <Modal
    open={true}
    title={$LL.files.modals.rename.title({ name: modalState.rename.target.name })}
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
        label={$LL.files.modals.rename.newName()}
        bind:value={modalState.rename.newName}
      />
      <div class="flex justify-end gap-2 pt-2 border-t border-border">
        <Button variant="ghost" size="md" onclick={() => (modalState.rename.isOpen = false)}>
          {$LL.common.cancel()}
        </Button>
        <Button
          type="submit"
          variant="primary"
          size="md"
          disabled={!modalState.rename.newName.trim() || modalState.rename.newName === modalState.rename.target.name}
        >
          <Check class="w-3.5 h-3.5 mr-1" />
          <span>{$LL.files.table.rename()}</span>
        </Button>
      </div>
    </form>
  </Modal>
{/if}

<!-- 4. Single Delete Modal -->
{#if modalState.delete.isOpen && modalState.delete.target}
  <Modal
    open={true}
    title={$LL.files.modals.delete.title()}
    onclose={() => (modalState.delete.isOpen = false)}
  >
    <div class="space-y-4 font-mono text-xs">
      <div class="p-3 bg-red-950/30 border border-red-800 rounded text-red-200 space-y-1">
        <p class="font-bold text-red-400">{$LL.files.modals.delete.confirmBody()}</p>
        <p class="text-[11px] text-red-300/80">
          {$LL.files.modals.delete.warning()}
        </p>
      </div>

      <code class="block p-2 rounded bg-card-sub border border-border text-red-400 font-bold truncate text-[11px]">
        {modalState.delete.target.path}
      </code>

      <div class="flex justify-end gap-2 pt-2 border-t border-border">
        <Button variant="ghost" size="md" onclick={() => (modalState.delete.isOpen = false)}>
          {$LL.common.cancel()}
        </Button>
        <Button variant="danger" size="md" onclick={() => onDelete?.()}>
          <Trash2 class="w-3.5 h-3.5 mr-1" />
          <span>{$LL.files.modals.delete.deletePermanently()}</span>
        </Button>
      </div>
    </div>
  </Modal>
{/if}

<!-- 5. Permissions (Chmod) Modal -->
{#if modalState.chmod.isOpen && modalState.chmod.target}
  <Modal
    open={true}
    title={$LL.files.modals.chmod.title({ name: modalState.chmod.target.name })}
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
        label={$LL.files.modals.chmod.octalLabel()}
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
        {$LL.files.modals.chmod.hint()}
      </p>
      <div class="flex justify-end gap-2 pt-2 border-t border-border">
        <Button variant="ghost" size="md" onclick={() => (modalState.chmod.isOpen = false)}>
          {$LL.common.cancel()}
        </Button>
        <Button type="submit" variant="primary" size="md" disabled={!modalState.chmod.mode}>
          <Check class="w-3.5 h-3.5 mr-1" />
          <span>{$LL.files.modals.chmod.applyMode()}</span>
        </Button>
      </div>
    </form>
  </Modal>
{/if}

<!-- 6. Compress Modal -->
{#if modalState.compress.isOpen && modalState.compress.target}
  <Modal
    open={true}
    title={$LL.files.modals.compress.title({ name: modalState.compress.target.name })}
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
        label={$LL.files.modals.compress.outputLabel()}
        placeholder="archive.zip"
        bind:value={modalState.compress.destName}
      />
      <p class="text-[11px] text-muted">
        {$LL.files.modals.compress.hint()}
      </p>
      <div class="flex justify-end gap-2 pt-2 border-t border-border">
        <Button variant="ghost" size="md" onclick={() => (modalState.compress.isOpen = false)}>
          {$LL.common.cancel()}
        </Button>
        <Button type="submit" variant="primary" size="md" disabled={!modalState.compress.destName.trim()}>
          <FileArchive class="w-3.5 h-3.5 mr-1" />
          <span>{$LL.files.modals.compress.createZip()}</span>
        </Button>
      </div>
    </form>
  </Modal>
{/if}

<!-- 7. Extract Modal -->
{#if modalState.extract.isOpen && modalState.extract.target}
  <Modal
    open={true}
    title={$LL.files.modals.extract.title({ name: modalState.extract.target.name })}
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
        label={$LL.files.modals.extract.destLabel()}
        bind:value={modalState.extract.destDir}
      />
      <p class="text-[11px] text-muted">
        {$LL.files.modals.extract.hint()}
      </p>
      <div class="flex justify-end gap-2 pt-2 border-t border-border">
        <Button variant="ghost" size="md" onclick={() => (modalState.extract.isOpen = false)}>
          {$LL.common.cancel()}
        </Button>
        <Button type="submit" variant="primary" size="md" disabled={!modalState.extract.destDir.trim()}>
          <FileArchive class="w-3.5 h-3.5 mr-1" />
          <span>{$LL.files.modals.extract.extractHere()}</span>
        </Button>
      </div>
    </form>
  </Modal>
{/if}

<!-- 8. Single Copy / Move Modal -->
{#if modalState.copyMove.isOpen && modalState.copyMove.target}
  <Modal
    open={true}
    title={modalState.copyMove.action === 'copy' ? $LL.files.modals.copyMove.titleCopy({ name: modalState.copyMove.target.name }) : $LL.files.modals.copyMove.titleMove({ name: modalState.copyMove.target.name })}
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
        label={$LL.files.modals.copyMove.destLabel()}
        bind:value={modalState.copyMove.destPath}
      />
      <div class="flex justify-end gap-2 pt-2 border-t border-border">
        <Button variant="ghost" size="md" onclick={() => (modalState.copyMove.isOpen = false)}>
          {$LL.common.cancel()}
        </Button>
        <Button type="submit" variant="primary" size="md" disabled={!modalState.copyMove.destPath.trim()}>
          {#if modalState.copyMove.action === 'copy'}
            <Copy class="w-3.5 h-3.5 mr-1" />
            <span>{$LL.files.modals.copyMove.copyHere()}</span>
          {:else}
            <FolderInput class="w-3.5 h-3.5 mr-1" />
            <span>{$LL.files.modals.copyMove.moveHere()}</span>
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
    title={$LL.files.modals.batchDelete.title({ count: selectedPaths.length })}
    onclose={() => (modalState.batch.deleteOpen = false)}
  >
    <div class="space-y-4 font-mono text-xs">
      <div class="p-3 bg-red-950/30 border border-red-800 rounded text-red-200 space-y-1">
        <p class="font-bold text-red-400">{$LL.files.modals.batchDelete.confirmBody()}</p>
        <p class="text-[11px] text-red-300/80">
          {$LL.files.modals.batchDelete.warning()}
        </p>
      </div>

      <div class="max-h-40 overflow-y-auto bg-card-sub border border-border rounded p-2.5 space-y-1">
        {#each selectedPaths.slice(0, 10) as p}
          <div class="text-muted truncate text-[11px]">• {p}</div>
        {/each}
        {#if selectedPaths.length > 10}
          <div class="text-accent font-bold pt-1 text-[11px]">{$LL.files.modals.batchDelete.moreItems({ count: selectedPaths.length - 10 })}</div>
        {/if}
      </div>

      <div class="flex justify-end gap-2 pt-2 border-t border-border">
        <Button variant="ghost" size="md" onclick={() => (modalState.batch.deleteOpen = false)}>
          {$LL.common.cancel()}
        </Button>
        <Button
          variant="danger"
          size="md"
          disabled={isBatchProcessing}
          onclick={() => onBatchDelete?.()}
        >
          <Trash2 class="w-3.5 h-3.5 mr-1" />
          <span>{isBatchProcessing ? $LL.files.modals.batchDelete.deleting() : $LL.files.modals.batchDelete.deleteAll()}</span>
        </Button>
      </div>
    </div>
  </Modal>
{/if}

<!-- 10. Batch Copy Modal -->
{#if modalState.batch.copyOpen}
  <Modal
    open={true}
    title={$LL.files.modals.batchCopy.title({ count: selectedPaths.length })}
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
        label={$LL.files.modals.copyMove.destLabel()}
        bind:value={modalState.batch.destDir}
      />
      <div>
        <span class="text-[10px] uppercase font-bold text-muted block mb-1">{$LL.files.modals.batchCopy.selectedItems()}</span>
        <div class="max-h-32 overflow-y-auto bg-card-sub border border-border rounded p-2 space-y-1">
          {#each selectedPaths.slice(0, 8) as p}
            <div class="text-muted truncate text-[11px]">• {p}</div>
          {/each}
          {#if selectedPaths.length > 8}
            <div class="text-accent font-bold text-[11px]">{$LL.files.modals.batchCopy.moreItems({ count: selectedPaths.length - 8 })}</div>
          {/if}
        </div>
      </div>
      <div class="flex justify-end gap-2 pt-2 border-t border-border">
        <Button variant="ghost" size="md" onclick={() => (modalState.batch.copyOpen = false)}>
          {$LL.common.cancel()}
        </Button>
        <Button type="submit" variant="primary" size="md" disabled={isBatchProcessing || !modalState.batch.destDir.trim()}>
          <Copy class="w-3.5 h-3.5 mr-1" />
          <span>{isBatchProcessing ? $LL.files.modals.batchCopy.copying() : $LL.files.modals.batchCopy.copyItems()}</span>
        </Button>
      </div>
    </form>
  </Modal>
{/if}

<!-- 11. Batch Move Modal -->
{#if modalState.batch.moveOpen}
  <Modal
    open={true}
    title={$LL.files.modals.batchMove.title({ count: selectedPaths.length })}
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
        label={$LL.files.modals.copyMove.destLabel()}
        bind:value={modalState.batch.destDir}
      />
      <div>
        <span class="text-[10px] uppercase font-bold text-muted block mb-1">{$LL.files.modals.batchCopy.selectedItems()}</span>
        <div class="max-h-32 overflow-y-auto bg-card-sub border border-border rounded p-2 space-y-1">
          {#each selectedPaths.slice(0, 8) as p}
            <div class="text-muted truncate text-[11px]">• {p}</div>
          {/each}
          {#if selectedPaths.length > 8}
            <div class="text-accent font-bold text-[11px]">{$LL.files.modals.batchCopy.moreItems({ count: selectedPaths.length - 8 })}</div>
          {/if}
        </div>
      </div>
      <div class="flex justify-end gap-2 pt-2 border-t border-border">
        <Button variant="ghost" size="md" onclick={() => (modalState.batch.moveOpen = false)}>
          {$LL.common.cancel()}
        </Button>
        <Button type="submit" variant="primary" size="md" disabled={isBatchProcessing || !modalState.batch.destDir.trim()}>
          <FolderInput class="w-3.5 h-3.5 mr-1" />
          <span>{isBatchProcessing ? $LL.files.modals.batchMove.moving() : $LL.files.modals.batchMove.moveItems()}</span>
        </Button>
      </div>
    </form>
  </Modal>
{/if}

<!-- 12. Batch Compress Modal -->
{#if modalState.batch.compressOpen}
  <Modal
    open={true}
    title={$LL.files.modals.batchCompress.title({ count: selectedPaths.length })}
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
        label={$LL.files.modals.compress.outputLabel()}
        placeholder="archive.zip"
        bind:value={modalState.batch.zipName}
      />
      <div>
        <span class="text-[10px] uppercase font-bold text-muted block mb-1">{$LL.files.modals.batchCompress.itemsIncluded()}</span>
        <div class="max-h-32 overflow-y-auto bg-card-sub border border-border rounded p-2 space-y-1">
          {#each selectedPaths.slice(0, 8) as p}
            <div class="text-muted truncate text-[11px]">• {p}</div>
          {/each}
          {#if selectedPaths.length > 8}
            <div class="text-accent font-bold text-[11px]">{$LL.files.modals.batchCopy.moreItems({ count: selectedPaths.length - 8 })}</div>
          {/if}
        </div>
      </div>
      <div class="flex justify-end gap-2 pt-2 border-t border-border">
        <Button variant="ghost" size="md" onclick={() => (modalState.batch.compressOpen = false)}>
          {$LL.common.cancel()}
        </Button>
        <Button type="submit" variant="primary" size="md" disabled={isBatchProcessing || !modalState.batch.zipName.trim()}>
          <FileArchive class="w-3.5 h-3.5 mr-1" />
          <span>{isBatchProcessing ? $LL.files.modals.batchCompress.compressing() : $LL.files.modals.batchCompress.createZipArchive()}</span>
        </Button>
      </div>
    </form>
  </Modal>
{/if}
