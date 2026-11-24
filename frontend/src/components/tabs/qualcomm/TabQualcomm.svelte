<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Zap,
    RefreshCw,
    Radio,
    Search,
    Filter,
    Layers,
    Sliders,
    AlertTriangle,
    Cpu,
    HardDrive,
    Database,
    Smartphone,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import QualcommBands from './QualcommBands.svelte'
  import QualcommCellLock from './QualcommCellLock.svelte'
  import QualcommProfiles from './QualcommProfiles.svelte'
  import QualcommSearcher from './QualcommSearcher.svelte'
  import QualcommCapability from './QualcommCapability.svelte'
  import QualcommMBN from './QualcommMBN.svelte'
  import QualcommEFS from './QualcommEFS.svelte'
  import QualcommNV from './QualcommNV.svelte'
  import QualcommIMEI from './QualcommIMEI.svelte'
  import { themeStore } from '../../../stores/theme.svelte'
  import SakugaTabs from '../../ui/SakugaTabs.svelte'
  import { LL } from '../../../i18n/i18n-svelte'
  interface BandConfig {
    engine: 'qualcomm_qrtr' | 'universal' | 'qualcomm_at' | 'intent'
    preferred_rat: 'hybrid' | '5g_only' | '4g_only' | '5g_4g' | '3g_only'
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
  let subTab = $state<'bands' | 'cell_lock' | 'searcher' | 'capability' | 'mbn' | 'profiles' | 'efs' | 'nv' | 'imei'>('bands')
  const SUBTAB_IDS: Array<typeof subTab> = ['bands', 'cell_lock', 'searcher', 'capability', 'mbn', 'profiles', 'efs', 'nv', 'imei']
  const SUBTAB_LABELS = $derived([$LL.qualcomm.subtabs.bands(), $LL.qualcomm.subtabs.cellLock(), $LL.qualcomm.subtabs.searcher(), $LL.qualcomm.subtabs.capability(), $LL.qualcomm.subtabs.mbn(), $LL.qualcomm.subtabs.profiles(), $LL.qualcomm.subtabs.efs(), $LL.qualcomm.subtabs.nv(), $LL.qualcomm.subtabs.imei()])
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

<div class="space-y-3 sm:space-y-6 pb-12 font-mono">
  <!-- TOP HERO & REFRESH -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b-2 border-border pb-4">
    <div>
      <div class="flex items-center gap-2">
        <Zap class="w-6 h-6 text-accent" />
        <h1 class="text-xl sm:text-2xl font-black uppercase tracking-wider text-foreground">
          {$LL.qualcomm.title()}
        </h1>
        <Badge variant="success" class="text-[10px]">AF_QIPCRTR</Badge>
      </div>
      <p class="text-xs text-muted mt-1">
        {$LL.qualcomm.subtitle()}
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
          {$LL.modem.compat.title()}
        </span>
      </div>
      {#if bandConfig.is_qualcomm}
        <Badge variant="success" class="flex items-center gap-1.5 w-fit">
          <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
          <span>{$LL.qualcomm.compatible()}</span>
        </Badge>
      {:else}
        <Badge variant="warning" class="flex items-center gap-1.5 w-fit">
          <AlertTriangle class="w-3.5 h-3.5" />
          <span>{$LL.qualcomm.nonQualcomm()}</span>
        </Badge>
      {/if}
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 text-xs">
      <div class="p-2.5 bg-card border-2 border-border rounded">
        <span class="text-[10px] text-muted uppercase font-bold block mb-1">{$LL.modem.compat.targetDevice()}</span>
        <p class="font-bold text-foreground">Google Pixel 1-5, Xiaomi, POCO, OnePlus, Sony, Moto, ASUS</p>
      </div>
      <div class="p-2.5 bg-card border-2 border-border rounded">
        <span class="text-[10px] text-muted uppercase font-bold block mb-1">{$LL.modem.compat.targetRom()}</span>
        <p class="font-bold text-foreground">AOSP, LineageOS, PixelOS, MIUI/HyperOS, OxygenOS (Non-OneUI)</p>
      </div>
      <div class="p-2.5 bg-card border-2 border-border rounded">
        <span class="text-[10px] text-muted uppercase font-bold block mb-1">{$LL.modem.compat.targetChipset()}</span>
        <p class="font-bold text-foreground">Qualcomm Snapdragon (All SoCs with Linux QRTR support)</p>
      </div>
      <div class="p-2.5 bg-card border-2 border-border rounded">
        <span class="text-[10px] text-muted uppercase font-bold block mb-1">{$LL.modem.compat.targetModem()}</span>
        <p class="font-bold text-foreground">Qualcomm MSS Baseband via Linux AF_QIPCRTR Socket</p>
      </div>
    </div>
  </div>

  <!-- SUB-TABS NAVIGATION (OPSI A) -->
{#if themeStore.useSakugaControls()}
    <SakugaTabs
      selected={SUBTAB_IDS.indexOf(subTab)}
      options={SUBTAB_LABELS}
      onchange={(e) => { subTab = SUBTAB_IDS[(e.target as any).selected] }}
    />
  {:else}
  <div class="flex items-center gap-2 border-b-2 border-border pb-2 overflow-x-auto text-xs">
    {#each [
      { id: 'bands', label: $LL.qualcomm.subtabs.bands(), icon: Radio },
      { id: 'cell_lock', label: $LL.qualcomm.subtabs.cellLock(), icon: Search },
      { id: 'searcher', label: $LL.qualcomm.subtabs.searcher(), icon: Filter },
      { id: 'capability', label: $LL.qualcomm.subtabs.capability(), icon: Zap },
      { id: 'mbn', label: $LL.qualcomm.subtabs.mbn(), icon: Layers },
      { id: 'profiles', label: $LL.qualcomm.subtabs.profiles(), icon: Sliders },
      { id: 'efs', label: $LL.qualcomm.subtabs.efs(), icon: HardDrive },
      { id: 'nv', label: $LL.qualcomm.subtabs.nv(), icon: Database },
      { id: 'imei', label: $LL.qualcomm.subtabs.imei(), icon: Smartphone },
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
  {/if}

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
  {:else if subTab === 'searcher'}
    <QualcommSearcher />
  {:else if subTab === 'capability'}
    <QualcommCapability />
  {:else if subTab === 'mbn'}
    <QualcommMBN />
  {:else if subTab === 'profiles'}
    <QualcommProfiles
      simSlot={bandConfig.sim_slot ?? 0}
    />
  {:else if subTab === 'efs'}
    <QualcommEFS />
  {:else if subTab === 'nv'}
    <QualcommNV />
  {:else if subTab === 'imei'}
    <QualcommIMEI />
  {/if}
</div>
