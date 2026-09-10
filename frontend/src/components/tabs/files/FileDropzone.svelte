<script lang="ts">
  import { Upload, RefreshCw, Folder } from '@lucide/svelte'

  interface Props {
    currentPath: string
    onUpload: (files: FileList | File[]) => Promise<void>
    isUploading?: boolean
  }

  let { currentPath, onUpload, isUploading = false }: Props = $props()

  let isDragging = $state(false)
  let dragDepth = $state(0)

  function handleDragEnter(e: DragEvent) {
    e.preventDefault()
    dragDepth++
    if (e.dataTransfer && e.dataTransfer.types.includes('Files')) {
      isDragging = true
    }
  }

  function handleDragOver(e: DragEvent) {
    e.preventDefault()
    if (e.dataTransfer) {
      e.dataTransfer.dropEffect = 'copy'
    }
  }

  function handleDragLeave(e: DragEvent) {
    e.preventDefault()
    dragDepth = Math.max(0, dragDepth - 1)
    if (dragDepth === 0) {
      isDragging = false
    }
  }

  async function handleDrop(e: DragEvent) {
    e.preventDefault()
    isDragging = false
    dragDepth = 0
    if (e.dataTransfer && e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      await onUpload(e.dataTransfer.files)
    }
  }
</script>

<svelte:window
  ondragenter={handleDragEnter}
  ondragover={handleDragOver}
  ondragleave={handleDragLeave}
  ondrop={handleDrop}
/>

{#if isDragging || isUploading}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4 sm:p-6 transition-all animate-in fade-in duration-150"
    onclick={() => { if (!isUploading) { isDragging = false; dragDepth = 0 } }}
  >
    <div
      class="neo-card bg-card border-4 border-dashed border-accent p-8 sm:p-12 rounded-xl max-w-lg w-full text-center shadow-neobrutal-lg space-y-4 select-none relative"
      onclick={(e) => e.stopPropagation()}
    >
      <div class="w-16 h-16 rounded-full bg-accent/15 border-2 border-accent mx-auto flex items-center justify-center text-accent">
        {#if isUploading}
          <RefreshCw class="w-8 h-8 animate-spin" />
        {:else}
          <Upload class="w-8 h-8 animate-pulse" />
        {/if}
      </div>

      <div class="space-y-1">
        <h3 class="text-base sm:text-lg font-bold font-mono uppercase tracking-wider text-foreground">
          {isUploading ? 'Uploading Files...' : 'Drop files to upload'}
        </h3>
        <p class="text-xs font-mono text-muted">
          Target destination:
        </p>
        <div class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-card-sub border border-border rounded font-mono text-xs text-accent font-bold max-w-full truncate">
          <Folder class="w-3.5 h-3.5 shrink-0 text-accent" />
          <span class="truncate">{currentPath}</span>
        </div>
      </div>

      {#if !isUploading}
        <p class="text-[11px] font-mono text-muted/80">
          Release files to upload directly into Android filesystem.
        </p>
      {/if}
    </div>
  </div>
{/if}
