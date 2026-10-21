<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Zap,
    RefreshCw,
    Save,
    AlertTriangle,
    Info,
    Radio,
    Layers,
    Cpu,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Modal from '../../ui/Modal.svelte'

  interface CapabilityFileStatus {
    path: string
    present: boolean
    readable: boolean
    write_only: boolean
    size?: number
    description: string
  }

  interface LTESectionState {
    lte_ca: boolean
    ca_filter: string
    ul_ca: boolean
    dl_256q: boolean
    ul_256q: boolean
    dl_mimo: boolean
    max_mimo_layer: number
    horxd: boolean
    rx0: boolean
    rx1: boolean
    rx2: boolean
    rx3: boolean
    volte: boolean
  }

  interface NSASectionState {
    nsa_lte: boolean
    nsa_lte_dsda: boolean
    nsa_ndds: boolean
    nsa_lte_dss: boolean
  }

  interface NRSectionState {
    nr_ca: boolean
    dl_256q: boolean
    ul_256q: boolean
    dl_mimo: boolean
    ul_mimo: boolean
    vonr: boolean
    redcap: boolean
    nr_nr_dsda: boolean
    limit_rf_mimo: boolean
  }

  interface CapabilityState {
    token_string: string
    lte: LTESectionState
    nsa: NSASectionState
    nr: NRSectionState
    files?: CapabilityFileStatus[]
  }

  let isLoading = $state(false)
  let isSaving = $state(false)
  let restartSSR = $state(false)
  let showFilesTable = $state(false)
  let showConfirmModal = $state(false)

  let capState = $state<CapabilityState>({
    token_string: '',
    lte: {
      lte_ca: true,
      ca_filter: '',
      ul_ca: false,
      dl_256q: true,
      ul_256q: false,
      dl_mimo: true,
      max_mimo_layer: 4,
      horxd: true,
      rx0: true,
      rx1: true,
      rx2: false,
      rx3: false,
      volte: true,
    },
    nsa: {
      nsa_lte: true,
      nsa_lte_dsda: false,
      nsa_ndds: false,
      nsa_lte_dss: true,
    },
    nr: {
      nr_ca: true,
      dl_256q: true,
      ul_256q: false,
      dl_mimo: true,
      ul_mimo: false,
      vonr: true,
      redcap: false,
      nr_nr_dsda: false,
      limit_rf_mimo: false,
    },
    files: [],
  })

  // Live Token String computed from local state
  let liveTokens = $derived.by(() => {
    const tokens: string[] = []
    if (capState.lte.lte_ca) tokens.push('[LTE_CA]')
    if (capState.lte.ca_filter) tokens.push('[LTE_CA_FILTER]')
    if (capState.lte.ul_ca) tokens.push('[LTE_UL_CA]')
    if (capState.lte.dl_256q) tokens.push('[LTE_DL_256Q]')
    if (capState.lte.ul_256q) tokens.push('[LTE_UL_256Q]')
    if (capState.lte.dl_mimo) tokens.push('[LTE_DL_MIMO]')
    if (capState.lte.horxd) tokens.push('[LTE_HORXD]')
    if (capState.lte.rx0) tokens.push('[LTE_RX0]')
    if (capState.lte.rx1) tokens.push('[LTE_RX1]')
    if (capState.lte.rx2) tokens.push('[LTE_RX2]')
    if (capState.lte.rx3) tokens.push('[LTE_RX3]')
    if (capState.lte.volte) tokens.push('[LTE_VOICE]')

    if (capState.nsa.nsa_lte) tokens.push('[NSA_LTE]')
    if (capState.nsa.nsa_lte_dss) tokens.push('[NSA_LTE_DSS]')
    if (capState.nsa.nsa_ndds) tokens.push('[NSA_NDDS]')
    if (capState.nsa.nsa_lte_dsda) tokens.push('[NSA_LTE_DSDA]')

    if (capState.nr.nr_ca) tokens.push('[NR_CA]')
    if (capState.nr.dl_256q) tokens.push('[NR_DL_256Q]')
    if (capState.nr.ul_256q) tokens.push('[NR_UL_256Q]')
    if (capState.nr.dl_mimo) tokens.push('[NR_DL_MIMO]')
    if (capState.nr.ul_mimo) tokens.push('[NR_UL_MIMO]')
    if (capState.nr.redcap) tokens.push('[NR_REDCAP]')
    if (capState.nr.vonr) tokens.push('[NR_VOICE]')
    if (capState.nr.nr_nr_dsda) tokens.push('[NR_NR_DSDA]')
    if (capState.nr.limit_rf_mimo) tokens.push('[NR_LIMIT_RF_MIMO]')

    return tokens.join('')
  })

  onMount(async () => {
    await loadCapability()
  })

  async function loadCapability() {
    isLoading = true
    try {
      const res = await api.get<{ ok: boolean; capability?: CapabilityState; files?: CapabilityFileStatus[]; error?: string }>('/api/efs/capability')
      if (res && res.ok && res.capability) {
        capState = res.capability
        if (res.files) {
          capState.files = res.files
        }
        toastStore.success('RRC capabilities loaded from modem EFS.')
      } else {
        toastStore.error(res?.error || 'Failed to read capabilities')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Read error')
    } finally {
      isLoading = false
    }
  }

  function promptApply() {
    showConfirmModal = true
  }

  async function confirmApply() {
    showConfirmModal = false
    isSaving = true
    try {
      const res = await api.post<{ ok: boolean; token_string?: string; ssr_triggered?: boolean; error?: string }>('/api/efs/capability/apply', {
        state: capState,
        backup: true,
        verify: true,
        ssr: restartSSR,
      })

      if (res && res.ok) {
        toastStore.success(`RRC capabilities applied successfully${res.ssr_triggered ? ' (SSR restart triggered)' : ''}!`)
        await loadCapability()
      } else {
        toastStore.error(res?.error || 'Failed to apply RRC capabilities')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Apply error')
    } finally {
      isSaving = false
    }
  }
</script>

<div class="space-y-6">
  <!-- HEADER & TOKEN DISPLAY -->
  <Card class="p-4 border-2 border-border shadow-neobrutal-sm bg-card space-y-3">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b-2 border-border/60 pb-3">
      <div class="flex items-center gap-2">
        <Zap class="w-5 h-5 text-accent" />
        <h3 class="text-sm font-black uppercase tracking-wider text-foreground">
          Qualcomm RRC & PHY Capabilities Editor
        </h3>
      </div>
      <div class="flex items-center gap-2">
        <Button variant="outline" size="sm" onclick={loadCapability} disabled={isLoading} class="gap-1.5">
          <RefreshCw class="w-3.5 h-3.5 {isLoading ? 'animate-spin' : ''}" />
          <span>Reload</span>
        </Button>
      </div>
    </div>

    <!-- LIVE TOKEN STRING RECONSTRUCTION -->
    <div class="space-y-1">
      <div class="flex items-center justify-between">
        <span class="text-[10px] text-muted uppercase font-bold tracking-wider">Reconstructed Token String (ModemCaps Format):</span>
        <span class="text-[10px] font-mono text-muted">{liveTokens.length} chars</span>
      </div>
      <div class="p-2.5 bg-card-sub border-2 border-border rounded font-mono text-xs text-foreground break-all select-all">
        {liveTokens || '[NO_ACTIVE_CAPABILITY_TOKENS]'}
      </div>
    </div>

    <!-- WRITE-ONLY FOOTER NOTICE (FROM MODEMCAPS FOOTER) -->
    <div class="p-2.5 bg-blue-500/10 border border-blue-500/30 rounded text-xs text-blue-900 dark:text-blue-200 flex items-start gap-2">
      <Info class="w-4 h-4 text-blue-700 dark:text-blue-400 shrink-0 mt-0.5" />
      <p class="leading-relaxed">
        <strong>Write-Only Advisory:</strong> Certain features (LTE CA, UL CA, DL 256-QAM, Rx chain, ARD) are designated <em>write-only</em> by modem firmware. Changes will be securely written to EFS and consumed on the next modem restart, but reading back live state depends on hardware capability.
      </p>
    </div>
  </Card>

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <!-- 1. LTE RRC SECTION -->
    <Card class="p-4 border-2 border-border shadow-neobrutal-sm bg-card space-y-4">
      <div class="flex items-center justify-between border-b-2 border-border/60 pb-2">
        <div class="flex items-center gap-2">
          <Radio class="w-4 h-4 text-accent" />
          <span class="text-xs font-black uppercase tracking-wider text-foreground">LTE Capabilities</span>
        </div>
        <Badge variant="default">12 Tokens</Badge>
      </div>

      <div class="space-y-3 text-xs">
        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">LTE Carrier Aggregation</span>
            <span class="text-[10px] text-muted font-mono">[LTE_CA]</span>
          </div>
          <div class="flex items-center gap-2">
            <span class="text-[10px] px-1 py-0.5 bg-amber-500/20 text-amber-700 dark:text-amber-300 rounded font-mono">Write-Only</span>
            <input type="checkbox" bind:checked={capState.lte.lte_ca} class="w-4 h-4 accent-accent" />
          </div>
        </label>

        <!-- CA FILTER -->
        <div class="p-2 bg-card-sub border border-border rounded space-y-1.5">
          <div class="flex items-center justify-between">
            <span class="font-bold text-foreground">Filter Band-Combo</span>
            <span class="text-[10px] text-muted font-mono">[LTE_CA_FILTER]</span>
          </div>
          <input
            type="text"
            placeholder="e.g. 1A-3A-7A; 3C-7A[0]"
            bind:value={capState.lte.ca_filter}
            class="w-full px-2 py-1 bg-card border-2 border-border rounded text-foreground font-mono text-xs"
          />
          <span class="text-[10px] text-muted block">Format: Band+Class+[Streams][UL]-...[BW Set]</span>
        </div>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">Uplink Carrier Aggregation</span>
            <span class="text-[10px] text-muted font-mono">[LTE_UL_CA]</span>
          </div>
          <input type="checkbox" bind:checked={capState.lte.ul_ca} class="w-4 h-4 accent-accent" />
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">Downlink 256-QAM</span>
            <span class="text-[10px] text-muted font-mono">[LTE_DL_256Q]</span>
          </div>
          <input type="checkbox" bind:checked={capState.lte.dl_256q} class="w-4 h-4 accent-accent" />
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">Uplink 256-QAM</span>
            <span class="text-[10px] text-muted font-mono">[LTE_UL_256Q]</span>
          </div>
          <input type="checkbox" bind:checked={capState.lte.ul_256q} class="w-4 h-4 accent-accent" />
        </label>

        <div class="p-2 bg-card-sub border border-border rounded space-y-2">
          <div class="flex items-center justify-between">
            <span class="font-bold text-foreground">Downlink MIMO Layers</span>
            <span class="text-[10px] text-muted font-mono">[LTE_DL_MIMO]</span>
          </div>
          <div class="flex items-center gap-4">
            <label class="flex items-center gap-1.5 cursor-pointer">
              <input type="radio" name="mimo" value={2} bind:group={capState.lte.max_mimo_layer} class="accent-accent" />
              <span>2 Layers (2x2)</span>
            </label>
            <label class="flex items-center gap-1.5 cursor-pointer">
              <input type="radio" name="mimo" value={4} bind:group={capState.lte.max_mimo_layer} class="accent-accent" />
              <span>4 Layers (4x4)</span>
            </label>
          </div>
        </div>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">Adaptive Rx Diversity (ARD)</span>
            <span class="text-[10px] text-muted font-mono">[LTE_HORXD]</span>
          </div>
          <input type="checkbox" bind:checked={capState.lte.horxd} class="w-4 h-4 accent-accent" />
        </label>

        <div class="p-2 bg-card-sub border border-border rounded space-y-1.5">
          <span class="font-bold text-foreground block">Rx Chains [LTE_RX0..3]</span>
          <div class="grid grid-cols-4 gap-2">
            <label class="flex items-center gap-1 cursor-pointer">
              <input type="checkbox" bind:checked={capState.lte.rx0} class="accent-accent" />
              <span>#0</span>
            </label>
            <label class="flex items-center gap-1 cursor-pointer">
              <input type="checkbox" bind:checked={capState.lte.rx1} class="accent-accent" />
              <span>#1</span>
            </label>
            <label class="flex items-center gap-1 cursor-pointer">
              <input type="checkbox" bind:checked={capState.lte.rx2} class="accent-accent" />
              <span>#2</span>
            </label>
            <label class="flex items-center gap-1 cursor-pointer">
              <input type="checkbox" bind:checked={capState.lte.rx3} class="accent-accent" />
              <span>#3</span>
            </label>
          </div>
        </div>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">Voice over LTE (VoLTE)</span>
            <span class="text-[10px] text-muted font-mono">[LTE_VOICE]</span>
          </div>
          <input type="checkbox" bind:checked={capState.lte.volte} class="w-4 h-4 accent-accent" />
        </label>
      </div>
    </Card>

    <!-- 2. NSA SECTION -->
    <Card class="p-4 border-2 border-border shadow-neobrutal-sm bg-card space-y-4">
      <div class="flex items-center justify-between border-b-2 border-border/60 pb-2">
        <div class="flex items-center gap-2">
          <Layers class="w-4 h-4 text-accent" />
          <span class="text-xs font-black uppercase tracking-wider text-foreground">NSA (EN-DC) Capabilities</span>
        </div>
        <Badge variant="default">4 Tokens</Badge>
      </div>

      <div class="space-y-3 text-xs">
        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">LTE + 5G NSA (EN-DC)</span>
            <span class="text-[10px] text-muted font-mono">[NSA_LTE]</span>
          </div>
          <input type="checkbox" bind:checked={capState.nsa.nsa_lte} class="w-4 h-4 accent-accent" />
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">Dynamic Spectrum Sharing (DSS)</span>
            <span class="text-[10px] text-muted font-mono">[NSA_LTE_DSS]</span>
          </div>
          <input type="checkbox" bind:checked={capState.nsa.nsa_lte_dss} class="w-4 h-4 accent-accent" />
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">NSA during Data-Call on Other SIM</span>
            <span class="text-[10px] text-muted font-mono">[NSA_NDDS]</span>
          </div>
          <input type="checkbox" bind:checked={capState.nsa.nsa_ndds} class="w-4 h-4 accent-accent" />
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">NSA Dual SIM Dual Active (DSDA)</span>
            <span class="text-[10px] text-muted font-mono">[NSA_LTE_DSDA]</span>
          </div>
          <input type="checkbox" bind:checked={capState.nsa.nsa_lte_dsda} class="w-4 h-4 accent-accent" />
        </label>
      </div>
    </Card>

    <!-- 3. NR (5G SA/NSA) SECTION -->
    <Card class="p-4 border-2 border-border shadow-neobrutal-sm bg-card space-y-4">
      <div class="flex items-center justify-between border-b-2 border-border/60 pb-2">
        <div class="flex items-center gap-2">
          <Cpu class="w-4 h-4 text-accent" />
          <span class="text-xs font-black uppercase tracking-wider text-foreground">NR (5G SA) Capabilities</span>
        </div>
        <Badge variant="default">9 Tokens</Badge>
      </div>

      <div class="space-y-3 text-xs">
        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">NR Carrier Aggregation (NR CA)</span>
            <span class="text-[10px] text-muted font-mono">[NR_CA]</span>
          </div>
          <input type="checkbox" bind:checked={capState.nr.nr_ca} class="w-4 h-4 accent-accent" />
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">NR Downlink 256-QAM</span>
            <span class="text-[10px] text-muted font-mono">[NR_DL_256Q]</span>
          </div>
          <input type="checkbox" bind:checked={capState.nr.dl_256q} class="w-4 h-4 accent-accent" />
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">NR Uplink 256-QAM</span>
            <span class="text-[10px] text-muted font-mono">[NR_UL_256Q]</span>
          </div>
          <input type="checkbox" bind:checked={capState.nr.ul_256q} class="w-4 h-4 accent-accent" />
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">NR Downlink MIMO</span>
            <span class="text-[10px] text-muted font-mono">[NR_DL_MIMO]</span>
          </div>
          <input type="checkbox" bind:checked={capState.nr.dl_mimo} class="w-4 h-4 accent-accent" />
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">NR Uplink MIMO</span>
            <span class="text-[10px] text-muted font-mono">[NR_UL_MIMO]</span>
          </div>
          <input type="checkbox" bind:checked={capState.nr.ul_mimo} class="w-4 h-4 accent-accent" />
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">Voice over NR (VoNR)</span>
            <span class="text-[10px] text-muted font-mono">[NR_VOICE]</span>
          </div>
          <input type="checkbox" bind:checked={capState.nr.vonr} class="w-4 h-4 accent-accent" />
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">NR RedCap (Reduced Capability)</span>
            <span class="text-[10px] text-muted font-mono">[NR_REDCAP]</span>
          </div>
          <input type="checkbox" bind:checked={capState.nr.redcap} class="w-4 h-4 accent-accent" />
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">NR Dual SIM Dual Active (DSDA)</span>
            <span class="text-[10px] text-muted font-mono">[NR_NR_DSDA]</span>
          </div>
          <input type="checkbox" bind:checked={capState.nr.nr_nr_dsda} class="w-4 h-4 accent-accent" />
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">Limit RF MIMO</span>
            <span class="text-[10px] text-muted font-mono">[NR_LIMIT_RF_MIMO]</span>
          </div>
          <input type="checkbox" bind:checked={capState.nr.limit_rf_mimo} class="w-4 h-4 accent-accent" />
        </label>
      </div>
    </Card>
  </div>

  <!-- ACTION CONTROLS & MODEM EFS FILES STATUS -->
  <Card class="p-4 border-2 border-border shadow-neobrutal-sm bg-card space-y-4">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <label class="flex items-center gap-2 cursor-pointer text-xs font-bold text-foreground">
          <input type="checkbox" bind:checked={restartSSR} class="w-4 h-4 accent-accent" />
          <span>Restart Modem (SSR) after write</span>
        </label>
        <button
          type="button"
          class="text-xs text-accent hover:underline font-bold cursor-pointer"
          onclick={() => (showFilesTable = !showFilesTable)}
        >
          {showFilesTable ? 'Hide File Mapping' : 'Show EFS File Mapping'}
        </button>
      </div>

      <Button variant="primary" onclick={promptApply} disabled={isSaving} class="gap-2">
        <Save class="w-4 h-4" />
        <span>Apply Capabilities to Modem</span>
      </Button>
    </div>

    <!-- EFS FILES STATUS TABLE -->
    {#if showFilesTable && capState.files && capState.files.length > 0}
      <div class="pt-3 border-t-2 border-border/60 space-y-2">
        <span class="text-[10px] text-muted uppercase font-bold tracking-wider block">Detected EFS Files on this Baseband:</span>
        <div class="overflow-x-auto">
          <table class="w-full text-left text-xs font-mono">
            <thead>
              <tr class="border-b border-border/60 text-muted">
                <th class="py-1.5 px-2">Path</th>
                <th class="py-1.5 px-2">Description</th>
                <th class="py-1.5 px-2">Status</th>
                <th class="py-1.5 px-2">Size</th>
              </tr>
            </thead>
            <tbody>
              {#each capState.files as f}
                <tr class="border-b border-border/30 hover:bg-card-sub">
                  <td class="py-1.5 px-2 text-foreground">{f.path}</td>
                  <td class="py-1.5 px-2 text-muted">{f.description}</td>
                  <td class="py-1.5 px-2">
                    {#if f.present}
                      <span class="text-emerald-400 font-bold">PRESENT</span>
                    {:else}
                      <span class="text-red-400">ABSENT</span>
                    {/if}
                    {#if f.write_only}
                      <span class="ml-1 text-[9px] px-1 bg-amber-500/20 text-amber-700 dark:text-amber-300 rounded">WRITE-ONLY</span>
                    {/if}
                  </td>
                  <td class="py-1.5 px-2 text-muted">{f.size !== undefined ? `${f.size} B` : '-'}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    {/if}
  </Card>
</div>

<!-- CONFIRMATION MODAL -->
<Modal bind:open={showConfirmModal} title="Confirm RRC Capabilities Write" size="md">
  {#snippet children()}
    <div class="space-y-3 text-xs text-foreground">
      <div class="p-3 bg-amber-500/10 border border-amber-500/30 rounded flex items-start gap-2">
        <AlertTriangle class="w-4 h-4 text-amber-600 dark:text-amber-400 shrink-0 mt-0.5" />
        <p class="leading-relaxed text-amber-900 dark:text-amber-200">
          This operation will write the configured capability state to modem EFS item files. The existing files will be automatically backed up before overwriting.
        </p>
      </div>

      <div class="space-y-1">
        <span class="text-[10px] uppercase font-bold text-muted block">Reconstructed Token Payload:</span>
        <div class="p-2.5 bg-card-sub border-2 border-border rounded font-mono break-all text-xs">
          {liveTokens || '[NO_TOKENS]'}
        </div>
      </div>

      <label class="flex items-center gap-2 cursor-pointer font-bold pt-2 border-t border-border/60">
        <input type="checkbox" bind:checked={restartSSR} class="w-4 h-4 accent-accent" />
        <span>Perform Subsystem Restart (SSR) immediately after write</span>
      </label>
    </div>
  {/snippet}

  {#snippet footer()}
    <div class="flex items-center justify-end gap-2">
      <Button variant="outline" size="sm" onclick={() => (showConfirmModal = false)}>
        Cancel
      </Button>
      <Button variant="primary" size="sm" onclick={confirmApply} disabled={isSaving}>
        {isSaving ? 'Writing to EFS...' : 'Confirm & Apply to EFS'}
      </Button>
    </div>
  {/snippet}
</Modal>
