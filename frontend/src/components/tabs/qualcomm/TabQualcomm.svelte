<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Zap,
    RefreshCw,
    Radio,
    Search,
    Sliders,
    AlertTriangle,
    Cpu,
    HardDrive,
    Database,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import QualcommBands from './QualcommBands.svelte'
  import QualcommCellLock from './QualcommCellLock.svelte'
  import QualcommProfiles from './QualcommProfiles.svelte'
  import QualcommEFS from './QualcommEFS.svelte'
  import QualcommNV from './QualcommNV.svelte'

  interface BandConfig {
    engine: 'qualcomm_qrtr' | 'universal' | 'qualcomm_at' | 'intent'
    preferred_rat: 'hybrid' | '5g_only' | '4g_only' | '3g_only'
    lte_bands: number[]
    nr_bands: number[]
    hex_bitmask?: string
    sim_slot?: number
    nr_mode?: 'both' | 'sa' | 'nsa'
    is_qualcomm?: boolean
    active_lte_bands?: number[]
    active_nr_bands?: number[]
  }

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

  let subTab = $state<'bands' | 'cell_lock' | 'profiles' | 'efs' | 'nv'>('bands')

  let bandConfig = $state<BandConfig>({
    engine: 'qualcomm_qrtr',
    preferred_rat: 'hybrid',
    lte_bands: [1, 3, 8],
    nr_bands: [1, 3, 40],
    hex_bitmask: '',
    sim_slot: 0,
    nr_mode: 'both',
    is_qualcomm: false,
    active_lte_bands: [],
    active_nr_bands: [],
  })

  let simStatus = $state<SimStatus | null>(null)
  let isLoading = $state(false)
  let isLoadingSim = $state(false)

  onMount(async () => {
    await fetchSimStatus()
    const initialSlot = simStatus && simStatus.active_slot !== undefined ? simStatus.active_slot : 0
    bandConfig.sim_slot = initialSlot
    await fetchBands(initialSlot)
  })

  async function fetchBands(slotIndex?: number) {
    const targetSlot = slotIndex !== undefined ? slotIndex : (bandConfig.sim_slot ?? 0)
    try {
      isLoading = true
      const res = await api.get<BandConfig>(`/api/modem/bands?slot=${targetSlot}`)
      if (res) {
        bandConfig = {
          engine: 'qualcomm_qrtr',
          preferred_rat: res.preferred_rat || 'hybrid',
          lte_bands:
            Array.isArray(res.lte_bands) && res.lte_bands.length > 0
              ? res.lte_bands
              : res.active_lte_bands && res.active_lte_bands.length > 0
                ? res.active_lte_bands
                : [1, 3, 8],
          nr_bands: Array.isArray(res.nr_bands) ? res.nr_bands : [1, 3, 40],
          hex_bitmask: res.hex_bitmask || '',
          sim_slot: targetSlot,
          nr_mode: res.nr_mode || 'both',
          is_qualcomm: res.is_qualcomm ?? false,
          active_lte_bands: res.active_lte_bands || [],
          active_nr_bands: res.active_nr_bands || [],
        }
      }
    } catch {
      // Retain fallback defaults
    } finally {
      isLoading = false
    }
  }

  async function fetchSimStatus() {
    try {
      isLoadingSim = true
      const res = await api.get<SimStatus>('/api/modem/sim/status')
      if (res) {
        simStatus = res
      }
    } catch {
      // Retain fallback defaults
    } finally {
      isLoadingSim = false
    }
  }

  function handleSlotChange(slotIndex: number) {
    bandConfig.sim_slot = slotIndex
    fetchBands(slotIndex)
  }
</script>

<div class="space-y-6 pb-12 font-mono">
  <!-- TOP HERO & REFRESH -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b-2 border-border pb-4">
    <div>
      <div class="flex items-center gap-2">
        <Zap class="w-6 h-6 text-accent" />
        <h1 class="text-xl sm:text-2xl font-black uppercase tracking-wider text-foreground">
          Snapdragon QMI Control Panel
        </h1>
        <Badge variant="success" class="text-[10px]">AF_QIPCRTR</Badge>
      </div>
      <p class="text-xs text-muted mt-1">
        Direct Qualcomm IPC Router socket modulation via QMI Network Access Service (NAS).
      </p>
    </div>

    <div class="flex items-center gap-2">
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoading || isLoadingSim}
        onclick={() => {
          fetchBands(bandConfig.sim_slot)
          fetchSimStatus()
        }}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoading || isLoadingSim ? 'animate-spin' : ''}" />
        <span>Refresh</span>
      </Button>
    </div>
  </div>

  <!-- TARGET ARCHITECTURE & HARDWARE SPECIFICATIONS -->
  <div class="p-4 bg-card-sub border-2 border-border rounded-lg space-y-3 shadow-neobrutal-sm">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b-2 border-border/60 pb-3">
      <div class="flex items-center gap-2">
        <Cpu class="w-4 h-4 text-accent" />
        <span class="text-xs font-bold uppercase tracking-wider text-foreground">
          Target Architecture & Hardware Compatibility
        </span>
      </div>
      {#if bandConfig.is_qualcomm}
        <Badge variant="success" class="flex items-center gap-1.5 w-fit">
          <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
          <span>COMPATIBLE & DETECTED ON THIS DEVICE</span>
        </Badge>
      {:else}
        <Badge variant="warning" class="flex items-center gap-1.5 w-fit">
          <AlertTriangle class="w-3.5 h-3.5" />
          <span>NON-QUALCOMM / QRTR NOT DETECTED</span>
        </Badge>
      {/if}
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 text-xs">
      <div class="p-2.5 bg-card border-2 border-border rounded">
        <span class="text-[10px] text-muted uppercase font-bold block mb-1">Target Device</span>
        <p class="font-bold text-foreground">Google Pixel 1-5, Xiaomi, POCO, OnePlus, Sony, Moto, ASUS</p>
      </div>
      <div class="p-2.5 bg-card border-2 border-border rounded">
        <span class="text-[10px] text-muted uppercase font-bold block mb-1">Target ROM</span>
        <p class="font-bold text-foreground">AOSP, LineageOS, PixelOS, MIUI/HyperOS, OxygenOS (Non-OneUI)</p>
      </div>
      <div class="p-2.5 bg-card border-2 border-border rounded">
        <span class="text-[10px] text-muted uppercase font-bold block mb-1">Target Chipset</span>
        <p class="font-bold text-foreground">Qualcomm Snapdragon (All SoCs with Linux QRTR support)</p>
      </div>
      <div class="p-2.5 bg-card border-2 border-border rounded">
        <span class="text-[10px] text-muted uppercase font-bold block mb-1">Target Modem</span>
        <p class="font-bold text-foreground">Qualcomm MSS Baseband via Linux AF_QIPCRTR Socket</p>
      </div>
    </div>
  </div>

  <!-- SUB-TABS NAVIGATION (OPSI A) -->
  <div class="flex items-center gap-2 border-b-2 border-border pb-2 overflow-x-auto text-xs">
    {#each [
      { id: 'bands', label: 'Band Locking', icon: Radio },
      { id: 'cell_lock', label: 'Cell & Tower Lock', icon: Search },
      { id: 'profiles', label: 'Hardware & Profiles', icon: Sliders },
      { id: 'efs', label: 'EFS & NV Browser', icon: HardDrive },
      { id: 'nv', label: 'NV Manager', icon: Database },
    ] as tab}
      <button
        type="button"
        class="neo-button flex items-center gap-2 px-3.5 py-2 rounded-t font-bold transition-all cursor-pointer {subTab === tab.id
          ? 'border-2 border-b-0 border-accent bg-accent/15 text-accent shadow-neobrutal-sm'
          : 'border-2 border-transparent text-muted hover:text-foreground'}"
        onclick={() => (subTab = tab.id as typeof subTab)}
      >
        <tab.icon class="w-3.5 h-3.5" />
        <span>{tab.label}</span>
      </button>
    {/each}
  </div>

  <!-- SUB-TAB CONTENT -->
  {#if subTab === 'bands'}
    <QualcommBands
      bind:bandConfig
      {simStatus}
      onSlotChange={handleSlotChange}
      onReload={() => fetchBands(bandConfig.sim_slot)}
    />
  {:else if subTab === 'cell_lock'}
    <QualcommCellLock
      simSlot={bandConfig.sim_slot ?? 0}
      {simStatus}
    />
  {:else if subTab === 'profiles'}
    <QualcommProfiles
      simSlot={bandConfig.sim_slot ?? 0}
    />
  {:else if subTab === 'efs'}
    <QualcommEFS />
  {:else if subTab === 'nv'}
    <QualcommNV />
  {/if}
</div>
