<script lang="ts">
  import {
    Shield,
    Play,
    Square,
    RotateCcw,
    Activity,
    Cpu,
    Eye,
    ExternalLink,
    Code2,
  } from '@lucide/svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'

  interface Props {
    isRunning: boolean
    activeCore: string
    activePid: number
    memoryUsage: string
    coreVersion: string
    watchdog: boolean
    isLoading: boolean
    onAction: (action: string) => Promise<void>
    onSwitchCore: (core: string) => Promise<void>
    onToggleWatchdog: () => Promise<void>
  }

  let {
    isRunning,
    activeCore,
    activePid,
    memoryUsage,
    coreVersion,
    watchdog,
    isLoading,
    onAction,
    onSwitchCore,
    onToggleWatchdog,
  }: Props = $props()

  const availableCores = [
    { id: 'mihomo', label: 'Mihomo', desc: 'Clash.Meta', repo: 'https://github.com/MetaCubeX/mihomo' },
    { id: 'sing-box', label: 'Sing-box', desc: 'Next-gen', repo: 'https://github.com/SagerNet/sing-box' },
    { id: 'clash', label: 'Clash', desc: 'Legacy', repo: 'https://github.com/Dreamacro/clash' },
  ]
</script>

<Card
  title="Box Core Engine Manager"
  subtitle="Multi-core proxy daemon lifecycle, memory telemetry & watchdog supervisor"
  tone="ice"
>
  {#snippet action()}
    <div class="flex items-center gap-2 flex-wrap">
      <button
        type="button"
        onclick={onToggleWatchdog}
        class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded text-xs font-mono font-bold border transition-colors cursor-pointer {watchdog ? 'bg-accent/15 border-accent text-accent' : 'bg-card border-border text-muted hover:text-foreground'}"
        title="Automatically restart core if it crashes or gets killed"
      >
        <Eye class="w-3.5 h-3.5 {watchdog ? 'text-accent' : 'text-muted'}" />
        <span>Watchdog: {watchdog ? 'ON' : 'OFF'}</span>
      </button>

      {#if isRunning}
        <Badge variant="success" class="flex items-center gap-1.5">
          <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
          <span>CORE RUNNING</span>
        </Badge>
      {:else}
        <Badge variant="danger" class="flex items-center gap-1.5">
          <span class="w-2 h-2 rounded-full bg-red-400"></span>
          <span>STOPPED</span>
        </Badge>
      {/if}
    </div>
  {/snippet}

  <div class="space-y-4 font-mono text-xs">
    <!-- Core Engine Selector Pills & Official Source Links -->
    <div>
      <div class="flex items-center justify-between mb-2">
        <span class="text-[10px] text-muted uppercase font-bold block">Active Core Binary</span>
        <div class="flex items-center gap-2">
          <a
            href="https://github.com/MetaCubeX/mihomo/releases"
            target="_blank"
            rel="noreferrer"
            class="inline-flex items-center gap-1 text-[10px] text-accent hover:underline font-bold"
            title="Official MetaCubeX Mihomo Core Releases"
          >
            <Code2 class="w-3 h-3" />
            <span>MetaCubeX/mihomo</span>
            <ExternalLink class="w-2.5 h-2.5" />
          </a>
        </div>
      </div>
      <div class="grid grid-cols-3 gap-2">
        {#each availableCores as c}
          <button
            type="button"
            onclick={() => onSwitchCore(c.id)}
            disabled={isLoading}
            class="neo-button p-2 text-center rounded border-2 transition-all cursor-pointer {activeCore.toLowerCase().includes(c.id) ? 'border-accent bg-accent/15 text-accent font-bold shadow-neobrutal-sm' : 'border-border bg-card-sub text-muted hover:text-foreground'}"
          >
            <div class="font-bold text-xs">{c.label}</div>
            <div class="text-[10px] opacity-75">{c.desc}</div>
          </button>
        {/each}
      </div>
    </div>

    <!-- Telemetry Cards -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
      <div class="p-2.5 bg-card-sub border border-border rounded space-y-0.5">
        <span class="text-[10px] text-muted uppercase font-bold">Process PID</span>
        <div class="font-bold text-foreground text-sm">{isRunning && activePid > 0 ? activePid : '-'}</div>
      </div>

      <div class="p-2.5 bg-card-sub border border-border rounded space-y-0.5">
        <span class="text-[10px] text-muted uppercase font-bold">Memory RSS</span>
        <div class="font-bold text-accent text-sm">{isRunning ? memoryUsage : '-'}</div>
      </div>

      <div class="p-2.5 bg-card-sub border border-border rounded space-y-0.5">
        <span class="text-[10px] text-muted uppercase font-bold">Version</span>
        <div class="font-bold text-foreground text-xs truncate" title={coreVersion}>{coreVersion || '-'}</div>
      </div>

      <div class="p-2.5 bg-card-sub border border-border rounded space-y-0.5">
        <span class="text-[10px] text-muted uppercase font-bold">Service Controller</span>
        <div class="font-bold text-emerald-400 text-xs">BFM box.service</div>
      </div>
    </div>

    <!-- Service Action Buttons -->
    <div class="pt-2 border-t border-border flex items-center justify-end gap-2 flex-wrap">
      {#if isRunning}
        <Button
          variant="danger"
          size="sm"
          disabled={isLoading}
          onclick={() => onAction('stop')}
        >
          <Square class="w-3.5 h-3.5 mr-1" />
          <span>Stop Core</span>
        </Button>

        <Button
          variant="secondary"
          size="sm"
          disabled={isLoading}
          onclick={() => onAction('restart')}
        >
          <RotateCcw class="w-3.5 h-3.5 mr-1" />
          <span>Restart</span>
        </Button>
      {:else}
        <Button
          variant="primary"
          size="sm"
          disabled={isLoading}
          onclick={() => onAction('start')}
        >
          <Play class="w-3.5 h-3.5 mr-1" />
          <span>Start Core</span>
        </Button>
      {/if}
    </div>
  </div>
</Card>
