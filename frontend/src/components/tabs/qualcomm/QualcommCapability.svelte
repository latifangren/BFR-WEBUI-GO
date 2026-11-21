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
  import { LL } from '../../../i18n/i18n-svelte'
  import { themeStore } from '../../../stores/theme.svelte'
  import SakugaCheckbox from '../../ui/SakugaCheckbox.svelte'
  import SakugaRadio from '../../ui/SakugaRadio.svelte'

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
        toastStore.success($LL.qualcomm.capability.loaded())
      } else {
        toastStore.error(res?.error || $LL.qualcomm.capability.loadFailed())
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.qualcomm.capability.readError())
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
        toastStore.success(res.ssr_triggered ? $LL.qualcomm.capability.appliedSsr() : $LL.qualcomm.capability.applied())
        await loadCapability()
      } else {
        toastStore.error(res?.error || $LL.qualcomm.capability.applyFailed())
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.qualcomm.capability.applyError())
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
          {$LL.qualcomm.capability.title()}
        </h3>
      </div>
      <div class="flex items-center gap-2">
        <Button variant="outline" size="sm" onclick={loadCapability} disabled={isLoading} class="gap-1.5">
          <RefreshCw class="w-3.5 h-3.5 {isLoading ? 'animate-spin' : ''}" />
          <span>{$LL.common.reload()}</span>
        </Button>
      </div>
    </div>

    <!-- LIVE TOKEN STRING RECONSTRUCTION -->
    <div class="space-y-1">
      <div class="flex items-center justify-between">
        <span class="text-[10px] text-muted uppercase font-bold tracking-wider">{$LL.qualcomm.capability.tokenLabel()}</span>
        <span class="text-[10px] font-mono text-muted">{$LL.qualcomm.capability.chars({ count: liveTokens.length })}</span>
      </div>
      <div class="p-2.5 bg-card-sub border-2 border-border rounded font-mono text-xs text-foreground break-all select-all">
        {liveTokens || '[NO_ACTIVE_CAPABILITY_TOKENS]'}
      </div>
    </div>

    <!-- WRITE-ONLY FOOTER NOTICE (FROM MODEMCAPS FOOTER) -->
    <div class="p-2.5 bg-blue-500/10 border border-blue-500/30 rounded text-xs text-blue-900 dark:text-blue-200 flex items-start gap-2">
      <Info class="w-4 h-4 text-blue-700 dark:text-blue-400 shrink-0 mt-0.5" />
      <p class="leading-relaxed">
        <strong>{$LL.qualcomm.capability.writeOnlyAdvisory()}</strong> {$LL.qualcomm.capability.writeOnlyBodyA()} <em>write-only</em> {$LL.qualcomm.capability.writeOnlyBodyB()}
      </p>
    </div>
  </Card>

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <!-- 1. LTE RRC SECTION -->
    <Card class="p-4 border-2 border-border shadow-neobrutal-sm bg-card space-y-4">
      <div class="flex items-center justify-between border-b-2 border-border/60 pb-2">
        <div class="flex items-center gap-2">
          <Radio class="w-4 h-4 text-accent" />
          <span class="text-xs font-black uppercase tracking-wider text-foreground">{$LL.qualcomm.capability.lteTitle()}</span>
        </div>
        <Badge variant="default">{$LL.qualcomm.capability.tokenCount({ count: 12 })}</Badge>
      </div>

      <div class="space-y-3 text-xs">
        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">{$LL.qualcomm.capability.lteCa()}</span>
            <span class="text-[10px] text-muted font-mono">[LTE_CA]</span>
          </div>
          <div class="flex items-center gap-2">
            <span class="text-[10px] px-1 py-0.5 bg-amber-500/20 text-amber-700 dark:text-amber-300 rounded font-mono">{$LL.qualcomm.capability.writeOnly()}</span>
            {#if themeStore.useSakugaControls()}
              <SakugaCheckbox bind:checked={capState.lte.lte_ca} />
            {:else}
            <input type="checkbox" bind:checked={capState.lte.lte_ca} class="w-4 h-4 accent-accent" />
            {/if}
          </div>
        </label>

        <!-- CA FILTER -->
        <div class="p-2 bg-card-sub border border-border rounded space-y-1.5">
          <div class="flex items-center justify-between">
            <span class="font-bold text-foreground">{$LL.qualcomm.capability.caFilter()}</span>
            <span class="text-[10px] text-muted font-mono">[LTE_CA_FILTER]</span>
          </div>
          <input
            type="text"
            placeholder={$LL.qualcomm.capability.caFilterPlaceholder()}
            bind:value={capState.lte.ca_filter}
            class="w-full px-2 py-1 bg-card border-2 border-border rounded text-foreground font-mono text-xs"
          />
          <span class="text-[10px] text-muted block">{$LL.qualcomm.capability.caFilterFormat()}</span>
        </div>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">{$LL.qualcomm.capability.ulCa()}</span>
            <span class="text-[10px] text-muted font-mono">[LTE_UL_CA]</span>
          </div>
          {#if themeStore.useSakugaControls()}
            <SakugaCheckbox bind:checked={capState.lte.ul_ca} />
          {:else}
          <input type="checkbox" bind:checked={capState.lte.ul_ca} class="w-4 h-4 accent-accent" />
          {/if}
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">{$LL.qualcomm.capability.dl256q()}</span>
            <span class="text-[10px] text-muted font-mono">[LTE_DL_256Q]</span>
          </div>
          {#if themeStore.useSakugaControls()}
            <SakugaCheckbox bind:checked={capState.lte.dl_256q} />
          {:else}
          <input type="checkbox" bind:checked={capState.lte.dl_256q} class="w-4 h-4 accent-accent" />
          {/if}
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">{$LL.qualcomm.capability.ul256q()}</span>
            <span class="text-[10px] text-muted font-mono">[LTE_UL_256Q]</span>
          </div>
          {#if themeStore.useSakugaControls()}
            <SakugaCheckbox bind:checked={capState.lte.ul_256q} />
          {:else}
          <input type="checkbox" bind:checked={capState.lte.ul_256q} class="w-4 h-4 accent-accent" />
          {/if}
        </label>

        <div class="p-2 bg-card-sub border border-border rounded space-y-2">
          <div class="flex items-center justify-between">
            <span class="font-bold text-foreground">{$LL.qualcomm.capability.dlMimo()}</span>
            <span class="text-[10px] text-muted font-mono">[LTE_DL_MIMO]</span>
          </div>
{#if themeStore.useSakugaControls()}
            <SakugaRadio value={String(capState.lte.max_mimo_layer)} label="Downlink MIMO Layers"
              options={[{ value: '2', label: $LL.qualcomm.capability.mimo2() }, { value: '4', label: $LL.qualcomm.capability.mimo4() }]}
              onchange={(e) => { capState.lte.max_mimo_layer = Number((e.target as any).value) }} />
          {:else}
          <div class="flex items-center gap-4">
            <label class="flex items-center gap-1.5 cursor-pointer">
              <input type="radio" name="mimo" value={2} bind:group={capState.lte.max_mimo_layer} class="accent-accent" />
              <span>{$LL.qualcomm.capability.mimo2()}</span>
            </label>
            <label class="flex items-center gap-1.5 cursor-pointer">
              <input type="radio" name="mimo" value={4} bind:group={capState.lte.max_mimo_layer} class="accent-accent" />
              <span>{$LL.qualcomm.capability.mimo4()}</span>
            </label>
          </div>
          {/if}
        </div>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">{$LL.qualcomm.capability.ard()}</span>
            <span class="text-[10px] text-muted font-mono">[LTE_HORXD]</span>
          </div>
          {#if themeStore.useSakugaControls()}
            <SakugaCheckbox bind:checked={capState.lte.horxd} />
          {:else}
          <input type="checkbox" bind:checked={capState.lte.horxd} class="w-4 h-4 accent-accent" />
          {/if}
        </label>

        <div class="p-2 bg-card-sub border border-border rounded space-y-1.5">
          <span class="font-bold text-foreground block">{$LL.qualcomm.capability.rxChains()}</span>
          <div class="grid grid-cols-4 gap-2">
            <label class="flex items-center gap-1 cursor-pointer">
              {#if themeStore.useSakugaControls()}
                <SakugaCheckbox bind:checked={capState.lte.rx0} />
              {:else}
              <input type="checkbox" bind:checked={capState.lte.rx0} class="accent-accent" />
              {/if}
              <span>#0</span>
            </label>
            <label class="flex items-center gap-1 cursor-pointer">
              {#if themeStore.useSakugaControls()}
                <SakugaCheckbox bind:checked={capState.lte.rx1} />
              {:else}
              <input type="checkbox" bind:checked={capState.lte.rx1} class="accent-accent" />
              {/if}
              <span>#1</span>
            </label>
            <label class="flex items-center gap-1 cursor-pointer">
              {#if themeStore.useSakugaControls()}
                <SakugaCheckbox bind:checked={capState.lte.rx2} />
              {:else}
              <input type="checkbox" bind:checked={capState.lte.rx2} class="accent-accent" />
              {/if}
              <span>#2</span>
            </label>
            <label class="flex items-center gap-1 cursor-pointer">
              {#if themeStore.useSakugaControls()}
                <SakugaCheckbox bind:checked={capState.lte.rx3} />
              {:else}
              <input type="checkbox" bind:checked={capState.lte.rx3} class="accent-accent" />
              {/if}
              <span>#3</span>
            </label>
          </div>
        </div>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">{$LL.qualcomm.capability.volte()}</span>
            <span class="text-[10px] text-muted font-mono">[LTE_VOICE]</span>
          </div>
          {#if themeStore.useSakugaControls()}
            <SakugaCheckbox bind:checked={capState.lte.volte} />
          {:else}
          <input type="checkbox" bind:checked={capState.lte.volte} class="w-4 h-4 accent-accent" />
          {/if}
        </label>
      </div>
    </Card>

    <!-- 2. NSA SECTION -->
    <Card class="p-4 border-2 border-border shadow-neobrutal-sm bg-card space-y-4">
      <div class="flex items-center justify-between border-b-2 border-border/60 pb-2">
        <div class="flex items-center gap-2">
          <Layers class="w-4 h-4 text-accent" />
          <span class="text-xs font-black uppercase tracking-wider text-foreground">{$LL.qualcomm.capability.nsaTitle()}</span>
        </div>
        <Badge variant="default">{$LL.qualcomm.capability.tokenCount({ count: 4 })}</Badge>
      </div>

      <div class="space-y-3 text-xs">
        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">{$LL.qualcomm.capability.nsaLte()}</span>
            <span class="text-[10px] text-muted font-mono">[NSA_LTE]</span>
          </div>
          {#if themeStore.useSakugaControls()}
            <SakugaCheckbox bind:checked={capState.nsa.nsa_lte} />
          {:else}
          <input type="checkbox" bind:checked={capState.nsa.nsa_lte} class="w-4 h-4 accent-accent" />
          {/if}
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">{$LL.qualcomm.capability.dss()}</span>
            <span class="text-[10px] text-muted font-mono">[NSA_LTE_DSS]</span>
          </div>
          {#if themeStore.useSakugaControls()}
            <SakugaCheckbox bind:checked={capState.nsa.nsa_lte_dss} />
          {:else}
          <input type="checkbox" bind:checked={capState.nsa.nsa_lte_dss} class="w-4 h-4 accent-accent" />
          {/if}
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">{$LL.qualcomm.capability.ndds()}</span>
            <span class="text-[10px] text-muted font-mono">[NSA_NDDS]</span>
          </div>
          {#if themeStore.useSakugaControls()}
            <SakugaCheckbox bind:checked={capState.nsa.nsa_ndds} />
          {:else}
          <input type="checkbox" bind:checked={capState.nsa.nsa_ndds} class="w-4 h-4 accent-accent" />
          {/if}
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">{$LL.qualcomm.capability.nsaDsda()}</span>
            <span class="text-[10px] text-muted font-mono">[NSA_LTE_DSDA]</span>
          </div>
          {#if themeStore.useSakugaControls()}
            <SakugaCheckbox bind:checked={capState.nsa.nsa_lte_dsda} />
          {:else}
          <input type="checkbox" bind:checked={capState.nsa.nsa_lte_dsda} class="w-4 h-4 accent-accent" />
          {/if}
        </label>
      </div>
    </Card>

    <!-- 3. NR (5G SA/NSA) SECTION -->
    <Card class="p-4 border-2 border-border shadow-neobrutal-sm bg-card space-y-4">
      <div class="flex items-center justify-between border-b-2 border-border/60 pb-2">
        <div class="flex items-center gap-2">
          <Cpu class="w-4 h-4 text-accent" />
          <span class="text-xs font-black uppercase tracking-wider text-foreground">{$LL.qualcomm.capability.nrTitle()}</span>
        </div>
        <Badge variant="default">{$LL.qualcomm.capability.tokenCount({ count: 9 })}</Badge>
      </div>

      <div class="space-y-3 text-xs">
        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">{$LL.qualcomm.capability.nrCa()}</span>
            <span class="text-[10px] text-muted font-mono">[NR_CA]</span>
          </div>
          {#if themeStore.useSakugaControls()}
            <SakugaCheckbox bind:checked={capState.nr.nr_ca} />
          {:else}
          <input type="checkbox" bind:checked={capState.nr.nr_ca} class="w-4 h-4 accent-accent" />
          {/if}
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">{$LL.qualcomm.capability.nrDl256q()}</span>
            <span class="text-[10px] text-muted font-mono">[NR_DL_256Q]</span>
          </div>
          {#if themeStore.useSakugaControls()}
            <SakugaCheckbox bind:checked={capState.nr.dl_256q} />
          {:else}
          <input type="checkbox" bind:checked={capState.nr.dl_256q} class="w-4 h-4 accent-accent" />
          {/if}
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">{$LL.qualcomm.capability.nrUl256q()}</span>
            <span class="text-[10px] text-muted font-mono">[NR_UL_256Q]</span>
          </div>
          {#if themeStore.useSakugaControls()}
            <SakugaCheckbox bind:checked={capState.nr.ul_256q} />
          {:else}
          <input type="checkbox" bind:checked={capState.nr.ul_256q} class="w-4 h-4 accent-accent" />
          {/if}
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">{$LL.qualcomm.capability.nrDlMimo()}</span>
            <span class="text-[10px] text-muted font-mono">[NR_DL_MIMO]</span>
          </div>
          {#if themeStore.useSakugaControls()}
            <SakugaCheckbox bind:checked={capState.nr.dl_mimo} />
          {:else}
          <input type="checkbox" bind:checked={capState.nr.dl_mimo} class="w-4 h-4 accent-accent" />
          {/if}
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">{$LL.qualcomm.capability.nrUlMimo()}</span>
            <span class="text-[10px] text-muted font-mono">[NR_UL_MIMO]</span>
          </div>
          {#if themeStore.useSakugaControls()}
            <SakugaCheckbox bind:checked={capState.nr.ul_mimo} />
          {:else}
          <input type="checkbox" bind:checked={capState.nr.ul_mimo} class="w-4 h-4 accent-accent" />
          {/if}
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">{$LL.qualcomm.capability.vonr()}</span>
            <span class="text-[10px] text-muted font-mono">[NR_VOICE]</span>
          </div>
          {#if themeStore.useSakugaControls()}
            <SakugaCheckbox bind:checked={capState.nr.vonr} />
          {:else}
          <input type="checkbox" bind:checked={capState.nr.vonr} class="w-4 h-4 accent-accent" />
          {/if}
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">{$LL.qualcomm.capability.redcap()}</span>
            <span class="text-[10px] text-muted font-mono">[NR_REDCAP]</span>
          </div>
          {#if themeStore.useSakugaControls()}
            <SakugaCheckbox bind:checked={capState.nr.redcap} />
          {:else}
          <input type="checkbox" bind:checked={capState.nr.redcap} class="w-4 h-4 accent-accent" />
          {/if}
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">{$LL.qualcomm.capability.nrDsda()}</span>
            <span class="text-[10px] text-muted font-mono">[NR_NR_DSDA]</span>
          </div>
          {#if themeStore.useSakugaControls()}
            <SakugaCheckbox bind:checked={capState.nr.nr_nr_dsda} />
          {:else}
          <input type="checkbox" bind:checked={capState.nr.nr_nr_dsda} class="w-4 h-4 accent-accent" />
          {/if}
        </label>

        <label class="flex items-center justify-between p-2 bg-card-sub border border-border rounded cursor-pointer">
          <div class="space-y-0.5">
            <span class="font-bold text-foreground block">{$LL.qualcomm.capability.limitRfMimo()}</span>
            <span class="text-[10px] text-muted font-mono">[NR_LIMIT_RF_MIMO]</span>
          </div>
          {#if themeStore.useSakugaControls()}
            <SakugaCheckbox bind:checked={capState.nr.limit_rf_mimo} />
          {:else}
          <input type="checkbox" bind:checked={capState.nr.limit_rf_mimo} class="w-4 h-4 accent-accent" />
          {/if}
        </label>
      </div>
    </Card>
  </div>

  <!-- ACTION CONTROLS & MODEM EFS FILES STATUS -->
  <Card class="p-4 border-2 border-border shadow-neobrutal-sm bg-card space-y-4">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <label class="flex items-center gap-2 cursor-pointer text-xs font-bold text-foreground">
          {#if themeStore.useSakugaControls()}
            <SakugaCheckbox bind:checked={restartSSR} />
          {:else}
          <input type="checkbox" bind:checked={restartSSR} class="w-4 h-4 accent-accent" />
          {/if}
          <span>{$LL.qualcomm.capability.restartAfterWrite()}</span>
        </label>
        <button
          type="button"
          class="text-xs text-accent hover:underline font-bold cursor-pointer"
          onclick={() => (showFilesTable = !showFilesTable)}
        >
          {showFilesTable ? $LL.qualcomm.capability.hideMapping() : $LL.qualcomm.capability.showMapping()}
        </button>
      </div>

      <Button variant="primary" onclick={promptApply} disabled={isSaving} class="gap-2">
        <Save class="w-4 h-4" />
        <span>{$LL.qualcomm.capability.apply()}</span>
      </Button>
    </div>

    <!-- EFS FILES STATUS TABLE -->
    {#if showFilesTable && capState.files && capState.files.length > 0}
      <div class="pt-3 border-t-2 border-border/60 space-y-2">
        <span class="text-[10px] text-muted uppercase font-bold tracking-wider block">{$LL.qualcomm.capability.detectedFiles()}</span>
        <div class="overflow-x-auto">
          <table class="w-full text-left text-xs font-mono">
            <thead>
              <tr class="border-b border-border/60 text-muted">
                <th class="py-1.5 px-2">{$LL.qualcomm.capability.thPath()}</th>
                <th class="py-1.5 px-2">{$LL.qualcomm.capability.thDesc()}</th>
                <th class="py-1.5 px-2">{$LL.common.status()}</th>
                <th class="py-1.5 px-2">{$LL.qualcomm.capability.thSize()}</th>
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
                      <span class="ml-1 text-[9px] px-1 bg-amber-500/20 text-amber-700 dark:text-amber-300 rounded">{$LL.qualcomm.capability.writeOnlyBadge()}</span>
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
<Modal bind:open={showConfirmModal} title={$LL.qualcomm.capability.confirmTitle()} size="md">
  {#snippet children()}
    <div class="space-y-3 text-xs text-foreground">
      <div class="p-3 bg-amber-500/10 border border-amber-500/30 rounded flex items-start gap-2">
        <AlertTriangle class="w-4 h-4 text-amber-600 dark:text-amber-400 shrink-0 mt-0.5" />
        <p class="leading-relaxed text-amber-900 dark:text-amber-200">
          {$LL.qualcomm.capability.confirmBody()}
        </p>
      </div>

      <div class="space-y-1">
        <span class="text-[10px] uppercase font-bold text-muted block">{$LL.qualcomm.capability.tokenPayload()}</span>
        <div class="p-2.5 bg-card-sub border-2 border-border rounded font-mono break-all text-xs">
          {liveTokens || '[NO_TOKENS]'}
        </div>
      </div>

      <label class="flex items-center gap-2 cursor-pointer font-bold pt-2 border-t border-border/60">
        {#if themeStore.useSakugaControls()}
          <SakugaCheckbox bind:checked={restartSSR} />
        {:else}
        <input type="checkbox" bind:checked={restartSSR} class="w-4 h-4 accent-accent" />
        {/if}
        <span>{$LL.qualcomm.capability.ssrAfterWrite()}</span>
      </label>
    </div>
  {/snippet}

  {#snippet footer()}
    <div class="flex items-center justify-end gap-2">
      <Button variant="outline" size="sm" onclick={() => (showConfirmModal = false)}>
        {$LL.common.cancel()}
      </Button>
      <Button variant="primary" size="sm" onclick={confirmApply} disabled={isSaving}>
        {isSaving ? $LL.qualcomm.capability.writing() : $LL.qualcomm.capability.confirmApply()}
      </Button>
    </div>
  {/snippet}
</Modal>
