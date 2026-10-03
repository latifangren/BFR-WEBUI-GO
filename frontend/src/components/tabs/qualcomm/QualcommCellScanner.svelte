<script lang="ts">
  import {
    Search,
    Lock,
    Bookmark,
    BookmarkCheck,
    Trash2,
    Radio,
    Signal,
    Layers,
    Navigation,
    Clock,
    RefreshCw,
    ExternalLink,
    ChevronDown,
    ChevronUp,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'

  export interface NearbyCell {
    type: string
    role: string // "serving", "scell", "neighbor"
    cell_id: number
    pci: number
    tac: number
    earfcn: number
    band: number
    band_name: string
    rsrp: number
    rsrq: number
    rssi: number
    sinr: number
    timing_advance: number
    distance_meters: number
    distance_str: string
    is_current_lock?: boolean
  }

  export interface CellScanResponse {
    success: boolean
    serving_cell?: NearbyCell
    carrier_aggr: NearbyCell[]
    neighbors: NearbyCell[]
    total_cells: number
    scan_time: string
    error_message?: string
  }

  interface SavedTower {
    id: string
    label: string
    earfcn: number
    pci: number
    band_name: string
    cell_id?: number
    last_rsrp?: number
    saved_at: string
  }

  let {
    simSlot = 0,
    onSelectTower,
    onLockTower,
  }: {
    simSlot: number
    onSelectTower: (earfcn: number, pci: number) => void
    onLockTower: (earfcn: number, pci: number) => void
  } = $props()

  let isScanning = $state(false)
  let scanResult = $state<CellScanResponse | null>(null)
  let savedTowers = $state<SavedTower[]>([])
  let newTowerLabel = $state('')
  let selectedForBookmark = $state<NearbyCell | null>(null)
  let showSavedTowers = $state(true)

  const storageKey = $derived(`bfr_saved_towers_slot_${simSlot}`)

  // Load saved bookmarks from localStorage
  $effect(() => {
    try {
      const data = localStorage.getItem(storageKey)
      if (data) {
        savedTowers = JSON.parse(data)
      } else {
        savedTowers = []
      }
    } catch {
      savedTowers = []
    }
  })

  function saveBookmarksToStorage(towers: SavedTower[]) {
    savedTowers = towers
    try {
      localStorage.setItem(storageKey, JSON.stringify(towers))
    } catch {
      // Ignore
    }
  }

  export async function performScan() {
    try {
      isScanning = true
      const res = await api.get<CellScanResponse>(`/api/modem/cell_scan?slot=${simSlot}`)
      if (res && res.success) {
        scanResult = res
        toastStore.success(`Cell scan complete: ${res.total_cells} tower(s) discovered.`)
      } else {
        toastStore.error(res?.error_message || 'Cell scan returned no cells from baseband')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Cell tower scan failed')
    } finally {
      isScanning = false
    }
  }

  function bookmarkTower(cell: NearbyCell) {
    selectedForBookmark = cell
    newTowerLabel = `${cell.band_name} - PCI ${cell.pci}`
  }

  function confirmBookmark() {
    if (!selectedForBookmark) return
    const newEntry: SavedTower = {
      id: `${selectedForBookmark.earfcn}_${selectedForBookmark.pci}_${Date.now()}`,
      label: newTowerLabel.trim() || `Tower ${selectedForBookmark.band_name} (PCI ${selectedForBookmark.pci})`,
      earfcn: selectedForBookmark.earfcn,
      pci: selectedForBookmark.pci,
      band_name: selectedForBookmark.band_name,
      cell_id: selectedForBookmark.cell_id > 0 ? selectedForBookmark.cell_id : undefined,
      last_rsrp: selectedForBookmark.rsrp,
      saved_at: new Date().toLocaleDateString('id-ID', { hour: '2-digit', minute: '2-digit' }),
    }

    const updated = [newEntry, ...savedTowers.filter((t) => !(t.earfcn === newEntry.earfcn && t.pci === newEntry.pci))]
    saveBookmarksToStorage(updated)
    selectedForBookmark = null
    toastStore.success(`Tower bookmarked: ${newEntry.label}`)
  }

  function deleteBookmark(id: string) {
    const updated = savedTowers.filter((t) => t.id !== id)
    saveBookmarksToStorage(updated)
    toastStore.info('Tower bookmark removed.')
  }
</script>

<div class="space-y-3 sm:space-y-6 font-mono">
  <!-- SCANNER CARD -->
  <Card
    title="Nearby Cell Towers & Signal Scanner"
    subtitle="Scan surrounding radio mast antennas, secondary CA cells, and neighbor base stations via QMI QRTR"
    tone="mint"
  >
    {#snippet action()}
      <div class="flex items-center gap-2">
        <Button
          variant="primary"
          size="sm"
          disabled={isScanning}
          onclick={performScan}
        >
          <Search class="w-3.5 h-3.5 mr-1.5 {isScanning ? 'animate-spin' : ''}" />
          <span>{isScanning ? 'Scanning RF...' : 'Scan Nearby Towers'}</span>
        </Button>
      </div>
    {/snippet}

    {#if !scanResult && !isScanning}
      <div class="p-6 text-center border-2 border-dashed border-border rounded-lg space-y-3">
        <Radio class="w-8 h-8 text-accent mx-auto opacity-70 animate-pulse" />
        <p class="text-xs text-foreground font-bold">No cell scan performed yet</p>
        <p class="text-[11px] text-muted max-w-md mx-auto">
          Tap <strong>Scan Nearby Towers</strong> to query the baseband modem for all surrounding cell towers with exact EARFCN, PCI, RSRP, SINR, and estimated distance.
        </p>
      </div>
    {:else if isScanning}
      <div class="p-8 text-center border-2 border-border rounded-lg space-y-3 bg-card-sub">
        <RefreshCw class="w-8 h-8 text-accent mx-auto animate-spin" />
        <p class="text-xs text-foreground font-bold">Scanning baseband RF environment...</p>
        <p class="text-[11px] text-muted">Reading intra-frequency and inter-frequency cellular antennas</p>
      </div>
    {:else if scanResult}
      <div class="space-y-4 text-xs">
        <div class="flex items-center justify-between text-[11px] text-muted border-b border-border pb-2">
          <span>Found <strong>{scanResult.total_cells}</strong> tower(s) • Scanned at {scanResult.scan_time}</span>
          <span class="text-emerald-400 font-bold">Live QMI Baseband Telemetry</span>
        </div>

        <!-- 1. SERVING CELL (CONNECTED) -->
        {#if scanResult.serving_cell}
          {@const sc = scanResult.serving_cell}
          <div class="p-3.5 rounded-lg border-2 border-emerald-500 bg-emerald-950/20 shadow-neobrutal-sm space-y-2">
            <div class="flex items-center justify-between flex-wrap gap-2">
              <div class="flex items-center gap-2">
                <Badge variant="success" class="text-[10px] font-bold">CONNECTED PRIMARY CELL</Badge>
                <span class="font-bold text-foreground text-sm">{sc.band_name}</span>
              </div>
              <div class="flex items-center gap-2">
                <Button variant="outline" size="sm" onclick={() => onSelectTower(sc.earfcn, sc.pci)}>
                  <span>Fill Form</span>
                </Button>
                <Button variant="secondary" size="sm" onclick={() => onLockTower(sc.earfcn, sc.pci)}>
                  <Lock class="w-3 h-3 mr-1" />
                  <span>Lock to This</span>
                </Button>
                <Button variant="ghost" size="sm" onclick={() => bookmarkTower(sc)}>
                  <Bookmark class="w-3.5 h-3.5 text-accent" />
                </Button>
              </div>
            </div>

            <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 pt-1 text-[11px]">
              <div class="p-2 bg-card rounded border border-border/80">
                <span class="text-muted block text-[10px]">EARFCN (Channel)</span>
                <strong class="text-accent">{sc.earfcn}</strong>
              </div>
              <div class="p-2 bg-card rounded border border-border/80">
                <span class="text-muted block text-[10px]">Physical Cell ID (PCI)</span>
                <strong class="text-foreground">{sc.pci}</strong>
              </div>
              <div class="p-2 bg-card rounded border border-border/80">
                <span class="text-muted block text-[10px]">Global Cell ID (eNodeB)</span>
                <strong class="text-foreground">{sc.cell_id > 0 ? sc.cell_id : 'Protected'}</strong>
              </div>
              <div class="p-2 bg-card rounded border border-border/80">
                <span class="text-muted block text-[10px]">Tracking Area (TAC)</span>
                <strong class="text-foreground">{sc.tac > 0 ? sc.tac : 'Protected'}</strong>
              </div>
            </div>

            <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 text-[11px]">
              <div class="p-2 bg-card rounded border border-border/80">
                <span class="text-muted block text-[10px]">RSRP (Power)</span>
                <strong class="{sc.rsrp >= -95 ? 'text-emerald-400' : sc.rsrp >= -110 ? 'text-amber-400' : 'text-red-400'}">
                  {sc.rsrp !== 0 && sc.rsrp !== -999 ? `${sc.rsrp.toFixed(1)} dBm` : 'N/A'}
                </strong>
              </div>
              <div class="p-2 bg-card rounded border border-border/80">
                <span class="text-muted block text-[10px]">RSRQ (Quality)</span>
                <strong class="text-foreground">{sc.rsrq !== 0 && sc.rsrq !== -999 ? `${sc.rsrq.toFixed(1)} dB` : 'N/A'}</strong>
              </div>
              <div class="p-2 bg-card rounded border border-border/80">
                <span class="text-muted block text-[10px]">SINR (SNR)</span>
                <strong class="{sc.sinr >= 10 ? 'text-emerald-400' : 'text-foreground'}">
                  {sc.sinr !== 0 && sc.sinr !== -999 ? `${sc.sinr.toFixed(1)} dB` : 'N/A'}
                </strong>
              </div>
              <div class="p-2 bg-card rounded border border-border/80">
                <span class="text-muted block text-[10px]">Estimated Distance</span>
                <strong class="text-cyan-300">{sc.distance_str}</strong>
              </div>
            </div>
          </div>
        {/if}

        <!-- 2. CARRIER AGGREGATION SECONDARY CELLS (SCell) -->
        {#if scanResult.carrier_aggr && scanResult.carrier_aggr.length > 0}
          <div class="space-y-2">
            <span class="text-[11px] font-bold text-cyan-400 flex items-center gap-1.5 uppercase">
              <Layers class="w-3.5 h-3.5" />
              <span>Active Carrier Aggregation Secondary Cells ({scanResult.carrier_aggr.length})</span>
            </span>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              {#each scanResult.carrier_aggr as scell}
                <div class="p-3 bg-cyan-950/20 border-2 border-cyan-800 rounded-lg flex flex-col justify-between gap-2">
                  <div class="flex items-center justify-between">
                    <Badge variant="info" class="text-[9px]">CA SCell</Badge>
                    <span class="font-bold text-foreground">{scell.band_name}</span>
                  </div>
                  <div class="grid grid-cols-3 gap-1 text-[10px]">
                    <div>EARFCN: <strong class="text-accent">{scell.earfcn}</strong></div>
                    <div>PCI: <strong class="text-foreground">{scell.pci}</strong></div>
                    <div>RSRP: <strong class="text-foreground">{scell.rsrp.toFixed(0)} dBm</strong></div>
                  </div>
                  <div class="flex items-center justify-end gap-1.5 pt-1">
                    <Button variant="outline" size="sm" onclick={() => onSelectTower(scell.earfcn, scell.pci)}>
                      <span>Select</span>
                    </Button>
                    <Button variant="secondary" size="sm" onclick={() => onLockTower(scell.earfcn, scell.pci)}>
                      <Lock class="w-3 h-3 mr-1" />
                      <span>Lock</span>
                    </Button>
                  </div>
                </div>
              {/each}
            </div>
          </div>
        {/if}

        <!-- 3. NEIGHBOR CELL TOWERS -->
        {#if scanResult.neighbors && scanResult.neighbors.length > 0}
          <div class="space-y-2">
            <span class="text-[11px] font-bold text-foreground flex items-center gap-1.5 uppercase">
              <Radio class="w-3.5 h-3.5 text-accent" />
              <span>Detected Neighbor Towers ({scanResult.neighbors.length})</span>
            </span>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5 max-h-80 overflow-y-auto pr-1">
              {#each scanResult.neighbors as n}
                <div class="p-2.5 bg-card-sub border-2 border-border rounded flex items-center justify-between gap-2 hover:border-accent/60 transition-all">
                  <div class="space-y-0.5">
                    <div class="flex items-center gap-2">
                      <span class="font-bold text-foreground text-xs">{n.band_name}</span>
                      <span class="text-[10px] text-muted">PCI: <strong class="text-foreground">{n.pci}</strong></span>
                    </div>
                    <div class="text-[10px] text-muted flex items-center gap-2">
                      <span>EARFCN: <strong class="text-accent">{n.earfcn}</strong></span>
                      <span>RSRP: <strong class="{n.rsrp >= -100 ? 'text-emerald-400' : 'text-amber-400'}">{n.rsrp !== 0 && n.rsrp !== -999 ? `${n.rsrp.toFixed(0)} dBm` : 'N/A'}</strong></span>
                    </div>
                  </div>
                  <div class="flex items-center gap-1.5 shrink-0">
                    <Button variant="outline" size="sm" onclick={() => onSelectTower(n.earfcn, n.pci)}>
                      <span>Select</span>
                    </Button>
                    <Button variant="secondary" size="sm" onclick={() => onLockTower(n.earfcn, n.pci)}>
                      <Lock class="w-3 h-3 mr-1" />
                      <span>Lock</span>
                    </Button>
                    <Button variant="ghost" size="sm" onclick={() => bookmarkTower(n)}>
                      <Bookmark class="w-3 h-3 text-muted hover:text-accent" />
                    </Button>
                  </div>
                </div>
              {/each}
            </div>
          </div>
        {/if}
      </div>
    {/if}
  </Card>

  <!-- SAVED / BOOKMARKED TOWERS -->
  <Card
    title="Saved & Favorite Cell Towers ({savedTowers.length})"
    subtitle="Personal list of bookmarked BTS towers for 1-click re-locking"
    tone="peach"
  >
    {#snippet action()}
      <Button
        variant="ghost"
        size="sm"
        onclick={() => (showSavedTowers = !showSavedTowers)}
      >
        {#if showSavedTowers}
          <ChevronUp class="w-3.5 h-3.5 mr-1" />
          <span>Hide</span>
        {:else}
          <ChevronDown class="w-3.5 h-3.5 mr-1" />
          <span>Show</span>
        {/if}
      </Button>
    {/snippet}

    {#if showSavedTowers}
      {#if savedTowers.length === 0}
        <div class="p-4 text-center border-2 border-dashed border-border rounded text-[11px] text-muted">
          No saved cell towers yet. Click the bookmark icon next to any scanned tower to save it here.
        </div>
      {:else}
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-2 text-xs">
          {#each savedTowers as t}
            <div class="p-3 bg-card-sub border-2 border-border rounded flex items-center justify-between gap-2 hover:border-amber-400/50 transition-all">
              <div class="space-y-0.5">
                <span class="font-bold text-foreground text-xs block">{t.label}</span>
                <span class="text-[10px] text-muted block">
                  {t.band_name} • EARFCN <strong class="text-accent">{t.earfcn}</strong> • PCI <strong class="text-foreground">{t.pci}</strong>
                </span>
                <span class="text-[9px] text-muted">Saved: {t.saved_at}</span>
              </div>
              <div class="flex items-center gap-1.5 shrink-0">
                <Button variant="outline" size="sm" onclick={() => onSelectTower(t.earfcn, t.pci)}>
                  <span>Fill</span>
                </Button>
                <Button variant="secondary" size="sm" onclick={() => onLockTower(t.earfcn, t.pci)}>
                  <Lock class="w-3 h-3 mr-1" />
                  <span>Lock</span>
                </Button>
                <button
                  type="button"
                  class="p-1.5 text-muted hover:text-red-400 transition-colors cursor-pointer"
                  onclick={() => deleteBookmark(t.id)}
                  title="Remove bookmark"
                >
                  <Trash2 class="w-3.5 h-3.5" />
                </button>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    {/if}
  </Card>

  <!-- BOOKMARK MODAL / DIALOG -->
  {#if selectedForBookmark}
    <div class="fixed inset-0 z-50 bg-black/70 backdrop-blur-xs flex items-center justify-center p-4">
      <div class="bg-card border-2 border-accent rounded-lg max-w-md w-full p-4 space-y-4 shadow-neobrutal font-mono text-xs">
        <div class="flex items-center gap-2 text-foreground font-bold">
          <Bookmark class="w-4 h-4 text-accent" />
          <span>Bookmark Cell Tower</span>
        </div>
        <p class="text-[11px] text-muted">
          Saving tower <strong>{selectedForBookmark.band_name}</strong> (EARFCN: {selectedForBookmark.earfcn}, PCI: {selectedForBookmark.pci})
        </p>
        <div class="space-y-1.5">
          <label for="tower-label-input" class="text-[10px] text-muted font-bold uppercase">Tower Nickname / Location</label>
          <input
            id="tower-label-input"
            type="text"
            bind:value={newTowerLabel}
            placeholder="e.g. Home Tower - B3 (Kuat)"
            class="neo-input w-full bg-card-sub border-2 border-border rounded px-3 py-2 text-xs font-mono text-foreground focus:outline-none focus:border-accent"
          />
        </div>
        <div class="flex items-center justify-end gap-2 pt-2">
          <Button variant="outline" size="sm" onclick={() => (selectedForBookmark = null)}>
            <span>Cancel</span>
          </Button>
          <Button variant="primary" size="sm" onclick={confirmBookmark}>
            <BookmarkCheck class="w-3.5 h-3.5 mr-1" />
            <span>Save Bookmark</span>
          </Button>
        </div>
      </div>
    </div>
  {/if}
</div>
