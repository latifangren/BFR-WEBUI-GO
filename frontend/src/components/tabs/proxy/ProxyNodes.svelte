<script lang="ts">
  import {
    Activity,
    Sliders,
    Zap,
    ChevronDown,
    ChevronUp,
    RefreshCw,
    Globe,
    Check,
  } from '@lucide/svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'

  interface ProxyGroup {
    name: string
    type: string
    now: string
    all: string[]
  }

  interface Props {
    currentMode: string
    onModeChange: (mode: string) => Promise<void>
  }

  let { currentMode, onModeChange }: Props = $props()

  let groups = $state<ProxyGroup[]>([])
  let isLoadingNodes = $state(false)
  let testingDelays = $state<Record<string, boolean>>({})
  let nodeDelays = $state<Record<string, number>>({})
  let expandedGroups = $state<Record<string, boolean>>({})

  const modeOptions = [
    { id: 'rule', label: 'Rule', desc: 'Bypass domestic / Proxy foreign' },
    { id: 'global', label: 'Global', desc: 'All traffic routed via proxy' },
    { id: 'direct', label: 'Direct', desc: 'Bypass proxy (direct outbound)' },
  ]

  $effect(() => {
    fetchNodes()
  })

  async function fetchNodes() {
    try {
      isLoadingNodes = true
      const res = await api.get<{ groups: ProxyGroup[] }>('/api/proxy/nodes')
      if (res?.groups && Array.isArray(res.groups)) {
        groups = res.groups
        // Expand first group by default
        if (groups.length > 0 && Object.keys(expandedGroups).length === 0) {
          expandedGroups[groups[0].name] = true
        }
      }
    } catch {
      // Keep empty if API not ready
    } finally {
      isLoadingNodes = false
    }
  }

  async function selectNode(groupName: string, nodeName: string) {
    try {
      const res = await api.post<{ success: boolean }>('/api/proxy/nodes/select', {
        group: groupName,
        node: nodeName,
      })
      if (res?.success) {
        toastStore.success(`Switched to node: ${nodeName}`)
        await fetchNodes()
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to switch node')
    }
  }

  async function testDelay(nodeName: string) {
    try {
      testingDelays[nodeName] = true
      const res = await api.post<{ delay: number; error?: string }>('/api/proxy/delay', {
        node: nodeName,
      })
      if (res && res.delay > 0) {
        nodeDelays[nodeName] = res.delay
      } else {
        nodeDelays[nodeName] = -1
      }
    } catch {
      nodeDelays[nodeName] = -1
    } finally {
      testingDelays[nodeName] = false
    }
  }

  function toggleGroup(groupName: string) {
    expandedGroups[groupName] = !expandedGroups[groupName]
  }
</script>

<Card
  title="Outbounds & Routing Mode"
  subtitle="Rule-based routing, proxy groups selector, and real-time node latency ping tester"
  tone="mint"
>
  {#snippet action()}
    <Button
      variant="outline"
      size="sm"
      disabled={isLoadingNodes}
      onclick={fetchNodes}
    >
      <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoadingNodes ? 'animate-spin' : ''}" />
      <span>Refresh Nodes</span>
    </Button>
  {/snippet}

  <div class="space-y-4 font-mono text-xs">
    <!-- Routing Mode (Rule, Global, Direct) -->
    <div>
      <span class="text-[10px] text-muted uppercase font-bold block mb-2">Clash / Mihomo Mode</span>
      <div class="grid grid-cols-3 gap-2">
        {#each modeOptions as m}
          <button
            type="button"
            onclick={() => onModeChange(m.id)}
            class="neo-button p-2 text-center rounded border-2 transition-all cursor-pointer {currentMode.toLowerCase() === m.id ? 'border-accent bg-accent/15 text-accent font-bold shadow-neobrutal-sm' : 'border-border bg-card-sub text-muted hover:text-foreground'}"
          >
            <div class="font-bold text-xs">{m.label}</div>
            <div class="text-[10px] opacity-75">{m.desc}</div>
          </button>
        {/each}
      </div>
    </div>

    <!-- Proxy Groups Accordion -->
    {#if groups.length > 0}
      <div class="space-y-3 pt-2 border-t border-border">
        <span class="text-[10px] text-muted uppercase font-bold block">Proxy Groups ({groups.length})</span>

        {#each groups as g}
          <div class="p-3 bg-card-sub border border-border rounded-lg space-y-2">
            <button
              type="button"
              onclick={() => toggleGroup(g.name)}
              class="w-full flex items-center justify-between text-left cursor-pointer select-none"
            >
              <div class="flex items-center gap-2">
                <Globe class="w-4 h-4 text-accent" />
                <span class="font-bold text-foreground text-xs">{g.name}</span>
                <span class="px-1.5 py-0.2 rounded bg-card border border-border text-[10px] text-muted">{g.type}</span>
              </div>
              <div class="flex items-center gap-2">
                <span class="text-[11px] text-accent font-bold">{g.now}</span>
                {#if expandedGroups[g.name]}
                  <ChevronUp class="w-4 h-4 text-muted" />
                {:else}
                  <ChevronDown class="w-4 h-4 text-muted" />
                {/if}
              </div>
            </button>

            {#if expandedGroups[g.name]}
              <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2 pt-2 border-t border-border/70">
                {#each g.all as node}
                  <div
                    class="p-2 rounded border transition-colors flex items-center justify-between gap-1.5 {g.now === node ? 'bg-accent/10 border-accent font-bold text-foreground' : 'bg-card border-border text-muted hover:border-accent/40'}"
                  >
                    <button
                      type="button"
                      onclick={() => selectNode(g.name, node)}
                      class="flex-1 text-left truncate text-[11px] cursor-pointer"
                      title={node}
                    >
                      {node}
                    </button>

                    <div class="flex items-center gap-1 shrink-0">
                      {#if nodeDelays[node] !== undefined}
                        <span class="text-[10px] {nodeDelays[node] > 0 ? 'text-emerald-400' : 'text-red-400'}">
                          {nodeDelays[node] > 0 ? `${nodeDelays[node]}ms` : 'Timeout'}
                        </span>
                      {/if}

                      <button
                        type="button"
                        onclick={() => testDelay(node)}
                        disabled={testingDelays[node]}
                        class="p-1 rounded hover:bg-card-sub text-muted hover:text-accent cursor-pointer"
                        title="Ping latency test"
                      >
                        <Zap class="w-3 h-3 {testingDelays[node] ? 'animate-spin text-amber-400' : ''}" />
                      </button>
                    </div>
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {:else if !isLoadingNodes}
      <div class="p-4 bg-card-sub border border-border rounded text-center text-muted">
        No proxy groups detected. Ensure Mihomo/Clash core is running and REST API port 9090 is active.
      </div>
    {/if}
  </div>
</Card>
