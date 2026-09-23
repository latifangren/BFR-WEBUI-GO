<script lang="ts">
  import {
    ShieldCheck,
    Zap,
    RefreshCw,
  } from '@lucide/svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'

  interface Props {
    isBackingUp: string | null
    autoDownload: boolean
    onToggleAutoDownload: (val: boolean) => void
    onBackupGroup: (group: 'radio_efs' | 'boot_core') => void
  }

  let {
    isBackingUp,
    autoDownload,
    onToggleAutoDownload,
    onBackupGroup,
  }: Props = $props()
</script>

<Card
  title="1-CLICK QUICK BACKUPS"
  subtitle="Instant safety archives for cellular/IMEI calibrations and core boot images"
  tone="peach"
>
  {#snippet action()}
    <label class="inline-flex items-center gap-2 cursor-pointer select-none">
      <input
        type="checkbox"
        checked={autoDownload}
        onchange={(e) => onToggleAutoDownload((e.target as HTMLInputElement).checked)}
        class="rounded bg-card-sub border-border text-accent focus:ring-accent w-4 h-4 cursor-pointer"
      />
      <span class="text-xs font-mono font-bold text-foreground uppercase tracking-wider">
        AUTO-DOWNLOAD (.ZIP)
      </span>
    </label>
  {/snippet}

  <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
    <!-- Radio & EFS Quick Backup -->
    <div
      class="p-4 bg-card-sub border-2 border-border hover:border-red-500/50 rounded flex flex-col justify-between space-y-3 transition-colors"
    >
      <div class="space-y-1">
        <div class="flex items-center justify-between">
          <span class="inline-flex items-center gap-1.5 text-xs font-mono font-bold uppercase tracking-wider text-red-400">
            <ShieldCheck size={16} />
            NVRAM / IMEI / MODEM
          </span>
          <span class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-red-950/60 text-red-400 border border-red-800">
            HIGH PRIORITY
          </span>
        </div>
        <h4 class="text-sm font-bold font-mono text-foreground">Backup Radio & EFS (.ZIP)</h4>
        <p class="text-xs text-muted">
          Packages modemst1, modemst2, fsg, fsc, efs, nvram, and baseband calibrations into a single compressed .zip file.
        </p>
      </div>

      <Button
        variant="secondary"
        size="sm"
        fullWidth
        disabled={isBackingUp !== null}
        onclick={() => onBackupGroup('radio_efs')}
        class="border-red-900/60 hover:bg-red-950/30 text-red-300"
      >
        {#if isBackingUp === 'radio_efs'}
          <RefreshCw size={14} class="animate-spin mr-1.5" />
          <span>Packaging Radio & EFS (.ZIP)...</span>
        {:else}
          <ShieldCheck size={14} class="mr-1.5 text-red-400" />
          <span>Backup Radio & EFS (.ZIP)</span>
        {/if}
      </Button>
    </div>

    <!-- Boot & Kernel Core Quick Backup -->
    <div
      class="p-4 bg-card-sub border-2 border-border hover:border-emerald-500/50 rounded flex flex-col justify-between space-y-3 transition-colors"
    >
      <div class="space-y-1">
        <div class="flex items-center justify-between">
          <span class="inline-flex items-center gap-1.5 text-xs font-mono font-bold uppercase tracking-wider text-emerald-400">
            <Zap size={16} />
            KERNEL / RECOVERY / VBMETA
          </span>
          <span class="text-[10px] font-mono px-1.5 py-0.5 rounded bg-emerald-950/60 text-emerald-400 border border-emerald-800">
            BOOT CORE
          </span>
        </div>
        <h4 class="text-sm font-bold font-mono text-foreground">Backup Boot & Kernel Core (.ZIP)</h4>
        <p class="text-xs text-muted">
          Packages active boot, init_boot, vendor_boot, recovery, dtbo, and vbmeta into a single compressed .zip file.
        </p>
      </div>

      <Button
        variant="secondary"
        size="sm"
        fullWidth
        disabled={isBackingUp !== null}
        onclick={() => onBackupGroup('boot_core')}
        class="border-emerald-900/60 hover:bg-emerald-950/30 text-emerald-300"
      >
        {#if isBackingUp === 'boot_core'}
          <RefreshCw size={14} class="animate-spin mr-1.5" />
          <span>Packaging Boot Core (.ZIP)...</span>
        {:else}
          <Zap size={14} class="mr-1.5 text-emerald-400" />
          <span>Backup Boot & Kernel Core (.ZIP)</span>
        {/if}
      </Button>
    </div>
  </div>
</Card>
