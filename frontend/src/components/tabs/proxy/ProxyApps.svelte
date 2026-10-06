<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Smartphone,
    Search,
    Check,
    RotateCcw,
    Sliders,
    RefreshCw,
    X,
  } from '@lucide/svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Input from '../../ui/Input.svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'

  interface InstalledApp {
    package_name: string
    label: string
    is_system: boolean
    is_proxy: boolean
  }

  let mode = $state<'blacklist' | 'whitelist'>('blacklist')
  let apps = $state<InstalledApp[]>([])
  let selected = $state<Record<string, boolean>>({})
  let searchQuery = $state('')
  let filterType = $state<'user' | 'all'>('user')
  let isLoading = $state(false)
  let isSaving = $state(false)

  onMount(async () => {
    await fetchApps()
  })

  async function fetchApps() {
    try {
      isLoading = true
      const res = await api.get<{
        mode: 'blacklist' | 'whitelist'
        selected: string[]
        apps: InstalledApp[]
      }>('/api/proxy/apps')

      if (res) {
        if (res.mode) mode = res.mode
        if (Array.isArray(res.apps)) apps = res.apps

        const selMap: Record<string, boolean> = {}
        if (Array.isArray(res.selected)) {
          for (const p of res.selected) {
            selMap[p] = true
          }
        }
        selected = selMap
      }
    } catch {
      // Ignored
    } finally {
      isLoading = false
    }
  }

  const filteredApps = $derived.by(() => {
    const q = searchQuery.trim().toLowerCase()
    return apps.filter((a) => {
      if (filterType === 'user' && a.is_system) {
        return false
      }
      if (!q) return true
      return a.package_name.toLowerCase().includes(q) || a.label.toLowerCase().includes(q)
    })
  })

  const selectedCount = $derived(
    Object.values(selected).filter(Boolean).length
  )

  function toggleApp(pkg: string) {
    selected[pkg] = !selected[pkg]
  }

  function selectAllUserApps() {
    for (const a of apps) {
      if (!a.is_system) {
        selected[a.package_name] = true
      }
    }
  }

  function clearSelection() {
    selected = {}
  }

  async function saveAppProxy() {
    try {
      isSaving = true
      const packages = Object.keys(selected).filter((k) => selected[k])
      await api.post('/api/proxy/apps', {
        mode,
        packages,
      })
      toastStore.success(`App proxy policy saved (${mode}: ${packages.length} apps)`)
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to save app proxy policy')
      isSaving = false
    }
  }
</script>

<Card
  title="Per-App Proxy Routing (package.list.cfg)"
  subtitle="Filter which Android apps are routed through proxy or bypassed directly (Blacklist / Whitelist)"
  tone="ice"
>
  {#snippet action()}
    <div class="flex items-center gap-2 flex-wrap">
      <Button
        variant="primary"
        size="sm"
        disabled={isSaving || isLoading}
        onclick={saveAppProxy}
      >
        <Check class="w-3.5 h-3.5 mr-1" />
        <span>{isSaving ? 'Saving...' : 'Apply Policy'}</span>
      </Button>
    </div>
  {/snippet}

  <div class="space-y-4 font-mono text-xs">
    <!-- Mode Switcher: Blacklist vs Whitelist -->
    <div class="p-3 bg-card-sub border border-border rounded-lg space-y-2">
      <span class="text-[10px] text-muted uppercase font-bold block">App Routing Policy</span>
      <div class="grid grid-cols-2 gap-2">
        <button
          type="button"
          onclick={() => (mode = 'blacklist')}
          class="p-2.5 rounded border-2 text-left cursor-pointer transition-all {mode === 'blacklist' ? 'border-accent bg-accent/15 text-accent font-bold shadow-neobrutal-sm' : 'border-border bg-card text-muted hover:text-foreground'}"
        >
          <div class="font-bold text-xs">Blacklist Mode</div>
          <div class="text-[10px] opacity-75 mt-0.5">Proxy ALL apps EXCEPT checked packages</div>
        </button>

        <button
          type="button"
          onclick={() => (mode = 'whitelist')}
          class="p-2.5 rounded border-2 text-left cursor-pointer transition-all {mode === 'whitelist' ? 'border-accent bg-accent/15 text-accent font-bold shadow-neobrutal-sm' : 'border-border bg-card text-muted hover:text-foreground'}"
        >
          <div class="font-bold text-xs">Whitelist Mode</div>
          <div class="text-[10px] opacity-75 mt-0.5">Proxy ONLY checked packages directly</div>
        </button>
      </div>
    </div>

    <!-- Search & Quick Selection Bar -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 pt-2 border-t border-border">
      <div class="flex items-center gap-2 flex-1">
        <div class="relative flex-1">
          <Input placeholder="Search apps or package name..." bind:value={searchQuery} />
        </div>
        <div class="flex items-center gap-1">
          <button
            type="button"
            onclick={() => (filterType = 'user')}
            class="px-2.5 py-1.5 rounded border text-[10px] font-bold cursor-pointer transition-colors {filterType === 'user' ? 'bg-accent/15 border-accent text-accent' : 'bg-card-sub border-border text-muted'}"
          >
            User Apps
          </button>
          <button
            type="button"
            onclick={() => (filterType = 'all')}
            class="px-2.5 py-1.5 rounded border text-[10px] font-bold cursor-pointer transition-colors {filterType === 'all' ? 'bg-accent/15 border-accent text-accent' : 'bg-card-sub border-border text-muted'}"
          >
            All Apps
          </button>
        </div>
      </div>

      <div class="flex items-center gap-1.5 justify-end">
        <span class="text-[11px] text-muted font-bold mr-1">
          {selectedCount} selected
        </span>
        <button
          type="button"
          onclick={selectAllUserApps}
          class="px-2 py-1 rounded bg-card-sub border border-border text-[10px] text-foreground hover:border-accent cursor-pointer"
        >
          Select All User Apps
        </button>
        <button
          type="button"
          onclick={clearSelection}
          class="px-2 py-1 rounded bg-card-sub border border-border text-[10px] text-muted hover:text-red-400 cursor-pointer"
        >
          Clear Selection
        </button>
      </div>
    </div>

    <!-- Installed Apps Scrollable Grid -->
    <div class="max-h-[360px] overflow-y-auto space-y-1.5 pr-1 border border-border rounded-lg p-2 bg-card">
      {#if isLoading && apps.length === 0}
        <div class="p-8 text-center text-muted">
          <RefreshCw class="w-5 h-5 animate-spin mx-auto mb-2 text-accent" />
          <span>Scanning Android installed application packages...</span>
        </div>
      {:else if filteredApps.length === 0}
        <div class="p-6 text-center text-muted">
          No apps matched your filter query.
        </div>
      {:else}
        {#each filteredApps as app (app.package_name)}
          <label
            class="p-2.5 rounded-lg border transition-colors flex items-center justify-between gap-3 cursor-pointer select-none {selected[app.package_name] ? 'bg-accent/10 border-accent font-bold text-foreground' : 'bg-card-sub border-border/80 text-muted hover:border-border'}"
          >
            <div class="flex items-center gap-2.5 min-w-0">
              <input
                type="checkbox"
                checked={!!selected[app.package_name]}
                onchange={() => toggleApp(app.package_name)}
                class="w-4 h-4 accent-accent cursor-pointer shrink-0"
              />
              <div class="min-w-0">
                <div class="font-bold text-foreground text-xs truncate">{app.label}</div>
                <div class="text-[10px] text-muted truncate">{app.package_name}</div>
              </div>
            </div>

            {#if app.is_system}
              <span class="px-1.5 py-0.2 rounded bg-card border border-border text-[9px] text-muted shrink-0">
                System
              </span>
            {/if}
          </label>
        {/each}
      {/if}
    </div>
  </div>
</Card>
