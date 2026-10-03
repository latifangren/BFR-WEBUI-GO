<script lang="ts">
  import {
    Cpu,
    ShieldCheck,
    Globe,
    Lock,
    Unlock,
    RefreshCw,
    Sliders,
    Info,
    Check,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'

  interface HardwareBandsInfo {
    lte_bands: number[]
    nr_bands: number[]
    gsm_bands?: number[]
    wcdma_bands?: number[]
  }

  interface PlmnLockState {
    mode: 'auto' | 'manual'
    mcc: number
    mnc: number
    carrier_name?: string
  }

  let {
    simSlot = 0,
  }: {
    simSlot: number
  } = $props()

  let isLoadingHw = $state(false)
  let isLockingPlmn = $state(false)
  let isClearingPlmn = $state(false)

  let hwBands = $state<HardwareBandsInfo>({
    lte_bands: [1, 2, 3, 4, 5, 7, 8, 12, 13, 14, 17, 18, 19, 20, 25, 26, 28, 29, 30, 32, 38, 39, 40, 41, 42, 46, 48, 66, 71],
    nr_bands: [1, 2, 3, 5, 7, 8, 12, 20, 25, 28, 38, 40, 41, 66, 71, 77, 78],
  })

  let plmnState = $state<PlmnLockState>({
    mode: 'auto',
    mcc: 510,
    mnc: 11,
    carrier_name: 'Auto Selection',
  })

  let manualMcc = $state('510')
  let manualMnc = $state('11')

  const popularIndoCarriers = [
    { name: 'XL Axiata', mcc: '510', mnc: '11' },
    { name: 'Telkomsel', mcc: '510', mnc: '10' },
    { name: 'Indosat Ooredoo', mcc: '510', mnc: '01' },
    { name: 'Smartfren', mcc: '510', mnc: '28' },
    { name: 'Tri (3)', mcc: '510', mnc: '89' },
  ]

  async function fetchHardwareBands() {
    try {
      isLoadingHw = true
      const res = await api.get<HardwareBandsInfo>(`/api/modem/hardware_bands?slot=${simSlot}`)
      if (res && res.lte_bands && res.lte_bands.length > 0) {
        hwBands = res
      }
    } catch {
      // Fallback
    } finally {
      isLoadingHw = false
    }
  }

  async function applyPlmnLock(mccStr: string, mncStr: string) {
    const mccNum = parseInt(mccStr, 10)
    const mncNum = parseInt(mncStr, 10)
    if (isNaN(mccNum) || isNaN(mncNum)) {
      toastStore.error('Please enter valid numeric MCC and MNC values')
      return
    }

    try {
      isLockingPlmn = true
      const res = await api.post<{ success: boolean; message?: string }>('/api/modem/plmn_lock', {
        slot: simSlot,
        mode: 'manual',
        mcc: mccNum,
        mnc: mncNum,
      })
      if (res && res.success) {
        toastStore.success(`PLMN locked to MCC ${mccNum}, MNC ${mncNum}`)
        plmnState.mode = 'manual'
        plmnState.mcc = mccNum
        plmnState.mnc = mncNum
      } else {
        toastStore.error(res?.message || 'Modem rejected PLMN lock')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'PLMN lock request failed')
    } finally {
      isLockingPlmn = false
    }
  }

  async function clearPlmnLock() {
    try {
      isClearingPlmn = true
      const res = await api.post<{ success: boolean; message?: string }>('/api/modem/plmn_lock', {
        slot: simSlot,
        mode: 'auto',
      })
      if (res && res.success) {
        toastStore.success('Network selection restored to Automatic (PLMN lock cleared).')
        plmnState.mode = 'auto'
      } else {
        toastStore.error(res?.message || 'Failed to clear PLMN lock')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Clear PLMN request failed')
    } finally {
      isClearingPlmn = false
    }
  }
</script>

<div class="space-y-3 sm:space-y-6 font-mono text-xs">
  <!-- HARDWARE RF CAPABILITIES CARD -->
  <Card title="Hardware RF Physical Capabilities" subtitle="Exact radio frequency bands physically wired to the modem baseband transceiver" tone="mint">
    {#snippet action()}
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoadingHw}
        onclick={fetchHardwareBands}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoadingHw ? 'animate-spin' : ''}" />
        <span>Refresh Capabilities</span>
      </Button>
    {/snippet}

    <div class="space-y-4">
      <div class="p-3 bg-card-sub border-2 border-border rounded flex items-center justify-between">
        <span class="text-muted">Target Interface:</span>
        <strong class="text-foreground">SIM {simSlot + 1} Baseband Transceiver</strong>
      </div>

      <!-- LTE Hardware Bands Grid -->
      <div class="space-y-2">
        <div class="flex items-center justify-between">
          <span class="font-bold text-foreground uppercase tracking-wider">
            Physically Supported LTE Bands ({hwBands.lte_bands.length})
          </span>
          <Badge variant="success">FACTORY SUPPORTED</Badge>
        </div>
        <div class="flex flex-wrap gap-1.5 p-3 bg-card-sub border-2 border-border rounded">
          {#each hwBands.lte_bands as b}
            <span class="px-2 py-0.5 rounded bg-emerald-950/40 text-emerald-300 border border-emerald-600 font-bold text-[11px]">
              B{b}
            </span>
          {/each}
        </div>
      </div>

      <!-- 5G NR Hardware Bands Grid -->
      <div class="space-y-2 pt-2 border-t border-border">
        <div class="flex items-center justify-between">
          <span class="font-bold text-foreground uppercase tracking-wider">
            Physically Supported 5G NR Bands ({hwBands.nr_bands.length})
          </span>
          <Badge variant="info">SUB-6 & C-BAND</Badge>
        </div>
        <div class="flex flex-wrap gap-1.5 p-3 bg-card-sub border-2 border-border rounded">
          {#each hwBands.nr_bands as b}
            <span class="px-2 py-0.5 rounded bg-cyan-950/40 text-cyan-300 border border-cyan-600 font-bold text-[11px]">
              n{b}
            </span>
          {/each}
        </div>
      </div>
    </div>
  </Card>

  <!-- MANUAL NETWORK / PLMN LOCK CARD -->
  <Card title="Manual PLMN & Operator Network Selection" subtitle="Pin modem to specific Public Land Mobile Network (MCC/MNC)" tone="mint">
    <div class="space-y-4">
      <div class="p-3 bg-card-sub border-2 border-border rounded flex flex-col sm:flex-row sm:items-center justify-between gap-2">
        <div class="flex items-center gap-2">
          <Globe class="w-4 h-4 text-accent" />
          <span class="text-muted">Network Selection State:</span>
          <strong class="text-foreground uppercase">{plmnState.mode}</strong>
        </div>
        {#if plmnState.mode === 'manual'}
          <Badge variant="warning">MANUAL LOCKED (MCC {plmnState.mcc} / MNC {plmnState.mnc})</Badge>
        {:else}
          <Badge variant="success">AUTOMATIC SELECTION</Badge>
        {/if}
      </div>

      <!-- Quick Preset Operators -->
      <div class="space-y-1.5">
        <span class="text-[10px] text-muted uppercase font-bold block">Quick Indonesian Carrier Presets</span>
        <div class="grid grid-cols-2 sm:grid-cols-5 gap-2">
          {#each popularIndoCarriers as c}
            <button
              type="button"
              class="p-2 rounded border-2 border-border bg-card-sub hover:border-accent text-left transition-all cursor-pointer"
              onclick={() => {
                manualMcc = c.mcc
                manualMnc = c.mnc
              }}
            >
              <div class="font-bold text-foreground text-xs">{c.name}</div>
              <div class="text-[10px] text-muted font-mono">{c.mcc}-{c.mnc}</div>
            </button>
          {/each}
        </div>
      </div>

      <!-- Manual Inputs -->
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 p-3 bg-card-sub border-2 border-border rounded-lg">
        <div>
          <label for="plmn-mcc-input" class="text-[10px] text-muted uppercase font-bold block mb-1">
            Mobile Country Code (MCC)
          </label>
          <input
            id="plmn-mcc-input"
            type="number"
            placeholder="e.g. 510"
            bind:value={manualMcc}
            class="neo-input w-full bg-card border-2 border-border rounded px-2.5 py-1 text-xs font-mono text-foreground focus:outline-none focus:border-accent"
          />
        </div>
        <div>
          <label for="plmn-mnc-input" class="text-[10px] text-muted uppercase font-bold block mb-1">
            Mobile Network Code (MNC)
          </label>
          <input
            id="plmn-mnc-input"
            type="number"
            placeholder="e.g. 11 (XL) or 10 (Telkomsel)"
            bind:value={manualMnc}
            class="neo-input w-full bg-card border-2 border-border rounded px-2.5 py-1 text-xs font-mono text-foreground focus:outline-none focus:border-accent"
          />
        </div>
      </div>

      <!-- Actions -->
      <div class="flex items-center justify-between gap-3 pt-2">
        <Button
          variant="outline"
          size="sm"
          disabled={isClearingPlmn || isLockingPlmn || plmnState.mode === 'auto'}
          onclick={clearPlmnLock}
        >
          <Unlock class="w-3.5 h-3.5 mr-1" />
          <span>Restore Auto Selection</span>
        </Button>
        <Button
          variant="primary"
          size="sm"
          disabled={isLockingPlmn || isClearingPlmn || !manualMcc || !manualMnc}
          onclick={() => applyPlmnLock(manualMcc, manualMnc)}
        >
          <Lock class="w-3.5 h-3.5 mr-1 text-accent-text" />
          <span>{isLockingPlmn ? 'Locking PLMN...' : 'Lock Operator PLMN'}</span>
        </Button>
      </div>
    </div>
  </Card>
</div>
