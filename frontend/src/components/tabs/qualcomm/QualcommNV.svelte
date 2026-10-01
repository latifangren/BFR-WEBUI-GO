<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Database,
    Search,
    RefreshCw,
    Edit3,
    Download,
    Upload,
    Save,
    AlertTriangle,
    Check,
    X,
    Copy,
    Cpu,
    MinusCircle,
    Key,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Button from '../../ui/Button.svelte'
  import Modal from '../../ui/Modal.svelte'

  // ─── Types ───────────────────────────────────────────────────────────────────

  interface NVItem {
    id: number
    name: string
    category: string
    description: string
    size: number
    is_extended: boolean
    data_type: 'hex' | 'bcd' | 'ascii' | 'binary'
    read_only: boolean
  }

  interface NVReadResult {
    id: number
    name: string
    category: string
    index: number
    data_hex: string
    formatted: string
    data_type: string
    size: number
    success: boolean
    error?: string
  }

  interface NVBackupStats {
    current: number
    total: number
    success: number
    failed: number
    skipped: number
    done: boolean
    error?: string
  }

  // ─── State ───────────────────────────────────────────────────────────────────

  let allItems = $state<NVItem[]>([])
  let categories = $state<string[]>([])
  let filteredItems = $state<NVItem[]>([])

  let selectedCategory = $state('all')
  let searchQuery = $state('')
  let isLoading = $state(false)

  // Selected item + read result
  let selectedItem = $state<NVItem | null>(null)
  let readResult = $state<NVReadResult | null>(null)
  let isReading = $state(false)

  // Write modal — use bind:open pattern
  let writeModalOpen = $state(false)
  let writeHex = $state('')
  let isWriting = $state(false)
  let writeError = $state('')

  // Backup modal
  let backupModalOpen = $state(false)
  let backupMode = $state<'catalog' | 'range'>('catalog')
  let backupFilename = $state('/sdcard/nv_backup.json')
  let backupRangeStart = $state(0)
  let backupRangeEnd = $state(1000)
  let isBackingUp = $state(false)
  let backupStats = $state<NVBackupStats | null>(null)

  // Restore modal
  let restoreModalOpen = $state(false)
  let restoreFilename = $state('/sdcard/nv_backup.json')
  let isRestoring = $state(false)
  let restoreStats = $state<NVBackupStats | null>(null)

  // SPC unlock state
  let showSpcModal = $state(false)
  let spcInput = $state('000000')
  let isUnlockingSpc = $state(false)
  let isSpcUnlocked = $state(false)

  // ─── Derived ─────────────────────────────────────────────────────────────────

  // A "not available" result means modem simply doesn't have this NV item —
  // not a real error, just absent. Bad command / rejected = not provisioned.
  const isNotAvailable = $derived(
    !!readResult && !readResult.success &&
    !!readResult.error && (
      readResult.error.toLowerCase().includes('rejected') ||
      readResult.error.toLowerCase().includes('bad command') ||
      readResult.error.toLowerCase().includes('bad_cmd') ||
      readResult.error.toLowerCase().includes('0x13') ||
      readResult.error.toLowerCase().includes('status=4') ||
      readResult.error.toLowerCase().includes('status=5') ||
      readResult.error.toLowerCase().includes('status=6') ||
      readResult.error.toLowerCase().includes('status=8') ||
      readResult.error.toLowerCase().includes('not active') ||
      readResult.error.toLowerCase().includes('not defined') ||
      readResult.error.toLowerCase().includes('unsupported')
    )
  )

  // ─── Lifecycle ────────────────────────────────────────────────────────────────

  onMount(async () => {
    try {
      const st = await api.get<{ ok: boolean; status?: { unlocked_spc: boolean } }>('/api/efs/status')
      if (st?.status?.unlocked_spc) {
        isSpcUnlocked = true
      }
    } catch {}
    await loadCatalog()
  })

  // ─── Catalog ─────────────────────────────────────────────────────────────────

  async function loadCatalog() {
    try {
      isLoading = true
      const res = await api.get<{ items: NVItem[]; categories: string[] }>('/api/efs/nv/items')
      if (res) {
        allItems = res.items
        categories = res.categories
        applyFilter()
      }
    } catch {
      toastStore.error('Failed to load NV catalog')
    } finally {
      isLoading = false
    }
  }

  function applyFilter() {
    let items = allItems
    if (selectedCategory !== 'all') {
      items = items.filter((i) => i.category === selectedCategory)
    }
    if (searchQuery.trim() !== '') {
      const q = searchQuery.toLowerCase()
      items = items.filter(
        (i) =>
          i.name.toLowerCase().includes(q) ||
          i.description.toLowerCase().includes(q) ||
          String(i.id).includes(q),
      )
    }
    filteredItems = items.slice().sort((a, b) => a.id - b.id)
  }

  $effect(() => {
    selectedCategory
    searchQuery
    applyFilter()
  })

  // ─── Read ─────────────────────────────────────────────────────────────────────

  async function readItem(item: NVItem) {
    selectedItem = item
    readResult = null
    writeError = ''
    writeHex = ''
    try {
      isReading = true
      const res = await api.post<{ ok: boolean; result: NVReadResult; error?: string }>(
        '/api/efs/nv/read',
        { id: item.id, index: 0 },
      )
      if (res?.result) {
        readResult = res.result
        if (res.result.success) {
          writeHex = res.result.data_hex
        }
      } else if (res && !res.ok) {
        readResult = {
          id: item.id, name: item.name, category: item.category,
          index: 0, data_hex: '', formatted: '', data_type: item.data_type,
          size: 0, success: false, error: res.error ?? 'Unknown error',
        }
      }
    } catch (e: any) {
      readResult = {
        id: item.id, name: item.name, category: item.category,
        index: 0, data_hex: '', formatted: '', data_type: item.data_type,
        size: 0, success: false, error: e?.message ?? 'Request failed',
      }
    } finally {
      isReading = false
    }
  }

  // ─── Write ───────────────────────────────────────────────────────────────────

  async function submitSpcUnlock() {
    try {
      isUnlockingSpc = true
      const res = await api.post<{ ok: boolean; unlocked: boolean; error?: string }>('/api/efs/spc', {
        spc: spcInput.trim(),
      })
      if (res?.ok && res.unlocked) {
        isSpcUnlocked = true
        toastStore.success('SPC accepted: Modem DIAG security session elevated.')
        showSpcModal = false
      } else {
        toastStore.error(res?.error || 'SPC rejected by modem. Ensure code is correct.')
      }
    } catch (err: any) {
      toastStore.error(err instanceof Error ? err.message : 'SPC verification failed')
    } finally {
      isUnlockingSpc = false
    }
  }

  function openWriteModal() {
    writeError = ''
    if (!writeHex && readResult?.data_hex) {
      writeHex = readResult.data_hex
    }
    writeModalOpen = true
  }

  async function writeItem() {
    if (!selectedItem) return
    writeError = ''
    try {
      isWriting = true
      const res = await api.post<{ ok: boolean; error?: string }>('/api/efs/nv/write', {
        id: selectedItem.id,
        index: 0,
        data_hex: writeHex.replace(/\s/g, ''),
      })
      if (res?.ok) {
        toastStore.success(`NV ${selectedItem.id} (${selectedItem.name}) written`)
        writeModalOpen = false
        await readItem(selectedItem)
      } else {
        writeError = res?.error ?? 'Write failed'
      }
    } catch (e: any) {
      writeError = e?.message ?? 'DIAG write error'
    } finally {
      isWriting = false
    }
  }

  // ─── Backup ──────────────────────────────────────────────────────────────────

  async function runBackup() {
    backupStats = null
    try {
      isBackingUp = true
      const body: Record<string, unknown> = { filename: backupFilename }
      if (backupMode === 'range') {
        body.start_id = backupRangeStart
        body.end_id = backupRangeEnd
      }
      const res = await api.post<{ ok: boolean; stats: NVBackupStats; error?: string }>(
        '/api/efs/nv/backup', body,
      )
      if (res?.ok) {
        backupStats = res.stats
        toastStore.success(`Backup: ${res.stats.success} ok, ${res.stats.failed} failed`)
      } else {
        toastStore.error(res?.error ?? 'Backup failed')
      }
    } catch (e: any) {
      toastStore.error(e?.message ?? 'Backup error')
    } finally {
      isBackingUp = false
    }
  }

  // ─── Restore ─────────────────────────────────────────────────────────────────

  async function runRestore() {
    restoreStats = null
    try {
      isRestoring = true
      const res = await api.post<{ ok: boolean; stats: NVBackupStats; error?: string }>(
        '/api/efs/nv/restore', { filename: restoreFilename },
      )
      if (res?.ok) {
        restoreStats = res.stats
        toastStore.success(`Restore: ${res.stats.success} ok, ${res.stats.failed} failed`)
      } else {
        toastStore.error(res?.error ?? 'Restore failed')
      }
    } catch (e: any) {
      toastStore.error(e?.message ?? 'Restore error')
    } finally {
      isRestoring = false
    }
  }

  // ─── Helpers ─────────────────────────────────────────────────────────────────

  function copyToClipboard(text: string) {
    navigator.clipboard.writeText(text).then(() => toastStore.success('Copied'))
  }

  function categoryColor(cat: string): string {
    const map: Record<string, string> = {
      Identity: 'text-cyan-400 border-cyan-500/40 bg-cyan-500/10',
      Network: 'text-blue-400 border-blue-500/40 bg-blue-500/10',
      Security: 'text-red-400 border-red-500/40 bg-red-500/10',
      Radio: 'text-purple-400 border-purple-500/40 bg-purple-500/10',
      System: 'text-orange-400 border-orange-500/40 bg-orange-500/10',
      User: 'text-green-400 border-green-500/40 bg-green-500/10',
      Advanced: 'text-yellow-400 border-yellow-500/40 bg-yellow-500/10',
    }
    return map[cat] ?? 'text-muted border-border bg-card'
  }

  function dataTypeBadge(dt: string): string {
    const map: Record<string, string> = {
      bcd: 'text-emerald-400 border-emerald-500/40 bg-emerald-500/10',
      ascii: 'text-sky-400 border-sky-500/40 bg-sky-500/10',
      binary: 'text-violet-400 border-violet-500/40 bg-violet-500/10',
      hex: 'text-amber-400 border-amber-500/40 bg-amber-500/10',
    }
    return map[dt] ?? 'text-muted border-border bg-card'
  }

  function fmtHex(h: string): string {
    return h.toUpperCase().match(/.{1,2}/g)?.join(' ') ?? h
  }
</script>

<!-- ─────────────────────────────────────────────────────────────────────────── -->
<!-- NV MANAGER UI                                                              -->
<!-- ─────────────────────────────────────────────────────────────────────────── -->

<div class="space-y-4 font-mono">

  <!-- Header bar -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
    <div class="flex items-center gap-2">
      <Database class="w-4 h-4 text-accent" />
      <span class="text-sm font-black uppercase tracking-wider text-foreground">NV Manager</span>
      <span class="text-[10px] text-muted font-normal">
        {filteredItems.length} / {allItems.length} items
      </span>
    </div>
    <div class="flex items-center gap-2">
      <Button variant="ghost" size="sm" onclick={() => { showSpcModal = true }}>
        <Key class="w-3.5 h-3.5 mr-1 {isSpcUnlocked ? 'text-emerald-400' : 'text-muted'}" />
        <span>{isSpcUnlocked ? 'SPC Unlocked' : 'Unlock SPC'}</span>
      </Button>
      <Button variant="ghost" size="sm" onclick={() => { backupModalOpen = true }}>
        <Download class="w-3.5 h-3.5 mr-1" />Backup
      </Button>
      <Button variant="ghost" size="sm" onclick={() => { restoreModalOpen = true }}>
        <Upload class="w-3.5 h-3.5 mr-1" />Restore
      </Button>
      <Button variant="secondary" size="sm" disabled={isLoading} onclick={loadCatalog}>
        <RefreshCw class="w-3.5 h-3.5 {isLoading ? 'animate-spin' : ''}" />
      </Button>
    </div>
  </div>

  <!-- Filter bar -->
  <div class="flex flex-col sm:flex-row gap-2">
    <div class="flex items-center gap-1 flex-wrap">
      <button
        class="px-2.5 py-1 text-[10px] font-bold uppercase rounded border-2 transition-all
               {selectedCategory === 'all'
                 ? 'border-accent bg-accent/15 text-accent'
                 : 'border-border text-muted hover:text-foreground hover:border-border/80'}"
        onclick={() => { selectedCategory = 'all' }}
      >All</button>
      {#each categories as cat}
        <button
          class="px-2.5 py-1 text-[10px] font-bold uppercase rounded border-2 transition-all
                 {selectedCategory === cat
                   ? 'border-accent bg-accent/15 text-accent'
                   : 'border-border text-muted hover:text-foreground'}"
          onclick={() => { selectedCategory = cat }}
        >{cat}</button>
      {/each}
    </div>
    <div class="relative flex-1 min-w-[180px]">
      <Search class="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted pointer-events-none" />
      <input
        type="text"
        placeholder="Search ID, name, description..."
        bind:value={searchQuery}
        class="w-full pl-8 pr-3 py-1.5 text-xs bg-card border-2 border-border rounded
               text-foreground placeholder:text-muted focus:outline-none focus:border-accent transition-colors"
      />
    </div>
  </div>

  <!-- Main split: item list | detail panel -->
  <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">

    <!-- ── LEFT: NV Item Table ─────────────────────────────────────────────── -->
    <div class="border-2 border-border rounded-lg overflow-hidden">
      <div class="grid grid-cols-[56px_1fr_80px_60px] text-[10px] font-black uppercase tracking-wider
                  text-muted bg-card border-b-2 border-border px-3 py-2">
        <span>ID</span><span>Name</span><span>Category</span><span class="text-right">Size</span>
      </div>
      <div class="overflow-y-auto max-h-[460px]">
        {#if isLoading}
          <div class="flex items-center justify-center py-16 text-muted text-xs gap-2">
            <RefreshCw class="w-4 h-4 animate-spin" />Loading catalog...
          </div>
        {:else if filteredItems.length === 0}
          <div class="flex flex-col items-center justify-center py-16 text-muted text-xs gap-2">
            <Database class="w-8 h-8 opacity-30" /><span>No items found</span>
          </div>
        {:else}
          {#each filteredItems as item (item.id)}
            <button
              type="button"
              class="w-full grid grid-cols-[56px_1fr_80px_60px] items-center px-3 py-2
                     text-left border-b border-border/40 transition-all cursor-pointer
                     {selectedItem?.id === item.id
                       ? 'bg-accent/10 border-l-2 border-l-accent'
                       : 'hover:bg-card/60'}"
              onclick={() => readItem(item)}
            >
              <span class="text-[11px] font-bold text-accent tabular-nums">{item.id}</span>
              <div class="min-w-0">
                <p class="text-xs font-bold text-foreground truncate leading-tight">{item.name}</p>
                <p class="text-[10px] text-muted truncate leading-tight">{item.description}</p>
              </div>
              <span class="text-[9px] font-bold px-1.5 py-0.5 rounded border {categoryColor(item.category)} truncate">
                {item.category}
              </span>
              <span class="text-[11px] text-muted text-right tabular-nums">{item.size}B</span>
            </button>
          {/each}
        {/if}
      </div>
    </div>

    <!-- ── RIGHT: Detail & Read Panel ─────────────────────────────────────── -->
    <div class="border-2 border-border rounded-lg overflow-hidden">
      {#if !selectedItem}
        <div class="flex flex-col items-center justify-center h-full min-h-[300px] text-muted text-xs gap-3">
          <Cpu class="w-10 h-10 opacity-20" />
          <p>Select an NV item to read from modem</p>
        </div>
      {:else}
        <!-- Item header -->
        <div class="p-3 bg-card border-b-2 border-border flex items-start justify-between gap-2">
          <div class="min-w-0">
            <div class="flex items-center gap-2 flex-wrap">
              <span class="text-base font-black text-accent">#{selectedItem.id}</span>
              <span class="text-sm font-bold text-foreground">{selectedItem.name}</span>
              {#if selectedItem.read_only}
                <span class="text-[9px] font-bold px-1.5 py-0.5 rounded border border-amber-500/40 bg-amber-500/10 text-amber-400">READ-ONLY</span>
              {/if}
            </div>
            <p class="text-[11px] text-muted mt-0.5">{selectedItem.description}</p>
            <div class="flex items-center gap-1.5 mt-1.5 flex-wrap">
              <span class="text-[9px] font-bold px-1.5 py-0.5 rounded border {categoryColor(selectedItem.category)}">
                {selectedItem.category}
              </span>
              <span class="text-[9px] font-bold px-1.5 py-0.5 rounded border {dataTypeBadge(selectedItem.data_type)}">
                {selectedItem.data_type.toUpperCase()}
              </span>
              <span class="text-[9px] text-muted border border-border rounded px-1.5 py-0.5">
                {selectedItem.size}B
              </span>
            </div>
          </div>
          <div class="flex items-center gap-1.5 shrink-0">
            <Button size="sm" variant="secondary" disabled={isReading} onclick={() => readItem(selectedItem!)}>
              <RefreshCw class="w-3 h-3 {isReading ? 'animate-spin' : ''}" />
            </Button>
            {#if !selectedItem.read_only}
              <Button size="sm" variant="primary" onclick={openWriteModal}>
                <Edit3 class="w-3 h-3 mr-1" />Write
              </Button>
            {/if}
          </div>
        </div>

        <!-- Read result body -->
        <div class="p-3 space-y-3">
          {#if isReading}
            <div class="flex items-center gap-2 text-muted text-xs py-8 justify-center">
              <RefreshCw class="w-4 h-4 animate-spin" />Reading from modem...
            </div>
          {:else if readResult}
            {#if readResult.success}
              <!-- ✅ Success -->
              <div class="space-y-1">
                <p class="text-[10px] text-muted uppercase font-bold">Formatted Value</p>
                <div class="flex items-center gap-2">
                  <p class="text-base font-black text-foreground break-all">{readResult.formatted}</p>
                  <button class="text-muted hover:text-accent transition-colors shrink-0"
                    onclick={() => copyToClipboard(readResult!.formatted)}>
                    <Copy class="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>
              <div class="space-y-1">
                <p class="text-[10px] text-muted uppercase font-bold">Raw Hex</p>
                <div class="flex items-start gap-2">
                  <code class="text-[11px] font-mono text-accent break-all leading-relaxed flex-1
                                bg-card border border-border rounded p-2">
                    {fmtHex(readResult.data_hex)}
                  </code>
                  <button class="text-muted hover:text-accent transition-colors mt-2 shrink-0"
                    onclick={() => copyToClipboard(readResult!.data_hex)}>
                    <Copy class="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>
              <div class="grid grid-cols-2 gap-2 text-[10px]">
                <div class="p-2 bg-card border border-border rounded">
                  <p class="text-muted uppercase font-bold mb-0.5">Data Size</p>
                  <p class="font-bold text-foreground">{readResult.size} bytes</p>
                </div>
                <div class="p-2 bg-card border border-border rounded">
                  <p class="text-muted uppercase font-bold mb-0.5">Data Type</p>
                  <p class="font-bold text-foreground">{readResult.data_type.toUpperCase()}</p>
                </div>
              </div>

            {:else if isNotAvailable}
              <!-- ℹ️ Not provisioned on this device — not an error -->
              <div class="flex flex-col items-center justify-center py-8 gap-2 text-muted">
                <MinusCircle class="w-8 h-8 opacity-30" />
                <p class="text-xs font-bold">Not Available on This Device</p>
                <p class="text-[10px] text-center max-w-[220px] opacity-70">
                  NV item #{readResult.id} is not provisioned on this modem. This is normal — not all devices have every NV item.
                </p>
              </div>

            {:else}
              <!-- ❌ Real error -->
              <div class="flex items-start gap-2 p-3 bg-red-500/10 border border-red-500/30 rounded text-xs text-red-400">
                <AlertTriangle class="w-4 h-4 shrink-0 mt-0.5" />
                <div>
                  <p class="font-bold">Read failed</p>
                  <p class="opacity-80 break-all">{readResult.error}</p>
                </div>
              </div>
            {/if}

          {:else}
            <div class="flex items-center justify-center py-8 text-muted text-xs">
              Click <RefreshCw class="w-3 h-3 mx-1" /> to read this item from the modem
            </div>
          {/if}
        </div>
      {/if}
    </div>
  </div>
</div>

<!-- ─────────────────────────────────────────────────────────────────────────── -->
<!-- WRITE MODAL — bind:open pattern                                             -->
<!-- ─────────────────────────────────────────────────────────────────────────── -->

<Modal title="Write NV {selectedItem?.id} — {selectedItem?.name}" bind:open={writeModalOpen}
  onclose={() => { writeError = '' }}>
  <div class="space-y-4 text-xs font-mono">
    <div class="p-3 bg-amber-500/10 border border-amber-500/30 rounded flex items-start gap-2 text-amber-400">
      <AlertTriangle class="w-4 h-4 shrink-0 mt-0.5" />
      <p>Writing incorrect values can destabilize the modem. Double-check before writing.</p>
    </div>

    <div class="space-y-1">
      <label class="text-[10px] text-muted uppercase font-bold block">
        Hex Data ({selectedItem?.size ?? 0} bytes expected)
      </label>
      <input
        type="text"
        bind:value={writeHex}
        placeholder="e.g. 080123456789ABFF00"
        class="w-full px-3 py-2 text-xs font-mono bg-card border-2 border-border rounded
               text-foreground placeholder:text-muted focus:outline-none focus:border-accent transition-colors"
      />
      <p class="text-[10px] text-muted">
        {Math.floor(writeHex.replace(/\s/g, '').length / 2)} / {selectedItem?.size ?? 0} bytes entered
      </p>
    </div>

    {#if writeError}
      <p class="text-red-400 text-[11px] flex items-center gap-1">
        <X class="w-3.5 h-3.5 shrink-0" />{writeError}
      </p>
    {/if}

    <div class="flex gap-2 pt-1">
      <Button variant="secondary" size="sm" class="flex-1" onclick={() => { writeModalOpen = false }}>
        Cancel
      </Button>
      <Button
        variant="primary"
        size="sm"
        class="flex-1"
        disabled={isWriting || writeHex.replace(/\s/g, '').length < 2}
        onclick={writeItem}
      >
        {#if isWriting}
          <RefreshCw class="w-3.5 h-3.5 animate-spin mr-1" />
        {:else}
          <Save class="w-3.5 h-3.5 mr-1" />
        {/if}
        Write to Modem
      </Button>
    </div>
  </div>
</Modal>

<!-- ─────────────────────────────────────────────────────────────────────────── -->
<!-- BACKUP MODAL                                                                -->
<!-- ─────────────────────────────────────────────────────────────────────────── -->

<Modal title="NV Backup" bind:open={backupModalOpen} onclose={() => { backupStats = null }}>
  <div class="space-y-4 text-xs font-mono">
    <div class="flex gap-1">
      {#each [{ key: 'catalog', label: 'Catalog Items' }, { key: 'range', label: 'ID Range' }] as mode}
        <button
          class="flex-1 py-1.5 text-[11px] font-bold rounded border-2 transition-all
                 {backupMode === mode.key
                   ? 'border-accent bg-accent/15 text-accent'
                   : 'border-border text-muted hover:text-foreground'}"
          onclick={() => { backupMode = mode.key as 'catalog' | 'range' }}
        >{mode.label}</button>
      {/each}
    </div>

    {#if backupMode === 'range'}
      <div class="grid grid-cols-2 gap-2">
        <div class="space-y-1">
          <label class="text-[10px] text-muted uppercase font-bold block">Start ID</label>
          <input type="number" bind:value={backupRangeStart} min="0" max="65535"
            class="w-full px-3 py-1.5 text-xs bg-card border-2 border-border rounded
                   text-foreground focus:outline-none focus:border-accent transition-colors" />
        </div>
        <div class="space-y-1">
          <label class="text-[10px] text-muted uppercase font-bold block">End ID</label>
          <input type="number" bind:value={backupRangeEnd} min="0" max="65535"
            class="w-full px-3 py-1.5 text-xs bg-card border-2 border-border rounded
                   text-foreground focus:outline-none focus:border-accent transition-colors" />
        </div>
      </div>
      <p class="text-[10px] text-muted">
        {backupRangeEnd - backupRangeStart + 1} items · ~{Math.ceil((backupRangeEnd - backupRangeStart + 1) * 0.25)}s est.
      </p>
    {:else}
      <p class="text-[10px] text-muted">Will backup all {allItems.length} known catalog items.</p>
    {/if}

    <div class="space-y-1">
      <label class="text-[10px] text-muted uppercase font-bold block">Output File Path</label>
      <input type="text" bind:value={backupFilename} placeholder="/sdcard/nv_backup.json"
        class="w-full px-3 py-1.5 text-xs font-mono bg-card border-2 border-border rounded
               text-foreground placeholder:text-muted focus:outline-none focus:border-accent transition-colors" />
    </div>

    {#if backupStats}
      <div class="p-3 bg-card border-2 border-border rounded space-y-2">
        <p class="text-[10px] uppercase font-bold text-muted">Result</p>
        <div class="grid grid-cols-3 gap-2 text-center">
          <div>
            <p class="text-lg font-black text-emerald-400">{backupStats.success}</p>
            <p class="text-[9px] text-muted uppercase">OK</p>
          </div>
          <div>
            <p class="text-lg font-black text-red-400">{backupStats.failed}</p>
            <p class="text-[9px] text-muted uppercase">Failed</p>
          </div>
          <div>
            <p class="text-lg font-black text-amber-400">{backupStats.skipped}</p>
            <p class="text-[9px] text-muted uppercase">Skipped</p>
          </div>
        </div>
        {#if !backupStats.error}
          <p class="text-emerald-400 text-[10px] flex items-center gap-1">
            <Check class="w-3 h-3" /> Saved to {backupFilename}
          </p>
        {:else}
          <p class="text-red-400 text-[10px]">{backupStats.error}</p>
        {/if}
      </div>
    {/if}

    <div class="flex gap-2 pt-1">
      <Button variant="secondary" size="sm" class="flex-1" onclick={() => { backupModalOpen = false; backupStats = null }}>Close</Button>
      <Button variant="accent" size="sm" class="flex-1" disabled={isBackingUp} onclick={runBackup}>
        {#if isBackingUp}
          <RefreshCw class="w-3.5 h-3.5 animate-spin mr-1" />Backing up...
        {:else}
          <Download class="w-3.5 h-3.5 mr-1" />Start Backup
        {/if}
      </Button>
    </div>
  </div>
</Modal>

<!-- ─────────────────────────────────────────────────────────────────────────── -->
<!-- RESTORE MODAL                                                               -->
<!-- ─────────────────────────────────────────────────────────────────────────── -->

<Modal title="NV Restore" bind:open={restoreModalOpen} onclose={() => { restoreStats = null }}>
  <div class="space-y-4 text-xs font-mono">
    <div class="p-3 bg-red-500/10 border border-red-500/30 rounded flex items-start gap-2 text-red-400">
      <AlertTriangle class="w-4 h-4 shrink-0 mt-0.5" />
      <p>Restoring will overwrite current modem NV values. Ensure the backup matches this device.</p>
    </div>

    <div class="space-y-1">
      <label class="text-[10px] text-muted uppercase font-bold block">Backup File Path</label>
      <input type="text" bind:value={restoreFilename} placeholder="/sdcard/nv_backup.json"
        class="w-full px-3 py-1.5 text-xs font-mono bg-card border-2 border-border rounded
               text-foreground placeholder:text-muted focus:outline-none focus:border-accent transition-colors" />
    </div>

    {#if restoreStats}
      <div class="p-3 bg-card border-2 border-border rounded space-y-2">
        <p class="text-[10px] uppercase font-bold text-muted">Result</p>
        <div class="grid grid-cols-2 gap-2 text-center">
          <div>
            <p class="text-lg font-black text-emerald-400">{restoreStats.success}</p>
            <p class="text-[9px] text-muted uppercase">Restored</p>
          </div>
          <div>
            <p class="text-lg font-black text-red-400">{restoreStats.failed}</p>
            <p class="text-[9px] text-muted uppercase">Failed</p>
          </div>
        </div>
        {#if !restoreStats.error}
          <p class="text-emerald-400 text-[10px] flex items-center gap-1">
            <Check class="w-3 h-3" /> Restore complete
          </p>
        {:else}
          <p class="text-red-400 text-[10px]">{restoreStats.error}</p>
        {/if}
      </div>
    {/if}

    <div class="flex gap-2 pt-1">
      <Button variant="secondary" size="sm" class="flex-1" onclick={() => { restoreModalOpen = false; restoreStats = null }}>Close</Button>
      <Button variant="danger" size="sm" class="flex-1" disabled={isRestoring} onclick={runRestore}>
        {#if isRestoring}
          <RefreshCw class="w-3.5 h-3.5 animate-spin mr-1" />Restoring...
        {:else}
          <Upload class="w-3.5 h-3.5 mr-1" />Restore Now
        {/if}
      </Button>
    </div>
  </div>
</Modal>

<!-- ─────────────────────────────────────────────────────────────────────────── -->
<!-- SPC UNLOCK MODAL                                                             -->
<!-- ─────────────────────────────────────────────────────────────────────────── -->

<Modal title="Unlock DIAG Security Session (SPC)" bind:open={showSpcModal} size="sm">
  <div class="space-y-3 font-mono text-xs">
    <p class="text-muted text-[11px]">
      Qualcomm NV writes require elevating the DIAG security level via the 6-digit Service Programming Code (SPC). Default is usually <code>000000</code>.
    </p>

    <div>
      <label class="block text-[10px] text-muted uppercase font-bold mb-1">
        <span>Service Programming Code (6 Digits)</span>
        <input
          type="text"
          maxlength="6"
          bind:value={spcInput}
          class="w-full bg-card border-2 border-border rounded px-3 py-2 font-mono text-sm tracking-widest text-center text-foreground focus:outline-none focus:border-accent transition-colors"
          placeholder="000000"
        />
      </label>
    </div>

    <div class="flex gap-2 pt-2">
      <Button variant="secondary" size="sm" class="flex-1" onclick={() => (showSpcModal = false)}>Cancel</Button>
      <Button variant="primary" size="sm" class="flex-1" disabled={isUnlockingSpc} onclick={submitSpcUnlock}>
        <span>{isUnlockingSpc ? 'Verifying...' : 'Unlock SPC'}</span>
      </Button>
    </div>
  </div>
</Modal>
