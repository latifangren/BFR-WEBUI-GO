<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Layers,
    RefreshCw,
    Lock,
    RotateCcw,
    AlertTriangle,
    ShieldCheck,
    Check,
    Cpu,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Modal from '../../ui/Modal.svelte'

  interface ShannonStatus {
    is_shannon: boolean
    device_model: string
    soc: string
    router_available: boolean
    dm_available: boolean
    active_slot: number
    mode: string // 'both' | 'sa' | 'nsa' | 'lte_only'
    active_lte_bands: number[]
    active_nr_bands: number[]
  }

  let status = $state<ShannonStatus>({
    is_shannon: false,
    device_model: '',
    soc: '',
    router_available: false,
    dm_available: false,
    active_slot: 0,
    mode: 'both',
    active_lte_bands: [],
    active_nr_bands: [],
  })

  let selectedSlot = $state<number>(0)
  let selectedMode = $state<string>('both')
  let selectedLTEBands = $state<number[]>([1, 3, 8])
  let selectedNRBands = $state<number[]>([1, 3, 28, 40, 77, 78])

  let isLoading = $state(false)
  let isApplying = $state(false)
  let isResetting = $state(false)
  let showResetModal = $state(false)

  // Pixel Tensor Supported LTE Bands
  const availableLTEBands = [
    { band: 1, label: 'B1', name: '2100 MHz FDD' },
    { band: 2, label: 'B2', name: '1900 MHz PCS' },
    { band: 3, label: 'B3', name: '1800 MHz FDD' },
    { band: 4, label: 'B4', name: '1700/2100 AWS' },
    { band: 5, label: 'B5', name: '850 MHz FDD' },
    { band: 7, label: 'B7', name: '2600 MHz FDD' },
    { band: 8, label: 'B8', name: '900 MHz FDD' },
    { band: 12, label: 'B12', name: '700 MHz Lower' },
    { band: 13, label: 'B13', name: '700 MHz Upper' },
    { band: 14, label: 'B14', name: '700 MHz FirstNet' },
    { band: 17, label: 'B17', name: '700 MHz Lower' },
    { band: 18, label: 'B18', name: '800 MHz Japan' },
    { band: 19, label: 'B19', name: '800 MHz Japan' },
    { band: 20, label: 'B20', name: '800 MHz EU' },
    { band: 25, label: 'B25', name: '1900 MHz Ext' },
    { band: 26, label: 'B26', name: '850 MHz Ext' },
    { band: 28, label: 'B28', name: '700 MHz APT' },
    { band: 29, label: 'B29', name: '700 MHz SDL' },
    { band: 30, label: 'B30', name: '2300 MHz WCS' },
    { band: 38, label: 'B38', name: '2600 MHz TDD' },
    { band: 39, label: 'B39', name: '1900 MHz TDD' },
    { band: 40, label: 'B40', name: '2300 MHz TDD' },
    { band: 41, label: 'B41', name: '2500 MHz TDD' },
    { band: 46, label: 'B46', name: '5200 MHz LAA' },
    { band: 48, label: 'B48', name: '3600 MHz CBRS' },
    { band: 66, label: 'B66', name: '1700/2100 Ext' },
    { band: 71, label: 'B71', name: '600 MHz US' },
  ]

  // Pixel Tensor Supported 5G NR Bands
  const availableNRBands = [
    { band: 1, label: 'n1', name: '2100 MHz Sub-6' },
    { band: 2, label: 'n2', name: '1900 MHz PCS' },
    { band: 3, label: 'n3', name: '1800 MHz Sub-6' },
    { band: 5, label: 'n5', name: '850 MHz Sub-6' },
    { band: 7, label: 'n7', name: '2600 MHz Sub-6' },
    { band: 8, label: 'n8', name: '900 MHz Sub-6' },
    { band: 12, label: 'n12', name: '700 MHz Sub-6' },
    { band: 20, label: 'n20', name: '800 MHz Sub-6' },
    { band: 25, label: 'n25', name: '1900 MHz Sub-6' },
    { band: 28, label: 'n28', name: '700 MHz Sub-6' },
    { band: 38, label: 'n38', name: '2600 MHz Sub-6' },
    { band: 40, label: 'n40', name: '2300 MHz Sub-6' },
    { band: 41, label: 'n41', name: '2500 MHz Sub-6' },
    { band: 48, label: 'n48', name: '3600 MHz CBRS' },
    { band: 66, label: 'n66', name: '1700/2100 Sub-6' },
    { band: 71, label: 'n71', name: '600 MHz Sub-6' },
    { band: 77, label: 'n77', name: '3700 MHz C-Band' },
    { band: 78, label: 'n78', name: '3500 MHz Mid-Band' },
  ]

  onMount(() => {
    fetchStatus()
  })

  async function fetchStatus() {
    try {
      isLoading = true
      const res = await api.get<ShannonStatus>('/api/shannon/status')
      if (res) {
        status = res
        selectedSlot = res.active_slot ?? 0
        selectedMode = res.mode || 'both'
        if (Array.isArray(res.active_lte_bands) && res.active_lte_bands.length > 0) {
          selectedLTEBands = [...res.active_lte_bands]
        }
        if (Array.isArray(res.active_nr_bands) && res.active_nr_bands.length > 0) {
          selectedNRBands = [...res.active_nr_bands]
        }
      }
    } catch {
      // Fallback
    } finally {
      isLoading = false
    }
  }

  async function applyLock() {
    try {
      isApplying = true
      const res = await api.post<{ success: boolean; error?: string }>('/api/shannon/lock', {
        slot: selectedSlot,
        mode: selectedMode,
        lte_bands: selectedLTEBands,
        nr_bands: selectedNRBands,
      })
      if (res && res.success) {
        toastStore.success('Google Tensor Shannon band lock applied via /dev/umts_router.')
        await fetchStatus()
      } else {
        toastStore.error(res?.error || 'Failed to apply Shannon band lock')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Band lock request failed')
    } finally {
      isApplying = false
    }
  }

  async function confirmReset() {
    try {
      isResetting = true
      const res = await api.post<{ success: boolean; error?: string }>('/api/shannon/reset', {
        slot: selectedSlot,
      })
      if (res && res.success) {
        toastStore.success('Shannon modem restored to factory default bands.')
        showResetModal = false
        await fetchStatus()
      } else {
        toastStore.error(res?.error || 'Failed to reset Shannon bands')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Reset request failed')
    } finally {
      isResetting = false
    }
  }

  function toggleLTEBand(b: number) {
    if (selectedLTEBands.includes(b)) {
      selectedLTEBands = selectedLTEBands.filter((x) => x !== b)
    } else {
      selectedLTEBands = [...selectedLTEBands, b].sort((a, b) => a - b)
    }
  }

  function toggleNRBand(b: number) {
    if (selectedNRBands.includes(b)) {
      selectedNRBands = selectedNRBands.filter((x) => x !== b)
    } else {
      selectedNRBands = [...selectedNRBands, b].sort((a, b) => a - b)
    }
  }

  function selectAllLTE() {
    selectedLTEBands = availableLTEBands.map((b) => b.band)
  }

  function clearAllLTE() {
    selectedLTEBands = []
  }
</script>

<div class="space-y-6 pb-12 font-mono">
  <!-- TOP HERO & REFRESH -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b-2 border-border pb-4">
    <div>
      <div class="flex items-center gap-2">
        <Layers class="w-6 h-6 text-accent" />
        <h1 class="text-xl sm:text-2xl font-black uppercase tracking-wider text-foreground">
          Google Tensor Shannon Band Locking
        </h1>
        <Badge variant="info" class="text-[10px]">Shannon</Badge>
      </div>
      <p class="text-xs text-muted mt-1">
        Direct Samsung Shannon modem AT/NV modulation via /dev/umts_router and /dev/umts_dm0 kernel interfaces.
      </p>
    </div>

    <div class="flex items-center gap-2">
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoading || isApplying}
        onclick={fetchStatus}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoading ? 'animate-spin' : ''}" />
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
      {#if status.is_shannon || status.router_available}
        <Badge variant="success" class="flex items-center gap-1.5 w-fit">
          <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
          <span>COMPATIBLE & DETECTED (GOOGLE TENSOR)</span>
        </Badge>
      {:else}
        <Badge variant="warning" class="flex items-center gap-1.5 w-fit">
          <AlertTriangle class="w-3.5 h-3.5" />
          <span>INCOMPATIBLE (REQUIRES GOOGLE TENSOR PIXEL 6+)</span>
        </Badge>
      {/if}
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 text-xs">
      <div class="p-2.5 bg-card border-2 border-border rounded">
        <span class="text-[10px] text-muted uppercase font-bold block mb-1">Target Device</span>
        <p class="font-bold text-foreground">Google Pixel 6/7/8/9, Pro, a-Series, Pixel Fold</p>
      </div>
      <div class="p-2.5 bg-card border-2 border-border rounded">
        <span class="text-[10px] text-muted uppercase font-bold block mb-1">Target ROM</span>
        <p class="font-bold text-foreground">Google Pixel Stock Android (12–15+), GrapheneOS, AOSP</p>
      </div>
      <div class="p-2.5 bg-card border-2 border-border rounded">
        <span class="text-[10px] text-muted uppercase font-bold block mb-1">Target Chipset</span>
        <p class="font-bold text-foreground">Google Tensor (Tensor G1, G2, G3, G4)</p>
      </div>
      <div class="p-2.5 bg-card border-2 border-border rounded">
        <span class="text-[10px] text-muted uppercase font-bold block mb-1">Target Modem</span>
        <p class="font-bold text-foreground">Samsung Shannon Baseband via /dev/umts_router</p>
      </div>
    </div>
  </div>

  <!-- CONTROL CARD -->
  <Card title="Tensor Frequency Band Configuration" subtitle="Modulate Google Tensor baseband NV parameters directly" tone="ice">
    <div class="space-y-6 text-xs">
      <!-- Target SIM & 5G Mode -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <!-- Target SIM Slot -->
        <div class="space-y-2 p-3 bg-card-sub border-2 border-border rounded">
          <div class="flex items-center justify-between">
            <span class="text-[10px] text-muted uppercase font-bold block">Target SIM Slot</span>
            <span class="text-[10px] text-accent font-bold">SIM {selectedSlot + 1}</span>
          </div>
          <div class="grid grid-cols-2 gap-2">
            {#each [
              { slot: 0, label: 'SIM 1 (Slot 0)', desc: 'Primary' },
              { slot: 1, label: 'SIM 2 (Slot 1)', desc: 'Secondary' }
            ] as sim}
              <button
                type="button"
                class="neo-button p-2 text-center rounded border-2 transition-all cursor-pointer {selectedSlot === sim.slot ? 'border-accent bg-accent/15 text-accent font-bold shadow-neobrutal-sm' : 'border-border bg-card text-muted hover:text-foreground'}"
                onclick={() => (selectedSlot = sim.slot)}
              >
                <div class="font-bold text-xs">{sim.label}</div>
                <div class="text-[9px] text-muted">{sim.desc}</div>
              </button>
            {/each}
          </div>
        </div>

        <!-- 5G NR Operation Mode -->
        <div class="space-y-2 p-3 bg-card-sub border-2 border-border rounded">
          <span class="text-[10px] text-muted uppercase font-bold block">5G NR Mode Configuration</span>
          <div class="grid grid-cols-2 gap-2">
            {#each [
              { id: 'both', label: 'SA + NSA', badge: 'All 5G' },
              { id: 'sa', label: 'SA Only', badge: 'Standalone' },
              { id: 'nsa', label: 'NSA Only', badge: 'Non-Standalone' },
              { id: 'lte_only', label: 'LTE Only', badge: '5G Off' }
            ] as m}
              <button
                type="button"
                class="neo-button p-2 text-center rounded border-2 transition-all cursor-pointer {selectedMode === m.id ? 'border-accent bg-accent/15 text-accent font-bold shadow-neobrutal-sm' : 'border-border bg-card text-muted hover:text-foreground'}"
                onclick={() => (selectedMode = m.id)}
              >
                <div class="font-bold text-xs">{m.label}</div>
                <div class="text-[9px] text-muted">{m.badge}</div>
              </button>
            {/each}
          </div>
        </div>
      </div>

      <!-- LTE Bands Checklist -->
      <div class="space-y-3">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <span class="font-bold text-sm text-foreground uppercase tracking-wider">LTE 4G Frequency Bands</span>
            <Badge variant="default">{selectedLTEBands.length} Bands Selected</Badge>
          </div>
          <div class="flex items-center gap-1.5">
            <button
              type="button"
              class="px-2 py-0.5 text-[10px] font-bold border-2 border-border rounded bg-card-sub hover:border-accent text-muted hover:text-foreground cursor-pointer"
              onclick={selectAllLTE}
            >
              Select All
            </button>
            <button
              type="button"
              class="px-2 py-0.5 text-[10px] font-bold border-2 border-border rounded bg-card-sub hover:border-red-500 text-muted hover:text-red-400 cursor-pointer"
              onclick={clearAllLTE}
            >
              Clear All
            </button>
          </div>
        </div>

        <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-6 gap-2.5">
          {#each availableLTEBands as b}
            {@const isChecked = selectedLTEBands.includes(b.band)}
            <button
              type="button"
              class="p-2.5 rounded-md border-2 transition-all text-left cursor-pointer flex flex-col justify-between {isChecked ? 'border-accent bg-accent/10 text-foreground shadow-neobrutal-sm' : 'border-border bg-card-sub text-muted hover:border-muted-foreground'}"
              onclick={() => toggleLTEBand(b.band)}
            >
              <div class="flex items-center justify-between">
                <span class="font-black text-sm {isChecked ? 'text-accent' : 'text-muted'}">{b.label}</span>
                <span class="w-4 h-4 rounded border flex items-center justify-center {isChecked ? 'bg-accent border-black text-black' : 'border-border bg-card'}">
                  {#if isChecked}
                    <Check class="w-3 h-3 stroke-[3]" />
                  {/if}
                </span>
              </div>
              <div class="text-[10px] text-muted mt-1 leading-tight">{b.name}</div>
            </button>
          {/each}
        </div>
      </div>

      <!-- 5G NR Bands Checklist -->
      <div class="space-y-3 pt-4 border-t-2 border-border">
        <div class="flex items-center gap-2">
          <span class="font-bold text-sm text-foreground uppercase tracking-wider">5G NR Frequency Bands</span>
          <Badge variant="info">{selectedNRBands.length} Bands Selected</Badge>
        </div>

        <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-6 gap-2.5">
          {#each availableNRBands as b}
            {@const isChecked = selectedNRBands.includes(b.band)}
            <button
              type="button"
              class="p-2.5 rounded-md border-2 transition-all text-left cursor-pointer flex flex-col justify-between {isChecked ? 'border-cyan-500 bg-cyan-950/20 text-foreground shadow-neobrutal-sm' : 'border-border bg-card-sub text-muted hover:border-muted-foreground'}"
              onclick={() => toggleNRBand(b.band)}
            >
              <div class="flex items-center justify-between">
                <span class="font-black text-sm {isChecked ? 'text-cyan-400' : 'text-muted'}">{b.label}</span>
                <span class="w-4 h-4 rounded border flex items-center justify-center {isChecked ? 'bg-cyan-500 border-black text-black' : 'border-border bg-card'}">
                  {#if isChecked}
                    <Check class="w-3 h-3 stroke-[3]" />
                  {/if}
                </span>
              </div>
              <div class="text-[10px] text-muted mt-1 leading-tight">{b.name}</div>
            </button>
          {/each}
        </div>
      </div>

      <!-- Action Bar -->
      <div class="pt-4 border-t-2 border-border flex items-center justify-end gap-3">
        <Button
          variant="outline"
          size="sm"
          disabled={isResetting || isApplying}
          onclick={() => (showResetModal = true)}
        >
          <RotateCcw class="w-3.5 h-3.5 mr-1" />
          <span>Reset Bands</span>
        </Button>
        <Button
          variant="primary"
          size="sm"
          disabled={isApplying}
          onclick={applyLock}
        >
          <Lock class="w-3.5 h-3.5 mr-1 text-accent-text" />
          <span>{isApplying ? 'Applying...' : 'Apply Shannon Lock'}</span>
        </Button>
      </div>
    </div>
  </Card>
</div>

<!-- RESET CONFIRMATION MODAL -->
<Modal
  open={showResetModal}
  title="Reset Shannon Modem Bands"
  onclose={() => (showResetModal = false)}
>
  <div class="space-y-3 text-xs">
    <div class="flex items-start gap-3 p-3 bg-red-950/30 border-2 border-red-800 rounded text-red-200">
      <AlertTriangle class="w-5 h-5 text-red-400 shrink-0 mt-0.5" />
      <div>
        <p class="font-bold text-red-400">Restore All Shannon Factory Bands?</p>
        <p class="text-[11px] mt-1 text-red-300/80">
          This will wipe Google Tensor manual band lock overrides and restore Shannon baseband modem to full hardware-supported bands.
        </p>
      </div>
    </div>
  </div>

  {#snippet footer()}
    <Button
      variant="outline"
      size="sm"
      disabled={isResetting}
      onclick={() => (showResetModal = false)}
    >
      Cancel
    </Button>
    <Button
      variant="danger"
      size="sm"
      disabled={isResetting}
      onclick={confirmReset}
    >
      <RotateCcw class="w-3.5 h-3.5 mr-1" />
      <span>{isResetting ? 'Resetting...' : 'Reset Modem Now'}</span>
    </Button>
  {/snippet}
</Modal>
