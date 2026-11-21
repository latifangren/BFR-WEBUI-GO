<script lang="ts">
  import {
    HardDriveDownload,
    Flame,
    RefreshCw,
    Search,
    X,
  } from '@lucide/svelte'
  import type { PartitionInfo, PartitionCategory } from '../../../types/partition'
  import { LL } from '../../../i18n/i18n-svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'

  interface Props {
    partitions: PartitionInfo[]
    activeFilter: string
    searchQuery: string
    isBackingUp: string | null
    onSearchChange: (q: string) => void
    onBackupPartition: (p: PartitionInfo) => void
    onOpenFlashModal: (p: PartitionInfo) => void
  }

  let {
    partitions,
    activeFilter,
    searchQuery,
    isBackingUp,
    onSearchChange,
    onBackupPartition,
    onOpenFlashModal,
  }: Props = $props()

  const FLASH_WHITELIST = new Set([
    'boot',
    'init_boot',
    'vendor_boot',
    'recovery',
    'dtbo',
    'vbmeta',
    'vbmeta_system',
    'vbmeta_vendor',
  ])

  function isFlashable(name: string): boolean {
    const lower = name.toLowerCase()
    const baseName =
      lower.endsWith('_a') || lower.endsWith('_b') ? lower.slice(0, -2) : lower
    return FLASH_WHITELIST.has(baseName)
  }

  const filteredPartitions = $derived(
    partitions.filter((p) => {
      if (activeFilter !== 'all' && activeFilter !== 'backups') {
        if (p.category !== activeFilter) return false
      }
      if (searchQuery.trim()) {
        const q = searchQuery.toLowerCase().trim()
        const matchesName = p.name.toLowerCase().includes(q)
        const matchesPath = p.device_path.toLowerCase().includes(q)
        const matchesCategory = p.category.toLowerCase().includes(q)
        return matchesName || matchesPath || matchesCategory
      }
      return true
    })
  )

  function getCategoryBadge(category: PartitionCategory) {
    switch (category) {
      case 'radio_efs':
        return { label: $LL.partitions.table.categoryRadio(), class: 'bg-red-950/60 text-red-300 border-red-800' }
      case 'boot_recovery':
        return { label: $LL.partitions.table.categoryBoot(), class: 'bg-emerald-950/60 text-emerald-300 border-emerald-800' }
      case 'system_os':
        return { label: $LL.partitions.table.categorySystem(), class: 'bg-cyan-950/60 text-cyan-300 border-cyan-800' }
      case 'other':
      default:
        return { label: $LL.partitions.table.categoryOther(), class: 'bg-card-sub text-muted border-border' }
    }
  }
</script>

<div class="space-y-4">
  <!-- Search Bar Header -->
  <div class="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 bg-card-sub p-2.5 border border-border rounded">
    <div class="text-xs font-mono text-muted flex items-center gap-1.5">
      <span class="font-bold text-foreground">{filteredPartitions.length}</span>
      <span>{$LL.partitions.table.countLine({ total: partitions.length })}</span>
    </div>

    <!-- Search Input -->
    <div class="relative w-full sm:w-72">
      <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-muted">
        <Search size={14} />
      </div>
      <input
        type="text"
        value={searchQuery}
        oninput={(e) => onSearchChange((e.target as HTMLInputElement).value)}
        placeholder={$LL.partitions.table.searchPlaceholder()}
        class="neo-input w-full bg-card border border-border rounded pl-9 pr-8 py-1.5 text-xs font-mono text-foreground placeholder:text-muted focus:outline-none focus:border-accent"
      />
      {#if searchQuery}
        <button
          type="button"
          onclick={() => onSearchChange('')}
          class="absolute inset-y-0 right-0 pr-2.5 flex items-center text-muted hover:text-foreground cursor-pointer"
          title={$LL.partitions.table.clearSearchTitle()}
        >
          <X size={13} />
        </button>
      {/if}
    </div>
  </div>

  <!-- Partitions Table Card -->
  <Card
    title={$LL.partitions.table.title()}
    subtitle={$LL.partitions.table.subtitle({ filtered: filteredPartitions.length, total: partitions.length })}
    tone="none"
  >
    {#if filteredPartitions.length === 0}
      <div class="p-8 text-center border-2 border-dashed border-border rounded bg-card-sub">
        <p class="font-mono text-sm text-foreground font-bold">{$LL.partitions.table.emptyTitle()}</p>
        <p class="text-xs text-muted mt-1">
          {$LL.partitions.table.emptyBody()}
        </p>
      </div>
    {:else}
      <div class="overflow-x-auto border border-border rounded custom-scrollbar">
        <table class="w-full text-left border-collapse text-xs font-mono">
          <thead>
            <tr class="bg-card-sub border-b border-border text-muted uppercase text-[10px] tracking-wider">
              <th class="p-3">{$LL.partitions.table.thName()}</th>
              <th class="p-3">{$LL.partitions.table.thSlot()}</th>
              <th class="p-3">{$LL.partitions.table.thCategory()}</th>
              <th class="p-3">{$LL.partitions.table.thSize()}</th>
              <th class="p-3 hidden md:table-cell">{$LL.partitions.table.thDevicePath()}</th>
              <th class="p-3 text-right">{$LL.partitions.table.thActions()}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            {#each filteredPartitions as part (part.name)}
              {@const categoryBadge = getCategoryBadge(part.category)}
              <tr class="hover:bg-card-sub/50 transition-colors">
                <!-- Partition Name & Critical Tag -->
                <td class="p-3">
                  <div class="flex items-center gap-2">
                    <span class="font-bold text-foreground text-sm tracking-wide">
                      {part.name}
                    </span>
                    {#if part.is_critical}
                      <span
                        class="px-1.5 py-0.2 rounded text-[9px] font-mono font-bold uppercase bg-red-950/60 text-red-400 border border-red-800"
                        title={$LL.partitions.table.criticalTitle()}
                      >
                        {$LL.partitions.table.critical()}
                      </span>
                    {/if}
                  </div>
                </td>

                <!-- Slot Badge -->
                <td class="p-3 whitespace-nowrap">
                  {#if part.slot}
                    {#if part.is_active_slot}
                      <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-mono font-bold uppercase bg-emerald-950/60 text-emerald-400 border border-emerald-800">
                        <span class="h-1.5 w-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
                        _{part.slot.toUpperCase()}
                      </span>
                    {:else}
                      <span class="inline-flex items-center px-2 py-0.5 rounded text-[10px] font-mono font-bold uppercase bg-card-sub text-muted border border-border">
                        _{part.slot.toUpperCase()}
                      </span>
                    {/if}
                  {:else}
                    <span class="text-muted text-[10px]">-</span>
                  {/if}
                </td>

                <!-- Category Badge -->
                <td class="p-3 whitespace-nowrap">
                  <span class="inline-flex items-center px-2 py-0.5 rounded text-[10px] font-mono font-bold uppercase border {categoryBadge.class}">
                    {categoryBadge.label}
                  </span>
                </td>

                <!-- Size Human -->
                <td class="p-3 whitespace-nowrap text-foreground font-bold">
                  {part.size_human}
                </td>

                <!-- Device Path -->
                <td class="p-3 hidden md:table-cell text-muted text-[11px] font-mono truncate max-w-xs" title={part.device_path}>
                  {part.device_path}
                </td>

                <!-- Actions -->
                <td class="p-3 whitespace-nowrap text-right">
                  <div class="inline-flex items-center gap-1.5 justify-end">
                    <Button
                      variant="secondary"
                      size="sm"
                      disabled={isBackingUp === part.name}
                      onclick={() => onBackupPartition(part)}
                      title={$LL.partitions.table.backupTitle()}
                    >
                      {#if isBackingUp === part.name}
                        <RefreshCw size={12} class="animate-spin mr-1" />
                        <span>{$LL.partitions.table.backingUp()}</span>
                      {:else}
                        <HardDriveDownload size={12} class="mr-1" />
                        <span>{$LL.partitions.table.backup()}</span>
                      {/if}
                    </Button>

                    {#if isFlashable(part.name)}
                      <Button
                        variant="danger"
                        size="sm"
                        onclick={() => onOpenFlashModal(part)}
                        title={$LL.partitions.table.flashTitle()}
                      >
                        <Flame size={12} class="mr-1" />
                        <span>{$LL.partitions.table.flash()}</span>
                      </Button>
                    {:else}
                      <button
                        type="button"
                        disabled
                        class="px-2.5 py-1 text-xs font-mono font-bold rounded border border-border bg-card-sub/40 text-muted/40 cursor-not-allowed opacity-40 select-none"
                        title={$LL.partitions.table.flashDisabledTitle()}
                      >
                        {$LL.partitions.table.flash()}
                      </button>
                    {/if}
                  </div>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </Card>
</div>
