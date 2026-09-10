<script lang="ts">
  import { onMount } from 'svelte'
  import {
    MessageSquare,
    RefreshCw,
    Search,
    Copy,
    Check,
    Phone,
    Calendar,
    KeyRound,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import type { SMSMessage, SMSResponse } from '../../../types/network'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Input from '../../ui/Input.svelte'

  let messages = $state<SMSMessage[]>([])
  let totalCount = $state(0)
  let isLoading = $state(false)
  let searchQuery = $state('')
  let copiedOTP = $state<string | null>(null)

  onMount(async () => {
    await fetchInbox()
  })

  async function fetchInbox() {
    try {
      isLoading = true
      const q = searchQuery ? `?q=${encodeURIComponent(searchQuery)}` : ''
      const res = await api.get<SMSResponse | { messages: SMSMessage[]; total: number }>(`/api/sms/inbox${q}`)
      if (res && Array.isArray(res.messages)) {
        messages = res.messages
        totalCount = res.total || res.messages.length
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to load SMS inbox')
    } finally {
      isLoading = false
    }
  }

  function extractOTP(body: string): string | null {
    if (!body) return null
    // Detect keyword followed by 4-8 digits
    const keywordMatch = body.match(/(?:code|kode|otp|pin|verification|verifikasi)[^\d]{0,10}([0-9]{4,8})\b/i)
    if (keywordMatch && keywordMatch[1]) {
      return keywordMatch[1]
    }
    // Fallback to isolated 4-8 digit number
    const standaloneMatch = body.match(/\b([0-9]{4,8})\b/)
    if (standaloneMatch && standaloneMatch[1]) {
      return standaloneMatch[1]
    }
    return null
  }

  async function copyOTP(otp: string) {
    try {
      await navigator.clipboard.writeText(otp)
      copiedOTP = otp
      toastStore.success(`OTP ${otp} copied to clipboard!`)
      setTimeout(() => {
        if (copiedOTP === otp) copiedOTP = null
      }, 3000)
    } catch {
      toastStore.error('Failed to copy to clipboard')
    }
  }

  function formatDate(timestamp: number): string {
    if (!timestamp) return ''
    try {
      // Handles both second and millisecond timestamps
      const ms = timestamp < 10000000000 ? timestamp * 1000 : timestamp
      return new Date(ms).toLocaleString()
    } catch {
      return String(timestamp)
    }
  }

  const filteredMessages = $derived(
    messages.filter((m) => {
      if (!searchQuery) return true
      const q = searchQuery.toLowerCase()
      return (
        m.address?.toLowerCase().includes(q) ||
        m.body?.toLowerCase().includes(q)
      )
    })
  )
</script>

<div class="space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <MessageSquare class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        SMS Messages & Verification Inbox
      </h2>
      <Badge variant="default">
        {totalCount} Total
      </Badge>
    </div>

    <div class="flex items-center gap-2">
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoading}
        onclick={fetchInbox}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoading ? 'animate-spin' : ''}" />
        <span>Refresh</span>
      </Button>
    </div>
  </div>

  <!-- Search & Filter Bar -->
  <div class="flex items-center gap-3">
    <div class="relative flex-1 max-w-md">
      <Input
        placeholder="Filter by sender number or keyword..."
        bind:value={searchQuery}
        onchange={fetchInbox}
      />
    </div>
    <Button variant="secondary" size="md" onclick={fetchInbox}>
      <Search class="w-3.5 h-3.5 mr-1.5" />
      <span>Search</span>
    </Button>
  </div>

  <!-- Messages List -->
  {#if isLoading && messages.length === 0}
    <div class="p-12 text-center text-muted font-mono text-xs">
      <RefreshCw class="w-6 h-6 animate-spin mx-auto mb-2 text-accent" />
      Reading SMS messages from Android Telephony provider...
    </div>
  {:else if filteredMessages.length === 0}
    <Card tone="ice">
      <div class="p-8 text-center text-muted font-mono text-xs">
        <MessageSquare class="w-6 h-6 mx-auto mb-2 opacity-50" />
        No SMS messages found matching your criteria.
      </div>
    </Card>
  {:else}
    <div class="space-y-3 font-mono text-xs">
      {#each filteredMessages as msg (msg.id)}
        {@const otp = extractOTP(msg.body)}
        <div class="p-4 bg-card-sub border border-border rounded-lg space-y-3 hover:border-accent/40 transition-all">
          <div class="flex items-center justify-between flex-wrap gap-2">
            <div class="flex items-center gap-2">
              <Phone class="w-3.5 h-3.5 text-accent" />
              <span class="font-bold text-sm text-foreground">{msg.address || 'Unknown'}</span>
            </div>

            <div class="flex items-center gap-3 text-muted text-[11px]">
              {#if msg.date}
                <span class="flex items-center gap-1">
                  <Calendar class="w-3 h-3" />
                  {formatDate(msg.date)}
                </span>
              {/if}
            </div>
          </div>

          <!-- Message Body Content -->
          <p class="text-foreground leading-relaxed whitespace-pre-wrap select-text">
            {msg.body}
          </p>

          <!-- 1-Click Copy OTP Action Pill -->
          {#if otp}
            <div class="flex items-center justify-between p-2.5 bg-card border border-accent/40 rounded-md">
              <div class="flex items-center gap-2">
                <KeyRound class="w-4 h-4 text-accent" />
                <div>
                  <span class="text-[10px] uppercase font-bold text-muted block">Detected OTP Code:</span>
                  <span class="font-mono text-base font-black tracking-widest text-accent">{otp}</span>
                </div>
              </div>

              <Button
                variant={copiedOTP === otp ? 'primary' : 'secondary'}
                size="sm"
                onclick={() => copyOTP(otp)}
              >
                {#if copiedOTP === otp}
                  <Check class="w-3.5 h-3.5 mr-1" />
                  <span>Copied!</span>
                {:else}
                  <Copy class="w-3.5 h-3.5 mr-1" />
                  <span>Copy OTP</span>
                {/if}
              </Button>
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>
