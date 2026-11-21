<script lang="ts">
  import {
    AlertTriangle,
    Copy,
    Download,
    Radio,
    Search,
  } from '@lucide/svelte'
  import Modal from '../../ui/Modal.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import { themeStore } from '../../../stores/theme.svelte'
  import SakugaCheckbox from '../../ui/SakugaCheckbox.svelte'
  import { LL } from '../../../i18n/i18n-svelte'

  interface Props {
    open: boolean
    activeFile: any
    status: any
    editorMode: 'text' | 'hex' | 'ca'
    editorTextContent: string
    filteredCaCombos: any[]
    caSearchBand: string
    caMimoFilter: 'all' | '4x4' | '2x2'
    caUlFilter: boolean
    isSavingFile: boolean
    formatBytes: (bytes: number) => string
    downloadFile: (path: string) => Promise<void> | void
    saveFile: () => Promise<void> | void
    copySingleCombo: (str: string) => void
    copyAllCombos: () => void
    exportCombosJson: () => void
  }

  let {
    open = $bindable(false),
    activeFile,
    status,
    editorMode = $bindable('text'),
    editorTextContent = $bindable(''),
    filteredCaCombos = [],
    caSearchBand = $bindable(''),
    caMimoFilter = $bindable('all'),
    caUlFilter = $bindable(false),
    isSavingFile = false,
    formatBytes,
    downloadFile,
    saveFile,
    copySingleCombo,
    copyAllCombos,
    exportCombosJson,
  }: Props = $props()
</script>

<Modal
  bind:open={open}
  title={activeFile ? $LL.qualcomm.efsEditor.fileTitle({ path: activeFile.path }) : $LL.qualcomm.efsEditor.viewFile()}
  size="xl"
>
  {#if activeFile}
    <div class="space-y-3 font-mono text-xs">
      <div class="flex items-center justify-between border-b border-border pb-2">
        <div class="flex items-center gap-2">
          <Badge variant="info">{formatBytes(activeFile.size)}</Badge>
          <Badge variant={activeFile.is_binary ? 'warning' : 'default'}>
            {activeFile.is_binary ? $LL.qualcomm.efsEditor.binary() : $LL.qualcomm.efsEditor.plainText()}
          </Badge>
        </div>

        <div class="flex items-center gap-1.5">
          <button
            type="button"
            class="px-2 py-1 rounded font-bold {editorMode === 'text' ? 'bg-accent text-accent-text' : 'bg-card-sub text-muted'} cursor-pointer"
            onclick={() => (editorMode = 'text')}
          >
            Text
          </button>
          <button
            type="button"
            class="px-2 py-1 rounded font-bold {editorMode === 'hex' ? 'bg-accent text-accent-text' : 'bg-card-sub text-muted'} cursor-pointer"
            onclick={() => (editorMode = 'hex')}
          >
            Hex
          </button>
          {#if activeFile.carrier_policy && activeFile.carrier_policy.total_combos > 0}
            <button
              type="button"
              class="px-2.5 py-1 rounded font-bold flex items-center gap-1.5 {editorMode === 'ca' ? 'bg-cyan-500 text-black shadow' : 'bg-cyan-950/40 text-cyan-300 border border-cyan-800/50'} cursor-pointer transition-colors"
              onclick={() => (editorMode = 'ca')}
            >
              <Radio class="w-3.5 h-3.5" />
              <span>{$LL.qualcomm.efsEditor.caCombos({ count: activeFile.carrier_policy.total_combos })}</span>
            </button>
          {/if}
        </div>
      </div>

      {#if editorMode === 'ca' && activeFile.carrier_policy}
        {@const policy = activeFile.carrier_policy}
        <div class="space-y-3">
          <!-- Policy Summary Cards -->
          <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
            <div class="bg-zinc-950 p-2 rounded border border-border">
              <span class="text-[10px] text-muted uppercase font-bold block">{$LL.qualcomm.efsEditor.policyConfig()}</span>
              <span class="text-xs font-bold text-foreground truncate block">
                {policy.policy_name || 'Carrier Policy'} {policy.policy_ver ? `(v${policy.policy_ver})` : ''}
              </span>
            </div>

            <div class="bg-zinc-950 p-2 rounded border border-border">
              <span class="text-[10px] text-muted uppercase font-bold block">{$LL.qualcomm.efsEditor.totalCombos()}</span>
              <span class="text-xs font-bold text-cyan-400 block">{$LL.qualcomm.efsEditor.combinations({ count: policy.total_combos })}</span>
            </div>

            <div class="bg-zinc-950 p-2 rounded border border-border">
              <span class="text-[10px] text-muted uppercase font-bold block">{$LL.qualcomm.efsEditor.maxDlStreams()}</span>
              <span class="text-xs font-bold text-amber-400 block">
                {$LL.qualcomm.efsEditor.streams({ count: policy.max_streams, mimo: policy.max_streams >= 4 ? '(4x4 MIMO)' : '(2x2 MIMO)' })}
              </span>
            </div>

            <div class="bg-zinc-950 p-2 rounded border border-border">
              <span class="text-[10px] text-muted uppercase font-bold block">{$LL.qualcomm.efsEditor.supportedBands()}</span>
              <span class="text-xs font-bold text-emerald-400 block truncate">
                {$LL.qualcomm.efsEditor.bandsCount({ count: policy.bands_supported.length, list: 'B' + policy.bands_supported.slice(0, 5).join(', B') + (policy.bands_supported.length > 5 ? '...' : '') })}
              </span>
            </div>
          </div>

          <!-- Quick Band Filter Chips -->
          {#if policy.bands_supported.length > 0}
            <div class="flex items-center gap-1.5 flex-wrap bg-zinc-950/70 p-2 rounded border border-border/60">
              <span class="text-[10px] font-bold text-muted uppercase mr-1">{$LL.qualcomm.efsEditor.quickBand()}</span>
              <button
                type="button"
                class="px-2 py-0.5 rounded text-[10px] font-bold cursor-pointer transition-colors {caSearchBand === '' ? 'bg-cyan-500 text-black' : 'bg-card-sub text-muted hover:text-foreground'}"
                onclick={() => (caSearchBand = '')}
              >
                {$LL.common.all()}
              </button>
              {#each policy.bands_supported as b}
                <button
                  type="button"
                  class="px-2 py-0.5 rounded text-[10px] font-bold cursor-pointer transition-colors {caSearchBand === b.toString() ? 'bg-cyan-500 text-black' : 'bg-card-sub text-cyan-300 border border-cyan-900/40 hover:border-cyan-500/50'}"
                  onclick={() => (caSearchBand = caSearchBand === b.toString() ? '' : b.toString())}
                >
                  B{b}
                </button>
              {/each}
            </div>
          {/if}

          <!-- Search & Filter Controls -->
          <div class="flex flex-wrap items-center justify-between gap-2 bg-card-sub/50 p-2 rounded border border-border">
            <div class="flex flex-wrap items-center gap-2 flex-1 min-w-[240px]">
              <!-- Search -->
              <div class="relative flex-1 min-w-[140px]">
                <Search class="w-3.5 h-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-muted" />
                <input
                  type="text"
                  bind:value={caSearchBand}
                  placeholder={$LL.qualcomm.efsEditor.filterBandPlaceholder()}
                  class="w-full bg-zinc-950 border border-border rounded pl-8 pr-3 py-1.5 text-xs text-foreground placeholder:text-muted focus:outline-none focus:border-cyan-500"
                />
              </div>

              <!-- MIMO Filter -->
              <div class="flex items-center rounded border border-border overflow-hidden text-[11px] font-bold">
                <button
                  type="button"
                  class="px-2 py-1 cursor-pointer {caMimoFilter === 'all' ? 'bg-cyan-500 text-black' : 'bg-zinc-950 text-muted'}"
                  onclick={() => (caMimoFilter = 'all')}
                >
                  {$LL.qualcomm.efsEditor.allMimo()}
                </button>
                <button
                  type="button"
                  class="px-2 py-1 cursor-pointer {caMimoFilter === '4x4' ? 'bg-cyan-500 text-black' : 'bg-zinc-950 text-muted'}"
                  onclick={() => (caMimoFilter = '4x4')}
                >
                  {$LL.qualcomm.efsEditor.mimo4Only()}
                </button>
                <button
                  type="button"
                  class="px-2 py-1 cursor-pointer {caMimoFilter === '2x2' ? 'bg-cyan-500 text-black' : 'bg-zinc-950 text-muted'}"
                  onclick={() => (caMimoFilter = '2x2')}
                >
                  {$LL.qualcomm.efsEditor.mimo2Only()}
                </button>
              </div>

              <!-- UL CA Checkbox -->
              <label class="flex items-center gap-1.5 text-xs text-muted cursor-pointer select-none bg-zinc-950 px-2 py-1 rounded border border-border">
                {#if themeStore.useSakugaControls()}
                  <SakugaCheckbox bind:checked={caUlFilter} />
                {:else}
                <input type="checkbox" bind:checked={caUlFilter} class="rounded border-border accent-cyan-500" />
                {/if}
                <span>{$LL.qualcomm.efsEditor.ulCaOnly()}</span>
              </label>
            </div>

            <!-- Action Buttons -->
            <div class="flex items-center gap-1.5">
              <button
                type="button"
                class="px-2.5 py-1 bg-zinc-900 hover:bg-zinc-800 text-foreground border border-border rounded text-[11px] font-bold flex items-center gap-1 cursor-pointer"
                onclick={copyAllCombos}
                title={$LL.qualcomm.efsEditor.copyListTitle()}
              >
                <Copy class="w-3.5 h-3.5 text-cyan-400" />
                <span>{$LL.qualcomm.efsEditor.copyList()}</span>
              </button>
              <button
                type="button"
                class="px-2.5 py-1 bg-zinc-900 hover:bg-zinc-800 text-foreground border border-border rounded text-[11px] font-bold flex items-center gap-1 cursor-pointer"
                onclick={exportCombosJson}
                title={$LL.qualcomm.efsEditor.exportJsonTitle()}
              >
                <Download class="w-3.5 h-3.5 text-emerald-400" />
                <span>JSON</span>
              </button>
            </div>
          </div>

          <!-- Table of Combinations -->
          <div class="border-2 border-border rounded overflow-hidden max-h-[380px] overflow-y-auto">
            <table class="w-full text-left text-xs border-collapse">
              <thead class="bg-card-sub sticky top-0 border-b-2 border-border z-10 text-[10px] uppercase font-bold text-muted">
                <tr>
                  <th class="p-2 w-12 text-center">{$LL.qualcomm.efsEditor.thIndex()}</th>
                  <th class="p-2">{$LL.qualcomm.efsEditor.thComponents()}</th>
                  <th class="p-2 w-28 text-center">{$LL.qualcomm.efsEditor.thDlStreams()}</th>
                  <th class="p-2 w-32 text-center">{$LL.qualcomm.efsEditor.thBcs()}</th>
                  <th class="p-2 w-16 text-right">{$LL.qualcomm.efsEditor.thAction()}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border bg-zinc-950 font-mono">
                {#if filteredCaCombos.length === 0}
                  <tr>
                    <td colspan="5" class="p-8 text-center text-muted">
                      <span>{$LL.qualcomm.efsEditor.noCombosMatch()}</span>
                    </td>
                  </tr>
                {:else}
                  {#each filteredCaCombos as combo, idx}
                    <tr class="hover:bg-zinc-900/80 transition-colors">
                      <!-- Index -->
                      <td class="p-2 text-center text-muted text-[11px]">
                        {idx + 1}
                      </td>

                      <!-- Components -->
                      <td class="p-2">
                        <div class="flex items-center gap-1.5 flex-wrap">
                          {#each combo.components as comp, cIdx}
                            {#if cIdx > 0}
                              <span class="text-muted font-bold text-xs">+</span>
                            {/if}
                            <div class="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-zinc-900 border border-zinc-700/60 shadow-sm">
                              <!-- Band & Class -->
                              <span class="font-bold text-cyan-300">
                                B{comp.band}{comp.dl_class}
                              </span>

                              <!-- MIMO Badge -->
                              {#if comp.dl_mimo.includes('4')}
                                <span class="px-1 py-0.2 rounded text-[9px] font-bold bg-cyan-950 text-cyan-300 border border-cyan-700/50">
                                  {comp.dl_mimo === '44' ? '4x4+4x4' : '4x4'}
                                </span>
                              {:else}
                                <span class="px-1 py-0.2 rounded text-[9px] font-bold bg-zinc-800 text-zinc-400">
                                  {comp.dl_mimo === '22' ? '2x2+2x2' : '2x2'}
                                </span>
                              {/if}

                              <!-- UL Badge -->
                              {#if comp.ul_class && comp.ul_class !== 'None'}
                                <span class="px-1 py-0.2 rounded text-[9px] font-bold bg-amber-950/80 text-amber-300 border border-amber-600/40">
                                  UL:{comp.ul_class}
                                </span>
                              {/if}
                            </div>
                          {/each}
                        </div>
                      </td>

                      <!-- DL Streams -->
                      <td class="p-2 text-center">
                        <span class="px-2 py-0.5 rounded text-[10px] font-bold {combo.total_streams >= 8 ? 'bg-cyan-950 text-cyan-300 border border-cyan-600/40' : combo.total_streams >= 4 ? 'bg-emerald-950 text-emerald-300 border border-emerald-600/40' : 'bg-zinc-800 text-zinc-300'}">
                          {$LL.qualcomm.efsEditor.streamsCount({ count: combo.total_streams })}
                        </span>
                      </td>

                      <!-- BCS -->
                      <td class="p-2 text-center text-muted text-[11px] truncate" title={combo.bcs}>
                        <span class="px-1.5 py-0.5 rounded text-[9px] font-bold bg-zinc-900 border border-zinc-800 text-foreground">
                          {combo.bcs}
                        </span>
                      </td>

                      <!-- Copy Single Action -->
                      <td class="p-2 text-right">
                        <button
                          type="button"
                          class="p-1 hover:text-cyan-400 rounded cursor-pointer text-muted transition-colors"
                          title={$LL.qualcomm.efsEditor.copyComboTitle()}
                          onclick={() => copySingleCombo(combo.raw_combo_string)}
                        >
                          <Copy class="w-3.5 h-3.5" />
                        </button>
                      </td>
                    </tr>
                  {/each}
                {/if}
              </tbody>
            </table>
          </div>
        </div>
      {:else if editorMode === 'text'}
        <textarea
          bind:value={editorTextContent}
          rows="18"
          class="neo-input w-full bg-zinc-950 border-2 border-border rounded p-3 font-mono text-xs text-foreground focus:outline-none focus:border-accent"
          placeholder={$LL.qualcomm.efsEditor.fileContentPlaceholder()}
        ></textarea>
      {:else}
        <div class="bg-zinc-950 p-3 rounded border-2 border-border max-h-[350px] overflow-y-auto break-all font-mono text-[11px] leading-relaxed text-cyan-300">
          {activeFile.data_hex}
        </div>
      {/if}

      {#if status.readonly}
        <div class="p-2 bg-amber-950/30 border border-amber-600/40 rounded text-amber-300 text-[11px] flex items-center gap-2">
          <AlertTriangle class="w-4 h-4 shrink-0 text-amber-400" />
          <span>{$LL.qualcomm.efsEditor.readonlyNote()}</span>
        </div>
      {/if}
    </div>
  {/if}

  {#snippet footer()}
    <div class="flex items-center justify-between w-full">
      <Button
        variant="outline"
        size="sm"
        onclick={() => downloadFile(activeFile?.path || '')}
      >
        <Download class="w-3.5 h-3.5 mr-1" />
        <span>{$LL.qualcomm.efsEditor.downloadFile()}</span>
      </Button>

      <div class="flex items-center gap-2">
        {#if editorMode === 'ca'}
          <span class="text-[11px] text-muted hidden sm:inline">
            {$LL.qualcomm.efsEditor.caViewNote()}
          </span>
        {/if}
        <Button variant="secondary" size="sm" onclick={() => (open = false)}>
          {$LL.common.cancel()}
        </Button>
        <Button
          variant="primary"
          size="sm"
          disabled={status.readonly || isSavingFile || editorMode === 'ca'}
          onclick={saveFile}
        >
          <span>{isSavingFile ? $LL.qualcomm.efsEditor.saving() : $LL.qualcomm.efsEditor.save()}</span>
        </Button>
      </div>
    </div>
  {/snippet}

</Modal>


