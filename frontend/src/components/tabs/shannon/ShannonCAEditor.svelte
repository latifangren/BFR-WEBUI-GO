<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Layers,
    RefreshCw,
    Download,
    Upload,
    Save,
    Play,
    CheckCircle2,
    AlertTriangle,
    XCircle,
    Plus,
    Trash2,
    Filter,
    FileSpreadsheet,
    ShieldAlert,
    Sliders,
    Search,
    Info,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Modal from '../../ui/Modal.svelte'

  interface Component {
    band: number
    bwClassMimoDl: number
    bwClassMimoUl: number
    dl_class: string
    dl_mimo: number
    ul_class: string
  }

  interface Combo {
    components: Component[]
    bcs: number
    configMaskLow: number
    configMaskHigh: number
    conf_ids: number[]
    combo_string: string
    total_streams: number
  }

  interface CAEditorStatus {
    supported_device: boolean
    device_model: string
    tensor_gen: string
    detected_paths: string[]
    active_file_path?: string
    combo_count: number
    version: number
    bitmask: number
    warning_message?: string
  }

  interface ValidationResult {
    valid: boolean
    error_count: number
    warning_count: number
    issues: string[]
  }

  let status = $state<CAEditorStatus>({
    supported_device: false,
    device_model: '',
    tensor_gen: '',
    detected_paths: [],
    combo_count: 0,
    version: 1,
    bitmask: 0,
  })

  let combos = $state<Combo[]>([])
  let isLoading = $state(false)
  let isSaving = $state(false)
  let isGenerating = $state(false)
  let searchQuery = $state('')
  let targetSavePath = $state('/vendor/firmware/stable_modem/uecap/uecap.binarypb')

  // Modals
  let showAutoGenModal = $state(false)
  let showPruneModal = $state(false)
  let showValidationModal = $state(false)
  let showAddComboModal = $state(false)
  let showUploadModal = $state(false)

  // Validation
  let validationResult = $state<ValidationResult | null>(null)

  // AutoGen Form
  let autoGenInput = $state('1+3+8B+40D+28')
  let autoGenMaxCC = $state(5)
  let autoGenUL = $state('fdd_aa')
  let autoGenBCS = $state(2147483648)
  let autoGenConfIds = $state('0, 1, 2')

  // Prune Form
  let pruneBands = $state('1, 3, 5, 8, 20, 28, 40')
  let pruneMaxCombos = $state(128)
  let pruneExcludeIntra = $state(true)

  // New Combo Form
  let newComboString = $state('1A4[A]+3C4+8A2')

  // File Upload
  let uploadFileInput: HTMLInputElement | null = null

  onMount(() => {
    fetchStatus()
  })

  async function fetchStatus() {
    try {
      isLoading = true
      const res = await api.get<CAEditorStatus>('/api/shannon/ca/status')
      if (res) {
        status = res
        if (res.active_file_path) {
          targetSavePath = res.active_file_path
        } else if (res.detected_paths && res.detected_paths.length > 0) {
          targetSavePath = res.detected_paths[0]
        }
      }
      // Load current document
      const docRes = await api.get<{ loaded: boolean; combos: Combo[] }>('/api/shannon/ca/document')
      if (docRes && docRes.loaded) {
        combos = docRes.combos || []
      }
    } catch {
      // Retain state
    } finally {
      isLoading = false
    }
  }

  async function loadFile(path: string) {
    try {
      isLoading = true
      const res = await api.post<{ success: boolean; combos: Combo[]; status: CAEditorStatus }>('/api/shannon/ca/load', { path })
      if (res.success) {
        combos = res.combos || []
        status = res.status
        toastStore.success(`Successfully loaded ${combos.length} LTE CA combos`)
      }
    } catch (err: any) {
      toastStore.danger(err.message || 'Failed to load binarypb file')
    } finally {
      isLoading = false
    }
  }

  async function handleFileUpload(e: Event) {
    const input = e.target as HTMLInputElement
    if (!input.files || input.files.length === 0) return

    const file = input.files[0]
    const formData = new FormData()
    formData.append('file', file)

    try {
      isLoading = true
      const res = await fetch('/api/shannon/ca/upload', {
        method: 'POST',
        body: formData,
      })
      const data = await res.json()
      if (data.success) {
        combos = data.combos || []
        status = data.status
        showUploadModal = false
        toastStore.success(`Imported ${combos.length} combos from ${file.name}`)
      } else {
        toastStore.danger(data.error || 'Upload failed')
      }
    } catch (err: any) {
      toastStore.danger(err.message || 'Failed to upload binarypb')
    } finally {
      isLoading = false
      if (input) input.value = ''
    }
  }

  async function saveFile() {
    try {
      isSaving = true
      const res = await api.post<{ success: boolean; message?: string }>('/api/shannon/ca/save', {
        path: targetSavePath,
        combos,
      })
      if (res.success) {
        toastStore.success(res.message || 'Saved successfully')
        fetchStatus()
      }
    } catch (err: any) {
      toastStore.danger(err.message || 'Failed to save binarypb file')
    } finally {
      isSaving = false
    }
  }

  async function runValidation() {
    try {
      const res = await api.post<ValidationResult>('/api/shannon/ca/validate', { combos })
      validationResult = res
      showValidationModal = true
    } catch (err: any) {
      toastStore.danger(err.message || 'Failed to validate combos')
    }
  }

  async function runAutoGen() {
    try {
      isGenerating = true
      const rawCombos = autoGenInput.split('\n').map(s => s.trim()).filter(Boolean)
      const confIds = autoGenConfIds.split(',').map(s => parseInt(s.trim())).filter(n => !isNaN(n))

      const res = await api.post<{ success: boolean; combos: Combo[]; count: number }>('/api/shannon/ca/autogen', {
        input_combos: rawCombos,
        max_cc: autoGenMaxCC,
        ul_option: autoGenUL,
        bcs_option: autoGenBCS,
        conf_ids: confIds,
      })

      if (res.success && res.combos) {
        combos = [...combos, ...res.combos]
        showAutoGenModal = false
        toastStore.success(`Generated and appended ${res.count} fallback combos`)
      }
    } catch (err: any) {
      toastStore.danger(err.message || 'Failed to generate combos')
    } finally {
      isGenerating = false
    }
  }

  async function runPrune() {
    try {
      const allowed = pruneBands.split(',').map(s => parseInt(s.trim())).filter(n => !isNaN(n))
      const res = await api.post<{ success: boolean; combos: Combo[]; count: number }>('/api/shannon/ca/prune', {
        combos,
        allowed_bands: allowed,
        max_combos: pruneMaxCombos,
        exclude_intra_band: pruneExcludeIntra,
      })
      if (res.success && res.combos) {
        combos = res.combos
        showPruneModal = false
        toastStore.success(`Pruned list to ${res.count} essential combinations`)
      }
    } catch (err: any) {
      toastStore.danger(err.message || 'Failed to prune combos')
    }
  }

  function deleteCombo(idx: number) {
    combos = combos.filter((_, i) => i !== idx)
  }

  function downloadFile() {
    window.location.href = '/api/shannon/ca/download'
  }

  const filteredCombos = $derived(
    searchQuery.trim() === ''
      ? combos
      : combos.filter(c =>
          c.combo_string.toLowerCase().includes(searchQuery.toLowerCase()) ||
          c.conf_ids.some(id => id.toString().includes(searchQuery))
        )
  )
</script>

<div class="space-y-4">
  <!-- HARDWARE & COMPATIBILITY NOTICE -->
  <div class="p-4 bg-card-sub border-2 border-border rounded-lg space-y-2">
    <div class="flex items-center justify-between flex-wrap gap-2">
      <div class="flex items-center gap-2">
        <Badge variant={status.supported_device ? 'success' : 'warning'}>
          {status.supported_device ? 'Pixel 9 / 10 Verified' : 'Pixel 9 & 10 Only'}
        </Badge>
        <span class="font-bold text-foreground text-xs">{status.tensor_gen || 'Tensor Architecture'}</span>
      </div>

      <div class="flex items-center gap-1.5 text-xs">
        <Button variant="secondary" size="sm" onclick={fetchStatus} disabled={isLoading}>
          <RefreshCw class="w-3.5 h-3.5 mr-1 {isLoading ? 'animate-spin' : ''}" />
          <span>Refresh</span>
        </Button>
      </div>
    </div>

    <!-- EXPLICIT HARDWARE NOTICE -->
    <div class="p-3 bg-amber-950/40 border-2 border-amber-600 rounded text-amber-200 text-xs space-y-1">
      <p class="font-bold flex items-center gap-1.5 text-amber-300">
        <AlertTriangle class="w-4 h-4 text-amber-400" />
        <span>Hardware Compatibility Notice: Google Pixel 9 & 10</span>
      </p>
      <p class="text-[11px] leading-relaxed text-amber-200/90">
        This editor is engineered specifically for modifying Carrier Aggregation configurations serialized via <strong>Google Protocol Buffers (`ShannonLteUECap` .binarypb)</strong> utilized exclusively on <strong>Google Pixel 9 (Tensor G4)</strong> and <strong>Pixel 10 (Tensor G5)</strong>.
      </p>
      <p class="text-[10px] text-amber-300/80 font-mono">
        ℹ️ <strong>Pixel 6, Pixel 7, and Pixel 8</strong> series devices utilize internal legacy Shannon baseband NV item tables without standalone .binarypb files. On non-Pixel 9/10 devices, you may still import, construct, edit, validate, and export `.binarypb` files for research, offline staging, or Magisk module overlays.
      </p>
    </div>
  </div>

  <!-- TOOLBAR CONTROLS -->
  <Card title="Shannon LTE CA Combinations" subtitle="Carrier aggregation combos, bandwidth classes, MIMO, and PLMN conf_id mapping" tone="ice">
    {#snippet action()}
      <div class="flex items-center flex-wrap gap-1.5">
        <Button variant="outline" size="sm" onclick={() => (showUploadModal = true)}>
          <Upload class="w-3.5 h-3.5 mr-1 text-accent" />
          <span>Import .binarypb</span>
        </Button>

        <Button variant="outline" size="sm" onclick={downloadFile} disabled={combos.length === 0}>
          <Download class="w-3.5 h-3.5 mr-1 text-cyan-400" />
          <span>Export .binarypb</span>
        </Button>

        <Button variant="outline" size="sm" onclick={() => (showAutoGenModal = true)}>
          <Sliders class="w-3.5 h-3.5 mr-1 text-emerald-400" />
          <span>Auto Generator</span>
        </Button>

        <Button variant="outline" size="sm" onclick={() => (showPruneModal = true)} disabled={combos.length === 0}>
          <Filter class="w-3.5 h-3.5 mr-1 text-purple-400" />
          <span>Prune (128 Cap)</span>
        </Button>

        <Button variant="outline" size="sm" onclick={runValidation} disabled={combos.length === 0}>
          <ShieldAlert class="w-3.5 h-3.5 mr-1 text-amber-400" />
          <span>Validate Rules</span>
        </Button>

        <Button variant="primary" size="sm" onclick={saveFile} disabled={isSaving || combos.length === 0}>
          <Save class="w-3.5 h-3.5 mr-1" />
          <span>{isSaving ? 'Saving...' : 'Save to System'}</span>
        </Button>
      </div>
    {/snippet}

    <div class="space-y-3 font-mono text-xs">
      <!-- FILE SELECTOR & SEARCH -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
        <div>
          <label class="block text-[10px] text-muted uppercase font-bold mb-1">
            <span>Target File Path on Device</span>
            <input
              type="text"
              bind:value={targetSavePath}
              class="neo-input mt-1 w-full bg-card-sub border-2 border-border rounded px-3 py-1.5 text-foreground focus:outline-none focus:border-accent text-xs"
              placeholder="/vendor/firmware/stable_modem/uecap/uecap.binarypb"
            />
          </label>
        </div>

        <div>
          <label class="block text-[10px] text-muted uppercase font-bold mb-1">
            <span>Search Filter (Band or Conf ID)</span>
            <div class="relative mt-1">
              <Search class="w-3.5 h-3.5 text-muted absolute left-2.5 top-2.5" />
              <input
                type="text"
                bind:value={searchQuery}
                class="neo-input w-full bg-card-sub border-2 border-border rounded pl-8 pr-3 py-1.5 text-foreground focus:outline-none focus:border-accent text-xs"
                placeholder="Search combo (e.g. 1A, 3C, 8)..."
              />
            </div>
          </label>
        </div>
      </div>

      <!-- DETECTED FILES SHORTCUTS -->
      {#if status.detected_paths && status.detected_paths.length > 0}
        <div class="flex items-center gap-2 flex-wrap text-[11px] p-2 bg-card-sub border border-border rounded">
          <span class="text-muted font-bold">Detected on Phone:</span>
          {#each status.detected_paths as path}
            <button
              type="button"
              class="px-2 py-0.5 bg-zinc-800 hover:bg-accent/20 hover:text-accent rounded border border-border transition-colors cursor-pointer text-[10px]"
              onclick={() => loadFile(path)}
            >
              {path}
            </button>
          {/each}
        </div>
      {/if}

      <!-- STATS BAR -->
      <div class="flex items-center justify-between text-[11px] p-2 bg-zinc-950 border border-border rounded">
        <div>
          <span class="text-muted">Total Combinations:</span>
          <span class="font-bold text-accent ml-1">{combos.length}</span>
          {#if combos.length > 128}
            <Badge variant="warning" class="ml-2 text-[9px]">Exceeds 128 (May need Pruning)</Badge>
          {:else if combos.length > 0}
            <Badge variant="success" class="ml-2 text-[9px]">Within 128 Limit</Badge>
          {/if}
        </div>

        <div>
          <span class="text-muted">Protobuf Version:</span>
          <span class="font-bold text-foreground ml-1">{status.version}</span>
        </div>
      </div>

      <!-- COMBO TABLE -->
      {#if filteredCombos.length === 0}
        <div class="p-8 text-center border-2 border-dashed border-border rounded space-y-2">
          <FileSpreadsheet class="w-8 h-8 text-muted mx-auto" />
          <p class="font-bold text-foreground">No LTE Carrier Aggregation combos loaded</p>
          <p class="text-[11px] text-muted max-w-md mx-auto">
            Click "Auto Generator" to create fallback combos from your target bands, or "Import .binarypb" to open an existing file.
          </p>
        </div>
      {:else}
        <div class="border-2 border-border rounded overflow-hidden">
          <div class="max-h-[500px] overflow-y-auto">
            <table class="w-full text-left border-collapse text-[11px]">
              <thead class="bg-card-sub text-muted font-bold uppercase sticky top-0 border-b-2 border-border z-10">
                <tr>
                  <th class="p-2 w-12 text-center">#</th>
                  <th class="p-2">CA Combo Definition</th>
                  <th class="p-2">Components Detail</th>
                  <th class="p-2 text-center w-24">Streams</th>
                  <th class="p-2 text-center w-28">PLMN Conf IDs</th>
                  <th class="p-2 text-center w-16">Action</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border font-mono">
                {#each filteredCombos as combo, idx}
                  <tr class="hover:bg-accent/5 transition-colors">
                    <td class="p-2 text-center text-muted">{idx + 1}</td>
                    <td class="p-2">
                      <span class="font-bold text-accent">{combo.combo_string}</span>
                    </td>
                    <td class="p-2">
                      <div class="flex items-center gap-1.5 flex-wrap">
                        {#each combo.components as comp}
                          <span class="px-1.5 py-0.5 rounded bg-zinc-900 border border-border text-[10px]">
                            B{comp.band}: {comp.dl_class} ({comp.dl_mimo}x{comp.dl_mimo})
                            {#if comp.ul_class !== 'None'}
                              <span class="text-amber-400 font-bold ml-0.5">UL:{comp.ul_class}</span>
                            {/if}
                          </span>
                        {/each}
                      </div>
                    </td>
                    <td class="p-2 text-center">
                      <Badge variant={combo.total_streams > 24 ? 'danger' : 'info'}>
                        {combo.total_streams} / 24
                      </Badge>
                    </td>
                    <td class="p-2 text-center">
                      {#if combo.conf_ids.length > 0}
                        <span class="text-muted text-[10px]">[{combo.conf_ids.join(', ')}]</span>
                      {:else}
                        <span class="text-muted text-[10px]">Default (0)</span>
                      {/if}
                    </td>
                    <td class="p-2 text-center">
                      <button
                        type="button"
                        class="p-1 text-red-400 hover:text-red-300 hover:bg-red-950/40 rounded transition-colors cursor-pointer"
                        onclick={() => deleteCombo(idx)}
                        title="Delete Combo"
                      >
                        <Trash2 class="w-3.5 h-3.5" />
                      </button>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        </div>
      {/if}
    </div>
  </Card>
</div>

<!-- AUTO GENERATOR MODAL -->
<Modal bind:open={showAutoGenModal} title="Auto CA Combo Generator" size="md">
  <div class="space-y-4 font-mono text-xs">
    <p class="text-muted text-[11px] leading-relaxed">
      Input carrier aggregation combinations (e.g. <code>1+3+8B+40D+28</code>). The generator will automatically generate fallback subsets (e.g. <code>1+3</code>, <code>1+8</code>, <code>1+3+8</code>), assign bandwidth classes, 4x4 MIMO, and limit spatial streams to 24.
    </p>

    <div>
      <label class="block text-[10px] text-muted uppercase font-bold mb-1">
        <span>Target Combos (One per line)</span>
        <textarea
          bind:value={autoGenInput}
          rows="3"
          class="neo-input mt-1 w-full bg-card-sub border-2 border-border rounded p-2 text-foreground focus:outline-none focus:border-accent text-xs"
          placeholder="1+3+8B+40D+28"
        ></textarea>
      </label>
    </div>

    <div class="grid grid-cols-2 gap-3">
      <div>
        <label class="block text-[10px] text-muted uppercase font-bold mb-1">
          <span>Max Component Carriers (CC)</span>
          <input
            type="number"
            min="1"
            max="5"
            bind:value={autoGenMaxCC}
            class="neo-input mt-1 w-full bg-card-sub border-2 border-border rounded px-2.5 py-1.5 text-foreground focus:outline-none focus:border-accent text-xs"
          />
        </label>
      </div>

      <div>
        <label class="block text-[10px] text-muted uppercase font-bold mb-1">
          <span>Uplink Carrier Aggregation</span>
          <select
            bind:value={autoGenUL}
            class="neo-input mt-1 w-full bg-card-sub border-2 border-border rounded px-2.5 py-1.5 text-foreground focus:outline-none focus:border-accent text-xs"
          >
            <option value="none">Single UL Only (Default)</option>
            <option value="fdd_aa">Allow FDD A+A Non-Contiguous</option>
            <option value="tdd_aa">Allow TDD A+A Non-Contiguous</option>
            <option value="tdd_fdd">Allow TDD+FDD Mixed UL</option>
          </select>
        </label>
      </div>
    </div>

    <div>
      <label class="block text-[10px] text-muted uppercase font-bold mb-1">
        <span>Operator Conf IDs Mapping</span>
        <input
          type="text"
          bind:value={autoGenConfIds}
          class="neo-input mt-1 w-full bg-card-sub border-2 border-border rounded px-2.5 py-1.5 text-foreground focus:outline-none focus:border-accent text-xs"
          placeholder="0, 1, 2"
        />
      </label>
    </div>
  </div>

  {#snippet footer()}
    <Button variant="secondary" size="sm" onclick={() => (showAutoGenModal = false)}>Cancel</Button>
    <Button variant="primary" size="sm" disabled={isGenerating} onclick={runAutoGen}>
      <span>{isGenerating ? 'Generating...' : 'Generate Fallbacks'}</span>
    </Button>
  {/snippet}
</Modal>

<!-- PRUNE MODAL -->
<Modal bind:open={showPruneModal} title="Prune Combinations (128 Cap)" size="md">
  <div class="space-y-4 font-mono text-xs">
    <p class="text-muted text-[11px] leading-relaxed">
      If a cellular network does not configure <code>requestedFrequencyBands-r11</code>, devices can only report up to 128 combos in the UE Capability message. Use this tool to filter essential bands and remove duplicates.
    </p>

    <div>
      <label class="block text-[10px] text-muted uppercase font-bold mb-1">
        <span>Essential Bands to Keep</span>
        <input
          type="text"
          bind:value={pruneBands}
          class="neo-input mt-1 w-full bg-card-sub border-2 border-border rounded px-2.5 py-1.5 text-foreground focus:outline-none focus:border-accent text-xs"
          placeholder="1, 3, 5, 8, 20, 28, 40"
        />
      </label>
    </div>

    <div class="grid grid-cols-2 gap-3">
      <div>
        <label class="block text-[10px] text-muted uppercase font-bold mb-1">
          <span>Max Combo Count</span>
          <input
            type="number"
            bind:value={pruneMaxCombos}
            class="neo-input mt-1 w-full bg-card-sub border-2 border-border rounded px-2.5 py-1.5 text-foreground focus:outline-none focus:border-accent text-xs"
          />
        </label>
      </div>

      <div class="flex items-center pt-5">
        <label class="flex items-center gap-2 cursor-pointer">
          <input type="checkbox" bind:checked={pruneExcludeIntra} class="accent-accent" />
          <span class="text-[11px] text-foreground font-bold">Exclude Intra-Band CA</span>
        </label>
      </div>
    </div>
  </div>

  {#snippet footer()}
    <Button variant="secondary" size="sm" onclick={() => (showPruneModal = false)}>Cancel</Button>
    <Button variant="primary" size="sm" onclick={runPrune}>
      <span>Apply Pruning</span>
    </Button>
  {/snippet}
</Modal>

<!-- VALIDATION RESULTS MODAL -->
<Modal bind:open={showValidationModal} title="Shannon CA Validation Report" size="md">
  <div class="space-y-3 font-mono text-xs">
    {#if validationResult}
      <div class="flex items-center justify-between p-3 rounded bg-zinc-950 border-2 border-border">
        <div class="flex items-center gap-2">
          {#if validationResult.valid}
            <CheckCircle2 class="w-5 h-5 text-emerald-400" />
            <span class="font-bold text-emerald-400 text-sm">Valid Configuration</span>
          {:else}
            <XCircle class="w-5 h-5 text-red-400" />
            <span class="font-bold text-red-400 text-sm">Rule Violations Detected</span>
          {/if}
        </div>
        <div class="flex items-center gap-2">
          <Badge variant={validationResult.error_count > 0 ? 'danger' : 'success'}>
            {validationResult.error_count} Errors
          </Badge>
          <Badge variant={validationResult.warning_count > 0 ? 'warning' : 'info'}>
            {validationResult.warning_count} Warnings
          </Badge>
        </div>
      </div>

      {#if validationResult.issues.length === 0}
        <p class="text-emerald-300 text-center py-4">
          All combinations comply with Shannon modem spatial stream limits, 4x4 MIMO rules, and uplink constraints!
        </p>
      {:else}
        <div class="max-h-64 overflow-y-auto space-y-1.5 p-2 bg-zinc-900 border border-border rounded text-[11px]">
          {#each validationResult.issues as issue}
            <p class="text-amber-200/90 leading-relaxed">• {issue}</p>
          {/each}
        </div>
      {/if}
    {/if}
  </div>

  {#snippet footer()}
    <Button variant="primary" size="sm" onclick={() => (showValidationModal = false)}>Close</Button>
  {/snippet}
</Modal>

<!-- UPLOAD MODAL -->
<Modal bind:open={showUploadModal} title="Import Shannon .binarypb File" size="sm">
  <div class="space-y-3 font-mono text-xs">
    <p class="text-muted text-[11px]">
      Select a <code>.binarypb</code> file from your computer or phone storage to decode and edit in the browser.
    </p>

    <div class="p-4 border-2 border-dashed border-border rounded text-center">
      <input
        type="file"
        accept=".binarypb,.pb,.bin"
        onchange={handleFileUpload}
        class="text-xs file:mr-2 file:py-1 file:px-2 file:rounded file:border-0 file:text-xs file:bg-accent file:text-accent-text cursor-pointer"
      />
    </div>
  </div>

  {#snippet footer()}
    <Button variant="secondary" size="sm" onclick={() => (showUploadModal = false)}>Cancel</Button>
  {/snippet}
</Modal>
