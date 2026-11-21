<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Filter,
    ShieldAlert,
    RefreshCw,
    Save,
    Trash2,
    Plus,
    Lock,
    Unlock,
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
  import { themeStore } from '../../../stores/theme.svelte'
  import SakugaCheckbox from '../../ui/SakugaCheckbox.svelte'
  import { LL } from '../../../i18n/i18n-svelte'

  interface SearcherItem {
    raw_value: number
    channel?: number
    code?: number
    cell_id?: number
    area?: number
    scs?: number
    formatted?: string
  }

  interface RATSearcherConfig {
    cell_blacklist: SearcherItem[]
    channel_blacklist: SearcherItem[]
    channel_code_whitelist: SearcherItem[]
    priority_channels: SearcherItem[]
  }

  interface SearcherPreferences {
    gsm: RATSearcherConfig
    umts: RATSearcherConfig
    lte: RATSearcherConfig
    nr: RATSearcherConfig
  }

  interface ExtraLocksConfig {
    geran_lock_arfcn_enabled: boolean
    geran_arfcn_list: number[]
    fake_bts_cell_barring_enabled: boolean
    wcdma_freq_lock_item: number
    lte_csp_pci_lock: number
    lte_csp_earfcn_lock: number
    nr_csp_pci_lock: number
    nr_csp_earfcn_lock: number
  }

  let selectedRat = $state<'lte' | 'nr' | 'gsm' | 'umts'>('lte')
  let isLoading = $state(false)
  let isSaving = $state(false)
  let restartSSR = $state(false)

  // Inputs for adding new item
  let newCellPci = $state('')
  let newCellCid = $state('')
  let newChannel = $state('')
  let newPriorityChannel = $state('')

  // Extra locks inputs
  let geranArfcnInput = $state('')

  let searcherPrefs = $state<SearcherPreferences>({
    gsm: { cell_blacklist: [], channel_blacklist: [], channel_code_whitelist: [], priority_channels: [] },
    umts: { cell_blacklist: [], channel_blacklist: [], channel_code_whitelist: [], priority_channels: [] },
    lte: { cell_blacklist: [], channel_blacklist: [], channel_code_whitelist: [], priority_channels: [] },
    nr: { cell_blacklist: [], channel_blacklist: [], channel_code_whitelist: [], priority_channels: [] },
  })

  let extraLocks = $state<ExtraLocksConfig>({
    geran_lock_arfcn_enabled: false,
    geran_arfcn_list: [],
    fake_bts_cell_barring_enabled: false,
    wcdma_freq_lock_item: 0,
    lte_csp_pci_lock: 0,
    lte_csp_earfcn_lock: 0,
    nr_csp_pci_lock: 0,
    nr_csp_earfcn_lock: 0,
  })

  onMount(async () => {
    await loadAll()
  })

  async function loadAll() {
    isLoading = true
    try {
      const [resSearcher, resLocks] = await Promise.all([
        api.get<{ ok: boolean; preferences?: SearcherPreferences; error?: string }>('/api/efs/searcher'),
        api.get<{ ok: boolean; locks?: ExtraLocksConfig; error?: string }>('/api/efs/locks'),
      ])

      if (resSearcher && resSearcher.ok && resSearcher.preferences) {
        searcherPrefs = resSearcher.preferences
      }
      if (resLocks && resLocks.ok && resLocks.locks) {
        extraLocks = resLocks.locks
        geranArfcnInput = (resLocks.locks.geran_arfcn_list || []).join(', ')
      }
      toastStore.success($LL.qualcomm.searcher.loaded())
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.qualcomm.searcher.loadFailed())
    } finally {
      isLoading = false
    }
  }

  function addCellBlacklist() {
    const pci = parseInt(newCellPci.trim(), 10)
    const cid = parseInt(newCellCid.trim(), 10)
    if (isNaN(pci) || isNaN(cid)) {
      toastStore.error($LL.qualcomm.searcher.invalidPciCid())
      return
    }

    const current = searcherPrefs[selectedRat].cell_blacklist
    if (current.length >= 6) {
      toastStore.error($LL.qualcomm.searcher.maxCell())
      return
    }

    // Pack PCI and CID: (pci << 32) | cid
    const raw = (pci * 4294967296) + cid
    current.push({
      raw_value: raw,
      code: pci,
      cell_id: cid,
      formatted: `PCI: ${pci}, CID: ${cid}`,
    })
    newCellPci = ''
    newCellCid = ''
  }

  function removeCellBlacklist(index: number) {
    searcherPrefs[selectedRat].cell_blacklist.splice(index, 1)
  }

  function addChannelBlacklist() {
    const ch = parseInt(newChannel.trim(), 10)
    if (isNaN(ch)) {
      toastStore.error($LL.qualcomm.searcher.invalidChannel())
      return
    }
    const current = searcherPrefs[selectedRat].channel_blacklist
    if (current.length >= 6) {
      toastStore.error($LL.qualcomm.searcher.maxChannel())
      return
    }
    current.push({
      raw_value: ch,
      channel: ch,
      formatted: `CH: ${ch}`,
    })
    newChannel = ''
  }

  function removeChannelBlacklist(index: number) {
    searcherPrefs[selectedRat].channel_blacklist.splice(index, 1)
  }

  function addPriorityChannel() {
    const ch = parseInt(newPriorityChannel.trim(), 10)
    if (isNaN(ch)) {
      toastStore.error($LL.qualcomm.searcher.invalidPriority())
      return
    }
    const current = searcherPrefs[selectedRat].priority_channels
    const maxEntries = selectedRat === 'umts' ? 1 : 6
    if (current.length >= maxEntries) {
      toastStore.error($LL.qualcomm.searcher.maxPriority({ count: maxEntries, rat: selectedRat.toUpperCase() }))
      return
    }
    current.push({
      raw_value: ch,
      channel: ch,
      formatted: `Priority: ${ch}`,
    })
    newPriorityChannel = ''
  }

  function removePriorityChannel(index: number) {
    searcherPrefs[selectedRat].priority_channels.splice(index, 1)
  }

  async function saveSearcherPreferences() {
    isSaving = true
    try {
      const res = await api.post<{ ok: boolean; ssr_triggered?: boolean; error?: string }>('/api/efs/searcher', {
        preferences: searcherPrefs,
        backup: true,
        verify: true,
        ssr: restartSSR,
      })

      if (res && res.ok) {
        toastStore.success(res.ssr_triggered ? $LL.qualcomm.searcher.appliedSsr() : $LL.qualcomm.searcher.applied())
      } else {
        toastStore.error(res?.error || $LL.qualcomm.searcher.applyFailed())
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.qualcomm.searcher.saveFailed())
    } finally {
      isSaving = false
    }
  }

  async function saveExtraLocks() {
    isSaving = true
    try {
      // Parse GERAN ARFCN list
      const arfcns = geranArfcnInput
        .split(',')
        .map((s) => parseInt(s.trim(), 10))
        .filter((n) => !isNaN(n) && n > 0)

      extraLocks.geran_arfcn_list = arfcns

      const res = await api.post<{ ok: boolean; ssr_triggered?: boolean; error?: string }>('/api/efs/locks', {
        locks: extraLocks,
        backup: true,
        verify: true,
        ssr: restartSSR,
      })

      if (res && res.ok) {
        toastStore.success(res.ssr_triggered ? $LL.qualcomm.searcher.locksSavedSsr() : $LL.qualcomm.searcher.locksSaved())
      } else {
        toastStore.error(res?.error || $LL.qualcomm.searcher.locksSaveFailed())
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.qualcomm.searcher.saveFailed())
    } finally {
      isSaving = false
    }
  }
</script>

<div class="space-y-6">
  <!-- HEADER & WARNING BANNER -->
  <Card class="p-4 border-2 border-border shadow-neobrutal-sm bg-card space-y-3">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b-2 border-border/60 pb-3">
      <div class="flex items-center gap-2">
        <Filter class="w-5 h-5 text-accent" />
        <h3 class="text-sm font-black uppercase tracking-wider text-foreground">
          {$LL.qualcomm.searcher.title()}
        </h3>
      </div>
      <div class="flex items-center gap-2">
        <Button variant="outline" size="sm" onclick={loadAll} disabled={isLoading} class="gap-1.5">
          <RefreshCw class="w-3.5 h-3.5 {isLoading ? 'animate-spin' : ''}" />
          <span>{$LL.common.reload()}</span>
        </Button>
      </div>
    </div>

    <!-- STATIC WARNING NOTE AS IN MODEMCAPS SPEC -->
    <div class="p-3 bg-amber-500/10 border-2 border-amber-500/30 rounded text-xs text-amber-900 dark:text-amber-200 flex items-start gap-2.5">
      <ShieldAlert class="w-4 h-4 text-amber-700 dark:text-amber-400 shrink-0 mt-0.5" />
      <div class="space-y-1">
        <span class="font-bold uppercase tracking-wider block text-amber-800 dark:text-amber-400">{$LL.qualcomm.searcher.warningTitle()}</span>
        <p>
          {$LL.qualcomm.searcher.warningA()} <strong>modem power cycle / subsystem restart (SSR)</strong>{$LL.qualcomm.searcher.warningB()} <strong>UMTS-only</strong> {$LL.qualcomm.searcher.warningC()}
        </p>
      </div>
    </div>
  </Card>

  <!-- RAT SELECTION TABS -->
  <div class="space-y-4">
    <div class="flex items-center gap-2 border-b-2 border-border pb-1 overflow-x-auto text-xs">
      {#each [
        { id: 'lte', label: 'LTE (4G)', badge: $LL.qualcomm.searcher.rules({ count: searcherPrefs.lte.cell_blacklist.length + searcherPrefs.lte.channel_blacklist.length }) },
        { id: 'nr', label: 'NR (5G SA/NSA)', badge: $LL.qualcomm.searcher.rules({ count: searcherPrefs.nr.cell_blacklist.length + searcherPrefs.nr.channel_blacklist.length }) },
        { id: 'gsm', label: 'GERAN (GSM)', badge: $LL.qualcomm.searcher.rules({ count: searcherPrefs.gsm.cell_blacklist.length }) },
        { id: 'umts', label: 'WCDMA (UMTS/3G)', badge: $LL.qualcomm.searcher.rules({ count: searcherPrefs.umts.priority_channels.length }) },
      ] as tab}
        <button
          type="button"
          class="flex items-center gap-2 px-3.5 py-2 font-bold rounded-t border-2 cursor-pointer transition-all {selectedRat === tab.id
            ? 'border-b-0 border-accent bg-accent/15 text-accent'
            : 'border-transparent text-muted hover:text-foreground'}"
          onclick={() => (selectedRat = tab.id as typeof selectedRat)}
        >
          <span>{tab.label}</span>
          <span class="text-[10px] px-1.5 py-0.2 bg-card border border-border rounded text-muted font-mono">{tab.badge}</span>
        </button>
      {/each}
    </div>

    <!-- RAT PREFERENCES CARD -->
    <Card class="p-4 border-2 border-border shadow-neobrutal-sm bg-card space-y-5">
      <div class="flex items-center justify-between border-b-2 border-border/60 pb-2">
        <span class="text-xs font-black uppercase tracking-wider text-foreground">
          {$LL.qualcomm.searcher.ratTitle({ rat: selectedRat.toUpperCase() })}
        </span>
        <span class="text-[11px] text-muted">{$LL.qualcomm.searcher.maxNote()}</span>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <!-- 1. CELL BLACKLIST -->
        <div class="p-3 bg-card-sub border-2 border-border rounded space-y-3">
          <div class="flex items-center justify-between">
            <span class="text-xs font-bold text-foreground">{$LL.qualcomm.searcher.cellBlacklist()}</span>
            <Badge variant="default">{searcherPrefs[selectedRat].cell_blacklist.length}/6</Badge>
          </div>

          <div class="space-y-1.5 max-h-48 overflow-y-auto">
            {#if searcherPrefs[selectedRat].cell_blacklist.length === 0}
              <div class="p-3 text-center text-xs text-muted font-mono border border-dashed border-border rounded">
                {$LL.qualcomm.searcher.noCells()}
              </div>
            {:else}
              {#each searcherPrefs[selectedRat].cell_blacklist as cell, i}
                <div class="flex items-center justify-between p-2 bg-card border border-border rounded text-xs font-mono">
                  <span>{cell.formatted || `PCI: ${cell.code}, CID: ${cell.cell_id}`}</span>
                  <button
                    type="button"
                    class="text-red-400 hover:text-red-300 p-1"
                    onclick={() => removeCellBlacklist(i)}
                    aria-label={$LL.qualcomm.searcher.removeCell()}
                  >
                    <Trash2 class="w-3.5 h-3.5" />
                  </button>
                </div>
              {/each}
            {/if}
          </div>

          <div class="flex items-center gap-1.5 pt-2 border-t border-border/60">
            <input
              type="text"
              placeholder={$LL.qualcomm.searcher.phPci()}
              bind:value={newCellPci}
              class="w-20 px-2 py-1 text-xs bg-card border-2 border-border rounded text-foreground font-mono"
            />
            <input
              type="text"
              placeholder={$LL.qualcomm.searcher.phCid()}
              bind:value={newCellCid}
              class="flex-1 px-2 py-1 text-xs bg-card border-2 border-border rounded text-foreground font-mono"
            />
            <Button size="sm" onclick={addCellBlacklist} class="px-2.5">
              <Plus class="w-3.5 h-3.5" />
            </Button>
          </div>
        </div>

        <!-- 2. CHANNEL BLACKLIST -->
        <div class="p-3 bg-card-sub border-2 border-border rounded space-y-3">
          <div class="flex items-center justify-between">
            <span class="text-xs font-bold text-foreground">{$LL.qualcomm.searcher.channelBlacklist()}</span>
            <Badge variant="default">{searcherPrefs[selectedRat].channel_blacklist.length}/6</Badge>
          </div>

          <div class="space-y-1.5 max-h-48 overflow-y-auto">
            {#if searcherPrefs[selectedRat].channel_blacklist.length === 0}
              <div class="p-3 text-center text-xs text-muted font-mono border border-dashed border-border rounded">
                {$LL.qualcomm.searcher.noChannels()}
              </div>
            {:else}
              {#each searcherPrefs[selectedRat].channel_blacklist as ch, i}
                <div class="flex items-center justify-between p-2 bg-card border border-border rounded text-xs font-mono">
                  <span>{ch.formatted || `CH: ${ch.channel}`}</span>
                  <button
                    type="button"
                    class="text-red-400 hover:text-red-300 p-1"
                    onclick={() => removeChannelBlacklist(i)}
                    aria-label={$LL.qualcomm.searcher.removeChannel()}
                  >
                    <Trash2 class="w-3.5 h-3.5" />
                  </button>
                </div>
              {/each}
            {/if}
          </div>

          <div class="flex items-center gap-1.5 pt-2 border-t border-border/60">
            <input
              type="text"
              placeholder={$LL.qualcomm.searcher.phChannel()}
              bind:value={newChannel}
              class="flex-1 px-2 py-1 text-xs bg-card border-2 border-border rounded text-foreground font-mono"
            />
            <Button size="sm" onclick={addChannelBlacklist} class="px-2.5">
              <Plus class="w-3.5 h-3.5" />
            </Button>
          </div>
        </div>

        <!-- 3. PRIORITY CHANNELS -->
        <div class="p-3 bg-card-sub border-2 border-border rounded space-y-3">
          <div class="flex items-center justify-between">
            <span class="text-xs font-bold text-foreground">{$LL.qualcomm.searcher.priorityChannels()}</span>
            <Badge variant="default">{searcherPrefs[selectedRat].priority_channels.length}/{selectedRat === 'umts' ? 1 : 6}</Badge>
          </div>

          <div class="space-y-1.5 max-h-48 overflow-y-auto">
            {#if searcherPrefs[selectedRat].priority_channels.length === 0}
              <div class="p-3 text-center text-xs text-muted font-mono border border-dashed border-border rounded">
                {$LL.qualcomm.searcher.noPriority()}
              </div>
            {:else}
              {#each searcherPrefs[selectedRat].priority_channels as ch, i}
                <div class="flex items-center justify-between p-2 bg-card border border-border rounded text-xs font-mono">
                  <span>#{i + 1} {ch.formatted || `Priority CH: ${ch.channel}`}</span>
                  <button
                    type="button"
                    class="text-red-400 hover:text-red-300 p-1"
                    onclick={() => removePriorityChannel(i)}
                    aria-label={$LL.qualcomm.searcher.removePriority()}
                  >
                    <Trash2 class="w-3.5 h-3.5" />
                  </button>
                </div>
              {/each}
            {/if}
          </div>

          <div class="flex items-center gap-1.5 pt-2 border-t border-border/60">
            <input
              type="text"
              placeholder={$LL.qualcomm.searcher.phPriority()}
              bind:value={newPriorityChannel}
              class="flex-1 px-2 py-1 text-xs bg-card border-2 border-border rounded text-foreground font-mono"
            />
            <Button size="sm" onclick={addPriorityChannel} class="px-2.5">
              <Plus class="w-3.5 h-3.5" />
            </Button>
          </div>
        </div>
      </div>

      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pt-3 border-t-2 border-border/60">
        <label class="flex items-center gap-2 cursor-pointer text-xs font-bold text-foreground">
          {#if themeStore.useSakugaControls()}
            <SakugaCheckbox bind:checked={restartSSR} />
          {:else}
          <input type="checkbox" bind:checked={restartSSR} class="w-4 h-4 accent-accent" />
          {/if}
          <span>{$LL.qualcomm.searcher.restartAfterWrite()}</span>
        </label>
        <Button variant="primary" onclick={saveSearcherPreferences} disabled={isSaving} class="gap-2">
          <Save class="w-4 h-4" />
          <span>{isSaving ? $LL.qualcomm.searcher.writing() : $LL.qualcomm.searcher.apply()}</span>
        </Button>
      </div>
    </Card>
  </div>

  <!-- SECTION 2: KUNCI EKSTRA / EXTRA LOCKS (FITUR C) -->
  <Card class="p-4 border-2 border-border shadow-neobrutal-sm bg-card space-y-4">
    <div class="flex items-center gap-2 border-b-2 border-border/60 pb-3">
      <Lock class="w-4 h-4 text-accent" />
      <span class="text-xs font-black uppercase tracking-wider text-foreground">
        {$LL.qualcomm.searcher.extraLocksTitle()}
      </span>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
      <!-- GERAN LOCKS -->
      <div class="p-3 bg-card-sub border-2 border-border rounded space-y-3">
        <span class="font-bold text-foreground block uppercase text-[11px] tracking-wider">{$LL.qualcomm.searcher.geranTitle()}</span>

        <label class="flex items-center gap-2 cursor-pointer">
          {#if themeStore.useSakugaControls()}
            <SakugaCheckbox bind:checked={extraLocks.geran_lock_arfcn_enabled} />
          {:else}
          <input type="checkbox" bind:checked={extraLocks.geran_lock_arfcn_enabled} class="w-4 h-4 accent-accent" />
          {/if}
          <span class="font-bold text-foreground">{$LL.qualcomm.searcher.enableGeran()}</span>
        </label>

        <div class="space-y-1">
          <label for="geran-arfcn-input" class="text-[10px] text-muted uppercase font-bold block">{$LL.qualcomm.searcher.geranListLabel()}</label>
          <input
            id="geran-arfcn-input"
            type="text"
            placeholder={$LL.qualcomm.searcher.geranPlaceholder()}
            bind:value={geranArfcnInput}
            class="w-full px-2.5 py-1.5 bg-card border-2 border-border rounded text-foreground font-mono"
          />
        </div>

        <label class="flex items-center gap-2 cursor-pointer pt-2 border-t border-border/40">
          {#if themeStore.useSakugaControls()}
            <SakugaCheckbox bind:checked={extraLocks.fake_bts_cell_barring_enabled} />
          {:else}
          <input type="checkbox" bind:checked={extraLocks.fake_bts_cell_barring_enabled} class="w-4 h-4 accent-accent" />
          {/if}
          <span class="font-bold text-foreground">{$LL.qualcomm.searcher.fakeBts()}</span>
        </label>
      </div>

      <!-- WCDMA & CSP LOCKS -->
      <div class="p-3 bg-card-sub border-2 border-border rounded space-y-3">
        <span class="font-bold text-foreground block uppercase text-[11px] tracking-wider">{$LL.qualcomm.searcher.wcdmaTitle()}</span>

        <div class="space-y-1">
          <label for="wcdma-freq-input" class="text-[10px] text-muted uppercase font-bold block">{$LL.qualcomm.searcher.wcdmaLabel()}</label>
          <input
            id="wcdma-freq-input"
            type="number"
            bind:value={extraLocks.wcdma_freq_lock_item}
            class="w-full px-2.5 py-1.5 bg-card border-2 border-border rounded text-foreground font-mono"
          />
        </div>

        <div class="grid grid-cols-2 gap-2">
          <div class="space-y-1">
            <label for="lte-csp-pci" class="text-[10px] text-muted uppercase font-bold block">{$LL.qualcomm.searcher.lteCspPci()}</label>
            <input
              id="lte-csp-pci"
              type="number"
              bind:value={extraLocks.lte_csp_pci_lock}
              class="w-full px-2.5 py-1.5 bg-card border-2 border-border rounded text-foreground font-mono"
            />
          </div>
          <div class="space-y-1">
            <label for="lte-csp-earfcn" class="text-[10px] text-muted uppercase font-bold block">{$LL.qualcomm.searcher.lteCspEarfcn()}</label>
            <input
              id="lte-csp-earfcn"
              type="number"
              bind:value={extraLocks.lte_csp_earfcn_lock}
              class="w-full px-2.5 py-1.5 bg-card border-2 border-border rounded text-foreground font-mono"
            />
          </div>
        </div>

        <div class="grid grid-cols-2 gap-2">
          <div class="space-y-1">
            <label for="nr-csp-pci" class="text-[10px] text-muted uppercase font-bold block">{$LL.qualcomm.searcher.nrCspPci()}</label>
            <input
              id="nr-csp-pci"
              type="number"
              bind:value={extraLocks.nr_csp_pci_lock}
              class="w-full px-2.5 py-1.5 bg-card border-2 border-border rounded text-foreground font-mono"
            />
          </div>
          <div class="space-y-1">
            <label for="nr-csp-earfcn" class="text-[10px] text-muted uppercase font-bold block">{$LL.qualcomm.searcher.nrCspEarfcn()}</label>
            <input
              id="nr-csp-earfcn"
              type="number"
              bind:value={extraLocks.nr_csp_earfcn_lock}
              class="w-full px-2.5 py-1.5 bg-card border-2 border-border rounded text-foreground font-mono"
            />
          </div>
        </div>
      </div>
    </div>

    <div class="flex items-center justify-end pt-3 border-t-2 border-border/60">
      <Button variant="primary" onclick={saveExtraLocks} disabled={isSaving} class="gap-2">
        <Lock class="w-4 h-4" />
        <span>{isSaving ? $LL.qualcomm.searcher.savingLocks() : $LL.qualcomm.searcher.applyLocks()}</span>
      </Button>
    </div>
  </Card>
</div>
