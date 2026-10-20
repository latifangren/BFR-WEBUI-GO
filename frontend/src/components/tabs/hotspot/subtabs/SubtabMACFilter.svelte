<script lang="ts">
  import {
    Shield,
    Plus,
    Trash2,
    ShieldAlert,
    ShieldCheck,
    Check,
  } from '@lucide/svelte'
  import { api } from '../../../../api/client'
  import { toastStore } from '../../../../stores/toast.svelte'
  import type { MACFilterResponse } from '../../../../types/hotspot'
  import Card from '../../../ui/Card.svelte'
  import Button from '../../../ui/Button.svelte'
  import Badge from '../../../ui/Badge.svelte'
  import Input from '../../../ui/Input.svelte'

  interface Props {
    externalAddMAC?: string
  }

  let { externalAddMAC = '' }: Props = $props()

  let mode = $state<'disabled' | 'blacklist' | 'whitelist'>('disabled')
  let blockedList = $state<string[]>([])
  let allowedList = $state<string[]>([])
  let newMAC = $state('')
  let isLoading = $state(false)
  let isSaving = $state(false)

  $effect(() => {
    if (externalAddMAC) {
      newMAC = externalAddMAC
    }
  })

  $effect(() => {
    void fetchMACFilter()
  })

  async function fetchMACFilter() {
    try {
      isLoading = true
      const res = await api.get<MACFilterResponse>('/api/hotspot/mac-filter')
      if (res && res.config) {
        mode = res.config.mode || 'disabled'
        blockedList = res.config.blocked_macs || []
        allowedList = res.config.allowed_macs || []
      }
    } catch {
      // Ignored if empty
    } finally {
      isLoading = false
    }
  }

  async function saveFilter(newMode = mode, bList = blockedList, aList = allowedList) {
    try {
      isSaving = true
      await api.post('/api/hotspot/mac-filter', {
        mode: newMode,
        blocked_macs: bList,
        allowed_macs: aList,
      })
      mode = newMode
      blockedList = bList
      allowedList = aList
      toastStore.success('MAC filter configuration updated.')
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to update MAC filter')
    } finally {
      isSaving = false
    }
  }

  async function setMode(targetMode: 'disabled' | 'blacklist' | 'whitelist') {
    await saveFilter(targetMode, blockedList, allowedList)
  }

  async function addRule() {
    if (!newMAC) return
    const clean = newMAC.trim().toUpperCase()
    if (clean.length < 11) {
      toastStore.warning('Invalid MAC address format')
      return
    }

    if (mode === 'whitelist') {
      if (allowedList.includes(clean)) return
      await saveFilter(mode, blockedList, [...allowedList, clean])
    } else {
      if (blockedList.includes(clean)) return
      await saveFilter(mode, [...blockedList, clean], allowedList)
    }
    newMAC = ''
  }

  async function removeRule(mac: string, type: 'blocked' | 'allowed') {
    if (type === 'allowed') {
      await saveFilter(mode, blockedList, allowedList.filter((m) => m !== mac))
    } else {
      await saveFilter(mode, blockedList.filter((m) => m !== mac), allowedList)
    }
  }
</script>

<Card
  title="Dual-Stack MAC Filtering & Access Control"
  subtitle="Hardware-level packet filtering across IPv4 and IPv6 forwarding chains"
  tone="mint"
>
  <div class="space-y-4 font-mono text-xs">
    <!-- Mode Switcher -->
    <div class="flex flex-wrap items-center gap-2 pb-3 border-b border-border">
      <span class="text-muted uppercase font-bold mr-1 text-[11px]">Enforcement Mode:</span>
      {#each (['disabled', 'blacklist', 'whitelist'] as const) as m}
        <button
          type="button"
          class="px-3 py-1.5 text-xs uppercase font-bold rounded-lg border-2 transition-all cursor-pointer {mode === m ? 'bg-accent text-accent-text border-accent shadow-sm' : 'bg-card-sub border-border text-foreground hover:border-accent'}"
          onclick={() => setMode(m)}
          disabled={isSaving}
        >
          {m}
        </button>
      {/each}
    </div>

    <!-- Input Form -->
    <div class="flex items-center gap-2">
      <div class="flex-1">
        <Input
          placeholder="AA:BB:CC:DD:EE:FF"
          bind:value={newMAC}
        />
      </div>
      <Button
        variant="secondary"
        size="md"
        disabled={!newMAC || isSaving}
        onclick={addRule}
      >
        <Plus class="w-3.5 h-3.5 mr-1" />
        <span>Add to {mode === 'whitelist' ? 'Whitelist' : 'Blacklist'}</span>
      </Button>
    </div>

    <!-- Lists Grid -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4 pt-2">
      <!-- Blacklist Box -->
      <div class="space-y-2 p-3 bg-card-sub border-2 border-border rounded-lg">
        <div class="flex items-center justify-between pb-1.5 border-b border-border">
          <div class="flex items-center gap-1.5 text-red-400 font-bold uppercase text-[11px]">
            <ShieldAlert class="w-3.5 h-3.5" />
            <span>Blocked MACs ({blockedList.length})</span>
          </div>
          {#if mode === 'blacklist'}
            <Badge variant="danger" class="text-[9px]">ENFORCING</Badge>
          {/if}
        </div>

        {#if blockedList.length === 0}
          <div class="p-4 text-center text-muted text-[11px]">
            No blocked MAC addresses.
          </div>
        {:else}
          <div class="space-y-1.5 max-h-48 overflow-y-auto">
            {#each blockedList as mac}
              <div class="flex items-center justify-between p-2 bg-card border border-border rounded">
                <span class="font-bold text-foreground text-xs">{mac}</span>
                <button
                  type="button"
                  class="text-red-400 hover:text-red-300 cursor-pointer p-1"
                  onclick={() => removeRule(mac, 'blocked')}
                  title="Remove rule"
                >
                  <Trash2 class="w-3.5 h-3.5" />
                </button>
              </div>
            {/each}
          </div>
        {/if}
      </div>

      <!-- Whitelist Box -->
      <div class="space-y-2 p-3 bg-card-sub border-2 border-border rounded-lg">
        <div class="flex items-center justify-between pb-1.5 border-b border-border">
          <div class="flex items-center gap-1.5 text-emerald-400 font-bold uppercase text-[11px]">
            <ShieldCheck class="w-3.5 h-3.5" />
            <span>Allowed MACs ({allowedList.length})</span>
          </div>
          {#if mode === 'whitelist'}
            <Badge variant="success" class="text-[9px]">ENFORCING</Badge>
          {/if}
        </div>

        {#if allowedList.length === 0}
          <div class="p-4 text-center text-muted text-[11px]">
            No allowed MAC addresses (all blocked in whitelist mode).
          </div>
        {:else}
          <div class="space-y-1.5 max-h-48 overflow-y-auto">
            {#each allowedList as mac}
              <div class="flex items-center justify-between p-2 bg-card border border-border rounded">
                <span class="font-bold text-foreground text-xs">{mac}</span>
                <button
                  type="button"
                  class="text-red-400 hover:text-red-300 cursor-pointer p-1"
                  onclick={() => removeRule(mac, 'allowed')}
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
