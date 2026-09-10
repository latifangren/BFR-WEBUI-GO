<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Wifi,
    RefreshCw,
    Users,
    Shield,
    Check,
    Power,
    Plus,
    Trash2,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import type { HotspotStatus, ConnectedClient, MACFilterResponse } from '../../../types/hotspot'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Input from '../../ui/Input.svelte'

  let status = $state<HotspotStatus>({ enabled: false, ssid: 'AndroidAP' })
  let ssidInput = $state('AndroidAP')
  let passwordInput = $state('')
  let clients = $state<ConnectedClient[]>([])
  let isLoading = $state(false)
  let isToggling = $state(false)
  let isSavingConfig = $state(false)

  // MAC Filter
  let macFilterMode = $state<'disabled' | 'blacklist' | 'whitelist'>('disabled')
  let blockedList = $state<string[]>([])
  let allowedList = $state<string[]>([])
  let newMAC = $state('')

  onMount(async () => {
    await fetchHotspotData()
  })

  async function fetchHotspotData() {
    try {
      isLoading = true
      const s = await api.get<HotspotStatus>('/api/hotspot/status')
      status = s
      ssidInput = s.ssid || 'AndroidAP'

      const c = await api.get<ConnectedClient[]>('/api/hotspot/clients')
      clients = Array.isArray(c) ? c : []

      try {
        const res = await api.get<MACFilterResponse>('/api/hotspot/mac-filter')
        macFilterMode = res.config?.mode || 'disabled'
        blockedList = res.config?.blocked_macs || []
        allowedList = res.config?.allowed_macs || []
      } catch {
        // Ignored if unconfigured
      }
    } catch {
      // Ignored
    } finally {
      isLoading = false
    }
  }

  async function toggleHotspot() {
    try {
      isToggling = true
      const nextEnable = !status.enabled
      await api.post('/api/hotspot/control', {
        enable: nextEnable,
        ssid: ssidInput,
        password: passwordInput || undefined,
      })
      status.enabled = nextEnable
      toastStore.success(`Hotspot ${nextEnable ? 'started' : 'stopped'} successfully.`)
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to toggle hotspot')
    } finally {
      isToggling = false
    }
  }

  async function saveConfig() {
    try {
      isSavingConfig = true
      await api.post('/api/hotspot/control', {
        enable: true,
        ssid: ssidInput,
        password: passwordInput || undefined,
      })
      status.ssid = ssidInput
      status.enabled = true
      passwordInput = ''
      toastStore.success('Hotspot SSID & password updated.')
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to update hotspot config')
    } finally {
      isSavingConfig = false
    }
  }

  async function saveMACFilter(mode = macFilterMode, bList = blockedList, aList = allowedList) {
    try {
      await api.post('/api/hotspot/mac-filter', {
        mode,
        blocked_macs: bList,
        allowed_macs: aList,
      })
      macFilterMode = mode
      blockedList = bList
      allowedList = aList
      toastStore.success('MAC filter configuration updated.')
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to update MAC filter')
    }
  }

  async function setFilterMode(mode: 'disabled' | 'blacklist' | 'whitelist') {
    await saveMACFilter(mode, blockedList, allowedList)
  }

  async function addMACRule() {
    if (!newMAC) return
    const macClean = newMAC.trim().toUpperCase()
    if (macFilterMode === 'whitelist') {
      if (allowedList.includes(macClean)) return
      const nextList = [...allowedList, macClean]
      await saveMACFilter(macFilterMode, blockedList, nextList)
    } else {
      if (blockedList.includes(macClean)) return
      const nextList = [...blockedList, macClean]
      await saveMACFilter(macFilterMode, nextList, allowedList)
    }
    newMAC = ''
  }

  async function removeMACRule(mac: string, type: 'blocked' | 'allowed') {
    if (type === 'allowed') {
      const nextList = allowedList.filter((m) => m !== mac)
      await saveMACFilter(macFilterMode, blockedList, nextList)
    } else {
      const nextList = blockedList.filter((m) => m !== mac)
      await saveMACFilter(macFilterMode, nextList, allowedList)
    }
  }
</script>

<div class="space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Wifi class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        SoftAP & Tethering Management
      </h2>
    </div>
    <div class="flex items-center gap-2">
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoading}
        onclick={fetchHotspotData}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoading ? 'animate-spin' : ''}" />
        <span>Refresh</span>
      </Button>
    </div>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <!-- Hotspot Control & Credentials -->
    <Card
      title="Wi-Fi Hotspot Controller"
      subtitle="Android SoftAP tethering daemon"
      tone="mint"
      class="lg:col-span-2"
    >
      {#snippet action()}
        <Button
          variant={status.enabled ? 'danger' : 'primary'}
          size="sm"
          disabled={isToggling}
          onclick={toggleHotspot}
        >
          <Power class="w-3.5 h-3.5 mr-1.5" />
          <span>{status.enabled ? 'Stop Hotspot' : 'Start Hotspot'}</span>
        </Button>
      {/snippet}

      <div class="space-y-4 font-mono text-xs">
        <div class="flex items-center justify-between p-3 bg-card-sub border border-border rounded">
          <div class="flex items-center gap-2">
            <span class="text-muted uppercase font-bold">Hotspot Status:</span>
            <Badge variant={status.enabled ? 'success' : 'default'}>
              {status.enabled ? 'Active / Broadcasting' : 'Disabled'}
            </Badge>
          </div>
          <span class="text-xs font-bold text-accent">{status.ssid || 'AndroidAP'}</span>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-2">
          <Input
            label="Broadcast SSID"
            placeholder="SSID Name"
            bind:value={ssidInput}
          />
          <Input
            type="password"
            label="WPA2/WPA3 Password"
            placeholder="Leave blank to keep current"
            bind:value={passwordInput}
          />
        </div>

        <div class="flex justify-end pt-2 border-t border-border">
          <Button
            variant="secondary"
            size="md"
            disabled={isSavingConfig || !ssidInput}
            onclick={saveConfig}
          >
            <Check class="w-3.5 h-3.5 mr-1.5" />
            <span>Apply & Restart SoftAP</span>
          </Button>
        </div>
      </div>
    </Card>

    <!-- Connected Clients Count -->
    <Card title="Client Overview" subtitle="Real-time ARP status" tone="mint">
      <div class="space-y-4 font-mono text-xs">
        <div class="p-4 bg-card-sub border border-border rounded text-center">
          <span class="text-3xl font-black text-accent block">{clients.length}</span>
          <span class="text-[11px] text-muted uppercase font-bold tracking-wider">
            Connected Devices
          </span>
        </div>

        <p class="text-[11px] text-muted">
          Devices connected to this Android access point receive automatic DHCP addresses.
          Inspect the live clients list below.
        </p>
      </div>
    </Card>
  </div>

  <!-- Connected Clients List -->
  <Card title="Connected Client Devices" subtitle="Discovered ARP IP and MAC bindings" tone="mint">
    {#if clients.length === 0}
      <div class="p-8 text-center font-mono text-xs text-muted">
        <Users class="w-6 h-6 mx-auto mb-2 opacity-50" />
        No active devices connected to the SoftAP network.
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-left font-mono text-xs">
          <thead>
            <tr class="border-b border-border text-muted uppercase text-[10px]">
              <th class="py-2.5 px-3">IP Address</th>
              <th class="py-2.5 px-3">MAC Address</th>
              <th class="py-2.5 px-3">Hostname / Device</th>
              <th class="py-2.5 px-3 text-right">Status</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            {#each clients as client}
              <tr class="hover:bg-card-sub transition-colors">
                <td class="py-2.5 px-3 font-bold text-foreground">{client.ip}</td>
                <td class="py-2.5 px-3 text-muted">{client.mac}</td>
                <td class="py-2.5 px-3 text-foreground">{client.device || 'Unknown Client'}</td>
                <td class="py-2.5 px-3 text-right">
                  <Badge variant="success">Connected</Badge>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </Card>

  <!-- MAC Filter Security -->
  <Card title="MAC Filtering & Access Control" subtitle="Allow or restrict clients by hardware MAC" tone="mint">
    <div class="space-y-4 font-mono text-xs">
      <!-- Mode Selection Tabs -->
      <div class="flex items-center gap-2 pb-2 border-b border-border">
        <span class="text-muted uppercase font-bold mr-1">Filter Mode:</span>
        {#each (['disabled', 'blacklist', 'whitelist'] as const) as mode}
          <button
            type="button"
            class="px-3 py-1 text-xs uppercase font-bold rounded border transition-colors cursor-pointer {macFilterMode === mode ? 'bg-accent text-accent-text border-accent' : 'bg-card-sub border-border text-foreground hover:border-accent'}"
            onclick={() => setFilterMode(mode)}
          >
            {mode}
          </button>
        {/each}
      </div>

      <!-- Add MAC Input -->
      <div class="flex items-center gap-2">
        <Input
          placeholder="AA:BB:CC:DD:EE:FF"
          bind:value={newMAC}
        />
        <Button
          variant="secondary"
          size="md"
          disabled={!newMAC}
          onclick={addMACRule}
        >
          <Plus class="w-3.5 h-3.5 mr-1.5" />
          <span>Add to {macFilterMode === 'whitelist' ? 'Whitelist' : 'Blacklist'}</span>
        </Button>
      </div>

      <!-- List Display -->
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4 pt-2">
        <!-- Blocked MACs -->
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <span class="font-bold text-red-400 uppercase text-[11px]">
              Blocked MACs ({blockedList.length})
            </span>
            {#if macFilterMode === 'blacklist'}
              <Badge variant="danger">Active Enforcement</Badge>
            {/if}
          </div>
          {#if blockedList.length === 0}
            <div class="p-3 text-center text-muted bg-card-sub border border-border rounded text-[11px]">
              No blocked MAC addresses.
            </div>
          {:else}
            <div class="space-y-1.5 max-h-48 overflow-y-auto">
              {#each blockedList as mac}
                <div class="flex items-center justify-between p-2 bg-card-sub border border-border rounded">
                  <span class="font-bold text-foreground">{mac}</span>
                  <button
                    type="button"
                    class="text-red-400 hover:text-red-300 cursor-pointer"
                    onclick={() => removeMACRule(mac, 'blocked')}
                    title="Remove rule"
                  >
                    <Trash2 class="w-3.5 h-3.5" />
                  </button>
                </div>
              {/each}
            </div>
          {/if}
        </div>

        <!-- Allowed MACs -->
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <span class="font-bold text-emerald-400 uppercase text-[11px]">
              Allowed MACs ({allowedList.length})
            </span>
            {#if macFilterMode === 'whitelist'}
              <Badge variant="success">Active Enforcement</Badge>
            {/if}
          </div>
          {#if allowedList.length === 0}
            <div class="p-3 text-center text-muted bg-card-sub border border-border rounded text-[11px]">
              No allowed MAC addresses.
            </div>
          {:else}
            <div class="space-y-1.5 max-h-48 overflow-y-auto">
              {#each allowedList as mac}
                <div class="flex items-center justify-between p-2 bg-card-sub border border-border rounded">
                  <span class="font-bold text-foreground">{mac}</span>
                  <button
                    type="button"
                    class="text-red-400 hover:text-red-300 cursor-pointer"
                    onclick={() => removeMACRule(mac, 'allowed')}
                    title="Remove rule"
                  >
                    <Trash2 class="w-3.5 h-3.5" />
                  </button>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      </div>
    </div>
  </Card>
</div>
