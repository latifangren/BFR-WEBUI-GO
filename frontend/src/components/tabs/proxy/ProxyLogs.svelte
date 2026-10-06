<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import {
    Terminal,
    Pause,
    Play,
    Trash2,
    ChevronDown,
    ChevronUp,
    Search,
  } from '@lucide/svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Input from '../../ui/Input.svelte'

  let logs = $state<string[]>([])
  let isStreaming = $state(true)
  let isExpanded = $state(true)
  let autoScroll = $state(true)
  let searchFilter = $state('')
  let eventSource: EventSource | null = null
  let logContainer = $state<HTMLDivElement | null>(null)

  onMount(() => {
    connectLogs()
  })

  onDestroy(() => {
    disconnectLogs()
  })

  function connectLogs() {
    if (eventSource) return
    eventSource = new EventSource('/api/proxy/logs')
    eventSource.onmessage = (event) => {
      if (!isStreaming) return
      const text = event.data?.trim()
      if (text) {
        logs.push(text)
        if (logs.length > 300) {
          logs.shift()
        }
        if (autoScroll && logContainer) {
          setTimeout(() => {
            if (logContainer) logContainer.scrollTop = logContainer.scrollHeight
          }, 20)
        }
      }
    }
  }

  function disconnectLogs() {
    if (eventSource) {
      eventSource.close()
      eventSource = null
    }
  }

  function toggleStreaming() {
    isStreaming = !isStreaming
  }

  function clearLogs() {
    logs = []
  }

  const filteredLogs = $derived.by(() => {
    const q = searchFilter.trim().toLowerCase()
    if (!q) return logs
    return logs.filter((l) => l.toLowerCase().includes(q))
  })
</script>

<Card
  title="Real-Time Proxy Daemon Logs"
  subtitle="Live streaming stdout/stderr console logs from Mihomo / Sing-box core process"
  tone="ice"
>
  {#snippet action()}
    <div class="flex items-center gap-1.5 flex-wrap">
      <Button
        variant="outline"
        size="sm"
        onclick={toggleStreaming}
        title={isStreaming ? 'Pause streaming' : 'Resume streaming'}
      >
        {#if isStreaming}
          <Pause class="w-3.5 h-3.5 mr-1" />
          <span>Pause</span>
        {:else}
          <Play class="w-3.5 h-3.5 mr-1" />
          <span>Resume</span>
        {/if}
      </Button>

      <Button
        variant="outline"
        size="sm"
        onclick={clearLogs}
        title="Bersihkan log di layar"
      >
        <Trash2 class="w-3.5 h-3.5 mr-1 text-muted" />
        <span>Clear</span>
      </Button>

      <Button
        variant="ghost"
        size="sm"
        onclick={() => (isExpanded = !isExpanded)}
      >
        {#if isExpanded}
          <ChevronUp class="w-4 h-4 text-muted" />
        {:else}
          <ChevronDown class="w-4 h-4 text-muted" />
        {/if}
      </Button>
    </div>
  {/snippet}

  {#if isExpanded}
    <div class="space-y-2 font-mono text-xs">
      <!-- Search Filter & Options -->
      <div class="flex items-center justify-between gap-2">
        <div class="max-w-xs flex-1">
          <Input placeholder="Filter log (e.g. error, info, tcp)..." bind:value={searchFilter} />
        </div>
        <label class="flex items-center gap-1.5 text-[11px] text-muted cursor-pointer select-none">
          <input
            type="checkbox"
            bind:checked={autoScroll}
            class="w-3.5 h-3.5 accent-accent"
          />
          <span>Auto-scroll</span>
        </label>
      </div>

      <!-- Log Terminal Box -->
      <div
        bind:this={logContainer}
        class="bg-black/90 text-neutral-300 p-3 rounded-lg border-2 border-border font-mono text-[11px] h-60 overflow-y-auto space-y-1 select-text shadow-inner"
      >
        {#if filteredLogs.length === 0}
          <div class="text-neutral-500 italic py-8 text-center">
            {logs.length === 0 ? 'Waiting for core daemon log output...' : 'No logs matching current filter.'}
          </div>
        {:else}
          {#each filteredLogs as logLine}
            <div class="leading-relaxed whitespace-pre-wrap break-all hover:bg-white/5 px-1 rounded transition-colors {logLine.includes('error') || logLine.includes('Error') ? 'text-rose-400 font-bold' : logLine.includes('warn') || logLine.includes('WARN') ? 'text-amber-300' : ''}">
              {logLine}
            </div>
          {/each}
        {/if}
      </div>
    </div>
  {/if}
</Card>
