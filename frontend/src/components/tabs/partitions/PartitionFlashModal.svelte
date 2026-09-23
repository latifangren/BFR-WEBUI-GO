<script lang="ts">
  import {
    AlertTriangle,
    Flame,
    RefreshCw,
    Check,
  } from '@lucide/svelte'
  import type { PartitionInfo } from '../../../types/partition'
  import Modal from '../../ui/Modal.svelte'
  import Button from '../../ui/Button.svelte'

  interface Props {
    isOpen: boolean
    partition: PartitionInfo | null
    isFlashing: boolean
    onClose: () => void
    onConfirmFlash: (file: File, autoBackup: boolean) => Promise<void>
  }

  let {
    isOpen,
    partition,
    isFlashing,
    onClose,
    onConfirmFlash,
  }: Props = $props()

  let autoBackup = $state(true)
  let confirmationText = $state('')
  let selectedFile = $state<File | null>(null)
  let fileInputEl: HTMLInputElement | null = $state(null)
  let isDragging = $state(false)
  let fileError = $state<string | null>(null)

  $effect(() => {
    if (!isOpen) {
      confirmationText = ''
      selectedFile = null
      fileError = null
      isDragging = false
    }
  })

  function handleFileSelect(file: File) {
    fileError = null
    if (!file.name.toLowerCase().endsWith('.img')) {
      fileError = 'Warning: Selected file does not have a .img extension.'
    }
    selectedFile = file
  }

  function handleFileInputChange(e: Event) {
    const input = e.target as HTMLInputElement
    if (input.files && input.files.length > 0) {
      handleFileSelect(input.files[0])
    }
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault()
    isDragging = false
    if (e.dataTransfer && e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      handleFileSelect(e.dataTransfer.files[0])
    }
  }

  function formatBytes(bytes: number): string {
    if (bytes <= 0) return '0 B'
    const units = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(1024))
    return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
  }

  async function handleConfirm() {
    if (!selectedFile || confirmationText.trim().toUpperCase() !== 'FLASH' || isFlashing) return
    await onConfirmFlash(selectedFile, autoBackup)
  }
</script>

<Modal
  open={isOpen}
  title="FLASH RAW BLOCK PARTITION"
  onclose={() => {
    if (!isFlashing) onClose()
  }}
>
  <div class="space-y-4 text-xs font-mono">
    <!-- Danger Warning Banner -->
    <div class="p-3.5 bg-red-950/40 border-2 border-red-600 rounded text-red-200 space-y-2">
      <div class="flex items-center gap-2 text-red-400 font-bold text-sm uppercase">
        <AlertTriangle size={18} class="shrink-0 text-red-500 animate-pulse" />
        <span>High-Risk Operation Warning</span>
      </div>
      <p class="leading-relaxed">
        Writing directly to raw block partitions can cause an unbootable device, bootloop, or hard-brick if the image is corrupted or incompatible. Ensure this image matches your exact hardware model and active slot.
      </p>
    </div>

    <!-- Target Partition Details -->
    {#if partition}
      <div class="p-3 bg-card-sub border border-border rounded space-y-1.5">
        <div class="text-[10px] text-muted uppercase font-bold tracking-wider">
          Target Partition to Overwrite
        </div>
        <div class="grid grid-cols-2 gap-2 text-foreground">
          <div>
            <span class="text-muted">Name:</span>
            <span class="font-bold ml-1 text-sm text-accent">{partition.name}</span>
          </div>
          <div>
            <span class="text-muted">Block Size:</span>
            <span class="font-bold ml-1">{partition.size_human}</span>
          </div>
          <div class="col-span-2 truncate">
            <span class="text-muted">Path:</span>
            <span class="font-mono ml-1 text-[11px]">{partition.device_path}</span>
          </div>
          {#if partition.slot}
            <div class="col-span-2">
              <span class="text-muted">Slot:</span>
              <span class="font-bold ml-1">_{partition.slot.toUpperCase()} {partition.is_active_slot ? '(Active Slot)' : ''}</span>
            </div>
          {/if}
        </div>
      </div>
    {/if}

    <!-- Auto-backup Pre-flash Toggle -->
    <div class="p-3 bg-card-sub border border-border rounded flex items-center justify-between">
      <div class="space-y-0.5 pr-2">
        <div class="font-bold text-foreground">Auto-Backup Before Flashing</div>
        <div class="text-[11px] text-muted">
          Automatically creates an emergency backup image of this partition before writing.
        </div>
      </div>
      <label class="relative inline-flex items-center cursor-pointer shrink-0">
        <input
          type="checkbox"
          bind:checked={autoBackup}
          disabled={isFlashing}
          class="sr-only peer"
        />
        <div class="w-11 h-6 bg-card peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-foreground after:border-border after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-accent border border-border"></div>
      </label>
    </div>

    <!-- File Picker Dropzone -->
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="border-2 border-dashed rounded p-4 text-center cursor-pointer transition-colors {isDragging ? 'border-accent bg-accent/10' : selectedFile ? 'border-emerald-500 bg-emerald-950/20' : 'border-border bg-card-sub hover:border-foreground/50'}"
      onclick={() => !isFlashing && fileInputEl?.click()}
      ondragover={(e) => { e.preventDefault(); if (!isFlashing) isDragging = true }}
      ondragleave={() => (isDragging = false)}
      ondrop={handleDrop}
    >
      <input
        type="file"
        accept=".img"
        bind:this={fileInputEl}
        onchange={handleFileInputChange}
        disabled={isFlashing}
        class="hidden"
      />

      {#if selectedFile}
        <div class="flex items-center justify-center gap-2 text-emerald-400 font-bold text-sm">
          <Check size={18} />
          <span class="break-all">{selectedFile.name}</span>
        </div>
        <div class="text-[11px] text-muted mt-1">
          Size: {formatBytes(selectedFile.size)} (Click to select a different file)
        </div>
      {:else}
        <Flame size={24} class="mx-auto text-muted mb-1 opacity-70" />
        <div class="font-bold text-foreground">Select partition image (.img)</div>
        <div class="text-[11px] text-muted mt-1">
          Click or drag-and-drop your kernel, boot, or recovery image here
        </div>
      {/if}
    </div>

    {#if fileError}
      <div class="p-2.5 bg-red-950/60 border border-red-700 rounded text-red-300 text-xs">
        {fileError}
      </div>
    {/if}

    <!-- Safety Confirmation Input Requirement -->
    <div class="space-y-1.5 pt-1">
      <label for="flash-confirm-input" class="block text-muted text-[11px] font-bold uppercase tracking-wider">
        Type <span class="text-red-400 font-extrabold tracking-widest">FLASH</span> to enable confirmation button:
      </label>
      <input
        id="flash-confirm-input"
        type="text"
        bind:value={confirmationText}
        disabled={isFlashing}
        placeholder="Type FLASH here..."
        autocomplete="off"
        class="neo-input w-full bg-card-sub border-2 border-border focus:border-red-500 rounded px-3 py-2 text-sm font-mono font-bold tracking-widest text-foreground uppercase placeholder:text-muted/50"
      />
    </div>

    {#if isFlashing}
      <div class="p-3 bg-red-950/40 border border-red-700 rounded text-center space-y-2">
        <div class="flex items-center justify-center gap-2 text-red-300 font-bold">
          <RefreshCw size={16} class="animate-spin text-red-400" />
          <span>Writing image to partition {partition?.name}...</span>
        </div>
        <p class="text-[11px] text-muted animate-pulse">
          Do not reboot the device or close the browser window!
        </p>
      </div>
    {/if}
  </div>

  {#snippet footer()}
    <div class="flex items-center justify-end gap-2 w-full">
      <Button
        variant="ghost"
        disabled={isFlashing}
        onclick={onClose}
      >
        Cancel
      </Button>

      <Button
        variant="danger"
        disabled={isFlashing || !selectedFile || confirmationText.trim().toUpperCase() !== 'FLASH'}
        onclick={handleConfirm}
      >
        {#if isFlashing}
          <RefreshCw size={14} class="animate-spin mr-1.5" />
          <span>Writing Image...</span>
        {:else}
          <Flame size={14} class="mr-1.5" />
          <span>Flash Image Now</span>
        {/if}
      </Button>
    </div>
  {/snippet}
</Modal>
