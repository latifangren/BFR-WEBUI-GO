<script lang="ts">
  import {
    RefreshCw,
    Cpu,
    Terminal,
    Layers,
    HardDrive,
  } from '@lucide/svelte'
  import type { SystemSlotInfo } from '../../../types/partition'
  import { LL } from '../../../i18n/i18n-svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'

  interface Props {
    slotInfo: SystemSlotInfo | null
    totalPartitions: number
    backupCount: number
    isLoading: boolean
    onRefresh: () => void
  }

  let {
    slotInfo,
    totalPartitions,
    backupCount,
    isLoading,
    onRefresh,
  }: Props = $props()
</script>

<Card
  title={$LL.partitions.header.title()}
  subtitle={$LL.partitions.header.subtitle()}
  tone="ice"
>
  {#snippet action()}
    <Button
      variant="outline"
      size="sm"
      disabled={isLoading}
      onclick={onRefresh}
      title={$LL.partitions.header.refreshTitle()}
    >
      <RefreshCw size={14} class={isLoading ? 'animate-spin' : ''} />
      <span class="hidden sm:inline ml-1.5">{$LL.common.refresh()}</span>
    </Button>
  {/snippet}

  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
    <!-- Active Slot Info -->
    <div class="p-3 bg-card-sub border border-border rounded flex flex-col justify-between">
      <span class="text-[11px] font-mono font-bold uppercase tracking-wider text-muted flex items-center gap-1.5">
        <Cpu size={13} class="text-accent" />
        {$LL.partitions.header.slotArchitecture()}
      </span>
      <div class="mt-2 flex items-center gap-2">
        {#if slotInfo?.is_ab_device}
          <span
            class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded text-xs font-mono font-bold tracking-wider uppercase border {slotInfo.current_slot === 'a' || slotInfo.current_slot === 'b' ? 'bg-emerald-950/60 text-emerald-400 border-emerald-800' : 'bg-card text-foreground border-border'}"
          >
            <span class="h-1.5 w-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
            {$LL.partitions.header.activeSlot({ slot: slotInfo.current_slot.toUpperCase() })}
          </span>
        {:else if slotInfo}
          <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-mono font-bold uppercase tracking-wider border bg-card text-muted border-border">
            {$LL.partitions.header.nonAbLegacy()}
          </span>
        {:else}
          <span class="text-xs font-mono text-muted">{$LL.partitions.header.detecting()}</span>
        {/if}
      </div>
    </div>

    <!-- Block Device Root Path -->
    <div class="p-3 bg-card-sub border border-border rounded flex flex-col justify-between">
      <span class="text-[11px] font-mono font-bold uppercase tracking-wider text-muted flex items-center gap-1.5">
        <Terminal size={13} class="text-accent" />
        {$LL.partitions.header.blockDeviceRoot()}
      </span>
      <div
        class="mt-2 font-mono text-xs text-foreground truncate"
        title={slotInfo?.block_device_root || '/dev/block/by-name'}
      >
        {slotInfo?.block_device_root || '/dev/block/by-name'}
      </div>
    </div>

    <!-- Total Partition Count -->
    <div class="p-3 bg-card-sub border border-border rounded flex flex-col justify-between">
      <span class="text-[11px] font-mono font-bold uppercase tracking-wider text-muted flex items-center gap-1.5">
        <Layers size={13} class="text-accent" />
        {$LL.partitions.header.discoveredPartitions()}
      </span>
      <div class="mt-2 flex items-baseline gap-2">
        <span class="font-mono text-xl font-bold text-foreground">
          {slotInfo?.total_partitions || totalPartitions}
        </span>
        <span class="text-[10px] font-mono text-muted uppercase">{$LL.partitions.header.blocksDetected()}</span>
      </div>
    </div>

    <!-- Saved Backups Count -->
    <div class="p-3 bg-card-sub border border-border rounded flex flex-col justify-between">
      <span class="text-[11px] font-mono font-bold uppercase tracking-wider text-muted flex items-center gap-1.5">
        <HardDrive size={13} class="text-accent" />
        {$LL.partitions.header.savedArchives()}
      </span>
      <div class="mt-2 flex items-baseline gap-2">
        <span class="font-mono text-xl font-bold text-foreground">
          {backupCount}
        </span>
        <span class="text-[10px] font-mono text-muted uppercase">{$LL.partitions.header.storedFiles()}</span>
      </div>
    </div>
  </div>
</Card>
