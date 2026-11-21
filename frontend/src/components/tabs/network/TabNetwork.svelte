<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Network,
    RefreshCw,
    Globe,
    Zap,
    Search,
    RotateCcw,
    CheckCircle2,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { LL } from '../../../i18n/i18n-svelte'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Input from '../../ui/Input.svelte'

  // Loading state
  let isRefreshing = $state(false)

  // TTL state
  let currentTTL = $state('64')
  let selectedTTL = $state('64')
  let isApplyingTTL = $state(false)

  // DNS state
  let activeDNS1 = $state('1.1.1.1')
  let activeDNS2 = $state('1.0.0.1')
  let customDNS1 = $state('')
  let customDNS2 = $state('')
  let isApplyingDNS = $state(false)
  let isResettingDNS = $state(false)

  const dnsPresets = [
    { name: 'Cloudflare', p: '1.1.1.1', s: '1.0.0.1' },
    { name: 'Google', p: '8.8.8.8', s: '8.8.4.4' },
    { name: 'AdGuard', p: '94.140.14.14', s: '94.140.15.15' },
    { name: 'Quad9', p: '9.9.9.9', s: '149.112.112.112' },
  ]

  // Ping state
  let pingHost = $state('1.1.1.1')
  let pingResult = $state<string | null>(null)
  let isPinging = $state(false)

  onMount(async () => {
    await fetchNetworkData()
  })

  async function fetchNetworkData() {
    isRefreshing = true
    try {
      const ttlRes = await api.get<{ ttl?: string | number; current_ttl?: string | number }>('/api/network/ttl')
      if (ttlRes?.current_ttl) {
        currentTTL = String(ttlRes.current_ttl)
        selectedTTL = String(ttlRes.current_ttl)
      } else if (ttlRes?.ttl) {
        currentTTL = String(ttlRes.ttl)
        selectedTTL = String(ttlRes.ttl)
      }
    } catch {
      // Ignored
    }

    try {
      const dnsRes = await api.get<{ primary?: string; secondary?: string; dns1?: string; dns2?: string }>('/api/network/dns')
      if (dnsRes?.primary) activeDNS1 = dnsRes.primary
      else if (dnsRes?.dns1) activeDNS1 = dnsRes.dns1

      if (dnsRes?.secondary) activeDNS2 = dnsRes.secondary
      else if (dnsRes?.dns2) activeDNS2 = dnsRes.dns2
    } catch {
      // Ignored
    } finally {
      isRefreshing = false
    }
  }

  async function applyTTL(ttl: string) {
    try {
      isApplyingTTL = true
      selectedTTL = ttl
      await api.post('/api/network/ttl', { enable: true, ttl: parseInt(ttl, 10) })
      currentTTL = ttl
      toastStore.success($LL.network.ttl.toasts.set({ ttl }))
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.network.ttl.toasts.applyFailed())
    } finally {
      isApplyingTTL = false
    }
  }

  async function setDNS(primary: string, secondary: string) {
    try {
      isApplyingDNS = true
      await api.post('/api/network/dns', { primary, secondary })
      activeDNS1 = primary
      activeDNS2 = secondary
      toastStore.success($LL.network.dns.toasts.updated({ primary, secondary }))
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.network.dns.toasts.setFailed())
    } finally {
      isApplyingDNS = false
    }
  }

  async function resetDNS() {
    try {
      isResettingDNS = true
      const res = await api.post<{ success: boolean; primary?: string; secondary?: string }>('/api/network/dns/reset')
      if (res && res.success) {
        activeDNS1 = res.primary || $LL.network.dns.defaultDhcp()
        activeDNS2 = res.secondary || $LL.common.none()
        customDNS1 = ''
        customDNS2 = ''
        toastStore.success($LL.network.dns.toasts.resetDone())
      } else {
        toastStore.error($LL.network.dns.toasts.resetFailed())
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.network.dns.toasts.resetError())
    } finally {
      isResettingDNS = false
    }
  }

  async function runPing() {
    if (!pingHost.trim()) return
    try {
      isPinging = true
      pingResult = null
      const res = await api.post<{ output: string; success?: boolean }>('/api/network/ping', { host: pingHost.trim() })
      pingResult = res.output || $LL.network.ping.noResponse()
    } catch (err: unknown) {
      pingResult = $LL.network.ping.error({ message: err instanceof Error ? err.message : $LL.network.ping.failedToReach() })
    } finally {
      isPinging = false
    }
  }
</script>

<div class="space-y-3 sm:space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Network class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        {$LL.network.title()}
      </h2>
    </div>
    <div class="flex items-center gap-2">
      <Button
        variant="secondary"
        size="sm"
        disabled={isRefreshing}
        onclick={fetchNetworkData}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isRefreshing ? 'animate-spin' : ''}" />
        <span>{$LL.common.refresh()}</span>
      </Button>
    </div>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-2 gap-3 sm:gap-6">
    <!-- TTL Modifier Card -->
    <Card title={$LL.network.ttl.title()} subtitle={$LL.network.ttl.subtitle()} tone="mint">
      {#snippet action()}
        <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-mono font-bold bg-emerald-500/15 text-emerald-500 border border-emerald-500/30">
          <CheckCircle2 class="w-3 h-3" />
          <span>{$LL.network.ttl.persistent()}</span>
        </span>
      {/snippet}

      <div class="space-y-4 font-mono text-xs">
        <div class="flex items-center justify-between p-3 bg-card-sub border border-border rounded">
          <div>
            <span class="text-muted uppercase font-bold block">{$LL.network.ttl.activeLabel()}</span>
            <span class="text-[10px] text-muted">{$LL.network.ttl.activeHint()}</span>
          </div>
          <span class="text-base font-black text-accent">{currentTTL}</span>
        </div>

        <div>
          <span class="text-[10px] text-muted uppercase font-bold block mb-2">{$LL.network.ttl.presets()}</span>
          <div class="grid grid-cols-3 gap-2">
            <button
              type="button"
              class="neo-button p-2 text-center rounded border border-border bg-card-sub hover:border-accent cursor-pointer {selectedTTL === '64' ? 'border-accent bg-accent/10 text-accent font-bold' : 'text-foreground'}"
              onclick={() => applyTTL('64')}
            >
              <div class="text-sm font-black">64</div>
              <div class="text-[10px] text-muted">{$LL.network.ttl.presetDefault()}</div>
            </button>

            <button
              type="button"
              class="neo-button p-2 text-center rounded border border-border bg-card-sub hover:border-accent cursor-pointer {selectedTTL === '65' ? 'border-accent bg-accent/10 text-accent font-bold' : 'text-foreground'}"
              onclick={() => applyTTL('65')}
            >
              <div class="text-sm font-black">65</div>
              <div class="text-[10px] text-muted">{$LL.network.ttl.presetHotspot()}</div>
            </button>

            <button
              type="button"
              class="neo-button p-2 text-center rounded border border-border bg-card-sub hover:border-accent cursor-pointer {selectedTTL === '128' ? 'border-accent bg-accent/10 text-accent font-bold' : 'text-foreground'}"
              onclick={() => applyTTL('128')}
            >
              <div class="text-sm font-black">128</div>
              <div class="text-[10px] text-muted">{$LL.network.ttl.presetWindows()}</div>
            </button>
          </div>
        </div>

        <div class="pt-2 border-t border-border flex items-center gap-2">
          <Input placeholder={$LL.network.ttl.customPlaceholder()} bind:value={selectedTTL} />
          <Button
            variant="secondary"
            size="md"
            disabled={isApplyingTTL || !selectedTTL}
            onclick={() => applyTTL(selectedTTL)}
          >
            <Zap class="w-3.5 h-3.5 mr-1" />
            <span>{$LL.network.ttl.apply()}</span>
          </Button>
        </div>
      </div>
    </Card>

    <!-- DNS Changer Card -->
    <Card title={$LL.network.dns.title()} subtitle={$LL.network.dns.subtitle()} tone="ice">
      {#snippet action()}
        <Button
          variant="outline"
          size="sm"
          disabled={isResettingDNS || isApplyingDNS}
          onclick={resetDNS}
          title={$LL.network.dns.resetTitle()}
        >
          <RotateCcw class="w-3 h-3 mr-1.5 {isResettingDNS ? 'animate-spin' : ''}" />
          <span>{$LL.network.dns.resetDefault()}</span>
        </Button>
      {/snippet}

      <div class="space-y-4 font-mono text-xs">
        <div class="flex items-center justify-between p-3 bg-card-sub border border-border rounded">
          <div>
            <span class="text-muted uppercase font-bold block">{$LL.network.dns.activeLabel()}</span>
            <span class="text-[10px] text-muted">{$LL.network.dns.activeHint()}</span>
          </div>
          <span class="text-xs font-bold text-accent">{activeDNS1} / {activeDNS2}</span>
        </div>

        <div>
          <span class="text-[10px] text-muted uppercase font-bold block mb-2">{$LL.network.dns.presets()}</span>
          <div class="grid grid-cols-2 gap-2">
            {#each dnsPresets as preset}
              <button
                type="button"
                class="neo-button p-2 text-left rounded border border-border bg-card-sub hover:border-accent cursor-pointer transition-colors"
                onclick={() => setDNS(preset.p, preset.s)}
                disabled={isApplyingDNS || isResettingDNS}
              >
                <div class="font-bold text-foreground flex items-center gap-1.5">
                  <Globe class="w-3 h-3 text-accent" />
                  <span>{preset.name}</span>
                </div>
                <div class="text-[10px] text-muted mt-0.5">{preset.p}</div>
              </button>
            {/each}
          </div>
        </div>

        <div class="pt-2 border-t border-border space-y-2">
          <span class="text-[10px] text-muted uppercase font-bold block">{$LL.network.dns.manualTitle()}</span>
          <div class="grid grid-cols-2 gap-2">
            <Input placeholder={$LL.network.dns.primaryPlaceholder()} bind:value={customDNS1} />
            <Input placeholder={$LL.network.dns.secondaryPlaceholder()} bind:value={customDNS2} />
          </div>
          <Button
            variant="secondary"
            size="md"
            fullWidth={true}
            disabled={isApplyingDNS || isResettingDNS || !customDNS1}
            onclick={() => setDNS(customDNS1, customDNS2)}
          >
            {$LL.network.dns.applyCustom()}
          </Button>
        </div>
      </div>
    </Card>
  </div>

  <!-- Network Ping Diagnostic -->
  <Card title={$LL.network.ping.title()} subtitle={$LL.network.ping.subtitle()} tone="mint">
    <div class="space-y-4 font-mono text-xs">
      <div class="flex items-center gap-2">
        <Input
          placeholder={$LL.network.ping.placeholder()}
          bind:value={pingHost}
        />
        <Button
          variant="primary"
          size="md"
          disabled={isPinging || !pingHost}
          onclick={runPing}
        >
          <Search class="w-3.5 h-3.5 mr-1.5 {isPinging ? 'animate-spin' : ''}" />
          <span>{isPinging ? $LL.network.ping.pinging() : $LL.network.ping.ping()}</span>
        </Button>
      </div>

      <!-- Quick Host Chips -->
      <div class="flex items-center gap-1.5 flex-wrap">
        <span class="text-[10px] text-muted uppercase font-bold mr-1">{$LL.network.ping.quickTarget()}</span>
        {#each ['1.1.1.1', '8.8.8.8', 'google.com', 'cloudflare.com'] as target}
          <button
            type="button"
            class="px-2 py-0.5 rounded text-[10px] font-mono bg-card-sub border border-border hover:border-accent hover:text-accent cursor-pointer transition-colors"
            onclick={() => { pingHost = target; runPing(); }}
          >
            {target}
          </button>
        {/each}
      </div>

      {#if pingResult}
        <pre class="bg-card-sub border border-border p-3 rounded text-[11px] font-mono text-muted overflow-x-auto whitespace-pre-wrap leading-relaxed">{pingResult}</pre>
      {/if}
    </div>
  </Card>
</div>
