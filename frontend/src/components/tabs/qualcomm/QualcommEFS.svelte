<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Folder,
    FileText,
    FileCode,
    Link2,
    Database,
    Download,
    Upload,
    Trash2,
    Edit3,
    RefreshCw,
    Lock,
    Unlock,
    Key,
    RotateCcw,
    AlertTriangle,
    Check,
    HardDrive,
    Search,
    ChevronRight,
    Plus,
    Radio,
    Copy,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Modal from '../../ui/Modal.svelte'
  import EfsEditorModal from './EfsEditorModal.svelte'
  import EfsModals from './EfsModals.svelte'

  interface EfsEntry {
    name: string
    path: string
    type: 'dir' | 'file' | 'link' | 'item' | 'unknown'
    mode: number
    size: number
    target?: string
    mtime?: string
  }

  interface StatFSInfo {
    path: string
    block_size: number
    total_blocks: number
    free_blocks: number
    total_bytes: number
    free_bytes: number
    used_bytes: number
    used_percent: number
  }

  interface EfsStatus {
    connected: boolean
    is_qualcomm: boolean
    readonly: boolean
    transport: string
    subsys: number
    unlocked_spc: boolean
    statfs?: StatFSInfo
    error?: string
  }

  interface CarrierPolicyComponent {
    band: number
    dl_class: string
    dl_mimo: string
    dl_streams: number
    ul_class: string
  }

  interface CarrierPolicyCombo {
    raw_combo_string: string
    components: CarrierPolicyComponent[]
    bcs: string
    total_streams: number
    combo_string: string
  }

  interface CarrierPolicyParseResult {
    total_combos: number
    policy_name?: string
    policy_ver?: string
    max_streams: number
    bands_supported: number[]
    combos: CarrierPolicyCombo[]
  }

  interface FileContent {
    path: string
    size: number
    is_binary: boolean
    text_content?: string
    data_hex?: string
    data_base64?: string
    carrier_policy?: CarrierPolicyParseResult
  }

  interface NVItemResult {
    item: number
    index: number
    data_hex: string
    status: number
    status_str: string
  }

  let status = $state<EfsStatus>({
    connected: false,
    is_qualcomm: false,
    readonly: true,
    transport: '',
    subsys: 0,
    unlocked_spc: false,
  })

  let currentPath = $state<string>('/')
  let entries = $state<EfsEntry[]>([])
  let isLoading = $state(false)
  let isConnecting = $state(false)
  let searchQuery = $state('')

  // Editor Modal States
  let showEditorModal = $state(false)
  let activeFile = $state<FileContent | null>(null)
  let editorMode = $state<'text' | 'hex' | 'ca'>('text')
  let editorTextContent = $state('')
  let isSavingFile = $state(false)

  // Carrier Policy Visualizer Filters
  let caSearchBand = $state('')
  let caMimoFilter = $state<'all' | '4x4' | '2x2'>('all')
  let caUlFilter = $state(false)

  let filteredCaCombos = $derived.by(() => {
    if (!activeFile?.carrier_policy?.combos) return []
    let list = activeFile.carrier_policy.combos

    if (caSearchBand.trim()) {
      const q = caSearchBand.trim().toLowerCase().replace(/^b/, '')
      list = list.filter((c) =>
        c.components.some((comp) => comp.band.toString() === q || comp.band.toString().includes(q))
      )
    }

    if (caMimoFilter === '4x4') {
      list = list.filter((c) => c.components.some((comp) => comp.dl_mimo.includes('4')))
    } else if (caMimoFilter === '2x2') {
      list = list.filter((c) => c.components.every((comp) => !comp.dl_mimo.includes('4')))
    }

    if (caUlFilter) {
      list = list.filter((c) => c.components.some((comp) => comp.ul_class && comp.ul_class !== 'None'))
    }

    return list
  })

  // Standalone CA Combos Parser Modal States
  let showCaModal = $state(false)
  let caInputContent = $state('')
  let isParsingCa = $state(false)

  async function parseCustomCarrierPolicy() {
    if (!caInputContent.trim()) {
      toastStore.warning('Please enter XML or combo string to parse.')
      return
    }
    try {
      isParsingCa = true
      const res = await api.post<{ ok: boolean; result: CarrierPolicyParseResult; error?: string }>(
        '/api/efs/carrier-policy/parse',
        { content: caInputContent }
      )
      if (res && res.ok && res.result && res.result.total_combos > 0) {
        activeFile = {
          path: 'Custom Carrier Policy / Combo Input',
          size: caInputContent.length,
          is_binary: false,
          carrier_policy: res.result,
        }
        editorMode = 'ca'
        caSearchBand = ''
        caMimoFilter = 'all'
        caUlFilter = false
        showCaModal = false
        showEditorModal = true
        toastStore.success(`Parsed ${res.result.total_combos} combinations successfully!`)
      } else {
        toastStore.error(res?.error || 'No valid CA combinations found in input.')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to parse carrier policy')
    } finally {
      isParsingCa = false
    }
  }

  function loadSampleCarrierPolicy() {
    caInputContent = `<ca_band_combos string="4A2A-2A2-mAll;4A2-2A2A-mAll;5A2A-2A2-mAll;5A2-2A2A-mAll;66A2A-2A2-mAll;66A2-2A2A-mAll;4A2A-4A2-mAll;5A2A-4A2-mAll;5A2-4A2A-mAll;66A2A-5A2-mAll;66A2-5A2A-mAll;26A2A-25A2-mAll;26A2-25A2A-mAll;41A2A-41A2-mAll;66A2A-66A2-mAll;1C22A-mAll;66C22A-mAll;5A2A-4A2-2A2-mAll;5A2-4A2A-2A2-mAll;5A2-4A2-2A2A-mAll;66A2A-5A2-2A2-mAll;66A2-5A2A-2A2-mAll;66A2-5A2-2A2A-mAll;66A2A-66A2-2A2-mAll;66A2-66A2-2A2A-mAll;5A2A-4A2-4A2-mAll;5A2-4A2A-4A2-mAll;66A2A-66C22-mAll;66A2-66C22A-mAll;66A2A-66A2-5A2-mAll;66A2-66A2-5A2A-mAll;66C22A-5A2-mAll;66C22-5A2A-mAll;41A2A-41C22-mAll;41A2-41C22A-mAll;41D222A-mAll;46C22-2A2A-mAll;46C22-4A2A-mAll;66A2A-46C22-mAll;" />`
  }

  // SPC Unlock Modal States
  let showSpcModal = $state(false)
  let spcInput = $state('000000')
  let isUnlockingSpc = $state(false)

  // NV Inspector Modal States
  let showNvModal = $state(false)
  let nvItemInput = $state<number>(550)
  let nvIndexInput = $state<number>(-1)
  let nvHexInput = $state('')
  let nvResult = $state<NVItemResult | null>(null)
  let isLoadingNv = $state(false)
  let isWritingNv = $state(false)

  // SSR Modal
  let showSsrModal = $state(false)
  let isRestartingModem = $state(false)

  // Read-Only Warning Modal
  let showUnlockWarningModal = $state(false)

  // Create Dir Modal
  let showMkdirModal = $state(false)
  let newDirName = $state('')
  let isCreatingDir = $state(false)

  // Delete Modal
  let showDeleteModal = $state(false)
  let entryToDelete = $state<EfsEntry | null>(null)
  let isDeleting = $state(false)

  onMount(async () => {
    await fetchStatus()
    if (status.is_qualcomm) {
      await loadDirectory('/')
    }
  })

  async function fetchStatus() {
    try {
      isLoading = true
      const res = await api.get<EfsStatus>('/api/efs/status')
      if (res) {
        status = res
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to query EFS status')
    } finally {
      isLoading = false
    }
  }

  async function connectEfs() {
    try {
      isConnecting = true
      const res = await api.post<{ ok: boolean; status: EfsStatus; error?: string }>('/api/efs/connect', {})
      if (res && res.ok) {
        status = res.status
        toastStore.success('Connected to Qualcomm DIAG service on QRTR.')
        await loadDirectory(currentPath)
      } else {
        toastStore.error(res?.error || 'Failed to connect to DIAG service')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Connection failed')
    } finally {
      isConnecting = false
    }
  }

  async function loadDirectory(dirPath: string) {
    try {
      isLoading = true
      const res = await api.get<{ ok: boolean; path: string; entries: EfsEntry[]; error?: string }>(
        `/api/efs/list?path=${encodeURIComponent(dirPath)}`
      )
      if (res && res.ok) {
        currentPath = res.path
        entries = res.entries || []
        searchQuery = ''
      } else {
        toastStore.error(res?.error || `Failed to list '${dirPath}'`)
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Directory listing failed')
    } finally {
      isLoading = false
    }
  }

  function navigateTo(pathSegment: string) {
    if (pathSegment === '/') {
      loadDirectory('/')
      return
    }
    const clean = pathSegment.startsWith('/') ? pathSegment : `/${pathSegment}`
    loadDirectory(clean)
  }

  function handleEntryClick(entry: EfsEntry) {
    if (entry.type === 'dir') {
      loadDirectory(entry.path)
    } else {
      openFile(entry.path)
    }
  }

  async function openFile(filePath: string) {
    try {
      isLoading = true
      const res = await api.get<FileContent & { ok: boolean; error?: string }>(
        `/api/efs/read?path=${encodeURIComponent(filePath)}`
      )
      if (res && res.ok) {
        activeFile = res
        editorTextContent = res.text_content || ''
        if (res.carrier_policy && res.carrier_policy.total_combos > 0) {
          editorMode = 'ca'
          caSearchBand = ''
          caMimoFilter = 'all'
          caUlFilter = false
        } else {
          editorMode = res.is_binary ? 'hex' : 'text'
        }
        showEditorModal = true
      } else {
        toastStore.error(res?.error || 'Failed to open file')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'File read failed')
    } finally {
      isLoading = false
    }
  }

  function copySingleCombo(str: string) {
    if (!str) return
    navigator.clipboard.writeText(str)
    toastStore.success('Combination string copied.')
  }

  function copyAllCombos() {
    if (!activeFile?.carrier_policy?.combos) return
    const rawList = filteredCaCombos.map((c) => c.raw_combo_string).join(';\n')
    navigator.clipboard.writeText(rawList)
    toastStore.success(`Copied ${filteredCaCombos.length} combinations to clipboard.`)
  }

  function exportCombosJson() {
    if (!activeFile?.carrier_policy) return
    const blob = new Blob([JSON.stringify(activeFile.carrier_policy, null, 2)], {
      type: 'application/json',
    })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `carrier_policy_combos_${Date.now()}.json`
    a.click()
    URL.revokeObjectURL(url)
    toastStore.success('Carrier policy exported to JSON.')
  }


  async function saveFile() {
    if (!activeFile) return
    if (status.readonly) {
      toastStore.warning('Modem filesystem is locked in Read-Only mode. Unlock write access first.')
      return
    }

    try {
      isSavingFile = true
      const payload = {
        path: activeFile.path,
        data: editorMode === 'text' ? editorTextContent : activeFile.data_hex,
        format: editorMode === 'text' ? 'text' : 'hex',
        mode: 0o644,
        force_item: activeFile.path.includes('/nv/item_files/') ? 1 : 0,
      }
      const res = await api.post<{ ok: boolean; error?: string }>('/api/efs/write', payload)
      if (res && res.ok) {
        toastStore.success(`Saved '${activeFile.path}' successfully.`)
        showEditorModal = false
        await loadDirectory(currentPath)
      } else {
        toastStore.error(res?.error || 'Failed to save file to EFS')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Save failed')
    } finally {
      isSavingFile = false
    }
  }

  async function toggleReadOnly() {
    if (status.readonly) {
      showUnlockWarningModal = true
    } else {
      await setReadOnlyMode(true)
    }
  }

  async function setReadOnlyMode(ro: boolean) {
    try {
      const res = await api.post<{ ok: boolean; readonly: boolean }>('/api/efs/readonly', { readonly: ro })
      if (res && res.ok) {
        status.readonly = res.readonly
        showUnlockWarningModal = false
        toastStore.info(ro ? 'Read-Only mode engaged.' : 'Write access unlocked. Be cautious with modifications!')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to toggle Read-Only')
    }
  }

  async function submitSpcUnlock() {
    try {
      isUnlockingSpc = true
      const res = await api.post<{ ok: boolean; unlocked: boolean; error?: string }>('/api/efs/spc', {
        spc: spcInput.trim(),
      })
      if (res && res.ok && res.unlocked) {
        status.unlocked_spc = true
        toastStore.success('SPC accepted: Modem DIAG security session elevated.')
        showSpcModal = false
      } else {
        toastStore.error(res?.error || 'SPC rejected by modem. Ensure code is correct.')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'SPC verification failed')
    } finally {
      isUnlockingSpc = false
    }
  }

  async function triggerSsr() {
    try {
      isRestartingModem = true
      const res = await api.post<{ ok: boolean; message?: string; error?: string }>('/api/efs/ssr', {})
      if (res && res.ok) {
        toastStore.success('Modem Subsystem Restart (SSR) initiated. Reconnecting in 3 seconds...')
        showSsrModal = false
        setTimeout(async () => {
          await connectEfs()
        }, 3000)
      } else {
        toastStore.error(res?.error || 'Modem restart failed')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Restart request failed')
    } finally {
      isRestartingModem = false
    }
  }

  async function createDirectory() {
    if (!newDirName.trim()) return
    const target = currentPath === '/' ? `/${newDirName.trim()}` : `${currentPath}/${newDirName.trim()}`
    try {
      isCreatingDir = true
      const res = await api.post<{ ok: boolean; error?: string }>('/api/efs/mkdir', { path: target, mode: 0o755 })
      if (res && res.ok) {
        toastStore.success(`Created directory '${target}'`)
        showMkdirModal = false
        newDirName = ''
        await loadDirectory(currentPath)
      } else {
        toastStore.error(res?.error || 'Failed to create directory')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Directory creation failed')
    } finally {
      isCreatingDir = false
    }
  }

  async function confirmDelete() {
    if (!entryToDelete) return
    try {
      isDeleting = true
      const res = await api.post<{ ok: boolean; error?: string }>('/api/efs/unlink', { path: entryToDelete.path })
      if (res && res.ok) {
        toastStore.success(`Deleted '${entryToDelete.name}'`)
        showDeleteModal = false
        entryToDelete = null
        await loadDirectory(currentPath)
      } else {
        toastStore.error(res?.error || 'Failed to delete entry')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Deletion failed')
    } finally {
      isDeleting = false
    }
  }

  async function readNvItem() {
    try {
      isLoadingNv = true
      nvResult = null
      const url = `/api/efs/nv/read?item=${nvItemInput}${nvIndexInput >= 0 ? `&index=${nvIndexInput}` : ''}`
      const res = await api.get<{ ok: boolean; data: NVItemResult; error?: string }>(url)
      if (res && res.ok) {
        nvResult = res.data
        nvHexInput = res.data.data_hex
      } else {
        toastStore.error(res?.error || 'Failed to read NV item')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'NV read failed')
    } finally {
      isLoadingNv = false
    }
  }

  async function writeNvItem() {
    if (status.readonly) {
      toastStore.warning('EFS is locked in Read-Only mode. Unlock write access first.')
      return
    }
    try {
      isWritingNv = true
      const payload = {
        item: nvItemInput,
        index: nvIndexInput,
        data_hex: nvHexInput.trim(),
      }
      const res = await api.post<{ ok: boolean; data: NVItemResult; error?: string }>('/api/efs/nv/write', payload)
      if (res && res.ok) {
        toastStore.success(`NV item ${nvItemInput} written successfully (Status: ${res.data.status_str}).`)
        nvResult = res.data
      } else {
        toastStore.error(res?.error || 'Failed to write NV item')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'NV write failed')
    } finally {
      isWritingNv = false
    }
  }

  function downloadFile(p: string) {
    window.open(`/api/efs/read?path=${encodeURIComponent(p)}&format=download`, '_blank')
  }

  function downloadBackup() {
    window.open(`/api/efs/backup?path=${encodeURIComponent(currentPath)}`, '_blank')
  }

  function formatBytes(bytes: number): string {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`
  }

  // Filtered entries
  const filteredEntries = $derived.by(() => {
    if (!searchQuery.trim()) return entries
    const q = searchQuery.toLowerCase()
    return entries.filter((e) => e.name.toLowerCase().includes(q))
  })

  // Breadcrumbs segments
  const breadcrumbSegments = $derived.by(() => {
    if (currentPath === '/') return []
    const parts = currentPath.split('/').filter(Boolean)
    let cumulative = ''
    return parts.map((part) => {
      cumulative += `/${part}`
      return { label: part, path: cumulative }
    })
  })
</script>

<div class="space-y-4 text-xs font-mono">
  <!-- STATUS & TOOLBAR HEADER -->
  <Card tone="ice" title="Qualcomm EFS2 Modem Filesystem Explorer" subtitle="Direct hardware EFS2 tree browsing and NV item management via DIAG QRTR service 4097">
    {#snippet action()}
      <div class="flex items-center gap-1.5 flex-wrap">
        <Button
          variant={status.readonly ? 'outline' : 'danger'}
          size="sm"
          onclick={toggleReadOnly}
        >
          {#if status.readonly}
            <Lock class="w-3.5 h-3.5 mr-1 text-amber-400" />
            <span>Read-Only</span>
          {:else}
            <Unlock class="w-3.5 h-3.5 mr-1 text-red-400 animate-pulse" />
            <span>Write Allowed</span>
          {/if}
        </Button>

        <Button
          variant="outline"
          size="sm"
          onclick={() => (showSpcModal = true)}
        >
          <Key class="w-3.5 h-3.5 mr-1 {status.unlocked_spc ? 'text-emerald-400' : 'text-muted'}" />
          <span>{status.unlocked_spc ? 'SPC Unlocked' : 'Unlock SPC'}</span>
        </Button>

        <Button
          variant="outline"
          size="sm"
          onclick={() => (showNvModal = true)}
        >
          <Database class="w-3.5 h-3.5 mr-1 text-cyan-400" />
          <span>NV Inspector</span>
        </Button>

        <Button
          variant="outline"
          size="sm"
          onclick={() => (showCaModal = true)}
        >
          <Radio class="w-3.5 h-3.5 mr-1 text-emerald-400" />
          <span>CA Visualizer</span>
        </Button>

        <Button
          variant="outline"
          size="sm"
          disabled={isLoading}
          onclick={downloadBackup}
        >
          <Download class="w-3.5 h-3.5 mr-1 text-accent" />
          <span>Dump .TAR</span>
        </Button>

        <Button
          variant="secondary"
          size="sm"
          disabled={status.readonly}
          onclick={() => (showSsrModal = true)}
        >
          <RotateCcw class="w-3.5 h-3.5 mr-1 text-amber-400" />
          <span>Restart Modem</span>
        </Button>

        <Button
          variant="primary"
          size="sm"
          disabled={isLoading || isConnecting}
          onclick={connectEfs}
        >
          <RefreshCw class="w-3.5 h-3.5 mr-1 text-accent-text {isLoading || isConnecting ? 'animate-spin' : ''}" />
          <span>{status.connected ? 'Refresh' : 'Connect'}</span>
        </Button>
      </div>
    {/snippet}

    <!-- METRICS STRIP -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-2.5 pt-1 text-xs">
      <div class="p-2 bg-card-sub border border-border rounded flex flex-col justify-between">
        <span class="text-[10px] text-muted uppercase font-bold">Baseband DIAG Bus</span>
        <div class="flex items-center gap-1.5 mt-1">
          <span class="w-2 h-2 rounded-full {status.connected ? 'bg-emerald-400 animate-pulse' : 'bg-red-400'}"></span>
          <span class="font-bold text-foreground truncate">{status.transport || 'QRTR Svc 4097'}</span>
        </div>
      </div>

      <div class="p-2 bg-card-sub border border-border rounded flex flex-col justify-between">
        <span class="text-[10px] text-muted uppercase font-bold">EFS Subsystem Protocol</span>
        <span class="font-bold text-cyan-400 mt-1">
          {status.subsys === 0x13 ? '0x13 (Standard)' : status.subsys === 0x3E ? '0x3E (Alternate)' : 'Auto-Detected'}
        </span>
      </div>

      <div class="p-2 bg-card-sub border border-border rounded flex flex-col justify-between">
        <span class="text-[10px] text-muted uppercase font-bold">Write Safety State</span>
        <span class="font-bold {status.readonly ? 'text-amber-400' : 'text-red-400'} mt-1">
          {status.readonly ? 'PROTECTED (Read-Only)' : 'UNLOCKED (Raw NV Write)'}
        </span>
      </div>

      <div class="p-2 bg-card-sub border border-border rounded flex flex-col justify-between">
        <span class="text-[10px] text-muted uppercase font-bold">EFS Storage Capacity</span>
        {#if status.statfs}
          <div class="flex items-center justify-between gap-1 mt-1">
            <span class="font-bold text-foreground">
              {formatBytes(status.statfs.free_bytes)} free
            </span>
            <span class="text-[10px] text-muted">
              {status.statfs.used_percent}% used
            </span>
          </div>
        {:else}
          <span class="font-bold text-muted mt-1">Ready on Query</span>
        {/if}
      </div>
    </div>
  </Card>

  <!-- BREADCRUMB NAVIGATION & DIRECTORY ACTIONS -->
  <div class="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-2 bg-card p-2.5 border-2 border-border rounded-lg shadow-neobrutal-sm">
    <div class="flex items-center gap-1 overflow-x-auto text-xs py-1">
      <button
        type="button"
        class="px-2 py-1 rounded hover:bg-accent/15 font-bold cursor-pointer text-accent flex items-center gap-1"
        onclick={() => navigateTo('/')}
      >
        <HardDrive class="w-3.5 h-3.5" />
        <span>efs://</span>
      </button>

      {#each breadcrumbSegments as segment}
        <ChevronRight class="w-3 h-3 text-muted shrink-0" />
        <button
          type="button"
          class="px-1.5 py-1 rounded hover:bg-accent/15 font-bold cursor-pointer text-foreground hover:text-accent whitespace-nowrap"
          onclick={() => navigateTo(segment.path)}
        >
          {segment.label}
        </button>
      {/each}
    </div>

    <div class="flex items-center gap-1.5 shrink-0">
      <div class="relative w-full sm:w-44">
        <Search class="w-3 h-3 absolute left-2 top-2.5 text-muted pointer-events-none" />
        <input
          type="text"
          bind:value={searchQuery}
          placeholder="Filter entries..."
          class="neo-input w-full bg-card-sub border border-border rounded pl-7 pr-2 py-1 text-[11px] text-foreground focus:outline-none focus:border-accent"
        />
      </div>

      <Button
        variant="outline"
        size="sm"
        disabled={status.readonly}
        onclick={() => (showMkdirModal = true)}
      >
        <Plus class="w-3.5 h-3.5 mr-1 text-emerald-400" />
        <span>New Folder</span>
      </Button>
    </div>
  </div>

  <!-- DIRECTORY FILE TABLE -->
  <div class="border-2 border-border rounded-lg overflow-hidden bg-card shadow-neobrutal-sm">
    <div class="overflow-x-auto max-h-[500px]">
      <table class="w-full text-left border-collapse text-[11px]">
        <thead class="bg-card-sub border-b-2 border-border sticky top-0 uppercase text-[10px] text-muted font-bold">
          <tr>
            <th class="p-2.5">Name</th>
            <th class="p-2.5 w-24">Type</th>
            <th class="p-2.5 w-24 text-right">Size</th>
            <th class="p-2.5 w-20 text-center">Mode</th>
            <th class="p-2.5 w-36">Modified</th>
            <th class="p-2.5 w-24 text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border/60">
          {#if currentPath !== '/'}
            <tr
              class="hover:bg-accent/10 cursor-pointer transition-colors"
              onclick={() => {
                const parent = currentPath.substring(0, currentPath.lastIndexOf('/')) || '/'
                navigateTo(parent)
              }}
            >
              <td colspan="6" class="p-2 text-accent font-bold flex items-center gap-2">
                <ChevronRight class="w-3.5 h-3.5 rotate-180" />
                <span>.. (Parent Directory)</span>
              </td>
            </tr>
          {/if}

          {#if filteredEntries.length === 0}
            <tr>
              <td colspan="6" class="p-8 text-center text-muted">
                {#if isLoading}
                  <div class="flex items-center justify-center gap-2">
                    <RefreshCw class="w-4 h-4 animate-spin text-accent" />
                    <span>Reading modem EFS2 tree...</span>
                  </div>
                {:else}
                  <span>Directory is empty or no matching entries found.</span>
                {/if}
              </td>
            </tr>
          {:else}
            {#each filteredEntries as item}
              <tr class="hover:bg-card-sub transition-colors group">
                <!-- Name -->
                <td class="p-2 font-medium">
                  <button
                    type="button"
                    class="flex items-center gap-2 text-left cursor-pointer hover:text-accent font-bold"
                    onclick={() => handleEntryClick(item)}
                  >
                    {#if item.type === 'dir'}
                      <Folder class="w-4 h-4 text-amber-400 shrink-0 fill-amber-400/20" />
                    {:else if item.type === 'item'}
                      <Database class="w-4 h-4 text-cyan-400 shrink-0" />
                    {:else if item.type === 'link'}
                      <Link2 class="w-4 h-4 text-purple-400 shrink-0" />
                    {:else}
                      <FileText class="w-4 h-4 text-muted-foreground shrink-0" />
                    {/if}
                    <span class="truncate max-w-[200px] sm:max-w-xs">{item.name}</span>
                    {#if item.target}
                      <span class="text-[10px] text-muted truncate">→ {item.target}</span>
                    {/if}
                  </button>
                </td>

                <!-- Type -->
                <td class="p-2">
                  <span class="px-1.5 py-0.5 rounded text-[9px] font-bold uppercase {item.type === 'dir' ? 'bg-amber-950/40 text-amber-300 border border-amber-600/40' : item.type === 'item' ? 'bg-cyan-950/40 text-cyan-300 border border-cyan-600/40' : item.type === 'link' ? 'bg-purple-950/40 text-purple-300 border border-purple-600/40' : 'bg-zinc-800 text-zinc-300'}">
                    {item.type}
                  </span>
                </td>

                <!-- Size -->
                <td class="p-2 text-right text-muted">
                  {item.type === 'dir' ? '-' : formatBytes(item.size)}
                </td>

                <!-- Mode -->
                <td class="p-2 text-center text-muted font-mono">
                  {item.mode ? (item.mode & 0o777).toString(8).padStart(3, '0') : '-'}
                </td>

                <!-- Modified -->
                <td class="p-2 text-muted truncate">
                  {item.mtime ? new Date(item.mtime).toLocaleDateString() : '-'}
                </td>

                <!-- Actions -->
                <td class="p-2 text-right">
                  <div class="flex items-center justify-end gap-1 opacity-80 group-hover:opacity-100">
                    {#if item.type !== 'dir'}
                      <button
                        type="button"
                        class="p-1 hover:text-accent rounded cursor-pointer"
                        title="Download"
                        onclick={() => downloadFile(item.path)}
                      >
                        <Download class="w-3.5 h-3.5" />
                      </button>
                      <button
                        type="button"
                        class="p-1 hover:text-cyan-400 rounded cursor-pointer"
                        title="View / Edit"
                        onclick={() => openFile(item.path)}
                      >
                        <Edit3 class="w-3.5 h-3.5" />
                      </button>
                    {/if}

                    <button
                      type="button"
                      class="p-1 hover:text-red-400 rounded cursor-pointer {status.readonly ? 'opacity-30 pointer-events-none' : ''}"
                      title="Delete"
                      disabled={status.readonly}
                      onclick={() => {
                        entryToDelete = item
                        showDeleteModal = true
                      }}
                    >
                      <Trash2 class="w-3.5 h-3.5" />
                    </button>
                  </div>
                </td>
              </tr>
            {/each}
          {/if}
        </tbody>
      </table>
    </div>
  </div>
</div>

<!-- FILE VIEWER / EDITOR MODAL -->
  <EfsEditorModal
    bind:open={showEditorModal}
    activeFile={activeFile}
    status={status}
    bind:editorMode={editorMode}
    bind:editorTextContent={editorTextContent}
    filteredCaCombos={filteredCaCombos}
    bind:caSearchBand={caSearchBand}
    bind:caMimoFilter={caMimoFilter}
    bind:caUlFilter={caUlFilter}
    isSavingFile={isSavingFile}
    formatBytes={formatBytes}
    downloadFile={downloadFile}
    saveFile={saveFile}
    copySingleCombo={copySingleCombo}
    copyAllCombos={copyAllCombos}
    exportCombosJson={exportCombosJson}
  />

  <EfsModals
    status={status}
    bind:showSpcModal={showSpcModal}
    bind:spcInput={spcInput}
    isUnlockingSpc={isUnlockingSpc}
    submitSpcUnlock={submitSpcUnlock}

    bind:showNvModal={showNvModal}
    bind:nvItemInput={nvItemInput}
    bind:nvIndexInput={nvIndexInput}
    isLoadingNv={isLoadingNv}
    nvResult={nvResult}
    bind:nvHexInput={nvHexInput}
    isWritingNv={isWritingNv}
    readNvItem={readNvItem}
    writeNvItem={writeNvItem}

    bind:showSsrModal={showSsrModal}
    isRestartingModem={isRestartingModem}
    triggerSsr={triggerSsr}

    bind:showUnlockWarningModal={showUnlockWarningModal}
    setReadOnlyMode={setReadOnlyMode}

    bind:showMkdirModal={showMkdirModal}
    bind:newDirName={newDirName}
    isCreatingDir={isCreatingDir}
    createDirectory={createDirectory}

    bind:showDeleteModal={showDeleteModal}
    entryToDelete={entryToDelete}
    isDeleting={isDeleting}
    confirmDelete={confirmDelete}

    bind:showCaModal={showCaModal}
    bind:caInputContent={caInputContent}
    isParsingCa={isParsingCa}
    loadSampleCarrierPolicy={loadSampleCarrierPolicy}
    parseCustomCarrierPolicy={parseCustomCarrierPolicy}
  />
