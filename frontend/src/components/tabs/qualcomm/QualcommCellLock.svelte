<script lang="ts">
  import {
    Search,
    Lock,
    Unlock,
    RotateCcw,
    AlertTriangle,
    ShieldCheck,
    CheckCircle,
    Info,
    RefreshCw,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import QualcommCellScanner from './QualcommCellScanner.svelte'

  interface SimSlotInfo {
    slot_index: number
    sub_id: number
    state: string
    is_active: boolean
    is_esim: boolean
    carrier: string
    label: string
  }

  interface SimStatus {
    active_slot: number
    active_sub_id: number
    slots: SimSlotInfo[]
  }

  interface CellLockState {
    supported: boolean
    active_lock: boolean
    earfcn: number
    pci: number
    nr_arfcn?: number
    nr_pci?: number
    message?: string
  }

  let {
    simSlot = 0,
    simStatus = null,
  }: {
    simSlot: number
    simStatus: SimStatus | null
  } = $props()

  let lteEarfcn = $state<string>('')
  let ltePci = $state<string>('')
  let isLoading = $state(false)
  let isLocking = $state(false)
  let isClearing = $state(false)

  let cellState = $state<CellLockState>({
    supported: true,
    active_lock: false,
    earfcn: 0,
    pci: 0,
  })

  const targetSimName = $derived(
    simStatus?.slots.find((s) => s.slot_index === simSlot)?.label || `SIM ${simSlot + 1}`
  )

  async function fetchCellLockState() {
    try {
      isLoading = true
      const res = await api.get<CellLockState>(`/api/modem/cell_lock?slot=${simSlot}`)
      if (res) {
        cellState = res
        if (res.active_lock) {
          lteEarfcn = res.earfcn > 0 ? String(res.earfcn) : ''
          ltePci = res.pci >= 0 ? String(res.pci) : ''
        }
      }
    } catch {
      // Retain fallback defaults
    } finally {
      isLoading = false
    }
  }

  async function applyCellLock() {
    const earfcnNum = parseInt(lteEarfcn.trim(), 10)
    const pciNum = parseInt(ltePci.trim(), 10)

    if (isNaN(earfcnNum) || earfcnNum <= 0) {
      toastStore.error('Please enter a valid EARFCN frequency channel')
      return
    }
    if (isNaN(pciNum) || pciNum < 0 || pciNum > 503) {
      toastStore.error('Physical Cell ID (PCI) must be between 0 and 503')
      return
    }

    try {
      isLocking = true
      const res = await api.post<{ success: boolean; message?: string }>('/api/modem/cell_lock', {
        slot: simSlot,
        type: 'lte',
        earfcn: earfcnNum,
        pci: pciNum,
      })

      if (res && res.success) {
        toastStore.success(res.message || `LTE Cell Lock applied: EARFCN ${earfcnNum}, PCI ${pciNum}`)
        await fetchCellLockState()
      } else {
        toastStore.error(res?.message || 'Modem rejected cell lock command')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Cell lock request failed')
    } finally {
      isLocking = false
    }
  }

  async function clearCellLock() {
    try {
      isClearing = true
      const res = await api.post<{ success: boolean; message?: string }>('/api/modem/cell_lock/clear', {
        slot: simSlot,
      })
      if (res && res.success) {
        toastStore.success('LTE Cell Lock cleared. Modem will connect to optimal tower automatically.')
        lteEarfcn = ''
        ltePci = ''
        await fetchCellLockState()
      } else {
        toastStore.error(res?.message || 'Failed to clear cell lock')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Clear cell lock failed')
    } finally {
      isClearing = false
    }
  }

  function handleSelectTower(earfcn: number, pci: number) {
    lteEarfcn = String(earfcn)
    ltePci = String(pci)
    toastStore.info(`Loaded tower into form: EARFCN ${earfcn}, PCI ${pci}`)
    document.getElementById('cell-earfcn-input')?.focus()
  }

  async function handleLockTower(earfcn: number, pci: number) {
    lteEarfcn = String(earfcn)
    ltePci = String(pci)
    await applyCellLock()
  }
</script>

<div class="space-y-6 font-mono">
  <!-- CELL LOCK STATUS HERO -->
  <Card title="Instant Cell & Tower Locking" subtitle="Pin baseband modem to a specific Physical Cell ID (PCI) and radio frequency channel (EARFCN)" tone="mint">
    {#snippet action()}
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoading || isLocking || isClearing}
        onclick={fetchCellLockState}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoading ? 'animate-spin' : ''}" />
        <span>Refresh</span>
      </Button>
    {/snippet}

    <div class="space-y-6 text-xs">
      <!-- Target Slot Indicator -->
      <div class="p-3 bg-card-sub border-2 border-border rounded flex flex-col sm:flex-row sm:items-center justify-between gap-2">
        <div class="flex items-center gap-2">
          <span class="text-muted">Target Interface:</span>
          <strong class="text-foreground">{targetSimName} (Slot {simSlot})</strong>
        </div>
        {#if cellState.active_lock}
          <Badge variant="warning" class="flex items-center gap-1.5 w-fit">
            <span class="w-2 h-2 rounded-full bg-amber-400 animate-pulse"></span>
            <span>CELL LOCK ACTIVE (EARFCN {cellState.earfcn} / PCI {cellState.pci})</span>
          </Badge>
        {:else}
          <Badge variant="success" class="flex items-center gap-1.5 w-fit">
            <span class="w-2 h-2 rounded-full bg-emerald-400"></span>
            <span>AUTO TOWER SELECTION (UNLOCKED)</span>
          </Badge>
        {/if}
      </div>

      <!-- Information Notice -->
      <div class="p-3 rounded bg-blue-950/20 border-2 border-blue-800 text-blue-300 text-xs flex items-start gap-2.5">
        <Info class="w-4 h-4 shrink-0 text-blue-400 mt-0.5" />
        <div class="space-y-1">
          <p class="font-bold text-foreground">How Cell Locking Works:</p>
          <p class="text-[11px] text-muted leading-relaxed">
            Cell locking forces the Qualcomm baseband modem to connect exclusively to the designated eNodeB transmitter cell tower. This eliminates ping spikes and prevents tower-hopping on borderline cellular signals.
          </p>
        </div>
      </div>

      <!-- Cell Lock Inputs -->
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 p-4 bg-card-sub border-2 border-border rounded-lg">
        <div class="space-y-1.5">
          <label for="cell-earfcn-input" class="text-[11px] font-bold text-foreground uppercase tracking-wider block">
            LTE Radio Frequency Channel (EARFCN)
          </label>
          <input
            id="cell-earfcn-input"
            type="number"
            placeholder="e.g. 1800 (Band 3) or 37900 (Band 40)"
            bind:value={lteEarfcn}
            class="neo-input w-full bg-card border-2 border-border rounded px-3 py-2 text-xs font-mono text-foreground focus:outline-none focus:border-accent"
          />
          <span class="text-[10px] text-muted block">
            Downlink center frequency channel of target cell (1-262143).
          </span>
        </div>

        <div class="space-y-1.5">
          <label for="cell-pci-input" class="text-[11px] font-bold text-foreground uppercase tracking-wider block">
            Physical Cell ID (PCI)
          </label>
          <input
            id="cell-pci-input"
            type="number"
            placeholder="0 to 503"
            bind:value={ltePci}
            min="0"
            max="503"
            class="neo-input w-full bg-card border-2 border-border rounded px-3 py-2 text-xs font-mono text-foreground focus:outline-none focus:border-accent"
          />
          <span class="text-[10px] text-muted block">
            Cell identity number broadcasted by specific mast antenna (0-503).
          </span>
        </div>
      </div>

      <!-- Action Buttons -->
      <div class="flex flex-col sm:flex-row items-center justify-between gap-3 pt-2">
        <div class="text-[11px] text-muted">
          {#if cellState.active_lock}
            Currently locked to: <strong class="text-amber-300">EARFCN {cellState.earfcn}, PCI {cellState.pci}</strong>
          {:else}
            No cell lock currently active.
          {/if}
        </div>

        <div class="flex items-center gap-2 w-full sm:w-auto justify-end">
          <Button
            variant="outline"
            size="sm"
            disabled={isClearing || isLocking || !cellState.active_lock}
            onclick={clearCellLock}
          >
            <Unlock class="w-3.5 h-3.5 mr-1" />
            <span>{isClearing ? 'Clearing...' : 'Clear Cell Lock'}</span>
          </Button>
          <Button
            variant="primary"
            size="sm"
            disabled={isLocking || isClearing || !lteEarfcn || !ltePci}
            onclick={applyCellLock}
          >
            <Lock class="w-3.5 h-3.5 mr-1 text-accent-text" />
            <span>{isLocking ? 'Locking Cell...' : 'Apply Cell Lock'}</span>
          </Button>
        </div>
      </div>
    </div>
  </Card>

  <!-- NEARBY TOWER SCANNER & BOOKMARKS -->
  <QualcommCellScanner
    {simSlot}
    onSelectTower={handleSelectTower}
    onLockTower={handleLockTower}
  />
</div>
