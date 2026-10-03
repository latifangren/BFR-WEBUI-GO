<script lang="ts">
  import {
    Radio,
    Lock,
    RotateCcw,
    AlertTriangle,
    ShieldCheck,
    Check,
    ArrowLeftRight,
    CheckCircle,
    XCircle,
    Copy,
    Info,
  } from '@lucide/svelte'
  import { api, ApiError } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Modal from '../../ui/Modal.svelte'

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

  interface BandLockVerificationResult {
    success: boolean
    verified: boolean
    target_slot: number
    target_slot_name: string
    requested_lte: number[]
    active_lte_bands: number[]
    active_nr_bands: number[]
    message: string
    step_logs: string[]
  }

  let {
    bandConfig = $bindable(),
    simStatus = null,
    onSlotChange,
    onReload,
  }: {
    bandConfig: BandConfig
    simStatus: SimStatus | null
    onSlotChange: (slotIndex: number) => void
    onReload: () => void
  } = $props()

  let isApplying = $state(false)
  let isResetting = $state(false)
  let isSwitchingSim = $state(false)
  let showResetModal = $state(false)
  let verificationResult = $state<BandLockVerificationResult | null>(null)
  let showLogs = $state(true)

  const availableLTEBands = [
    { band: 1, label: 'B1', name: '2100 MHz FDD' },
    { band: 2, label: 'B2', name: '1900 MHz PCS' },
    { band: 3, label: 'B3', name: '1800 MHz FDD' },
    { band: 4, label: 'B4', name: '1700/2100 AWS' },
    { band: 5, label: 'B5', name: '850 MHz FDD' },
    { band: 7, label: 'B7', name: '2600 MHz FDD' },
    { band: 8, label: 'B8', name: '900 MHz FDD' },
    { band: 12, label: 'B12', name: '700 MHz Lower' },
    { band: 18, label: 'B18', name: '800 MHz Japan' },
    { band: 19, label: 'B19', name: '800 MHz Japan' },
    { band: 20, label: 'B20', name: '800 MHz EU' },
    { band: 26, label: 'B26', name: '850 MHz Ext' },
    { band: 28, label: 'B28', name: '700 MHz APT' },
    { band: 38, label: 'B38', name: '2600 MHz TDD' },
    { band: 39, label: 'B39', name: '1900 MHz TDD' },
    { band: 40, label: 'B40', name: '2300 MHz TDD' },
    { band: 41, label: 'B41', name: '2500 MHz TDD' },
    { band: 66, label: 'B66', name: '1700/2100 Ext' },
  ]

  const availableNRBands = [
    { band: 1, label: 'n1', name: '2100 MHz Sub-6' },
    { band: 3, label: 'n3', name: '1800 MHz Sub-6' },
    { band: 28, label: 'n28', name: '700 MHz Sub-6' },
    { band: 40, label: 'n40', name: '2300 MHz Sub-6' },
    { band: 77, label: 'n77', name: '3700 MHz C-Band' },
    { band: 78, label: 'n78', name: '3500 MHz Mid-Band' },
  ]

  async function switchDataSim(slotIndex: number) {
    try {
      isSwitchingSim = true
      const res = await api.post<{ success: boolean; message: string }>('/api/modem/sim/switch', {
        slot: slotIndex,
      })
      if (res && res.success) {
        toastStore.success(res.message || `Switched data SIM to slot ${slotIndex + 1}`)
        onSlotChange(slotIndex)
      } else {
        toastStore.error('Failed to switch data SIM')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'SIM switch failed')
    } finally {
      isSwitchingSim = false
    }
  }

  async function applyBandLock() {
    try {
      isApplying = true
      verificationResult = null

      const res = await api.post<{
        success: boolean
        error?: string
        config?: BandConfig
        result?: BandLockVerificationResult
      }>('/api/modem/bands', {
        engine: 'qualcomm_qrtr',
        preferred_rat: bandConfig.preferred_rat,
        lte_bands: bandConfig.lte_bands,
        nr_bands: bandConfig.nr_bands,
        hex_bitmask: bandConfig.hex_bitmask || '',
        sim_slot: bandConfig.sim_slot ?? 0,
        nr_mode: bandConfig.nr_mode || 'both',
      })

      if (res && res.result) {
        verificationResult = res.result
        if (res.result.verified) {
          toastStore.success('Band lock verified by modem baseband hardware!')
        } else if (res.result.success) {
          toastStore.warning(res.result.message || 'Band lock applied, but baseband reports different bands.')
        } else {
          toastStore.error(res.result.message || 'Modem baseband rejected band lock command.')
        }
      } else if (res && res.success) {
        toastStore.success('Qualcomm QRTR band lock applied directly to modem baseband.')
      }

      if (res?.config) {
        if (res.config.active_lte_bands) bandConfig.active_lte_bands = res.config.active_lte_bands
        if (res.config.active_nr_bands) bandConfig.active_nr_bands = res.config.active_nr_bands
      }
      onReload()
    } catch (err: unknown) {
      if (
        err instanceof ApiError &&
        err.data &&
        typeof err.data === 'object' &&
        'result' in err.data &&
        (err.data as { result?: BandLockVerificationResult }).result
      ) {
        verificationResult = (err.data as { result: BandLockVerificationResult }).result
      } else {
        const msg = err instanceof Error ? err.message : 'Band lock request failed'
        verificationResult = {
          success: false,
          verified: false,
          target_slot: bandConfig.sim_slot ?? 0,
          target_slot_name: `SIM ${(bandConfig.sim_slot ?? 0) + 1}`,
          requested_lte: bandConfig.lte_bands,
          active_lte_bands: [],
          active_nr_bands: [],
          message: msg,
          step_logs: ['Execution failed: ' + msg],
        }
      }
      toastStore.error(verificationResult?.message || (err instanceof Error ? err.message : 'Band lock request failed'))
    } finally {
      isApplying = false
    }
  }

  async function confirmReset() {
    try {
      isResetting = true
      verificationResult = null
      const res = await api.post<{ success: boolean; error?: string }>('/api/modem/reset')
      if (res && res.success) {
        toastStore.success('Modem restored to all factory default bands.')
        showResetModal = false
        onReload()
      } else {
        toastStore.error(res?.error || 'Failed to reset modem bands')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Reset request failed')
    } finally {
      isResetting = false
    }
  }

  function toggleLTEBand(b: number) {
    if (bandConfig.lte_bands.includes(b)) {
      bandConfig.lte_bands = bandConfig.lte_bands.filter((x) => x !== b)
    } else {
      bandConfig.lte_bands = [...bandConfig.lte_bands, b].sort((a, b) => a - b)
    }
  }

  function toggleNRBand(b: number) {
    if (bandConfig.nr_bands.includes(b)) {
      bandConfig.nr_bands = bandConfig.nr_bands.filter((x) => x !== b)
    } else {
      bandConfig.nr_bands = [...bandConfig.nr_bands, b].sort((a, b) => a - b)
    }
  }

  function selectAllLTE() {
    bandConfig.lte_bands = availableLTEBands.map((b) => b.band)
  }

  function clearAllLTE() {
    bandConfig.lte_bands = []
  }

  function copyLogs() {
    if (!verificationResult) return
    const text = [
      `Target: ${verificationResult.target_slot_name} (Slot ${verificationResult.target_slot})`,
      `Status: ${verificationResult.verified ? 'VERIFIED' : verificationResult.success ? 'PARTIAL' : 'FAILED'}`,
      `Message: ${verificationResult.message}`,
      `Requested LTE: [${verificationResult.requested_lte.join(', ')}]`,
      `Active LTE: [${verificationResult.active_lte_bands.join(', ')}]`,
      `--- Execution Steps ---`,
      ...verificationResult.step_logs.map((l, idx) => `[${idx + 1}] ${l}`),
    ].join('\n')

    if (navigator?.clipboard) {
      navigator.clipboard.writeText(text)
      toastStore.success('Verification logs copied to clipboard')
    }
  }

  const selectedSlotInfo = $derived(
    simStatus?.slots.find((s) => s.slot_index === (bandConfig.sim_slot ?? 0))
  )
</script>

<Card title="Frequency Band Configuration" subtitle="Modulate active cellular frequencies instantly without reboot" tone="mint">
  <div class="space-y-3 sm:space-y-6 text-xs">
    <!-- TARGET SIM SLOT & ACTIVE DATA STATUS -->
    <div class="space-y-3 p-3.5 bg-card-sub border-2 border-border rounded-lg">
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
        <div>
          <span class="text-xs font-bold uppercase tracking-wider text-foreground block">
            Multi-SIM Slot Selection & Active Data Route
          </span>
          <span class="text-[11px] text-muted">
            Choose the SIM slot you want to configure. Band locks apply to the target modem port directly.
          </span>
        </div>
        {#if simStatus}
          <Badge variant="default" class="text-[10px] w-fit">
            Active Data: Slot {(simStatus.active_slot ?? 0) + 1} (SubID {simStatus.active_sub_id})
          </Badge>
        {/if}
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-1">
        {#each (simStatus?.slots || [
          { slot_index: 0, sub_id: 1, state: 'READY', is_active: true, is_esim: false, carrier: 'Auto', label: 'SIM 1 (pSIM)' },
          { slot_index: 1, sub_id: 2, state: 'READY', is_active: false, is_esim: true, carrier: 'Auto', label: 'SIM 2 (eSIM)' }
        ]) as slot}
          {@const isSelected = (bandConfig.sim_slot ?? 0) === slot.slot_index}
          {@const isActiveData = simStatus ? simStatus.active_slot === slot.slot_index : slot.is_active}
          {@const isAbsent = slot.state === 'ABSENT'}
          <button
            type="button"
            class="p-3 rounded-lg border-2 text-left transition-all cursor-pointer flex flex-col justify-between gap-2.5 {isSelected
              ? 'border-accent bg-accent/15 text-foreground shadow-neobrutal-sm ring-1 ring-accent'
              : 'border-border bg-card text-muted hover:border-muted-foreground'}"
            onclick={() => onSlotChange(slot.slot_index)}
          >
            <div class="flex items-start justify-between gap-2">
              <div class="flex items-center gap-2.5">
                <div
                  class="w-7 h-7 rounded border-2 flex items-center justify-center font-bold text-xs {isSelected
                    ? 'border-accent bg-accent text-accent-text'
                    : 'border-border bg-card-sub text-foreground'}"
                >
                  {slot.slot_index + 1}
                </div>
                <div>
                  <div class="font-bold text-xs text-foreground flex items-center gap-1.5">
                    <span>SIM {slot.slot_index + 1}</span>
                    <span
                      class="px-1.5 py-0.2 text-[9px] font-bold rounded border uppercase {slot.is_esim
                        ? 'border-purple-600 bg-purple-950/40 text-purple-300'
                        : 'border-blue-600 bg-blue-950/40 text-blue-300'}"
                    >
                      {slot.is_esim ? 'eSIM' : 'pSIM'}
                    </span>
                  </div>
                  <span class="text-[10px] text-muted block">
                    {slot.carrier || (isAbsent ? 'No SIM Card Inserted' : 'Unknown Operator')}
                  </span>
                </div>
              </div>

              <div class="flex flex-col items-end gap-1">
                {#if isActiveData}
                  <Badge variant="success" class="text-[9px] flex items-center gap-1">
                    <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
                    <span>ACTIVE DATA</span>
                  </Badge>
                {:else if isAbsent}
                  <Badge variant="danger" class="text-[9px]">ABSENT</Badge>
                {:else}
                  <Badge variant="default" class="text-[9px]">STANDBY</Badge>
                {/if}
              </div>
            </div>

            <div class="flex items-center justify-between text-[10px] pt-2 border-t border-border/40">
              <span class="text-muted">
                SubID: <strong class="text-foreground">{slot.sub_id > 0 ? slot.sub_id : '--'}</strong>
              </span>
              <span class="font-bold {isSelected ? 'text-accent' : 'text-muted'}">
                {isSelected ? '✓ Selected for Band Lock' : 'Click to select'}
              </span>
            </div>
          </button>
        {/each}
      </div>

      {#if selectedSlotInfo}
        {#if selectedSlotInfo.state === 'ABSENT'}
          <div class="p-2.5 rounded bg-red-950/30 border-2 border-red-800 text-red-300 text-xs flex items-center gap-2">
            <AlertTriangle class="w-4 h-4 shrink-0 text-red-400" />
            <span>
              <strong>Warning:</strong> Slot {(bandConfig.sim_slot ?? 0) + 1} has no physical or eSIM inserted (ABSENT). Band locks applied here will not affect cellular data.
            </span>
          </div>
        {:else if simStatus && simStatus.active_slot !== (bandConfig.sim_slot ?? 0)}
          <div class="p-2.5 rounded bg-amber-950/30 border-2 border-amber-800 text-amber-300 text-xs flex flex-col sm:flex-row sm:items-center justify-between gap-2">
            <div class="flex items-center gap-2">
              <Info class="w-4 h-4 shrink-0 text-amber-400" />
              <span>
                <strong>Notice:</strong> Mobile Data is currently routing through <strong>SIM {simStatus.active_slot + 1}</strong>. You are configuring <strong>SIM {(bandConfig.sim_slot ?? 0) + 1}</strong>.
              </span>
            </div>
            <Button
              variant="outline"
              size="sm"
              disabled={isSwitchingSim}
              onclick={() => switchDataSim(bandConfig.sim_slot ?? 0)}
              class="shrink-0 text-[11px]"
            >
              <ArrowLeftRight class="w-3 h-3 mr-1" />
              <span>Switch Data to SIM {(bandConfig.sim_slot ?? 0) + 1}</span>
            </Button>
          </div>
        {/if}
      {/if}
    </div>

    <!-- PREFERRED NETWORK MODE (RAT) -->
    <div class="space-y-2 p-3 bg-card-sub border-2 border-border rounded-lg">
      <span class="text-[10px] text-muted uppercase font-bold block">Preferred Network Technology (RAT)</span>
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
        {#each [
          { id: 'hybrid', label: 'Hybrid (Auto)', badge: '5G/4G/3G' },
          { id: '5g_only', label: '5G NR Only', badge: 'SA/NSA' },
          { id: '4g_only', label: '4G LTE Only', badge: 'LTE-A' },
          { id: '3g_only', label: '3G Only', badge: 'WCDMA' }
        ] as rat}
          <button
            type="button"
            class="neo-button p-2 text-center rounded border-2 transition-all cursor-pointer {bandConfig.preferred_rat === rat.id ? 'border-accent bg-accent/15 text-accent font-bold shadow-neobrutal-sm' : 'border-border bg-card text-muted hover:text-foreground'}"
            onclick={() => (bandConfig.preferred_rat = rat.id as BandConfig['preferred_rat'])}
          >
            <div class="font-bold text-xs">{rat.label}</div>
            <div class="text-[9px] text-muted">{rat.badge}</div>
          </button>
        {/each}
      </div>
    </div>

    <!-- HARDWARE ACTIVE BANDS BANNER -->
    {#if bandConfig.active_lte_bands && bandConfig.active_lte_bands.length > 0}
      <div class="p-3 rounded-lg border-2 border-emerald-500/40 bg-emerald-950/20 flex flex-wrap items-center justify-between gap-3 shadow-neobrutal-sm">
        <div class="flex items-center gap-2.5">
          <ShieldCheck class="w-5 h-5 text-emerald-400 shrink-0" />
          <div>
            <div class="text-xs font-bold text-foreground">
              Baseband Active LTE Bands (SIM {(bandConfig.sim_slot ?? 0) + 1}):
            </div>
            <div class="text-xs font-mono text-emerald-300 font-bold mt-0.5">
              {bandConfig.active_lte_bands.map(b => 'B' + b).join(', ')}
            </div>
          </div>
        </div>
        <button
          type="button"
          class="px-2.5 py-1 text-[11px] font-bold border-2 border-emerald-500/60 rounded bg-emerald-500/20 text-emerald-300 hover:bg-emerald-500/30 cursor-pointer"
          onclick={() => {
            if (bandConfig.active_lte_bands) {
              bandConfig.lte_bands = [...bandConfig.active_lte_bands]
            }
          }}
        >
          Sync Selection to Active
        </button>
      </div>
    {/if}

    <!-- GROUND TRUTH VERIFICATION & EXECUTION CARD -->
    {#if verificationResult}
      <div class="p-4 rounded-lg border-2 space-y-3 shadow-neobrutal-sm transition-all {verificationResult.verified
        ? 'border-emerald-500 bg-emerald-950/20'
        : verificationResult.success
          ? 'border-amber-500 bg-amber-950/20'
          : 'border-red-500 bg-red-950/25'}">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b-2 border-border/50 pb-2">
          <div class="flex items-center gap-2">
            {#if verificationResult.verified}
              <CheckCircle class="w-5 h-5 text-emerald-400 shrink-0" />
              <span class="font-bold text-sm text-emerald-300 uppercase">
                Hardware Verified (Ground Truth Confirmed)
              </span>
            {:else if verificationResult.success}
              <AlertTriangle class="w-5 h-5 text-amber-400 shrink-0" />
              <span class="font-bold text-sm text-amber-300 uppercase">
                Partial Acceptance / Baseband Notice
              </span>
            {:else}
              <XCircle class="w-5 h-5 text-red-400 shrink-0" />
              <span class="font-bold text-sm text-red-300 uppercase">
                Command Rejected by Modem Baseband
              </span>
            {/if}
          </div>

          <div class="flex items-center gap-2">
            <Button variant="ghost" size="sm" onclick={copyLogs} class="text-[11px]">
              <Copy class="w-3.5 h-3.5 mr-1" />
              <span>Copy Logs</span>
            </Button>
            <button
              type="button"
              class="text-xs text-muted hover:text-foreground cursor-pointer px-1.5 py-0.5"
              onclick={() => (verificationResult = null)}
            >
              ✕ Dismiss
            </button>
          </div>
        </div>

        <p class="text-xs font-bold {verificationResult.verified ? 'text-emerald-200' : verificationResult.success ? 'text-amber-200' : 'text-red-200'}">
          {verificationResult.message}
        </p>

        <div class="grid grid-cols-1 sm:grid-cols-3 gap-2 text-[11px]">
          <div class="p-2 bg-card border rounded">
            <span class="text-muted block text-[10px]">Target SIM Slot</span>
            <strong class="text-foreground">{verificationResult.target_slot_name} (Slot {verificationResult.target_slot})</strong>
          </div>
          <div class="p-2 bg-card border rounded">
            <span class="text-muted block text-[10px]">Requested LTE Bands</span>
            <strong class="text-foreground">
              {verificationResult.requested_lte.length > 0 ? verificationResult.requested_lte.map(b => 'B' + b).join(', ') : 'None'}
            </strong>
          </div>
          <div class="p-2 bg-card border rounded">
            <span class="text-muted block text-[10px]">Baseband Active LTE</span>
            <strong class="{verificationResult.verified ? 'text-emerald-400' : 'text-amber-400'}">
              {verificationResult.active_lte_bands.length > 0 ? verificationResult.active_lte_bands.map(b => 'B' + b).join(', ') : 'None / Not Reported'}
            </strong>
          </div>
        </div>

        <div class="space-y-1">
          <div class="flex items-center justify-between text-[10px] text-muted">
            <span>Step-by-Step Baseband Transaction Logs</span>
            <button
              type="button"
              class="hover:underline cursor-pointer"
              onclick={() => (showLogs = !showLogs)}
            >
              {showLogs ? 'Hide details' : 'Show details'}
            </button>
          </div>
          {#if showLogs}
            <div class="p-2.5 bg-black/80 rounded border border-border text-[11px] font-mono text-zinc-300 space-y-1 max-h-40 overflow-y-auto">
              {#each verificationResult.step_logs as log, idx}
                <div class="flex items-start gap-2">
                  <span class="text-zinc-500 shrink-0">[{idx + 1}]</span>
                  <span class="{log.includes('rejected') || log.includes('failed') ? 'text-red-400 font-bold' : log.includes('accepted') || log.includes('PASSED') ? 'text-emerald-300' : 'text-zinc-300'}">
                    {log}
                  </span>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      </div>
    {/if}

    <!-- LTE BANDS CHECKLIST -->
    <div class="space-y-3">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <span class="font-bold text-sm text-foreground uppercase tracking-wider">LTE 4G Frequency Bands</span>
          <Badge variant="default">{bandConfig.lte_bands.length} Bands Selected</Badge>
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
          {@const isChecked = bandConfig.lte_bands.includes(b.band)}
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

    <!-- 5G NR BANDS CHECKLIST -->
    <div class="space-y-3 pt-4 border-t-2 border-border">
      <div class="flex items-center gap-2">
        <span class="font-bold text-sm text-foreground uppercase tracking-wider">5G NR Frequency Bands</span>
        <Badge variant="info">{bandConfig.nr_bands.length} Bands Selected</Badge>
      </div>

      <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-6 gap-2.5">
        {#each availableNRBands as b}
          {@const isChecked = bandConfig.nr_bands.includes(b.band)}
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

    <!-- HEX BITMASK & ACTION BAR -->
    <div class="pt-4 border-t-2 border-border flex flex-col sm:flex-row items-center justify-between gap-3">
      <div class="flex items-center gap-2 w-full sm:w-auto">
        <span class="text-[11px] text-muted whitespace-nowrap">Hex Bitmask Override:</span>
        <input
          type="text"
          placeholder="Auto-calculated"
          bind:value={bandConfig.hex_bitmask}
          class="neo-input w-full sm:w-64 bg-card-sub border-2 border-border rounded px-2.5 py-1 text-xs font-mono text-foreground focus:outline-none focus:border-accent"
        />
      </div>

      <div class="flex items-center gap-2 w-full sm:w-auto justify-end">
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
          onclick={applyBandLock}
        >
          <Lock class="w-3.5 h-3.5 mr-1 text-accent-text" />
          <span>{isApplying ? 'Applying & Verifying...' : 'Lock Selected Bands'}</span>
        </Button>
      </div>
    </div>
  </div>
</Card>

<Modal
  open={showResetModal}
  title="Reset Snapdragon Modem Bands"
  onclose={() => (showResetModal = false)}
>
  <div class="space-y-3 text-xs">
    <div class="flex items-start gap-3 p-3 bg-red-950/30 border-2 border-red-800 rounded text-red-200">
      <AlertTriangle class="w-5 h-5 text-red-400 shrink-0 mt-0.5" />
      <div>
        <p class="font-bold text-red-400">Restore All Qualcomm Factory Bands?</p>
        <p class="text-[11px] mt-1 text-red-300/80">
          This will wipe all active frequency band lock overrides and restore Qualcomm baseband modem to full hardware-supported bands.
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
