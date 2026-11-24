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
  import { LL } from '../../../i18n/i18n-svelte'

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
    persistent_enabled?: boolean
    persistent_earfcn?: number
    persistent_pci?: number
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
          lteEarfcn = String(res.earfcn)
          ltePci = String(res.pci)
        } else if (res.persistent_enabled && res.persistent_earfcn) {
          lteEarfcn = String(res.persistent_earfcn)
          ltePci = String(res.persistent_pci || '')
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
      toastStore.error($LL.qualcomm.cellLock.invalidEarfcn())
      return
    }
    if (isNaN(pciNum) || pciNum < 0 || pciNum > 503) {
      toastStore.error($LL.qualcomm.cellLock.invalidPci())
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
        toastStore.success(res.message || $LL.qualcomm.cellLock.applied({ earfcn: earfcnNum, pci: pciNum }))
        await fetchCellLockState()
      } else {
        toastStore.error(res?.message || $LL.qualcomm.cellLock.rejected())
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.qualcomm.cellLock.requestFailed())
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
        toastStore.success($LL.qualcomm.cellLock.cleared())
        lteEarfcn = ''
        ltePci = ''
        await fetchCellLockState()
      } else {
        toastStore.error(res?.message || $LL.qualcomm.cellLock.clearFailed())
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.qualcomm.cellLock.clearRequestFailed())
    } finally {
      isClearing = false
    }
  }

  function handleSelectTower(earfcn: number, pci: number) {
    lteEarfcn = String(earfcn)
    ltePci = String(pci)
    toastStore.info($LL.qualcomm.cellLock.loadedTower({ earfcn, pci }))
    document.getElementById('cell-earfcn-input')?.focus()
  }

  async function handleLockTower(earfcn: number, pci: number) {
    lteEarfcn = String(earfcn)
    ltePci = String(pci)
    await applyCellLock()
  }
</script>

<div class="space-y-3 sm:space-y-6 font-mono">
  <!-- CELL LOCK STATUS HERO -->
  <Card title={$LL.qualcomm.cellLock.title()} subtitle={$LL.qualcomm.cellLock.subtitle()} tone="mint">
    {#snippet action()}
      <div class="flex items-center gap-2 flex-wrap">
        {#if cellState.active_lock || cellState.persistent_enabled}
          <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-mono font-bold bg-emerald-500/15 text-emerald-500 border border-emerald-500/30">
            <ShieldCheck class="w-3 h-3" />
            <span>{$LL.qualcomm.cellLock.persistentBoot()}</span>
          </span>
        {/if}
        <Button
          variant="secondary"
          size="sm"
          disabled={isLoading || isLocking || isClearing}
          onclick={fetchCellLockState}
        >
          <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoading ? 'animate-spin' : ''}" />
          <span>{$LL.common.refresh()}</span>
        </Button>
      </div>
    {/snippet}

    <div class="space-y-3 sm:space-y-6 text-xs">
      <!-- Target Slot Indicator -->
      <div class="p-3 bg-card-sub border-2 border-border rounded flex flex-col sm:flex-row sm:items-center justify-between gap-2">
        <div class="flex items-center gap-2">
          <span class="text-muted">{$LL.qualcomm.cellLock.targetInterface()}</span>
          <strong class="text-foreground">{targetSimName} (Slot {simSlot})</strong>
        </div>
        {#if cellState.active_lock}
          <Badge variant="warning" class="flex items-center gap-1.5 w-fit">
            <span class="w-2 h-2 rounded-full bg-amber-400 animate-pulse"></span>
            <span>{$LL.qualcomm.cellLock.lockActive({ earfcn: cellState.earfcn, pci: cellState.pci })}</span>
          </Badge>
        {:else}
          <Badge variant="success" class="flex items-center gap-1.5 w-fit">
            <span class="w-2 h-2 rounded-full bg-emerald-400"></span>
            <span>{$LL.qualcomm.cellLock.autoSelection()}</span>
          </Badge>
        {/if}
      </div>

      <!-- Multi-CA and Coverage Warning Notice (English) -->
      <div class="p-3.5 bg-card-sub border-2 border-amber-500/50 rounded-lg space-y-2 text-xs">
        <div class="flex items-center gap-2 text-amber-400 font-bold">
          <AlertTriangle class="w-4 h-4 shrink-0" />
          <span>{$LL.qualcomm.cellLock.caWarningTitle()}</span>
        </div>
        <div class="space-y-1.5 text-[11px] text-muted leading-relaxed pl-6">
          <p>
            <strong class="text-foreground">{$LL.qualcomm.cellLock.warnPccA()}</strong> {$LL.qualcomm.cellLock.warnPccB()}
          </p>
          <p>
            <strong class="text-foreground">{$LL.qualcomm.cellLock.warnMultiA()}</strong> {$LL.qualcomm.cellLock.warnMultiB()} <em>{$LL.qualcomm.cellLock.warnMultiTab()}</em>{$LL.qualcomm.cellLock.warnMultiC()}
          </p>
          <p>
            <strong class="text-foreground">{$LL.qualcomm.cellLock.warnHoA()}</strong> {$LL.qualcomm.cellLock.warnHoB()} <em>Out of Service</em> {$LL.qualcomm.cellLock.warnHoC()}
          </p>
        </div>
      </div>
      <!-- Cell Lock Inputs -->
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 p-4 bg-card-sub border-2 border-border rounded-lg">
        <div class="space-y-1.5">
          <label for="cell-earfcn-input" class="text-[11px] font-bold text-foreground uppercase tracking-wider block">
            {$LL.qualcomm.cellLock.earfcnLabel()}
          </label>
          <input
            id="cell-earfcn-input"
            type="number"
            placeholder={$LL.qualcomm.cellLock.earfcnPlaceholder()}
            bind:value={lteEarfcn}
            class="neo-input w-full bg-card border-2 border-border rounded px-3 py-2 text-xs font-mono text-foreground focus:outline-none focus:border-accent"
          />
          <span class="text-[10px] text-muted block">
            {$LL.qualcomm.cellLock.earfcnHint()}
          </span>
        </div>

        <div class="space-y-1.5">
          <label for="cell-pci-input" class="text-[11px] font-bold text-foreground uppercase tracking-wider block">
            {$LL.qualcomm.cellLock.pciLabel()}
          </label>
          <input
            id="cell-pci-input"
            type="number"
            placeholder={$LL.qualcomm.cellLock.pciPlaceholder()}
            bind:value={ltePci}
            min="0"
            max="503"
            class="neo-input w-full bg-card border-2 border-border rounded px-3 py-2 text-xs font-mono text-foreground focus:outline-none focus:border-accent"
          />
          <span class="text-[10px] text-muted block">
            {$LL.qualcomm.cellLock.pciHint()}
          </span>
        </div>
      </div>

      <!-- Action Buttons -->
      <div class="flex flex-col sm:flex-row items-center justify-between gap-3 pt-2">
        <div class="text-[11px] text-muted">
          {#if cellState.active_lock}
            {$LL.qualcomm.cellLock.lockedToA()} <strong class="text-amber-300">{$LL.qualcomm.cellLock.lockedToB({ earfcn: cellState.earfcn, pci: cellState.pci })}</strong>
          {:else}
            {$LL.qualcomm.cellLock.noLock()}
          {/if}
        </div>

        <div class="flex items-center gap-2 w-full sm:w-auto justify-end">
          <Button
            variant="outline"
            size="sm"
            disabled={isClearing || isLocking}
            onclick={clearCellLock}
            title={$LL.qualcomm.cellLock.resetTitle()}
          >
            <RotateCcw class="w-3.5 h-3.5 mr-1 {isClearing ? 'animate-spin' : ''}" />
            <span>{isClearing ? $LL.qualcomm.cellLock.resetting() : $LL.qualcomm.cellLock.resetToAuto()}</span>
          </Button>
          <Button
            variant="primary"
            size="sm"
            disabled={isLocking || isClearing || !lteEarfcn || !ltePci}
            onclick={applyCellLock}
          >
            <Lock class="w-3.5 h-3.5 mr-1 text-accent-text" />
            <span>{isLocking ? $LL.qualcomm.cellLock.locking() : $LL.qualcomm.cellLock.apply()}</span>
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
