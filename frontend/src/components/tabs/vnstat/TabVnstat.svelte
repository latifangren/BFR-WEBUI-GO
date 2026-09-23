<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Activity,
    RefreshCw,
    RotateCcw,
    AlertTriangle,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import type { StatsResponse } from '../../../types/vnstat'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Modal from '../../ui/Modal.svelte'
  import VnstatSummaryCards from './VnstatSummaryCards.svelte'
  import VnstatHistoryChart from './VnstatHistoryChart.svelte'
  import VnstatInterfaceTable from './VnstatInterfaceTable.svelte'

  let stats = $state<StatsResponse | null>(null)
  let isLoading = $state(false)
  let isResetting = $state(false)
  let isResetModalOpen = $state(false)

  onMount(async () => {
    await fetchVnstat()
  })

  async function fetchVnstat() {
    try {
      isLoading = true
      const res = await api.get<StatsResponse>('/api/vnstat/stats')
      stats = res
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to fetch vnstat stats')
    } finally {
      isLoading = false
    }
  }

  function openResetModal() {
    isResetModalOpen = true
  }

  function closeResetModal() {
    if (isResetting) return
    isResetModalOpen = false
  }

  async function confirmReset() {
    try {
      isResetting = true
      await api.post('/api/vnstat/reset')
      toastStore.success('Vnstat traffic database counters reset successfully.')
      closeResetModal()
      await fetchVnstat()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to reset vnstat database')
    } finally {
      isResetting = false
    }
  }

  function formatTime(dateStr?: string): string {
    if (!dateStr) return 'Never'
    try {
      const d = new Date(dateStr)
      return d.toLocaleTimeString('en-US', {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      })
    } catch {
      return dateStr
    }
  }
</script>

<div class="space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
    <div class="flex items-center gap-2 flex-wrap">
      <Activity class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        Vnstat Bandwidth Accounting
      </h2>
      <Badge variant="default" class="text-[10px]">
        Updated: {formatTime(stats?.last_updated)}
      </Badge>
    </div>

    <div class="flex items-center gap-2">
      <!-- Reset Database Button (Triggers Safety Modal) -->
      <Button
        variant="outline"
        size="sm"
        disabled={isResetting}
        onclick={openResetModal}
        class="border-red-900/60 hover:bg-red-950/30 text-red-400 font-mono text-xs"
        title="Reset all interface bandwidth counters"
      >
        <RotateCcw class="w-3.5 h-3.5 mr-1" />
        <span>Reset Database</span>
      </Button>

      <!-- Refresh Button -->
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoading}
        onclick={fetchVnstat}
        title="Refresh bandwidth telemetry"
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoading ? 'animate-spin' : ''}" />
        <span>Refresh</span>
      </Button>
    </div>
  </div>

  <!-- 1. 4-Cards Summary Grid -->
  <VnstatSummaryCards {stats} />

  <!-- 2. 7-Day History Chart -->
  <VnstatHistoryChart history={stats?.daily_history} />

  <!-- 3. Per-Interface Breakdown Table -->
  <VnstatInterfaceTable {stats} />
</div>

<!-- Safety Reset Confirmation Modal -->
<Modal
  open={isResetModalOpen}
  title="RESET VNSTAT DATABASE"
  onclose={closeResetModal}
>
  <div class="space-y-3.5 text-xs font-mono">
    <div class="p-3.5 bg-red-950/40 border-2 border-red-600 rounded text-red-200 space-y-2">
      <div class="flex items-center gap-2 text-red-400 font-bold text-sm uppercase">
        <AlertTriangle size={18} class="shrink-0 text-red-500 animate-pulse" />
        <span>Database Reset Warning</span>
      </div>
      <p class="leading-relaxed">
        Are you sure you want to reset all bandwidth statistics? This will permanently erase all interface counters (cellular, Wi-Fi, hotspot) and historical daily records. This operation <strong class="text-white underline">cannot be undone</strong>!
      </p>
    </div>
  </div>

  {#snippet footer()}
    <div class="flex items-center justify-end gap-2 w-full">
      <Button
        variant="ghost"
        disabled={isResetting}
        onclick={closeResetModal}
      >
        Cancel
      </Button>

      <Button
        variant="danger"
        disabled={isResetting}
        onclick={confirmReset}
      >
        {#if isResetting}
          <RefreshCw size={14} class="animate-spin mr-1.5" />
          <span>Resetting...</span>
        {:else}
          <RotateCcw size={14} class="mr-1.5" />
          <span>Reset Database Now</span>
        {/if}
      </Button>
    </div>
  {/snippet}
</Modal>
