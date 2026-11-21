<script lang="ts">
  import { onMount } from 'svelte'
  import {
    RefreshCw,
    Download,
    Upload,
    Save,
    CheckCircle2,
    AlertTriangle,
    XCircle,
    Trash2,
    Filter,
    FileSpreadsheet,
    ShieldAlert,
    Sliders,
    Search,
    Radio,
    Sparkles,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Modal from '../../ui/Modal.svelte'
  import { themeStore } from '../../../stores/theme.svelte'
  import SakugaCheckbox from '../../ui/SakugaCheckbox.svelte'
  import { LL } from '../../../i18n/i18n-svelte'

  // --- LTE CA TYPES ---
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

  // --- 5G NR & EN-DC TYPES ---
  interface NRComponent {
    band: number
    is_nr: boolean
    is_fr2: boolean
    bw_class_dl: string
    bw_class_ul: string
    mimo_dl: number
    mimo_ul: number
    modulation_dl: string
    modulation_ul: string
    max_scs: number
    max_bw_dl: number
    max_bw_ul: number
    bw_90mhz_supported: boolean
    srs_tx_switch?: string
  }

  interface NRCombo {
    type: 'ENDC' | 'NRCA' | 'NRDC' | string
    components_lte: NRComponent[]
    components_nr: NRComponent[]
    bcs_nr: number
    bcs_eutra: number
    bcs_intra_endc: number
    sku_bitmask: number
    power_class: string
    combo_string: string
    total_dl_streams: number
  }

  interface NREditorStatus {
    supported_device: boolean
    device_model: string
    tensor_gen: string
    detected_paths: string[]
    active_file_path?: string
    total_combos: number
    endc_count: number
    nrca_count: number
    nrdc_count: number
    version: number
    id: number
    warning_message?: string
  }

  // Active view mode within CA Editor
  let viewMode = $state<'lte' | 'nr'>('nr')

  // --- LTE STATE ---
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
  let isLteLoading = $state(false)
  let isSaving = $state(false)
  let isGenerating = $state(false)
  let lteSearchQuery = $state('')
  let targetSavePath = $state('/vendor/firmware/stable_modem/uecap/uecap.binarypb')

  // Modals for LTE
  let showAutoGenModal = $state(false)
  let showPruneModal = $state(false)
  let showValidationModal = $state(false)
  let showLteUploadModal = $state(false)
  let validationResult = $state<ValidationResult | null>(null)

  let autoGenInput = $state('1+3+8B+40D+28')
  let autoGenMaxCC = $state(5)
  let autoGenUL = $state('fdd_aa')
  let autoGenBCS = $state(2147483648)
  let autoGenConfIds = $state('0, 1, 2')
  let pruneBands = $state('1, 3, 5, 8, 20, 28, 40')
  let pruneMaxCombos = $state(128)
  let pruneExcludeIntra = $state(true)

  // --- 5G NR & EN-DC STATE ---
  let nrStatus = $state<NREditorStatus>({
    supported_device: false,
    device_model: '',
    tensor_gen: '',
    detected_paths: [],
    total_combos: 0,
    endc_count: 0,
    nrca_count: 0,
    nrdc_count: 0,
    version: 0,
    id: 0,
  })
  let nrCombos = $state<NRCombo[]>([])
  let isNrLoading = $state(false)
  let nrSearchQuery = $state('')
  let nrTypeFilter = $state<'ALL' | 'ENDC' | 'NRCA' | 'NRDC'>('ALL')
  let showNrUploadModal = $state(false)

  onMount(() => {
    fetchLteStatus()
    fetchNrStatus()
  })

  // --- LTE ACTIONS ---
  async function fetchLteStatus() {
    try {
      isLteLoading = true
      const res = await api.get<CAEditorStatus>('/api/shannon/ca/status')
      if (res) {
        status = res
        if (res.active_file_path) {
          targetSavePath = res.active_file_path
        } else if (res.detected_paths && res.detected_paths.length > 0) {
          targetSavePath = res.detected_paths[0]
        }
      }
      const docRes = await api.get<{ loaded: boolean; combos: Combo[] }>('/api/shannon/ca/document')
      if (docRes && docRes.loaded) {
        combos = docRes.combos || []
      }
    } catch {
      // Retain state
    } finally {
      isLteLoading = false
    }
  }

  async function loadLteFile(path: string) {
    try {
      isLteLoading = true
      const res = await api.post<{ success: boolean; combos: Combo[]; status: CAEditorStatus }>('/api/shannon/ca/load', { path })
      if (res.success) {
        combos = res.combos || []
        status = res.status
        toastStore.success($LL.shannon.ca.loadedLte({ count: combos.length }))
      }
    } catch (err: any) {
      toastStore.danger(err.message || $LL.shannon.ca.loadFailed())
    } finally {
      isLteLoading = false
    }
  }

  async function handleLteFileUpload(e: Event) {
    const input = e.target as HTMLInputElement
    if (!input.files || input.files.length === 0) return

    const file = input.files[0]
    const formData = new FormData()
    formData.append('file', file)

    try {
      isLteLoading = true
      const res = await fetch('/api/shannon/ca/upload', { method: 'POST', body: formData })
      const data = await res.json()
      if (data.success) {
        combos = data.combos || []
        status = data.status
        showLteUploadModal = false
        toastStore.success($LL.shannon.ca.importedLte({ count: combos.length, name: file.name }))
      } else {
        toastStore.danger(data.error || $LL.shannon.ca.uploadFailed())
      }
    } catch (err: any) {
      toastStore.danger(err.message || $LL.shannon.ca.uploadError())
    } finally {
      isLteLoading = false
      if (input) input.value = ''
    }
  }

  async function saveLteFile() {
    try {
      isSaving = true
      const res = await api.post<{ success: boolean; message?: string }>('/api/shannon/ca/save', {
        path: targetSavePath,
        combos,
      })
      if (res.success) {
        toastStore.success(res.message || $LL.shannon.ca.saved())
        fetchLteStatus()
      }
    } catch (err: any) {
      toastStore.danger(err.message || $LL.shannon.ca.saveFailed())
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
      toastStore.danger(err.message || $LL.shannon.ca.validateFailed())
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
        toastStore.success($LL.shannon.ca.generated({ count: res.count }))
      }
    } catch (err: any) {
      toastStore.danger(err.message || $LL.shannon.ca.generateFailed())
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
        toastStore.success($LL.shannon.ca.pruned({ count: res.count }))
      }
    } catch (err: any) {
      toastStore.danger(err.message || $LL.shannon.ca.pruneFailed())
    }
  }

  function deleteLteCombo(idx: number) {
    combos = combos.filter((_, i) => i !== idx)
  }

  function downloadLteFile() {
    window.location.href = '/api/shannon/ca/download'
  }

  async function unloadLteDocument() {
    try {
      isLteLoading = true
      await api.post('/api/shannon/ca/clear', {})
      combos = []
      status = {
        ...status,
        combo_count: 0,
        active_file_path: '',
      }
      toastStore.info($LL.shannon.ca.unloadedLte())
    } catch (err: any) {
      toastStore.danger(err.message || $LL.shannon.ca.unloadFailed())
    } finally {
      isLteLoading = false
    }
  }

  const filteredLteCombos = $derived(
    lteSearchQuery.trim() === ''
      ? combos
      : combos.filter(c =>
          (c.combo_string || '').toLowerCase().includes(lteSearchQuery.toLowerCase()) ||
          (c.conf_ids || []).some(id => id.toString().includes(lteSearchQuery))
        )
  )


  // --- 5G NR & EN-DC ACTIONS ---
  async function fetchNrStatus() {
    try {
      isNrLoading = true
      const res = await api.get<NREditorStatus>('/api/shannon/nr/status')
      if (res) {
        nrStatus = res
      }
      const docRes = await api.get<{
        loaded: boolean
        combos: NRCombo[]
        total_combos: number
        endc_count: number
        nrca_count: number
        nrdc_count: number
      }>('/api/shannon/nr/document')
      if (docRes && docRes.loaded) {
        nrCombos = docRes.combos || []
      }
    } catch {
      // Retain state
    } finally {
      isNrLoading = false
    }
  }

  async function loadNrFile(path: string) {
    try {
      isNrLoading = true
      const res = await api.post<{ success: boolean; combos: NRCombo[]; status: NREditorStatus }>('/api/shannon/nr/load', { path })
      if (res.success) {
        nrCombos = res.combos || []
        nrStatus = res.status
        toastStore.success($LL.shannon.ca.loadedNr({ count: nrCombos.length }))
      }
    } catch (err: any) {
      toastStore.danger(err.message || $LL.shannon.ca.loadNrFailed())
    } finally {
      isNrLoading = false
    }
  }

  async function loadNrDemo() {
    try {
      isNrLoading = true
      const res = await api.get<{ success: boolean; combos: NRCombo[]; status: NREditorStatus }>('/api/shannon/nr/demo')
      if (res.success) {
        nrCombos = res.combos || []
        nrStatus = res.status
        toastStore.success($LL.shannon.ca.demoLoaded({ count: nrCombos.length }))
      }
    } catch (err: any) {
      toastStore.danger(err.message || $LL.shannon.ca.demoFailed())
    } finally {
      isNrLoading = false
    }
  }

  async function handleNrFileUpload(e: Event) {
    const input = e.target as HTMLInputElement
    if (!input.files || input.files.length === 0) return

    const file = input.files[0]
    const formData = new FormData()
    formData.append('file', file)

    try {
      isNrLoading = true
      const res = await fetch('/api/shannon/nr/upload', { method: 'POST', body: formData })
      const data = await res.json()
      if (data.success) {
        nrCombos = data.combos || []
        nrStatus = data.status
        showNrUploadModal = false
        toastStore.success($LL.shannon.ca.importedNr({ count: nrCombos.length, name: file.name }))
      } else {
        toastStore.danger(data.error || $LL.shannon.ca.uploadFailed())
      }
    } catch (err: any) {
      toastStore.danger(err.message || $LL.shannon.ca.uploadError())
    } finally {
      isNrLoading = false
      if (input) input.value = ''
    }
  }

  function downloadNrFile() {
    window.location.href = '/api/shannon/nr/download'
  }

  async function unloadNrDocument() {
    try {
      isNrLoading = true
      await api.post('/api/shannon/nr/clear', {})
      nrCombos = []
      nrStatus = {
        ...nrStatus,
        total_combos: 0,
        endc_count: 0,
        nrca_count: 0,
        nrdc_count: 0,
        active_file_path: '',
      }
      toastStore.info($LL.shannon.ca.unloadedNr())
    } catch (err: any) {
      toastStore.danger(err.message || $LL.shannon.ca.unloadFailed())
    } finally {
      isNrLoading = false
    }
  }

  const filteredNrCombos = $derived(
    nrCombos.filter(combo => {
      // Type filter
      if (nrTypeFilter !== 'ALL' && combo.type !== nrTypeFilter) {
        return false
      }
      // Search query filter
      if (nrSearchQuery.trim() !== '') {
        const q = nrSearchQuery.toLowerCase()
        const matchesString = (combo.combo_string || '').toLowerCase().includes(q)
        const matchesLteBand = (combo.components_lte || []).some(c => `b${c.band}`.includes(q) || `${c.band}` === q)
        const matchesNrBand = (combo.components_nr || []).some(c => `n${c.band}`.includes(q) || `${c.band}` === q)
        return matchesString || matchesLteBand || matchesNrBand
      }
      return true
    })
  )
</script>

<div class="space-y-4">
  <!-- HARDWARE & COMPATIBILITY BANNER (PIXEL 6 THROUGH PIXEL 10) -->
  <div class="p-4 bg-card-sub border-2 border-border rounded-lg space-y-2">
    <div class="flex items-center justify-between flex-wrap gap-2">
      <div class="flex items-center gap-2">
        <Badge variant={status.supported_device || nrStatus.supported_device ? 'success' : 'warning'}>
          {status.supported_device || nrStatus.supported_device ? $LL.shannon.ca.verified() : $LL.shannon.ca.architecture()}
        </Badge>
        <span class="font-bold text-foreground text-xs">
          {status.tensor_gen || nrStatus.tensor_gen || $LL.shannon.ca.tensorFallback()}
        </span>
      </div>

      <div class="flex items-center gap-1.5 text-xs">
        <Button
          variant="secondary"
          size="sm"
          onclick={() => {
            fetchLteStatus()
            fetchNrStatus()
          }}
          disabled={isLteLoading || isNrLoading}
        >
          <RefreshCw class="w-3.5 h-3.5 mr-1 {isLteLoading || isNrLoading ? 'animate-spin' : ''}" />
          <span>{$LL.common.refresh()}</span>
        </Button>
      </div>
    </div>

    <div class="p-3 bg-zinc-950/70 border border-border rounded text-xs space-y-1">
      <p class="font-bold flex items-center gap-1.5 text-accent">
        <Radio class="w-4 h-4 text-accent" />
        <span>{$LL.shannon.ca.scopeTitle()}</span>
      </p>
      <p class="text-[11px] leading-relaxed text-muted">
        {$LL.shannon.ca.scopeBodyA()} <strong>Shannon Modem Protocol Buffers</strong> (<code>ShannonNrUECap</code> {$LL.shannon.ca.scopeBodyB()} <code>ShannonLteUECap</code> {$LL.shannon.ca.scopeBodyC()}
      </p>
      <p class="text-[10px] text-muted/80 font-mono">
        {$LL.shannon.ca.scopeNote()}
      </p>
    </div>
  </div>

  <!-- UNIFIED CA & UE CAPABILITY MODE SWITCHER -->
  <div class="flex items-center gap-2 border-b-2 border-border pb-2 text-xs">
    <button
      type="button"
      class="neo-button relative z-10 flex items-center gap-2 px-3 py-2 rounded font-bold transition-all cursor-pointer select-none {viewMode === 'nr'
        ? 'border-2 border-accent bg-accent/20 text-accent shadow-neobrutal-sm'
        : 'border-2 border-border bg-card-sub text-muted hover:text-foreground'}"
      onclick={(e) => {
        e.preventDefault()
        viewMode = 'nr'
      }}
    >
      <Radio class="w-3.5 h-3.5 text-cyan-400 pointer-events-none" />
      <span class="pointer-events-none">{$LL.shannon.ca.modeNr()}</span>
      {#if nrCombos.length > 0}
        <span class="px-1.5 py-0.2 rounded bg-card text-[10px] text-cyan-300 font-mono pointer-events-none">
          {nrCombos.length}
        </span>
      {/if}
    </button>

    <button
      type="button"
      class="neo-button relative z-10 flex items-center gap-2 px-3 py-2 rounded font-bold transition-all cursor-pointer select-none {viewMode === 'lte'
        ? 'border-2 border-accent bg-accent/20 text-accent shadow-neobrutal-sm'
        : 'border-2 border-border bg-card-sub text-muted hover:text-foreground'}"
      onclick={(e) => {
        e.preventDefault()
        viewMode = 'lte'
      }}
    >
      <FileSpreadsheet class="w-3.5 h-3.5 text-amber-400 pointer-events-none" />
      <span class="pointer-events-none">{$LL.shannon.ca.modeLte()}</span>
      {#if combos.length > 0}
        <span class="px-1.5 py-0.2 rounded bg-card text-[10px] text-amber-300 font-mono pointer-events-none">
          {combos.length}
        </span>
      {/if}
    </button>
  </div>

  <!-- VIEW 1: 5G NR & EN-DC DUAL CONNECTIVITY -->
  {#if viewMode === 'nr'}
    <Card
      title={$LL.shannon.ca.nrTitle()}
      subtitle={$LL.shannon.ca.nrSubtitle()}
      tone="ice"
    >
      {#snippet action()}
        <div class="flex items-center flex-wrap gap-1.5">
          <Button variant="outline" size="sm" onclick={loadNrDemo}>
            <Sparkles class="w-3.5 h-3.5 mr-1 text-accent" />
            <span>{$LL.shannon.ca.loadDemo()}</span>
          </Button>

          {#if nrCombos.length > 0}
            <Button variant="outline" size="sm" onclick={unloadNrDocument} title={$LL.shannon.ca.unloadTitle()}>
              <Trash2 class="w-3.5 h-3.5 mr-1 text-red-400" />
              <span>{$LL.shannon.ca.unload()}</span>
            </Button>
          {/if}

          <Button variant="outline" size="sm" onclick={() => (showNrUploadModal = true)}>
            <Upload class="w-3.5 h-3.5 mr-1 text-cyan-400" />
            <span>{$LL.shannon.ca.importFile()}</span>
          </Button>

          <Button variant="outline" size="sm" onclick={downloadNrFile} disabled={nrCombos.length === 0}>
            <Download class="w-3.5 h-3.5 mr-1 text-emerald-400" />
            <span>{$LL.shannon.ca.exportFile()}</span>
          </Button>
        </div>
      {/snippet}

      <div class="space-y-3 font-mono text-xs">
        <!-- SEARCH & TYPE FILTER -->
        <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
          <div class="md:col-span-2">
            <label class="block text-[10px] text-muted uppercase font-bold mb-1">
              <span>{$LL.shannon.ca.filterLabel()}</span>
              <div class="relative mt-1">
                <Search class="w-3.5 h-3.5 text-muted absolute left-2.5 top-2.5" />
                <input
                  type="text"
                  bind:value={nrSearchQuery}
                  class="neo-input w-full bg-card-sub border-2 border-border rounded pl-8 pr-3 py-1.5 text-foreground focus:outline-none focus:border-accent text-xs"
                  placeholder={$LL.shannon.ca.filterPlaceholder()}
                />
              </div>
            </label>
          </div>

          <div>
            <span class="block text-[10px] text-muted uppercase font-bold mb-1">{$LL.shannon.ca.comboCategory()}</span>
            <div class="flex items-center gap-1 mt-1">
              {#each [
                { id: 'ALL', label: $LL.common.all(), count: nrCombos.length },
                { id: 'ENDC', label: 'EN-DC', count: nrStatus.endc_count },
                { id: 'NRCA', label: 'NR-CA', count: nrStatus.nrca_count },
                { id: 'NRDC', label: 'NR-DC', count: nrStatus.nrdc_count },
              ] as f}
                <button
                  type="button"
                  class="flex-1 px-2 py-1.5 rounded border text-[10px] font-bold transition-all cursor-pointer {nrTypeFilter === f.id
                    ? 'border-accent bg-accent/20 text-accent'
                    : 'border-border bg-card-sub text-muted hover:text-foreground'}"
                  onclick={() => (nrTypeFilter = f.id as typeof nrTypeFilter)}
                >
                  {f.label} ({f.count})
                </button>
              {/each}
            </div>
          </div>
        </div>

        <!-- DETECTED FILES SHORTCUTS ON DEVICE -->
        {#if nrStatus.detected_paths && nrStatus.detected_paths.length > 0}
          <div class="flex items-center gap-2 flex-wrap text-[11px] p-2 bg-card-sub border border-border rounded">
            <span class="text-muted font-bold">{$LL.shannon.ca.detectedOnPhone()}</span>
            {#each nrStatus.detected_paths as path}
              <button
                type="button"
                class="px-2 py-0.5 bg-zinc-800 hover:bg-accent/20 hover:text-accent rounded border border-border transition-colors cursor-pointer text-[10px]"
                onclick={() => loadNrFile(path)}
              >
                {path}
              </button>
            {/each}
          </div>
        {/if}

        <!-- STATS BAR -->
        <div class="flex items-center justify-between text-[11px] p-2 bg-zinc-950 border border-border rounded flex-wrap gap-2">
          <div class="flex items-center gap-2">
            <span class="text-muted">{$LL.shannon.ca.totalCombos()}</span>
            <span class="font-bold text-accent">{nrCombos.length}</span>
            <span class="text-muted">•</span>
            <span class="text-muted">EN-DC:</span>
            <span class="font-bold text-cyan-400">{nrStatus.endc_count}</span>
            <span class="text-muted">•</span>
            <span class="text-muted">NR-CA:</span>
            <span class="font-bold text-emerald-400">{nrStatus.nrca_count}</span>
            {#if nrStatus.nrdc_count > 0}
              <span class="text-muted">•</span>
              <span class="text-muted">NR-DC:</span>
              <span class="font-bold text-purple-400">{nrStatus.nrdc_count}</span>
            {/if}
          </div>

          <div>
            {#if nrStatus.version > 0}
              <span class="text-muted">{$LL.shannon.ca.protoVersion()}</span>
              <span class="font-bold text-foreground ml-1">{nrStatus.version}</span>
            {/if}
            {#if nrStatus.id > 0}
              <span class="text-muted ml-2">ID:</span>
              <span class="font-bold text-foreground ml-1">{nrStatus.id}</span>
            {/if}
          </div>
        </div>

        <!-- 5G NR TABLE -->
        {#if filteredNrCombos.length === 0}
          <div class="p-8 text-center border-2 border-dashed border-border rounded space-y-2">
            <Radio class="w-8 h-8 text-muted mx-auto" />
            <p class="font-bold text-foreground">{$LL.shannon.ca.nrEmptyTitle()}</p>
            <p class="text-[11px] text-muted max-w-md mx-auto">
              {$LL.shannon.ca.click()} <strong>"{$LL.shannon.ca.loadDemo()}"</strong> {$LL.shannon.ca.nrEmptyBodyB()} <strong>"{$LL.shannon.ca.importFile()}"</strong> {$LL.shannon.ca.nrEmptyBodyC()}
            </p>
          </div>
        {:else}
          <div class="border-2 border-border rounded overflow-hidden">
            <div class="max-h-[520px] overflow-y-auto">
              <table class="w-full text-left border-collapse text-[11px]">
                <thead class="bg-card-sub text-muted font-bold uppercase sticky top-0 border-b-2 border-border z-10">
                  <tr>
                    <th class="p-2 w-12 text-center">#</th>
                    <th class="p-2 w-20 text-center">{$LL.shannon.ca.thType()}</th>
                    <th class="p-2">{$LL.shannon.ca.thCombo()}</th>
                    <th class="p-2">{$LL.shannon.ca.thLteAnchor()}</th>
                    <th class="p-2">{$LL.shannon.ca.thNrCarriers()}</th>
                    <th class="p-2 text-center w-20">{$LL.shannon.ca.thStreams()}</th>
                    <th class="p-2 text-center w-24">{$LL.shannon.ca.thPowerBcs()}</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-border font-mono">
                  {#each filteredNrCombos as combo, idx}
                    <tr class="hover:bg-accent/5 transition-colors">
                      <td class="p-2 text-center text-muted">{idx + 1}</td>
                      <td class="p-2 text-center">
                        {#if combo.type === 'ENDC'}
                          <Badge variant="info" class="text-[9px]">EN-DC</Badge>
                        {:else if combo.type === 'NRDC'}
                          <Badge variant="warning" class="text-[9px]">NR-DC</Badge>
                        {:else}
                          <Badge variant="success" class="text-[9px]">NR-CA</Badge>
                        {/if}
                      </td>
                      <td class="p-2">
                        <span class="font-bold text-accent">{combo.combo_string}</span>
                      </td>
                      <td class="p-2">
                        <div class="flex items-center gap-1 flex-wrap">
                          {#if !combo.components_lte || combo.components_lte.length === 0}
                            <span class="text-muted text-[10px]">{$LL.shannon.ca.noneSa()}</span>
                          {:else}
                            {#each combo.components_lte as comp}
                              <span class="px-1.5 py-0.5 rounded bg-amber-950/40 border border-amber-600/60 text-amber-300 text-[10px]">
                                B{comp.band}{comp.bw_class_dl} [{comp.mimo_dl}x{comp.mimo_dl}]
                              </span>
                            {/each}
                          {/if}
                        </div>
                      </td>
                      <td class="p-2">
                        <div class="flex items-center gap-1 flex-wrap">
                          {#if !combo.components_nr || combo.components_nr.length === 0}
                            <span class="text-muted text-[10px]">{$LL.common.none()}</span>
                          {:else}
                            {#each combo.components_nr as comp}
                              <span class="px-1.5 py-0.5 rounded bg-cyan-950/40 border border-cyan-500/60 text-cyan-300 text-[10px]">
                                n{comp.band}{comp.bw_class_dl} [{comp.mimo_dl}x{comp.mimo_dl}]
                                {#if comp.max_bw_dl > 0}
                                  <span class="text-muted text-[9px]">({comp.max_bw_dl}MHz)</span>
                                {/if}
                                {#if comp.max_scs > 0}
                                  <span class="text-emerald-400 text-[9px]">{comp.max_scs}kHz</span>
                                {/if}
                              </span>
                            {/each}
                          {/if}
                        </div>
                      </td>
                      <td class="p-2 text-center">
                        <Badge variant="info">{combo.total_dl_streams} DL</Badge>
                      </td>
                      <td class="p-2 text-center text-[10px]">
                        {#if combo.power_class && combo.power_class !== 'Default'}
                          <span class="text-amber-400 font-bold block">{combo.power_class}</span>
                        {/if}
                        {#if combo.bcs_nr > 0}
                          <span class="text-muted">BCS:{combo.bcs_nr}</span>
                        {:else}
                          <span class="text-muted">BCS:0</span>
                        {/if}
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
  {/if}

  <!-- VIEW 2: 4G LTE CARRIER AGGREGATION -->
  {#if viewMode === 'lte'}
    <Card
      title={$LL.shannon.ca.lteTitle()}
      subtitle={$LL.shannon.ca.lteSubtitle()}
      tone="ice"
    >
      {#snippet action()}
        <div class="flex items-center flex-wrap gap-1.5">
          <Button variant="outline" size="sm" onclick={() => (showLteUploadModal = true)}>
            <Upload class="w-3.5 h-3.5 mr-1 text-accent" />
            <span>{$LL.shannon.ca.importFile()}</span>
          </Button>

          <Button variant="outline" size="sm" onclick={downloadLteFile} disabled={combos.length === 0}>
            <Download class="w-3.5 h-3.5 mr-1 text-cyan-400" />
            <span>{$LL.shannon.ca.exportFile()}</span>
          </Button>

          <Button variant="outline" size="sm" onclick={() => (showAutoGenModal = true)}>
            <Sliders class="w-3.5 h-3.5 mr-1 text-emerald-400" />
            <span>{$LL.shannon.ca.autoGen()}</span>
          </Button>

          <Button variant="outline" size="sm" onclick={() => (showPruneModal = true)} disabled={combos.length === 0}>
            <Filter class="w-3.5 h-3.5 mr-1 text-purple-400" />
            <span>{$LL.shannon.ca.prune()}</span>
          </Button>

          <Button variant="outline" size="sm" onclick={runValidation} disabled={combos.length === 0}>
            <ShieldAlert class="w-3.5 h-3.5 mr-1 text-amber-400" />
            <span>{$LL.shannon.ca.validate()}</span>
          </Button>

          <Button variant="primary" size="sm" onclick={saveLteFile} disabled={isSaving || combos.length === 0}>
            <Save class="w-3.5 h-3.5 mr-1" />
            <span>{isSaving ? $LL.shannon.ca.saving() : $LL.shannon.ca.saveToSystem()}</span>
          </Button>
        </div>
      {/snippet}

      <div class="space-y-3 font-mono text-xs">
        <!-- FILE SELECTOR & SEARCH -->
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div>
            <label class="block text-[10px] text-muted uppercase font-bold mb-1">
              <span>{$LL.shannon.ca.targetPath()}</span>
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
              <span>{$LL.shannon.ca.searchLabel()}</span>
              <div class="relative mt-1">
                <Search class="w-3.5 h-3.5 text-muted absolute left-2.5 top-2.5" />
                <input
                  type="text"
                  bind:value={lteSearchQuery}
                  class="neo-input w-full bg-card-sub border-2 border-border rounded pl-8 pr-3 py-1.5 text-foreground focus:outline-none focus:border-accent text-xs"
                  placeholder={$LL.shannon.ca.searchPlaceholder()}
                />
              </div>
            </label>
          </div>
        </div>

        <!-- DETECTED FILES SHORTCUTS -->
        {#if status.detected_paths && status.detected_paths.length > 0}
          <div class="flex items-center gap-2 flex-wrap text-[11px] p-2 bg-card-sub border border-border rounded">
            <span class="text-muted font-bold">{$LL.shannon.ca.detectedOnPhone()}</span>
            {#each status.detected_paths as path}
              <button
                type="button"
                class="px-2 py-0.5 bg-zinc-800 hover:bg-accent/20 hover:text-accent rounded border border-border transition-colors cursor-pointer text-[10px]"
                onclick={() => loadLteFile(path)}
              >
                {path}
              </button>
            {/each}
          </div>
        {/if}

        <!-- STATS BAR -->
        <div class="flex items-center justify-between text-[11px] p-2 bg-zinc-950 border border-border rounded">
          <div>
            <span class="text-muted">{$LL.shannon.ca.totalCombos()}</span>
            <span class="font-bold text-accent ml-1">{combos.length}</span>
            {#if combos.length > 128}
              <Badge variant="warning" class="ml-2 text-[9px]">{$LL.shannon.ca.exceeds128()}</Badge>
            {:else if combos.length > 0}
              <Badge variant="success" class="ml-2 text-[9px]">{$LL.shannon.ca.within128()}</Badge>
            {/if}
          </div>

          <div>
            <span class="text-muted">{$LL.shannon.ca.protoBufVersion()}</span>
            <span class="font-bold text-foreground ml-1">{status.version}</span>
          </div>
        </div>

        <!-- COMBO TABLE -->
        {#if filteredLteCombos.length === 0}
          <div class="p-8 text-center border-2 border-dashed border-border rounded space-y-2">
            <FileSpreadsheet class="w-8 h-8 text-muted mx-auto" />
            <p class="font-bold text-foreground">{$LL.shannon.ca.lteEmptyTitle()}</p>
            <p class="text-[11px] text-muted max-w-md mx-auto">
              {$LL.shannon.ca.click()} "{$LL.shannon.ca.autoGen()}" {$LL.shannon.ca.lteEmptyBodyB()} "{$LL.shannon.ca.importFile()}" {$LL.shannon.ca.lteEmptyBodyC()}
            </p>
          </div>
        {:else}
          <div class="border-2 border-border rounded overflow-hidden">
            <div class="max-h-[500px] overflow-y-auto">
              <table class="w-full text-left border-collapse text-[11px]">
                <thead class="bg-card-sub text-muted font-bold uppercase sticky top-0 border-b-2 border-border z-10">
                  <tr>
                    <th class="p-2 w-12 text-center">#</th>
                    <th class="p-2">{$LL.shannon.ca.thCaCombo()}</th>
                    <th class="p-2">{$LL.shannon.ca.thComponents()}</th>
                    <th class="p-2 text-center w-24">{$LL.shannon.ca.thStreams()}</th>
                    <th class="p-2 text-center w-28">{$LL.shannon.ca.thConfIds()}</th>
                    <th class="p-2 text-center w-16">{$LL.shannon.ca.thAction()}</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-border font-mono">
                  {#each filteredLteCombos as combo, idx}
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
                          onclick={() => deleteLteCombo(idx)}
                          title={$LL.shannon.ca.deleteCombo()}
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
  {/if}
</div>

<!-- AUTO GENERATOR MODAL -->
<Modal open={showAutoGenModal} onclose={() => (showAutoGenModal = false)} title={$LL.shannon.ca.autogenTitle()} size="md">
  <div class="space-y-4 font-mono text-xs">
    <p class="text-muted text-[11px] leading-relaxed">
      {$LL.shannon.ca.autogenBodyA()} <code>1+3+8B+40D+28</code>{$LL.shannon.ca.autogenBodyB()}
    </p>

    <div>
      <label class="block text-[10px] text-muted uppercase font-bold mb-1">
        <span>{$LL.shannon.ca.autogenTarget()}</span>
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
          <span>{$LL.shannon.ca.autogenMaxCc()}</span>
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
          <span>{$LL.shannon.ca.autogenUl()}</span>
          <select
            bind:value={autoGenUL}
            class="neo-input mt-1 w-full bg-card-sub border-2 border-border rounded px-2.5 py-1.5 text-foreground focus:outline-none focus:border-accent text-xs"
          >
            <option value="none">{$LL.shannon.ca.ulSingle()}</option>
            <option value="fdd_aa">{$LL.shannon.ca.ulFddAa()}</option>
            <option value="tdd_aa">{$LL.shannon.ca.ulTddAa()}</option>
            <option value="tdd_fdd">{$LL.shannon.ca.ulTddFdd()}</option>
          </select>
        </label>
      </div>
    </div>

    <div>
      <label class="block text-[10px] text-muted uppercase font-bold mb-1">
        <span>{$LL.shannon.ca.autogenConfIds()}</span>
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
    <Button variant="secondary" size="sm" onclick={() => (showAutoGenModal = false)}>{$LL.common.cancel()}</Button>
    <Button variant="primary" size="sm" disabled={isGenerating} onclick={runAutoGen}>
      <span>{isGenerating ? $LL.shannon.ca.generating() : $LL.shannon.ca.generate()}</span>
    </Button>
  {/snippet}
</Modal>

<!-- PRUNE MODAL -->
<Modal open={showPruneModal} onclose={() => (showPruneModal = false)} title={$LL.shannon.ca.pruneTitle()} size="md">
  <div class="space-y-4 font-mono text-xs">
    <p class="text-muted text-[11px] leading-relaxed">
      {$LL.shannon.ca.pruneBodyA()} <code>requestedFrequencyBands-r11</code>{$LL.shannon.ca.pruneBodyB()}
    </p>

    <div>
      <label class="block text-[10px] text-muted uppercase font-bold mb-1">
        <span>{$LL.shannon.ca.pruneBands()}</span>
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
          <span>{$LL.shannon.ca.pruneMaxCount()}</span>
          <input
            type="number"
            bind:value={pruneMaxCombos}
            class="neo-input mt-1 w-full bg-card-sub border-2 border-border rounded px-2.5 py-1.5 text-foreground focus:outline-none focus:border-accent text-xs"
          />
        </label>
      </div>

      <div class="flex items-center pt-5">
        <label class="flex items-center gap-2 cursor-pointer">
          {#if themeStore.useSakugaControls()}
            <SakugaCheckbox bind:checked={pruneExcludeIntra} />
          {:else}
          <input type="checkbox" bind:checked={pruneExcludeIntra} class="accent-accent" />
          {/if}
          <span class="text-[11px] text-foreground font-bold">{$LL.shannon.ca.pruneExcludeIntra()}</span>
        </label>
      </div>
    </div>
  </div>

  {#snippet footer()}
    <Button variant="secondary" size="sm" onclick={() => (showPruneModal = false)}>{$LL.common.cancel()}</Button>
    <Button variant="primary" size="sm" onclick={runPrune}>
      <span>{$LL.shannon.ca.applyPruning()}</span>
    </Button>
  {/snippet}
</Modal>

<!-- VALIDATION RESULTS MODAL -->
<Modal open={showValidationModal} onclose={() => (showValidationModal = false)} title={$LL.shannon.ca.validationTitle()} size="md">
  <div class="space-y-3 font-mono text-xs">
    {#if validationResult}
      <div class="flex items-center justify-between p-3 rounded bg-zinc-950 border-2 border-border">
        <div class="flex items-center gap-2">
          {#if validationResult.valid}
            <CheckCircle2 class="w-5 h-5 text-emerald-400" />
            <span class="font-bold text-emerald-400 text-sm">{$LL.shannon.ca.validConfig()}</span>
          {:else}
            <XCircle class="w-5 h-5 text-red-400" />
            <span class="font-bold text-red-400 text-sm">{$LL.shannon.ca.ruleViolations()}</span>
          {/if}
        </div>
        <div class="flex items-center gap-2">
          <Badge variant={validationResult.error_count > 0 ? 'danger' : 'success'}>
            {$LL.shannon.ca.errorCount({count: validationResult.error_count})}
          </Badge>
          <Badge variant={validationResult.warning_count > 0 ? 'warning' : 'info'}>
            {$LL.shannon.ca.warningCount({count: validationResult.warning_count})}
          </Badge>
        </div>
      </div>

      {#if validationResult.issues.length === 0}
        <p class="text-emerald-300 text-center py-4">
          {$LL.shannon.ca.allValid()}
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
    <Button variant="primary" size="sm" onclick={() => (showValidationModal = false)}>{$LL.common.close()}</Button>
  {/snippet}
</Modal>

<!-- LTE UPLOAD MODAL -->
<Modal open={showLteUploadModal} onclose={() => (showLteUploadModal = false)} title={$LL.shannon.ca.lteUploadTitle()} size="sm">
  <div class="space-y-3 font-mono text-xs">
    <p class="text-muted text-[11px]">
      {$LL.shannon.ca.lteUploadBodyA()} <code>.binarypb</code> {$LL.shannon.ca.lteUploadBodyB()}
    </p>

    <div class="p-4 border-2 border-dashed border-border rounded text-center">
      <input
        type="file"
        accept=".binarypb,.pb,.bin"
        onchange={handleLteFileUpload}
        class="text-xs file:mr-2 file:py-1 file:px-2 file:rounded file:border-0 file:text-xs file:bg-accent file:text-accent-text cursor-pointer"
      />
    </div>
  </div>

  {#snippet footer()}
    <Button variant="secondary" size="sm" onclick={() => (showLteUploadModal = false)}>{$LL.common.cancel()}</Button>
  {/snippet}
</Modal>

<!-- NR UPLOAD MODAL -->
<Modal open={showNrUploadModal} onclose={() => (showNrUploadModal = false)} title={$LL.shannon.ca.nrUploadTitle()} size="sm">
  <div class="space-y-3 font-mono text-xs">
    <p class="text-muted text-[11px]">
      {$LL.shannon.ca.nrUploadBodyA()} <code>.binarypb</code> {$LL.shannon.ca.nrUploadBodyB()} <code>sub6.binarypb</code>, <code>mmWave.binarypb</code>{$LL.shannon.ca.nrUploadBodyC()}
    </p>

    <div class="p-4 border-2 border-dashed border-border rounded text-center">
      <input
        type="file"
        accept=".binarypb,.pb,.bin"
        onchange={handleNrFileUpload}
        class="text-xs file:mr-2 file:py-1 file:px-2 file:rounded file:border-0 file:text-xs file:bg-cyan-500 file:text-black cursor-pointer"
      />
    </div>
  </div>

  {#snippet footer()}
    <Button variant="secondary" size="sm" onclick={() => (showNrUploadModal = false)}>{$LL.common.cancel()}</Button>
  {/snippet}
</Modal>
