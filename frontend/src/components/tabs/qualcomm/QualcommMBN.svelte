<script lang="ts">
  import { onMount } from 'svelte'
  import {
    HardDrive,
    RefreshCw,
    CheckCircle,
    AlertTriangle,
    RotateCcw,
    ShieldAlert,
    Search,
    Radio,
    Sliders,
    Layers,
    Cpu,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'

  interface MbnConfigItem {
    name: string
    path: string
    type: 'sw' | 'hw'
    category?: string
    size: number
    active: boolean
  }

  interface MbnAutoselectStatus {
    sim1_mode: number
    sim2_mode: number
    sim1_auto: boolean
    sim2_auto: boolean
  }

  interface MbnOverview {
    active_profile: string
    previous_profile?: string
    autoselect: MbnAutoselectStatus
    sw_configs: MbnConfigItem[]
    hw_configs: MbnConfigItem[]
  }

  let selectedSlot = $state(0)
  let isLoading = $state(false)
  let isActivating = $state(false)
  let isReverting = $state(false)
  let restartSSR = $state(true)
  let searchQuery = $state('')
  let activeTab = $state<'sw' | 'hw'>('sw')

  // Selected config for confirmation modal
  let pendingConfig = $state<MbnConfigItem | null>(null)
  let showConfirmModal = $state(false)

  let overview = $state<MbnOverview>({
    active_profile: '',
    previous_profile: '',
    autoselect: {
      sim1_mode: 1,
      sim2_mode: 1,
      sim1_auto: true,
      sim2_auto: true,
    },
    sw_configs: [],
    hw_configs: [],
  })

  // Filtered SW configs
  let filteredSwConfigs = $derived.by(() => {
    const q = searchQuery.trim().toLowerCase()
    if (!q) return overview.sw_configs
    return overview.sw_configs.filter(
      (c) =>
        c.name.toLowerCase().includes(q) ||
        c.path.toLowerCase().includes(q) ||
        (c.category && c.category.toLowerCase().includes(q))
    )
  })

  // Filtered HW configs
  let filteredHwConfigs = $derived.by(() => {
    const q = searchQuery.trim().toLowerCase()
    if (!q) return overview.hw_configs
    return overview.hw_configs.filter(
      (c) =>
        c.name.toLowerCase().includes(q) ||
        c.path.toLowerCase().includes(q) ||
        (c.category && c.category.toLowerCase().includes(q))
    )
  })

  onMount(async () => {
    await loadOverview()
  })

  async function loadOverview() {
    isLoading = true
    try {
      const res = await api.get<{ ok: boolean; overview?: MbnOverview; error?: string }>(
        `/api/qualcomm/mbn?slot=${selectedSlot}`
      )
      if (res && res.ok && res.overview) {
        overview = res.overview
        toastStore.success('MBN carrier configurations loaded.')
      } else {
        toastStore.error(res?.error || 'Failed to load MBN configurations')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Read error')
    } finally {
      isLoading = false
    }
  }

  function promptActivate(cfg: MbnConfigItem) {
    pendingConfig = cfg
    showConfirmModal = true
  }

  async function confirmActivate() {
    if (!pendingConfig) return
    isActivating = true
    showConfirmModal = false

    try {
      const res = await api.post<{ ok: boolean; ssr_triggered?: boolean; error?: string }>(
        '/api/qualcomm/mbn/activate',
        {
          profile: pendingConfig.path || pendingConfig.name,
          slot: selectedSlot,
          ssr: restartSSR,
        }
      )

      if (res && res.ok) {
        toastStore.success(
          `Activated profile '${pendingConfig.name}' successfully${res.ssr_triggered ? ' (SSR modem restart triggered)' : ''}!`
        )
        await loadOverview()
      } else {
        toastStore.error(res?.error || 'Failed to activate profile')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Activation error')
    } finally {
      isActivating = false
      pendingConfig = null
    }
  }

  async function revertPreviousProfile() {
    if (!overview.previous_profile) return
    isReverting = true
    try {
      const res = await api.post<{ ok: boolean; ssr_triggered?: boolean; error?: string }>(
        '/api/qualcomm/mbn/revert',
        {
          slot: selectedSlot,
          ssr: restartSSR,
        }
      )

      if (res && res.ok) {
        toastStore.success(
          `Reverted back to '${overview.previous_profile}' successfully${res.ssr_triggered ? ' (SSR restart triggered)' : ''}!`
        )
        await loadOverview()
      } else {
        toastStore.error(res?.error || 'Failed to revert profile')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Revert error')
    } finally {
      isReverting = false
    }
  }

  async function updateAutoselect(slot: number, mode: number) {
    try {
      const res = await api.post<{ ok: boolean; ssr_triggered?: boolean; error?: string }>(
        '/api/qualcomm/mbn/autoselect',
        {
          slot,
          mode,
          ssr: false,
        }
      )

      if (res && res.ok) {
        toastStore.success(`Autoselect by UIM for SIM${slot + 1} updated to mode ${mode}.`)
        await loadOverview()
      } else {
        toastStore.error(res?.error || 'Failed to update autoselect')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Autoselect update failed')
    }
  }
</script>

<div class="space-y-6">
  <!-- HEADER CARD & CURRENT ACTIVE PROFILE -->
  <Card class="p-4 border-2 border-border shadow-neobrutal-sm bg-card space-y-3">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b-2 border-border/60 pb-3">
      <div class="flex items-center gap-2">
        <HardDrive class="w-5 h-5 text-accent" />
        <h3 class="text-sm font-black uppercase tracking-wider text-foreground">
          Qualcomm MBN / PDC Carrier Profile Switcher
        </h3>
      </div>
      <div class="flex items-center gap-2">
        <!-- SIM SLOT TOGGLE -->
        <div class="flex items-center border-2 border-border rounded overflow-hidden text-xs font-bold">
          <button
            type="button"
            class="px-2.5 py-1 cursor-pointer transition-all {selectedSlot === 0 ? 'bg-accent text-accent-foreground' : 'bg-card text-muted hover:text-foreground'}"
            onclick={() => { selectedSlot = 0; loadOverview(); }}
          >
            SIM 1
          </button>
          <button
            type="button"
            class="px-2.5 py-1 cursor-pointer transition-all {selectedSlot === 1 ? 'bg-accent text-accent-foreground' : 'bg-card text-muted hover:text-foreground'}"
            onclick={() => { selectedSlot = 1; loadOverview(); }}
          >
            SIM 2
          </button>
        </div>
        <Button variant="outline" size="sm" onclick={loadOverview} disabled={isLoading} class="gap-1.5">
          <RefreshCw class="w-3.5 h-3.5 {isLoading ? 'animate-spin' : ''}" />
          <span>Reload</span>
        </Button>
      </div>
    </div>

    <!-- ACTIVE PROFILE BADGE -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-3 text-xs">
      <div class="p-3 bg-card-sub border-2 border-border rounded space-y-1">
        <span class="text-[10px] text-muted uppercase font-bold tracking-wider block">Active Baseband MBN Profile (SIM {selectedSlot + 1}):</span>
        <div class="flex items-center gap-2">
          {#if overview.active_profile}
            <Badge variant="success" class="font-mono text-xs font-bold flex items-center gap-1.5 py-1 px-2.5">
              <CheckCircle class="w-3.5 h-3.5" />
              <span>{overview.active_profile}</span>
            </Badge>
          {:else}
            <span class="text-muted font-mono italic">Automatic / UIM Auto-selected (Default)</span>
          {/if}
        </div>
      </div>

      <!-- AUTOSELECT TOGGLE BY UIM -->
      <div class="p-3 bg-card-sub border-2 border-border rounded space-y-2">
        <span class="text-[10px] text-muted uppercase font-bold tracking-wider block">MCFG Autoselect on SIM Change:</span>
        <div class="flex items-center gap-3">
          <label class="flex items-center gap-1.5 cursor-pointer">
            <input
              type="radio"
              name="autoselect_sim"
              value={1}
              checked={selectedSlot === 0 ? overview.autoselect.sim1_mode === 1 : overview.autoselect.sim2_mode === 1}
              onchange={() => updateAutoselect(selectedSlot, 1)}
              class="accent-accent"
            />
            <span class="font-bold text-foreground">Auto by IMSI</span>
          </label>
          <label class="flex items-center gap-1.5 cursor-pointer">
            <input
              type="radio"
              name="autoselect_sim"
              value={0}
              checked={selectedSlot === 0 ? overview.autoselect.sim1_mode === 0 : overview.autoselect.sim2_mode === 0}
              onchange={() => updateAutoselect(selectedSlot, 0)}
              class="accent-accent"
            />
            <span class="font-bold text-foreground">Manual Lock</span>
          </label>
          <label class="flex items-center gap-1.5 cursor-pointer">
            <input
              type="radio"
              name="autoselect_sim"
              value={2}
              checked={selectedSlot === 0 ? overview.autoselect.sim1_mode === 2 : overview.autoselect.sim2_mode === 2}
              onchange={() => updateAutoselect(selectedSlot, 2)}
              class="accent-accent"
            />
            <span class="font-bold text-foreground">With MBN update only</span>
          </label>
        </div>
      </div>
    </div>

    <!-- ONE-CLICK ROLLBACK SAFETY BANNER -->
    {#if overview.previous_profile}
      <div class="p-3 bg-amber-500/10 border-2 border-amber-500/30 rounded text-xs flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div class="flex items-start gap-2">
          <ShieldAlert class="w-4 h-4 text-amber-400 shrink-0 mt-0.5" />
          <div>
            <span class="font-bold text-amber-300 block">Rollback Safety Shield:</span>
            <span class="text-muted">Profil sebelumnya tersimpan: <strong class="text-foreground font-mono">{overview.previous_profile}</strong></span>
          </div>
        </div>
        <Button
          variant="outline"
          size="sm"
          onclick={revertPreviousProfile}
          disabled={isReverting}
          class="border-amber-500/50 text-amber-300 hover:bg-amber-500/20 gap-1.5 shrink-0"
        >
          <RotateCcw class="w-3.5 h-3.5" />
          <span>{isReverting ? 'Reverting...' : 'Revert to Previous'}</span>
        </Button>
      </div>
    {/if}
  </Card>

  <!-- LIST OF PROFILES (SW & HW LISTVIEW) -->
  <Card class="p-4 border-2 border-border shadow-neobrutal-sm bg-card space-y-4">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b-2 border-border/60 pb-3">
      <div class="flex items-center gap-2">
        <button
          type="button"
          class="px-3 py-1.5 text-xs font-black uppercase tracking-wider rounded border-2 cursor-pointer transition-all {activeTab === 'sw' ? 'border-accent bg-accent/15 text-accent' : 'border-transparent text-muted hover:text-foreground'}"
          onclick={() => (activeTab = 'sw')}
        >
          SW Carrier Configs ({overview.sw_configs.length})
        </button>
        <button
          type="button"
          class="px-3 py-1.5 text-xs font-black uppercase tracking-wider rounded border-2 cursor-pointer transition-all {activeTab === 'hw' ? 'border-accent bg-accent/15 text-accent' : 'border-transparent text-muted hover:text-foreground'}"
          onclick={() => (activeTab = 'hw')}
        >
          HW Modem Configs ({overview.hw_configs.length})
        </button>
      </div>

      <!-- SEARCH FILTER -->
      <div class="relative w-full sm:w-64">
        <Search class="w-3.5 h-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-muted" />
        <input
          type="text"
          placeholder="Search carrier / mbn..."
          bind:value={searchQuery}
          class="w-full pl-8 pr-2.5 py-1 text-xs bg-card border-2 border-border rounded text-foreground font-mono"
        />
      </div>
    </div>

    <!-- TABLE VIEW -->
    <div class="overflow-x-auto max-h-[500px] overflow-y-auto">
      {#if activeTab === 'sw'}
        {#if filteredSwConfigs.length === 0}
          <div class="p-6 text-center text-xs text-muted font-mono border-2 border-dashed border-border rounded">
            {overview.sw_configs.length === 0 ? 'No MBN files found under /mcfg_sw/generic on modem EFS' : 'No matching carrier profiles found'}
          </div>
        {:else}
          <table class="w-full text-left text-xs font-mono">
            <thead class="sticky top-0 bg-card z-10 border-b-2 border-border">
              <tr class="text-muted">
                <th class="py-2 px-3">Profile Name</th>
                <th class="py-2 px-3">Category / Region</th>
                <th class="py-2 px-3">Modem EFS Path</th>
                <th class="py-2 px-3 text-right">Action</th>
              </tr>
            </thead>
            <tbody>
              {#each filteredSwConfigs as cfg}
                <tr class="border-b border-border/40 hover:bg-card-sub transition-colors {cfg.active ? 'bg-accent/10' : ''}">
                  <td class="py-2 px-3 font-bold text-foreground flex items-center gap-1.5">
                    {#if cfg.active}
                      <CheckCircle class="w-3.5 h-3.5 text-emerald-400 shrink-0" />
                    {/if}
                    <span>{cfg.name}</span>
                  </td>
                  <td class="py-2 px-3 text-muted uppercase text-[10px] font-bold">{cfg.category || 'generic'}</td>
                  <td class="py-2 px-3 text-muted text-[11px] truncate max-w-xs">{cfg.path}</td>
                  <td class="py-2 px-3 text-right">
                    {#if cfg.active}
                      <Badge variant="success">Active</Badge>
                    {:else}
                      <Button
                        size="sm"
                        variant="outline"
                        onclick={() => promptActivate(cfg)}
                        disabled={isActivating}
                        class="px-2 py-0.5 text-[11px]"
                      >
                        Activate
                      </Button>
                    {/if}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        {/if}
      {:else}
        {#if filteredHwConfigs.length === 0}
          <div class="p-6 text-center text-xs text-muted font-mono border-2 border-dashed border-border rounded">
            {overview.hw_configs.length === 0 ? 'No HW configs found under /mcfg_hw/generic' : 'No matching hardware configs'}
          </div>
        {:else}
          <table class="w-full text-left text-xs font-mono">
            <thead class="sticky top-0 bg-card z-10 border-b-2 border-border">
              <tr class="text-muted">
                <th class="py-2 px-3">Config Name</th>
                <th class="py-2 px-3">Modem EFS Path</th>
                <th class="py-2 px-3 text-right">Size</th>
              </tr>
            </thead>
            <tbody>
              {#each filteredHwConfigs as cfg}
                <tr class="border-b border-border/40 hover:bg-card-sub">
                  <td class="py-2 px-3 font-bold text-foreground">{cfg.name}</td>
                  <td class="py-2 px-3 text-muted text-[11px]">{cfg.path}</td>
                  <td class="py-2 px-3 text-right text-muted">{cfg.size} B</td>
                </tr>
              {/each}
            </tbody>
          </table>
        {/if}
      {/if}
    </div>
  </Card>
</div>

<!-- CONFIRMATION MODAL -->
{#if showConfirmModal && pendingConfig}
  <div class="fixed inset-0 z-50 bg-black/70 backdrop-blur-xs flex items-center justify-center p-4">
    <div class="p-5 bg-card border-2 border-border rounded-lg shadow-neobrutal-md max-w-md w-full space-y-4">
      <div class="flex items-center gap-2 border-b-2 border-border/60 pb-3">
        <AlertTriangle class="w-5 h-5 text-amber-400" />
        <h4 class="text-sm font-bold uppercase tracking-wider text-foreground">
          Konfirmasi Pergantian Profil MBN
        </h4>
      </div>

      <div class="space-y-2 text-xs text-foreground">
        <p>Anda akan mengaktifkan profil baseband berikut pada <strong>SIM {selectedSlot + 1}</strong>:</p>
        <div class="p-2.5 bg-card-sub border-2 border-border rounded font-mono break-all font-bold">
          {pendingConfig.name}
        </div>
        <p class="text-muted">
          Profil saat ini ({overview.active_profile || 'Default'}) akan dicatat di penyimpanan rollback otomatis. Jika modem kehilangan sinyal sesudah pergantian, Anda dapat menggunakan tombol <strong>Revert to Previous</strong>.
        </p>
      </div>

      <label class="flex items-center gap-2 text-xs font-bold text-foreground cursor-pointer pt-2 border-t border-border/60">
        <input type="checkbox" bind:checked={restartSSR} class="w-4 h-4 accent-accent" />
        <span>Jalankan Restart Modem (SSR) sesudah aktivasi</span>
      </label>

      <div class="flex items-center justify-end gap-2 pt-2">
        <Button variant="outline" size="sm" onclick={() => (showConfirmModal = false)}>
          Batal
        </Button>
        <Button variant="default" size="sm" onclick={confirmActivate} disabled={isActivating}>
          {isActivating ? 'Mengaktifkan...' : 'Ya, Aktifkan Profil'}
        </Button>
      </div>
    </div>
  </div>
{/if}
