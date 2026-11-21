<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Smartphone,
    Radio,
    Lock,
    RotateCcw,
    RefreshCw,
    AlertTriangle,
    CheckCircle,
    Sliders,
    ShieldCheck,
    Zap,
    Check,
    Search,
    Layers,
    Info,
    Cpu,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import { LL } from '../../../i18n/i18n-svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Modal from '../../ui/Modal.svelte'

  interface SamsungDeviceStatus {
    is_samsung: boolean
    manufacturer: string
    model: string
    device: string
    oneui_version: string
    dex_exists: boolean
    cli_exists: boolean
    cli_path: string
    supported_slots: number
    active_data_slot?: number
  }

  interface BandItem {
    id: number
    name: string
    active: boolean
  }

  interface SamsungBandStatus {
    slot: number
    mode: string // "AUTOMATIC" | "LOCKED"
    active_bands: number[]
    bands: BandItem[]
    error?: string
  }

  // State
  let deviceStatus = $state<SamsungDeviceStatus | null>(null)
  let isLoadingStatus = $state(false)

  let selectedSlot = $state<number>(0)
  let bandData = $state<SamsungBandStatus | null>(null)
  let isLoadingBands = $state(false)
  let bandsError = $state<string | null>(null)

  let selectedBandIds = $state<number[]>([])
  let searchQuery = $state('')
  let isLocking = $state(false)
  let isResetting = $state(false)
  let showResetModal = $state(false)

  // Popular LTE bands in Indonesia (Telkomsel, Indosat, XL, Smartfren)
  const popularBandIds = [1, 3, 8, 40]

  // Derived computations
  const isSamsung = $derived(deviceStatus?.is_samsung ?? false)
  const activeDataSlot = $derived(deviceStatus?.active_data_slot ?? 0)
  const isLockedMode = $derived(bandData?.mode === 'LOCKED')
  const allBands = $derived(bandData?.bands ?? [])
  const activeBandsList = $derived(
    allBands.filter((b) => b.active || (bandData?.active_bands ?? []).includes(b.id))
  )

  const filteredBands = $derived(
    allBands.filter((b) => {
      if (!searchQuery.trim()) return true
      const q = searchQuery.toLowerCase().trim()
      return (
        b.name.toLowerCase().includes(q) ||
        b.id.toString().includes(q) ||
        `b${b.id}`.toLowerCase().includes(q)
      )
    })
  )

  const isAllSelected = $derived(
    allBands.length > 0 && selectedBandIds.length === allBands.length
  )

  const availablePopularIds = $derived(
    allBands.filter((b) => popularBandIds.includes(b.id)).map((b) => b.id)
  )

  const isPopularSelected = $derived(
    availablePopularIds.length > 0 &&
      availablePopularIds.every((id) => selectedBandIds.includes(id)) &&
      selectedBandIds.length === availablePopularIds.length
  )

  onMount(async () => {
    await loadInitialData()
  })

  async function loadInitialData() {
    await Promise.allSettled([
      fetchDeviceStatus(),
      fetchBands(selectedSlot),
    ])
  }

  async function fetchDeviceStatus() {
    try {
      isLoadingStatus = true
      const res = await api.get<SamsungDeviceStatus>('/api/samsung/status')
      deviceStatus = res
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : $LL.samsung.toasts.statusFailed()
      toastStore.error(msg, $LL.samsung.toasts.statusError())
    } finally {
      isLoadingStatus = false
    }
  }

  async function fetchBands(slot: number) {
    try {
      isLoadingBands = true
      bandsError = null
      const res = await api.get<SamsungBandStatus>(`/api/samsung/bands?slot=${slot}`)
      bandData = res

      // Update selected bands based on server reported active bands if in LOCKED mode
      if (res.mode === 'LOCKED' && Array.isArray(res.active_bands) && res.active_bands.length > 0) {
        selectedBandIds = [...res.active_bands]
      } else if (res.mode === 'LOCKED') {
        selectedBandIds = res.bands.filter((b) => b.active).map((b) => b.id)
      } else {
        selectedBandIds = []
      }

      if (res.error) {
        bandsError = res.error
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : $LL.samsung.toasts.bandsFailed()
      bandsError = msg
      toastStore.error(msg, $LL.samsung.toasts.bandsQueryError())
    } finally {
      isLoadingBands = false
    }
  }

  async function handleSlotChange(slot: number) {
    if (slot === selectedSlot || isLoadingBands) return
    selectedSlot = slot
    await fetchBands(slot)
  }

  function toggleBand(id: number) {
    if (selectedBandIds.includes(id)) {
      selectedBandIds = selectedBandIds.filter((b) => b !== id)
    } else {
      selectedBandIds = [...selectedBandIds, id]
    }
  }

  function selectPopular() {
    if (availablePopularIds.length === 0) {
      toastStore.warning($LL.samsung.toasts.popularUnavailable())
      return
    }
    selectedBandIds = [...availablePopularIds]
    toastStore.info($LL.samsung.toasts.popularSelected({ count: selectedBandIds.length }))
  }

  function selectAll() {
    if (allBands.length === 0) return
    selectedBandIds = allBands.map((b) => b.id)
    toastStore.info($LL.samsung.toasts.allSelected({ count: allBands.length }))
  }

  function deselectAll() {
    selectedBandIds = []
  }

  function clearAll() {
    deselectAll()
  }

  async function applyLock() {
    if (selectedBandIds.length === 0) {
      toastStore.warning($LL.samsung.toasts.selectAtLeastOne())
      return
    }

    try {
      isLocking = true
      await api.post<{ status: string; message: string }>('/api/samsung/lock', {
        slot: selectedSlot,
        bands: selectedBandIds,
      })
      toastStore.success(
        $LL.samsung.toasts.lockApplied({ slot: selectedSlot + 1 }),
        $LL.samsung.toasts.lockAppliedTitle()
      )
      await fetchBands(selectedSlot)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : $LL.samsung.toasts.lockSendFailed()
      toastStore.error(msg, $LL.samsung.toasts.lockFailed())
    } finally {
      isLocking = false
    }
  }

  async function confirmResetAuto() {
    try {
      isResetting = true
      await api.post<{ status: string; message: string }>('/api/samsung/auto', {
        slot: selectedSlot,
      })
      toastStore.success(
        $LL.samsung.toasts.resetDone({ slot: selectedSlot + 1 }),
        $LL.samsung.toasts.resetDoneTitle()
      )
      showResetModal = false
      await fetchBands(selectedSlot)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : $LL.samsung.toasts.resetSendFailed()
      toastStore.error(msg, $LL.samsung.toasts.resetFailed())
    } finally {
      isResetting = false
    }
  }

  function formatBandDisplay(band: BandItem): { label: string; desc: string } {
    const raw = band.name || ''
    const match = raw.match(/^(B\d+|n\d+|LTE\s*B\d+|NR\s*n\d+)\s*(.*)$/i)
    if (match) {
      let label = match[1].replace(/LTE\s*/i, '').replace(/NR\s*/i, '')
      let desc = match[2].replace(/[()]/g, '').trim()
      return { label: label.toUpperCase(), desc: desc || $LL.samsung.bands.bandFallback({ id: band.id }) }
    }
    return {
      label: `B${band.id}`,
      desc: raw.replace(/[()]/g, '').trim() || $LL.samsung.bands.lteBand(),
    }
  }
</script>

<div class="space-y-3 sm:space-y-6 pb-12 font-mono">
  <!-- TOP HERO & REFRESH -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b-2 border-border pb-4">
    <div>
      <div class="flex items-center gap-2">
        <Smartphone class="w-6 h-6 text-accent" />
        <h1 class="text-xl sm:text-2xl font-black uppercase tracking-wider text-foreground">
          {$LL.samsung.title()}
        </h1>
        <Badge variant="info" class="text-[10px]">SecRIL</Badge>
      </div>
      <p class="text-xs text-muted mt-1">
        {$LL.samsung.subtitle()}
      </p>
    </div>

    <div class="flex items-center gap-2">
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoadingStatus || isLoadingBands}
        onclick={loadInitialData}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoadingStatus || isLoadingBands ? 'animate-spin' : ''}" />
        <span>{$LL.common.refresh()}</span>
      </Button>
    </div>
  </div>

  <!-- TARGET ARCHITECTURE & HARDWARE SPECIFICATIONS -->
  <div class="p-4 bg-card-sub border-2 border-border rounded-lg space-y-3 shadow-neobrutal-sm">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b-2 border-border/60 pb-3">
      <div class="flex items-center gap-2">
        <Cpu class="w-4 h-4 text-accent" />
        <span class="text-xs font-bold uppercase tracking-wider text-foreground">
          {$LL.samsung.compat.title()}
        </span>
      </div>
      {#if isSamsung}
        <Badge variant="success" class="flex items-center gap-1.5 w-fit">
          <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
          <span>{$LL.samsung.compat.compatible()}</span>
        </Badge>
      {:else}
        <Badge variant="warning" class="flex items-center gap-1.5 w-fit">
          <AlertTriangle class="w-3.5 h-3.5" />
          <span>{$LL.samsung.compat.incompatible()}</span>
        </Badge>
      {/if}
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 text-xs">
      <div class="p-2.5 bg-card border-2 border-border rounded">
        <span class="text-[10px] text-muted uppercase font-bold block mb-1">{$LL.samsung.compat.targetDevice()}</span>
        <p class="font-bold text-foreground">Galaxy Z Flip 3-6, Fold 3-6, S20-S24, Note, A & M Series</p>
      </div>
      <div class="p-2.5 bg-card border-2 border-border rounded">
        <span class="text-[10px] text-muted uppercase font-bold block mb-1">{$LL.samsung.compat.targetRom()}</span>
        <p class="font-bold text-foreground">Official Samsung OneUI (One UI 3.x - 7.x with SecRIL)</p>
      </div>
      <div class="p-2.5 bg-card border-2 border-border rounded">
        <span class="text-[10px] text-muted uppercase font-bold block mb-1">{$LL.samsung.compat.targetChipset()}</span>
        <p class="font-bold text-foreground">Qualcomm Snapdragon & Samsung Exynos (Running OneUI)</p>
      </div>
      <div class="p-2.5 bg-card border-2 border-border rounded">
        <span class="text-[10px] text-muted uppercase font-bold block mb-1">{$LL.samsung.compat.targetModem()}</span>
        <p class="font-bold text-foreground">SecRIL Telephony Subsystem (service call phone 73/74)</p>
      </div>
    </div>
  </div>

  <!-- NON-SAMSUNG WARNING BANNER -->
  {#if deviceStatus && !isSamsung}
    <div
      class="p-4 rounded-lg border-2 border-amber-500 bg-amber-950/25 shadow-neobrutal-sm text-foreground space-y-2 select-none"
      role="alert"
    >
      <div class="flex items-center gap-2 text-amber-400 font-bold uppercase text-xs tracking-wider">
        <AlertTriangle class="w-5 h-5 shrink-0 text-amber-400" />
        <span>{$LL.samsung.warning.title()}</span>
      </div>
      <p class="text-xs text-muted leading-relaxed">
        {$LL.samsung.warning.bodyA()} <span class="text-foreground font-bold">{deviceStatus.manufacturer || 'Non-Samsung'}</span> {$LL.samsung.warning.bodyB()} <span class="text-foreground font-bold">{deviceStatus.model || $LL.common.unknown()}</span>{$LL.samsung.warning.bodyC()}
      </p>
    </div>
  {/if}

  <!-- DEVICE HARDWARE & TOOL ENVIRONMENT -->
  <Card title={$LL.samsung.hardware.title()} subtitle={$LL.samsung.hardware.subtitle()} tone="ice">
    {#snippet action()}
      {#if isSamsung}
        <Badge variant="success" class="flex items-center gap-1">
          <ShieldCheck class="w-3 h-3" />
          <span>{$LL.samsung.hardware.verified()}</span>
        </Badge>
      {:else}
        <Badge variant="warning" class="flex items-center gap-1">
          <AlertTriangle class="w-3 h-3" />
          <span>{$LL.samsung.hardware.nonSamsung()}</span>
        </Badge>
      {/if}
    {/snippet}

    <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
      <!-- Device & Model -->
      <div class="p-3 bg-card-sub border-2 border-border rounded space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold block">{$LL.samsung.hardware.deviceModel()}</span>
        <p class="font-bold text-foreground truncate">
          {deviceStatus?.manufacturer || $LL.common.unknown()} {deviceStatus?.model || '-'}
        </p>
        <span class="text-[10px] text-muted block truncate font-mono">
          {$LL.samsung.hardware.codename({ device: deviceStatus?.device || '-' })}
        </span>
      </div>

      <!-- One UI Version -->
      <div class="p-3 bg-card-sub border-2 border-border rounded space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold block">{$LL.samsung.hardware.oneUiVersion()}</span>
        <p class="font-bold text-foreground truncate">
          {deviceStatus?.oneui_version ? $LL.samsung.hardware.oneUi({ version: deviceStatus.oneui_version }) : $LL.samsung.hardware.standardAndroid()}
        </p>
        <span class="text-[10px] text-muted block truncate">
          Samsung RIL Framework
        </span>
      </div>

      <!-- DEX Hook Status -->
      <div class="p-3 bg-card-sub border-2 border-border rounded space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold block">{$LL.samsung.hardware.dexHook()}</span>
        <div class="pt-0.5">
          {#if deviceStatus?.dex_exists}
            <Badge variant="success" class="flex items-center gap-1 w-fit">
              <CheckCircle class="w-3 h-3" />
              <span>{$LL.samsung.hardware.dexReady()}</span>
            </Badge>
          {:else}
            <Badge variant="danger" class="flex items-center gap-1 w-fit">
              <AlertTriangle class="w-3 h-3" />
              <span>{$LL.samsung.hardware.dexMissing()}</span>
            </Badge>
          {/if}
        </div>
        <span class="text-[10px] text-muted block truncate">
          RIL Class Injector
        </span>
      </div>

      <!-- Native Go Runner Status -->
      <div class="p-3 bg-card-sub border-2 border-border rounded space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold block">{$LL.samsung.hardware.executionEngine()}</span>
        <div class="pt-0.5">
          {#if deviceStatus?.dex_exists}
            <Badge variant="success" class="flex items-center gap-1 w-fit">
              <CheckCircle class="w-3 h-3" />
              <span>NATIVE RUNNER</span>
            </Badge>
          {:else}
            <Badge variant="warning" class="flex items-center gap-1 w-fit">
              <AlertTriangle class="w-3 h-3" />
              <span>{$LL.samsung.hardware.notReady()}</span>
            </Badge>
          {/if}
        </div>
        <span class="text-[10px] text-muted block truncate font-mono">
          Go UID 1001 Radio Runner
        </span>
      </div>
    </div>
  </Card>

  <!-- SIM SELECTOR & CURRENT MODE STATUS -->
  <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
    <!-- SIM SELECTOR (1 col) -->
    <Card title={$LL.samsung.sim.title()} subtitle={$LL.samsung.sim.subtitle()} class="md:col-span-1">
      <div class="flex flex-col gap-2.5">
        <button
          type="button"
          class="neo-button p-3 rounded border-2 text-left cursor-pointer flex items-center justify-between transition-all {selectedSlot === 0 ? 'border-accent bg-accent/15 text-foreground shadow-neobrutal-sm font-bold' : 'border-border bg-card-sub text-muted hover:border-foreground/30 hover:bg-card'}"
          disabled={isLoadingBands}
          onclick={() => handleSlotChange(0)}
        >
          <div class="flex items-center gap-2.5">
            <Radio class="w-4 h-4 {selectedSlot === 0 ? 'text-accent' : 'text-muted'}" />
            <div>
              <div class="text-sm font-bold">SIM 1</div>
              <div class="text-[10px] text-muted">{$LL.samsung.sim.slotIndex({ index: 0 })}</div>
            </div>
          </div>
          <div class="flex flex-col items-end gap-1">
            {#if activeDataSlot === 0}
              <Badge variant="success" class="text-[9px] px-1.5 py-0 flex items-center gap-1">
                <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
                <span>{$LL.samsung.sim.activeData()}</span>
              </Badge>
            {/if}
            {#if selectedSlot === 0}
              <Badge variant="info">{$LL.samsung.sim.selected()}</Badge>
            {/if}
          </div>
        </button>

        <button
          type="button"
          class="neo-button p-3 rounded border-2 text-left cursor-pointer flex items-center justify-between transition-all {selectedSlot === 1 ? 'border-accent bg-accent/15 text-foreground shadow-neobrutal-sm font-bold' : 'border-border bg-card-sub text-muted hover:border-foreground/30 hover:bg-card'}"
          disabled={isLoadingBands}
          onclick={() => handleSlotChange(1)}
        >
          <div class="flex items-center gap-2.5">
            <Radio class="w-4 h-4 {selectedSlot === 1 ? 'text-accent' : 'text-muted'}" />
            <div>
              <div class="text-sm font-bold">SIM 2</div>
              <div class="text-[10px] text-muted">{$LL.samsung.sim.slotIndex({ index: 1 })}</div>
            </div>
          </div>
          <div class="flex flex-col items-end gap-1">
            {#if activeDataSlot === 1}
              <Badge variant="success" class="text-[9px] px-1.5 py-0 flex items-center gap-1">
                <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
                <span>{$LL.samsung.sim.activeData()}</span>
              </Badge>
            {/if}
            {#if selectedSlot === 1}
              <Badge variant="info">{$LL.samsung.sim.selected()}</Badge>
            {/if}
          </div>
        </button>
      </div>
    </Card>

    <!-- STATUS MODE CARD (2 cols) -->
    <Card title={$LL.samsung.status.title()} subtitle={$LL.samsung.status.subtitle()} tone="mint" class="md:col-span-2">
      {#snippet action()}
        {#if bandData}
          {#if isLockedMode}
            <Badge variant="warning" class="px-2 py-1 text-xs">
              <Lock class="w-3 h-3 mr-1 inline" />
              <span>{$LL.samsung.status.locked()}</span>
            </Badge>
          {:else}
            <Badge variant="success" class="px-2 py-1 text-xs">
              <Zap class="w-3 h-3 mr-1 inline" />
              <span>{$LL.samsung.status.automatic()}</span>
            </Badge>
          {/if}
        {/if}
      {/snippet}

      <div class="space-y-3 font-mono text-xs">
        <div class="flex items-center justify-between p-2.5 rounded bg-card-sub border-2 border-border">
          <span class="text-muted text-[11px] uppercase font-bold">{$LL.samsung.status.currentMode()}</span>
          <span class="font-bold flex items-center gap-1.5 {isLockedMode ? 'text-amber-400' : 'text-emerald-400'}">
            {#if isLockedMode}
              <CheckCircle class="w-3.5 h-3.5 text-amber-400" />
              <span>{$LL.samsung.status.lockedToBands({ count: activeBandsList.length })}</span>
            {:else}
              <CheckCircle class="w-3.5 h-3.5 text-emerald-400" />
              <span>{$LL.samsung.status.automaticSelection()}</span>
            {/if}
          </span>
        </div>

        <div class="flex items-center justify-between p-2.5 rounded bg-card-sub border-2 border-border">
          <span class="text-muted text-[11px] uppercase font-bold">{$LL.samsung.status.registeredFrequencies()}</span>
          <span class="font-bold text-foreground">
            {$LL.samsung.status.registered({ active: activeBandsList.length, total: allBands.length })}
          </span>
        </div>

        <!-- Active Bands Chips Preview -->
        <div>
          <span class="text-[10px] text-muted uppercase font-bold block mb-1.5">
            {$LL.samsung.status.activeFrequencies()}
          </span>
          {#if activeBandsList.length === 0}
            <p class="text-xs text-muted italic">
              {$LL.samsung.status.noBands()}
            </p>
          {:else}
            <div class="flex flex-wrap gap-1.5 max-h-24 overflow-y-auto p-1.5 bg-card-sub border border-border/60 rounded">
              {#each activeBandsList as b}
                {@const fmt = formatBandDisplay(b)}
                <span class="px-2 py-0.5 rounded text-[11px] font-bold bg-accent/20 border border-accent text-accent">
                  {fmt.label}
                </span>
              {/each}
            </div>
          {/if}
        </div>
      </div>
    </Card>
  </div>

  <!-- BAND SELECTION MATRIX -->
  <Card
    title={$LL.samsung.bands.title()}
    subtitle={$LL.samsung.bands.subtitle()}
    tone="lavender"
  >
    {#snippet action()}
      <div class="flex items-center gap-2">
        <Badge variant="default" class="text-xs">
          {$LL.samsung.bands.bandsSelected({ count: selectedBandIds.length })}
        </Badge>
      </div>
    {/snippet}

    <div class="space-y-4">
      <!-- Search and Quick Filter Controls -->
      <div class="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 pb-3 border-b-2 border-border">
        <!-- Search Input -->
        <div class="relative flex-1">
          <Search class="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-muted" />
          <input
            type="text"
            placeholder={$LL.samsung.bands.searchPlaceholder()}
            bind:value={searchQuery}
            class="neo-input w-full pl-9 pr-3 py-1.5 text-xs bg-card-sub border-2 border-border rounded text-foreground focus:outline-none focus:border-accent"
          />
        </div>

        <!-- Quick Filter Buttons -->
        <div class="flex items-center gap-1.5 self-end sm:self-auto">
          <Button
            variant={isPopularSelected ? 'primary' : 'secondary'}
            size="sm"
            onclick={selectPopular}
            title={$LL.samsung.bands.selectPopularTitle()}
          >
            <Zap class="w-3.5 h-3.5 mr-1 text-amber-400" />
            <span>{$LL.samsung.bands.selectPopular()}</span>
          </Button>

          <Button
            variant={isAllSelected ? 'primary' : 'outline'}
            size="sm"
            onclick={isAllSelected ? deselectAll : selectAll}
          >
            <Check class="w-3.5 h-3.5 mr-1" />
            <span>{isAllSelected ? $LL.common.deselectAll() : $LL.common.selectAll()}</span>
          </Button>

          {#if selectedBandIds.length > 0}
            <Button variant="ghost" size="sm" onclick={clearAll}>
              <span>{$LL.common.clear()}</span>
            </Button>
          {/if}
        </div>
      </div>

      <!-- Loading State -->
      {#if isLoadingBands}
        <div class="p-8 text-center text-xs text-muted flex flex-col items-center justify-center gap-2">
          <RefreshCw class="w-6 h-6 animate-spin text-accent" />
          <span>{$LL.samsung.bands.loading()}</span>
        </div>
      {:else if bandsError && allBands.length === 0}
        <!-- Error State -->
        <div class="p-4 rounded-lg border-2 border-red-500 bg-red-950/30 text-red-200 space-y-2">
          <div class="flex items-center gap-2 font-bold text-xs uppercase">
            <AlertTriangle class="w-4 h-4 text-red-400" />
            <span>{$LL.samsung.bands.readFailed()}</span>
          </div>
          <p class="text-xs text-muted">{bandsError}</p>
          <Button variant="secondary" size="sm" onclick={() => fetchBands(selectedSlot)}>
            <RefreshCw class="w-3 h-3 mr-1" />
            <span>{$LL.samsung.bands.retryQuery()}</span>
          </Button>
        </div>
      {:else if filteredBands.length === 0}
        <!-- Empty Results -->
        <div class="p-8 text-center text-xs text-muted space-y-2">
          <Layers class="w-8 h-8 mx-auto opacity-40 text-muted" />
          <p class="font-bold text-foreground">{$LL.samsung.bands.noBandsFound()}</p>
          <p class="text-[11px] text-muted max-w-md mx-auto">
            {searchQuery
              ? $LL.samsung.bands.noMatch({ query: searchQuery })
              : $LL.samsung.bands.noBandsBody()}
          </p>
        </div>
      {:else}
        <!-- BANDS GRID -->
        <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-2.5">
          {#each filteredBands as band (band.id)}
            {@const isSelected = selectedBandIds.includes(band.id)}
            {@const isLiveActive = band.active}
            {@const fmt = formatBandDisplay(band)}
            {@const isNR = fmt.label.startsWith('N')}

            <button
              type="button"
              class="p-2.5 rounded-lg border-2 text-left cursor-pointer transition-all flex flex-col justify-between select-none {isSelected
                ? isNR
                  ? 'border-cyan-500 bg-cyan-950/30 shadow-neobrutal-sm ring-1 ring-cyan-500/50'
                  : 'border-accent bg-accent/20 shadow-neobrutal-sm ring-1 ring-accent/50'
                : 'border-border bg-card-sub hover:border-foreground/30 hover:bg-card'}"
              onclick={() => toggleBand(band.id)}
            >
              <div class="flex items-start justify-between gap-1">
                <div>
                  <span class="text-sm font-black {isSelected ? (isNR ? 'text-cyan-300' : 'text-accent') : 'text-foreground'}">
                    {fmt.label}
                  </span>
                  <span class="text-[9px] text-muted block truncate" title={fmt.desc}>
                    {fmt.desc}
                  </span>
                </div>

                <!-- Checkbox Indicator -->
                <div
                  class="w-4 h-4 rounded border-2 flex items-center justify-center shrink-0 mt-0.5 {isSelected
                    ? isNR
                      ? 'border-cyan-500 bg-cyan-500 text-black'
                      : 'border-accent bg-accent text-accent-text'
                    : 'border-border bg-card'}"
                >
                  {#if isSelected}
                    <Check class="w-3 h-3 stroke-[3]" />
                  {/if}
                </div>
              </div>

              <!-- Live Active Badge -->
              <div class="flex items-center justify-between gap-1 pt-2 mt-2 border-t border-border/40 text-[9px]">
                <span class="text-muted font-mono">ID: {band.id}</span>
                {#if isLiveActive}
                  <Badge variant="success" class="text-[8px] px-1 py-0 font-bold">{$LL.samsung.bands.activeBadge()}</Badge>
                {/if}
              </div>
            </button>
          {/each}
        </div>
      {/if}
    </div>
  </Card>

  <!-- ACTION BAR (LOCKED / RESET / APPLY) -->
  <div class="p-4 bg-card border-2 border-border shadow-neobrutal rounded-lg flex flex-col sm:flex-row items-center justify-between gap-4 sticky bottom-4 z-20 backdrop-blur-md">
    <div class="flex items-center gap-3 w-full sm:w-auto">
      <div class="p-2 rounded bg-accent/20 border border-accent text-accent">
        <Sliders class="w-5 h-5" />
      </div>
      <div>
        <div class="text-xs font-bold uppercase text-foreground">
          {$LL.samsung.actionBar.simConfig({ slot: selectedSlot + 1 })}
        </div>
        <div class="text-[11px] text-muted">
          {#if selectedBandIds.length === 0}
            <span class="text-red-400 font-bold">{$LL.samsung.actionBar.selectAtLeastOne()}</span>
          {:else}
            <span class="text-foreground font-bold">{$LL.samsung.actionBar.bandsCount({ count: selectedBandIds.length })}</span> {$LL.samsung.actionBar.readyToLock()}
          {/if}
        </div>
      </div>
    </div>

    <div class="flex flex-wrap items-center justify-end gap-2.5 w-full sm:w-auto">
      <!-- Reset to Auto Button -->
      <Button
        variant="secondary"
        size="md"
        disabled={isLoadingBands || isResetting || isLocking}
        onclick={() => (showResetModal = true)}
        title={$LL.samsung.actionBar.resetTitle()}
      >
        <RotateCcw class="w-4 h-4 mr-1.5" />
        <span>{$LL.samsung.actionBar.resetToAuto()}</span>
      </Button>

      <!-- Lock Selected Bands Button -->
      <Button
        variant="primary"
        size="md"
        disabled={selectedBandIds.length === 0 || isLoadingBands || isLocking}
        onclick={applyLock}
        title={$LL.samsung.actionBar.applyLockTitle()}
      >
        {#if isLocking}
          <RefreshCw class="w-4 h-4 mr-1.5 animate-spin" />
          <span>{$LL.samsung.actionBar.locking()}</span>
        {:else}
          <Lock class="w-4 h-4 mr-1.5" />
          <span>{$LL.samsung.actionBar.lockSelected()}</span>
        {/if}
      </Button>
    </div>
  </div>
</div>

<!-- RESET CONFIRMATION MODAL -->
<Modal
  open={showResetModal}
  title={$LL.samsung.resetModal.title()}
  onclose={() => (showResetModal = false)}
>
  <div class="space-y-4 font-mono text-xs">
    <div class="p-3.5 rounded border-2 border-amber-500/40 bg-amber-950/30 flex items-start gap-3 text-amber-200">
      <AlertTriangle class="w-5 h-5 shrink-0 text-amber-400 mt-0.5" />
      <div class="space-y-1">
        <p class="font-bold text-sm text-foreground">
          {$LL.samsung.resetModal.heading({ slot: selectedSlot + 1 })}
        </p>
        <p class="text-muted leading-relaxed">
          {$LL.samsung.resetModal.body({ slot: selectedSlot + 1 })}
        </p>
      </div>
    </div>

    <p class="text-muted text-[11px] leading-relaxed">
      {$LL.samsung.resetModal.note()}
    </p>
  </div>

  {#snippet footer()}
    <div class="flex items-center justify-end gap-2 w-full font-mono">
      <Button
        variant="outline"
        size="sm"
        disabled={isResetting}
        onclick={() => (showResetModal = false)}
      >
        {$LL.common.cancel()}
      </Button>
      <Button
        variant="danger"
        size="sm"
        disabled={isResetting}
        onclick={confirmResetAuto}
      >
        {#if isResetting}
          <RefreshCw class="w-3.5 h-3.5 mr-1.5 animate-spin" />
          <span>{$LL.samsung.resetModal.resettingModem()}</span>
        {:else}
          <RotateCcw class="w-3.5 h-3.5 mr-1.5" />
          <span>{$LL.samsung.resetModal.confirmReset()}</span>
        {/if}
      </Button>
    </div>
  {/snippet}
</Modal>
