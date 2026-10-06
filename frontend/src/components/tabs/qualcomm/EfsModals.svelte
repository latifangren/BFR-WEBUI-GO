<script lang="ts">
  import { AlertTriangle, Radio, RefreshCw } from '@lucide/svelte'
  import Modal from '../../ui/Modal.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'

  interface Props {
    status?: any
    showSpcModal: boolean
    spcInput: string
    isUnlockingSpc: boolean
    submitSpcUnlock: () => Promise<void> | void

    showNvModal: boolean
    nvItemInput: number
    nvIndexInput: number
    isLoadingNv: boolean
    nvResult: any
    nvHexInput: string
    isWritingNv: boolean
    readNvItem: () => Promise<void> | void
    writeNvItem: () => Promise<void> | void

    showSsrModal: boolean
    isRestartingModem: boolean
    triggerSsr: () => Promise<void> | void

    showUnlockWarningModal: boolean
    setReadOnlyMode: (ro: boolean) => Promise<void> | void

    showMkdirModal: boolean
    newDirName: string
    isCreatingDir: boolean
    createDirectory: () => Promise<void> | void

    showDeleteModal: boolean
    entryToDelete: any
    isDeleting: boolean
    confirmDelete: () => Promise<void> | void

    showCaModal: boolean
    caInputContent: string
    isParsingCa: boolean
    loadSampleCarrierPolicy: () => void
    parseCustomCarrierPolicy: () => Promise<void> | void
  }

  let {
    status = null,
    showSpcModal = $bindable(false),
    spcInput = $bindable(''),
    isUnlockingSpc = false,
    submitSpcUnlock,

    showNvModal = $bindable(false),
    nvItemInput = $bindable(550),
    nvIndexInput = $bindable(-1),
    isLoadingNv = false,
    nvResult = null,
    nvHexInput = $bindable(''),
    isWritingNv = false,
    readNvItem,
    writeNvItem,

    showSsrModal = $bindable(false),
    isRestartingModem = false,
    triggerSsr,

    showUnlockWarningModal = $bindable(false),
    setReadOnlyMode,

    showMkdirModal = $bindable(false),
    newDirName = $bindable(''),
    isCreatingDir = false,
    createDirectory,

    showDeleteModal = $bindable(false),
    entryToDelete = null,
    isDeleting = false,
    confirmDelete,

    showCaModal = $bindable(false),
    caInputContent = $bindable(''),
    isParsingCa = false,
    loadSampleCarrierPolicy,
    parseCustomCarrierPolicy,
  }: Props = $props()
</script>

<!-- SPC UNLOCK MODAL -->
<Modal bind:open={showSpcModal} title="Unlock DIAG Security Session (SPC)" size="sm">
  <div class="space-y-3 font-mono text-xs">
    <p class="text-muted text-[11px]">
      Qualcomm NV item writes require raising the DIAG security level via the 6-digit Service Programming Code (SPC). Default is usually <code>000000</code>.
    </p>

    <div>
      <label class="block text-[10px] text-muted uppercase font-bold mb-1">
        <span>Service Programming Code (6 Digits)</span>
        <input
          type="text"
          maxlength="6"
          bind:value={spcInput}
          class="neo-input mt-1 w-full bg-card-sub border-2 border-border rounded px-3 py-2 font-mono text-sm tracking-widest text-center text-foreground focus:outline-none focus:border-accent"
          placeholder="000000"
        />
      </label>
    </div>
  </div>

  {#snippet footer()}
    <Button variant="secondary" size="sm" onclick={() => (showSpcModal = false)}>Cancel</Button>
    <Button variant="primary" size="sm" disabled={isUnlockingSpc} onclick={submitSpcUnlock}>
      <span>{isUnlockingSpc ? 'Verifying...' : 'Submit Code'}</span>
    </Button>
  {/snippet}
</Modal>

<!-- NV ITEM INSPECTOR MODAL -->
<Modal bind:open={showNvModal} title="Qualcomm NV Item Inspector" size="lg">
  <div class="space-y-4 font-mono text-xs">
    <div class="grid grid-cols-2 gap-3">
      <div>
        <label class="block text-[10px] text-muted uppercase font-bold mb-1">
          <span>NV Item ID (0-65535)</span>
          <input
            type="number"
            bind:value={nvItemInput}
            class="neo-input mt-1 w-full bg-card-sub border-2 border-border rounded px-2.5 py-1.5 text-foreground focus:outline-none focus:border-accent"
            placeholder="550"
          />
        </label>
      </div>
      <div>
        <label class="block text-[10px] text-muted uppercase font-bold mb-1">
          <span>Subsystem Index (-1 for Default)</span>
          <input
            type="number"
            bind:value={nvIndexInput}
            class="neo-input mt-1 w-full bg-card-sub border-2 border-border rounded px-2.5 py-1.5 text-foreground focus:outline-none focus:border-accent"
            placeholder="-1"
          />
        </label>
      </div>
    </div>

    <div class="flex items-center justify-end gap-2">
      <Button variant="secondary" size="sm" disabled={isLoadingNv} onclick={readNvItem}>
        <RefreshCw class="w-3.5 h-3.5 mr-1 {isLoadingNv ? 'animate-spin' : ''}" />
        <span>Read NV Item</span>
      </Button>
    </div>

    {#if nvResult}
      <div class="p-3 bg-zinc-950 border-2 border-border rounded space-y-2">
        <div class="flex items-center justify-between text-[11px]">
          <span class="text-muted">Return Status:</span>
          <Badge variant={nvResult.status === 0 ? 'success' : 'danger'}>
            {nvResult.status_str} ({nvResult.status})
          </Badge>
        </div>

        <div>
          <label class="block text-[10px] text-muted uppercase font-bold mb-1">
            <span>NV Payload (128 Bytes Hex)</span>
            <textarea
              bind:value={nvHexInput}
              rows="4"
              class="neo-input mt-1 w-full bg-zinc-900 border border-border rounded p-2 text-cyan-300 font-mono text-[11px] focus:outline-none focus:border-accent"
            ></textarea>
          </label>
        </div>
      </div>
    {/if}
  </div>

  {#snippet footer()}
    <Button variant="secondary" size="sm" onclick={() => (showNvModal = false)}>Close</Button>
    <Button
      variant="danger"
      size="sm"
      disabled={status.readonly || isWritingNv || !nvResult}
      onclick={writeNvItem}
    >
      <span>{isWritingNv ? 'Writing...' : 'Write NV to Modem'}</span>
    </Button>
  {/snippet}
</Modal>

<!-- RESTART MODEM (SSR) MODAL -->
<Modal bind:open={showSsrModal} title="Confirm Modem Subsystem Restart (SSR)" size="sm">
  <div class="space-y-3 font-mono text-xs">
    <div class="p-3 bg-amber-950/40 border-2 border-amber-600 rounded text-amber-200 space-y-1">
      <p class="font-bold flex items-center gap-1.5 text-amber-300">
        <AlertTriangle class="w-4 h-4 text-amber-400" />
        <span>Temporary Cellular Interruption</span>
      </p>
      <p class="text-[11px] text-amber-200/90 leading-relaxed">
        This command flushes the dirty EFS journal to flash and issues a hardware-level Subsystem Restart (SSR) to the Qualcomm modem. Cellular data and calls will reset for approximately 2 to 3 seconds.
      </p>
    </div>
  </div>

  {#snippet footer()}
    <Button variant="secondary" size="sm" onclick={() => (showSsrModal = false)}>Cancel</Button>
    <Button variant="danger" size="sm" disabled={isRestartingModem} onclick={triggerSsr}>
      <span>{isRestartingModem ? 'Rebooting Modem...' : 'Restart Radio Now'}</span>
    </Button>
  {/snippet}
</Modal>

<!-- UNLOCK WRITE ACCESS WARNING MODAL -->
<Modal bind:open={showUnlockWarningModal} title="Unlock Modem Write Access" size="sm">
  <div class="space-y-3 font-mono text-xs">
    <div class="p-3 bg-red-950/40 border-2 border-red-600 rounded text-red-200 space-y-1">
      <p class="font-bold flex items-center gap-1.5 text-red-300">
        <AlertTriangle class="w-4 h-4 text-red-400" />
        <span>Caution: Raw Baseband Filesystem</span>
      </p>
      <p class="text-[11px] text-red-200/90 leading-relaxed">
        Writing incorrect or corrupt configurations directly to Qualcomm EFS2 can cause modem bootloops or loss of IMEI / radio calibrations.
      </p>
      <p class="text-[10px] text-red-300/80 mt-1 font-bold">
        Always download a .TAR backup before making modifications!
      </p>
    </div>
  </div>

  {#snippet footer()}
    <Button variant="secondary" size="sm" onclick={() => (showUnlockWarningModal = false)}>
      Keep Read-Only
    </Button>
    <Button variant="danger" size="sm" onclick={() => setReadOnlyMode(false)}>
      I Understand, Unlock
    </Button>
  {/snippet}
</Modal>

<!-- CREATE DIRECTORY MODAL -->
<Modal bind:open={showMkdirModal} title="Create Directory" size="sm">
  <div class="space-y-3 font-mono text-xs">
    <div>
      <label class="block text-[10px] text-muted uppercase font-bold mb-1">
        <span>Directory Name</span>
        <input
          type="text"
          bind:value={newDirName}
          class="neo-input mt-1 w-full bg-card-sub border-2 border-border rounded px-3 py-2 text-foreground focus:outline-none focus:border-accent"
          placeholder="my_folder"
        />
      </label>
    </div>
  </div>

  {#snippet footer()}
    <Button variant="secondary" size="sm" onclick={() => (showMkdirModal = false)}>Cancel</Button>
    <Button variant="primary" size="sm" disabled={isCreatingDir || !newDirName.trim()} onclick={createDirectory}>
      <span>{isCreatingDir ? 'Creating...' : 'Create Folder'}</span>
    </Button>
  {/snippet}
</Modal>

<!-- DELETE CONFIRM MODAL -->
<Modal bind:open={showDeleteModal} title="Confirm Deletion" size="sm">
  <div class="space-y-3 font-mono text-xs">
    <p>
      Are you sure you want to permanently delete <strong class="text-red-400">{entryToDelete?.path}</strong> from the Qualcomm modem filesystem?
    </p>
  </div>

  {#snippet footer()}
    <Button variant="secondary" size="sm" onclick={() => (showDeleteModal = false)}>Cancel</Button>
    <Button variant="danger" size="sm" disabled={isDeleting} onclick={confirmDelete}>
      <span>{isDeleting ? 'Deleting...' : 'Delete Permanently'}</span>
    </Button>
  {/snippet}
</Modal>

<!-- STANDALONE CARRIER POLICY / CA COMBOS MODAL -->
<Modal bind:open={showCaModal} title="Qualcomm Carrier Policy / CA Combos Visualizer" size="lg">
  <div class="space-y-3 font-mono text-xs">
    <div class="flex items-center justify-between">
      <p class="text-muted text-[11px]">
        Paste raw carrier policy XML, <code>&lt;ca_band_combos&gt;</code> tag, or semicolon-separated combo list below:
      </p>
      <button
        type="button"
        class="text-[10px] text-cyan-400 hover:underline font-bold cursor-pointer"
        onclick={loadSampleCarrierPolicy}
      >
        Load Demo Combos
      </button>
    </div>

    <div>
      <textarea
        rows="9"
        bind:value={caInputContent}
        class="neo-input w-full bg-zinc-950 border-2 border-border rounded p-3 font-mono text-xs text-foreground focus:outline-none focus:border-cyan-500 placeholder:text-muted"
        placeholder="Paste carrier_policy.xml or ca_band_combos string here..."
      ></textarea>
    </div>

    <div class="p-2.5 bg-card-sub rounded border border-border text-[11px] text-muted space-y-1">
      <div class="font-bold text-foreground">Supported Formats:</div>
      <div>• Qualcomm carrier policy XML: <code>&lt;ca_band_combos string="..." /&gt;</code></div>
      <div>• Raw combo list: <code>4A2A-2A2-mAll; 66C44A-2A4-m3; 5B22A-m1; ...</code></div>
    </div>
  </div>

  {#snippet footer()}
    <Button variant="secondary" size="sm" onclick={() => (showCaModal = false)}>Cancel</Button>
    <Button
      variant="primary"
      size="sm"
      disabled={isParsingCa || !caInputContent.trim()}
      onclick={parseCustomCarrierPolicy}
    >
      <Radio class="w-3.5 h-3.5 mr-1 text-black" />
      <span>{isParsingCa ? 'Analyzing...' : 'Parse & Visualize Combos'}</span>
    </Button>
  {/snippet}
</Modal>

