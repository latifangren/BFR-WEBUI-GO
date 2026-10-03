<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Gauge,
    RefreshCw,
    Check,
    RotateCcw,
    Gamepad2,
    PhoneCall,
    Sliders,
    Zap,
    Scale,
    Shield,
    Smartphone,
    Laptop,
    Plus,
    Trash2,
    Activity,
    ArrowDown,
    ArrowUp,
    Wifi,
    Info,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import type { QoSConfig, QoSStatus, ConnectedHotspotClient, ClientLimit } from '../../../types/qos'

  let config = $state<QoSConfig>({
    enabled: false,
    engine: 'auto',
    preset: 'gaming',
    global_download: 50,
    global_upload: 20,
    prioritize_gaming: true,
    prioritize_voip: true,
    client_limits: [],
  })

  let status = $state<QoSStatus>({
    active: false,
    engine_used: 'none',
    active_rules_count: 0,
    detected_iface: 'wlan1',
  })

  let connectedClients = $state<ConnectedHotspotClient[]>([])
  let isLoading = $state(false)
  let isSaving = $state(false)
  let newClientIP = $state('')
  let newClientName = $state('')

  onMount(async () => {
    await refreshAll()
  })

  async function refreshAll() {
    isLoading = true
    try {
      await Promise.all([fetchQoS(), fetchHotspotClients()])
    } finally {
      isLoading = false
    }
  }

  async function fetchQoS() {
    try {
      const res = await api.get<{ config?: QoSConfig; status?: QoSStatus }>('/api/qos/status')
      if (res) {
        if (res.config) {
          config = {
            ...config,
            ...res.config,
            client_limits: res.config.client_limits || [],
          }
        }
        if (res.status) {
          status = res.status
        }
      }
    } catch {
      // Fallback
    }
  }

  async function fetchHotspotClients() {
    try {
      const clients = await api.get<ConnectedHotspotClient[]>('/api/hotspot/clients')
      if (Array.isArray(clients)) {
        connectedClients = clients
      }
    } catch {
      connectedClients = []
    }
  }

  function applyPreset(presetType: 'gaming' | 'fair' | 'saver' | 'custom') {
    if (typeof navigator !== 'undefined' && navigator.vibrate) {
      navigator.vibrate(8)
    }
    config.preset = presetType
    if (presetType === 'gaming') {
      config.enabled = true
      config.global_download = 50
      config.global_upload = 20
      config.prioritize_gaming = true
      config.prioritize_voip = true
    } else if (presetType === 'fair') {
      config.enabled = true
      config.global_download = 30
      config.global_upload = 15
      config.prioritize_gaming = true
      config.prioritize_voip = false
    } else if (presetType === 'saver') {
      config.enabled = true
      config.global_download = 10
      config.global_upload = 5
      config.prioritize_gaming = false
      config.prioritize_voip = false
    }
  }

  function addClientLimit(client: { ip: string; mac?: string; device?: string }) {
    if (!client.ip) return
    const exists = config.client_limits.some((c) => c.ip === client.ip)
    if (exists) {
      toastStore.warning(`Client ${client.ip} is already in limiter list`)
      return
    }

    if (typeof navigator !== 'undefined' && navigator.vibrate) {
      navigator.vibrate(10)
    }

    config.client_limits = [
      ...config.client_limits,
      {
        ip: client.ip,
        mac: client.mac || '',
        hostname: client.device || 'Device',
        download_limit: 10,
        upload_limit: 5,
        priority: 'browsing',
        enabled: true,
      },
    ]
    newClientIP = ''
    newClientName = ''
  }

  function removeClientLimit(index: number) {
    if (typeof navigator !== 'undefined' && navigator.vibrate) {
      navigator.vibrate(10)
    }
    config.client_limits = config.client_limits.filter((_, i) => i !== index)
  }

  function setClientSpeed(client: ClientLimit, down: number, up: number) {
    if (typeof navigator !== 'undefined' && navigator.vibrate) {
      navigator.vibrate(6)
    }
    client.download_limit = down
    client.upload_limit = up
  }

  async function applyQoS() {
    if (typeof navigator !== 'undefined' && navigator.vibrate) {
      navigator.vibrate(12)
    }
    try {
      isSaving = true
      const res = await api.post<{ success: boolean; error?: string; status?: QoSStatus }>('/api/qos/apply', config)
      if (res && res.status) {
        status = res.status
      }
      toastStore.success('QoS router bandwidth rules applied successfully.')
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to apply QoS')
    } finally {
      isSaving = false
    }
  }

  async function clearQoS() {
    if (typeof navigator !== 'undefined' && navigator.vibrate) {
      navigator.vibrate(15)
    }
    try {
      isSaving = true
      const res = await api.post<{ success: boolean; status?: QoSStatus }>('/api/qos/clear')
      config.enabled = false
      if (res && res.status) {
        status = res.status
      } else {
        status = { active: false, engine_used: 'none', active_rules_count: 0, detected_iface: status.detected_iface }
      }
      toastStore.info('QoS bandwidth rules cleared.')
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to clear QoS')
    } finally {
      isSaving = false
    }
  }
</script>

<div class="space-y-6">
  <!-- Top Router QoS Telemetry Banner -->
  <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 font-mono">
    <div class="p-3 bg-card border-2 border-border shadow-[2px_2px_0px_var(--neo-shadow)] rounded-lg">
      <div class="flex items-center gap-2 mb-1">
        <Activity class="w-4 h-4 text-accent shrink-0" />
        <span class="text-[10px] text-muted uppercase font-bold">Engine Status</span>
      </div>
      <div class="flex items-center gap-1.5">
        <span class="w-2 h-2 rounded-full {status.active ? 'bg-emerald-500 animate-pulse' : 'bg-red-500'}"></span>
        <span class="text-xs font-black uppercase text-foreground">
          {status.active ? 'ACTIVE' : 'IDLE'}
        </span>
      </div>
    </div>

    <div class="p-3 bg-card border-2 border-border shadow-[2px_2px_0px_var(--neo-shadow)] rounded-lg">
      <div class="flex items-center gap-2 mb-1">
        <Gauge class="w-4 h-4 text-purple-400 shrink-0" />
        <span class="text-[10px] text-muted uppercase font-bold">Kernel Engine</span>
      </div>
      <div class="text-xs font-black uppercase text-foreground truncate">
        {status.engine_used === 'tc_htb' ? 'Linux TC HTB' : status.engine_used === 'iptables_mangle' ? 'iptables Mangle' : 'None'}
      </div>
    </div>

    <div class="p-3 bg-card border-2 border-border shadow-[2px_2px_0px_var(--neo-shadow)] rounded-lg">
      <div class="flex items-center gap-2 mb-1">
        <Wifi class="w-4 h-4 text-sky-400 shrink-0" />
        <span class="text-[10px] text-muted uppercase font-bold">Hotspot Iface</span>
      </div>
      <div class="text-xs font-black uppercase text-foreground truncate">
        {status.detected_iface || 'wlan1'}
      </div>
    </div>

    <div class="p-3 bg-card border-2 border-border shadow-[2px_2px_0px_var(--neo-shadow)] rounded-lg">
      <div class="flex items-center gap-2 mb-1">
        <Sliders class="w-4 h-4 text-amber-400 shrink-0" />
        <span class="text-[10px] text-muted uppercase font-bold">Active Rules</span>
      </div>
      <div class="text-xs font-black uppercase text-foreground">
        {status.active_rules_count || 0} Rules
      </div>
    </div>
  </div>

  <!-- Router Preset Profiles -->
  <Card title="Router QoS Profiles" subtitle="Instant 1-click traffic shaping presets" tone="butter">
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-2.5 font-mono">
      <button
        type="button"
        onclick={() => applyPreset('gaming')}
        class="p-3 rounded-lg border-2 text-left transition flex flex-col justify-between cursor-pointer {config.preset === 'gaming' ? 'bg-accent/15 border-accent shadow-[2px_2px_0px_var(--neo-shadow)]' : 'bg-card-sub border-border hover:border-accent/60'}"
      >
        <div class="flex items-center justify-between mb-2">
          <Gamepad2 class="w-4 h-4 text-emerald-400" />
          {#if config.preset === 'gaming'}
            <Badge variant="success">ACTIVE</Badge>
          {/if}
        </div>
        <div>
          <div class="font-black text-xs text-foreground uppercase">Gaming Anti-Lag</div>
          <div class="text-[10px] text-muted mt-0.5 leading-tight">Priority Ping & VoIP, 50/20M cap</div>
        </div>
      </button>

      <button
        type="button"
        onclick={() => applyPreset('fair')}
        class="p-3 rounded-lg border-2 text-left transition flex flex-col justify-between cursor-pointer {config.preset === 'fair' ? 'bg-accent/15 border-accent shadow-[2px_2px_0px_var(--neo-shadow)]' : 'bg-card-sub border-border hover:border-accent/60'}"
      >
        <div class="flex items-center justify-between mb-2">
          <Scale class="w-4 h-4 text-sky-400" />
          {#if config.preset === 'fair'}
            <Badge variant="info">ACTIVE</Badge>
          {/if}
        </div>
        <div>
          <div class="font-black text-xs text-foreground uppercase">Fair Share</div>
          <div class="text-[10px] text-muted mt-0.5 leading-tight">Anti-Monopoly per client, 30/15M</div>
        </div>
      </button>

      <button
        type="button"
        onclick={() => applyPreset('saver')}
        class="p-3 rounded-lg border-2 text-left transition flex flex-col justify-between cursor-pointer {config.preset === 'saver' ? 'bg-accent/15 border-accent shadow-[2px_2px_0px_var(--neo-shadow)]' : 'bg-card-sub border-border hover:border-accent/60'}"
      >
        <div class="flex items-center justify-between mb-2">
          <Shield class="w-4 h-4 text-amber-400" />
          {#if config.preset === 'saver'}
            <Badge variant="warning">ACTIVE</Badge>
          {/if}
        </div>
        <div>
          <div class="font-black text-xs text-foreground uppercase">Quota Saver</div>
          <div class="text-[10px] text-muted mt-0.5 leading-tight">Strict 10/5M limit for mobile data</div>
        </div>
      </button>

      <button
        type="button"
        onclick={() => applyPreset('custom')}
        class="p-3 rounded-lg border-2 text-left transition flex flex-col justify-between cursor-pointer {config.preset === 'custom' ? 'bg-accent/15 border-accent shadow-[2px_2px_0px_var(--neo-shadow)]' : 'bg-card-sub border-border hover:border-accent/60'}"
      >
        <div class="flex items-center justify-between mb-2">
          <Sliders class="w-4 h-4 text-purple-400" />
          {#if config.preset === 'custom'}
            <Badge variant="pop-purple">ACTIVE</Badge>
          {/if}
        </div>
        <div>
          <div class="font-black text-xs text-foreground uppercase">Custom Tuner</div>
          <div class="text-[10px] text-muted mt-0.5 leading-tight">Granular bandwidth manual tuning</div>
        </div>
      </button>
    </div>
  </Card>

  <!-- Global Bandwidth Shaper & Latency Shield -->
  <Card title="Global Traffic Shaper" subtitle="Hotspot interface pipe management and queueing" tone="lavender">
    {#snippet action()}
      <div class="flex items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          disabled={isSaving || isLoading}
          onclick={clearQoS}
        >
          <RotateCcw class="w-3.5 h-3.5 mr-1 text-red-400" />
          <span>Clear Rules</span>
        </Button>
        <Button
          variant="primary"
          size="sm"
          disabled={isSaving || isLoading}
          onclick={applyQoS}
        >
          <Check class="w-3.5 h-3.5 mr-1" />
          <span>Apply QoS</span>
        </Button>
      </div>
    {/snippet}

    <div class="space-y-5 font-mono text-xs">
      <!-- Master Enable & Engine Selection -->
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <label class="flex items-center justify-between p-3.5 bg-card-sub border-2 border-border rounded-lg cursor-pointer hover:border-accent transition-colors">
          <div>
            <span class="font-bold text-foreground block">Master QoS Switch</span>
            <span class="text-[10px] text-muted">Enforce kernel bandwidth limiting on hotspot</span>
          </div>
          <input
            type="checkbox"
            bind:checked={config.enabled}
            class="w-5 h-5 accent-accent cursor-pointer"
          />
        </label>

        <div class="p-3.5 bg-card-sub border-2 border-border rounded-lg flex items-center justify-between gap-2">
          <div>
            <span class="font-bold text-foreground block">Kernel Engine</span>
            <span class="text-[10px] text-muted">Linux traffic queue subsystem</span>
          </div>
          <select
            bind:value={config.engine}
            class="bg-card border-2 border-border px-2.5 py-1.5 rounded text-xs font-bold text-foreground focus:outline-none focus:border-accent"
          >
            <option value="auto">Auto (TC HTB Preferred)</option>
            <option value="tc">Linux TC HTB</option>
            <option value="iptables">iptables Mangle</option>
          </select>
        </div>
      </div>

      <!-- Global Bandwidth Sliders -->
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-1">
        <div class="bg-card-sub border-2 border-border p-3.5 rounded-lg space-y-2.5">
          <div class="flex items-center justify-between">
            <span class="text-muted uppercase font-bold flex items-center gap-1.5">
              <ArrowDown class="w-3.5 h-3.5 text-accent" />
              Global Max Download
            </span>
            <span class="text-sm font-black text-accent">{config.global_download > 0 ? `${config.global_download} Mbps` : 'Uncapped'}</span>
          </div>
          <input
            type="range"
            min="0"
            max="200"
            step="5"
            bind:value={config.global_download}
            class="w-full accent-accent cursor-pointer"
          />
          <div class="flex justify-between text-[10px] text-muted">
            <span>Uncapped</span>
            <span>50 Mbps</span>
            <span>100 Mbps</span>
            <span>200 Mbps</span>
          </div>
        </div>

        <div class="bg-card-sub border-2 border-border p-3.5 rounded-lg space-y-2.5">
          <div class="flex items-center justify-between">
            <span class="text-muted uppercase font-bold flex items-center gap-1.5">
              <ArrowUp class="w-3.5 h-3.5 text-purple-400" />
              Global Max Upload
            </span>
            <span class="text-sm font-black text-purple-400">{config.global_upload > 0 ? `${config.global_upload} Mbps` : 'Uncapped'}</span>
          </div>
          <input
            type="range"
            min="0"
            max="100"
            step="5"
            bind:value={config.global_upload}
            class="w-full accent-purple-500 cursor-pointer"
          />
          <div class="flex justify-between text-[10px] text-muted">
            <span>Uncapped</span>
            <span>25 Mbps</span>
            <span>50 Mbps</span>
            <span>100 Mbps</span>
          </div>
        </div>
      </div>

      <!-- Latency Shield & Priority Classification -->
      <div class="space-y-2.5 pt-2 border-t border-border/80">
        <span class="text-[10px] text-muted uppercase font-bold block">
          Latency Shield & Priority Traffic Classification (Bufferbloat Killer)
        </span>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
          <label class="flex items-center justify-between p-3 bg-card-sub border-2 border-border rounded-lg cursor-pointer hover:border-accent transition-colors">
            <div class="flex items-center gap-2.5">
              <div class="w-8 h-8 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/30 flex items-center justify-center shrink-0">
                <Gamepad2 class="w-4 h-4" />
              </div>
              <div>
                <span class="font-bold text-foreground block">Gaming & Ping Priority</span>
                <span class="text-[10px] text-muted">ICMP, DNS (53), Mobile Games UDP (prio 1)</span>
              </div>
            </div>
            <input
              type="checkbox"
              bind:checked={config.prioritize_gaming}
              class="w-4 h-4 accent-accent cursor-pointer"
            />
          </label>

          <label class="flex items-center justify-between p-3 bg-card-sub border-2 border-border rounded-lg cursor-pointer hover:border-accent transition-colors">
            <div class="flex items-center gap-2.5">
              <div class="w-8 h-8 rounded bg-sky-500/10 text-sky-400 border border-sky-500/30 flex items-center justify-center shrink-0">
                <PhoneCall class="w-4 h-4" />
              </div>
              <div>
                <span class="font-bold text-foreground block">VoIP & Discord Priority</span>
                <span class="text-[10px] text-muted">WhatsApp Call, Discord & SIP RTP (prio 2)</span>
              </div>
            </div>
            <input
              type="checkbox"
              bind:checked={config.prioritize_voip}
              class="w-4 h-4 accent-accent cursor-pointer"
            />
          </label>
        </div>
      </div>
    </div>
  </Card>

  <!-- Live Connected Clients Discovery -->
  <Card title="Active Hotspot Clients" subtitle="Discovered devices connected to Android Hotspot" tone="mint">
    {#snippet action()}
      <Button variant="ghost" size="sm" onclick={fetchHotspotClients} disabled={isLoading}>
        <RefreshCw class="w-3.5 h-3.5 {isLoading ? 'animate-spin' : ''}" />
        <span class="hidden sm:inline ml-1">Scan Clients</span>
      </Button>
    {/snippet}

    <div class="space-y-3 font-mono text-xs">
      {#if connectedClients.length === 0}
        <div class="p-4 text-center text-muted bg-card-sub border-2 border-dashed border-border rounded-lg">
          No active hotspot clients detected right now. Turn on Wi-Fi Hotspot on Android and connect clients.
        </div>
      {:else}
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2.5">
          {#each connectedClients as client}
            {@const isLimited = config.client_limits.some((l) => l.ip === client.ip)}
            <div class="p-3 bg-card-sub border-2 border-border rounded-lg flex items-center justify-between gap-2">
              <div class="min-w-0">
                <div class="flex items-center gap-1.5">
                  <Smartphone class="w-3.5 h-3.5 text-accent shrink-0" />
                  <span class="font-bold text-foreground truncate">{client.device || 'Android Device'}</span>
                </div>
                <div class="text-[10px] text-muted mt-0.5 truncate">
                  {client.ip} &bull; {client.mac || 'No MAC'}
                </div>
              </div>

              <div>
                {#if isLimited}
                  <Badge variant="warning">LIMITED</Badge>
                {:else}
                  <Button
                    variant="outline"
                    size="sm"
                    onclick={() => addClientLimit(client)}
                  >
                    <Plus class="w-3 h-3 mr-1" />
                    <span>Limit</span>
                  </Button>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      {/if}

      <!-- Custom Manual IP Add Bar -->
      <div class="pt-2 flex flex-col sm:flex-row gap-2">
        <input
          type="text"
          placeholder="Custom IP (e.g. 192.168.43.150)"
          bind:value={newClientIP}
          class="bg-card-sub border-2 border-border px-3 py-2 rounded text-xs font-mono text-foreground focus:outline-none focus:border-accent flex-1"
        />
        <input
          type="text"
          placeholder="Device Label (optional)"
          bind:value={newClientName}
          class="bg-card-sub border-2 border-border px-3 py-2 rounded text-xs font-mono text-foreground focus:outline-none focus:border-accent flex-1 sm:w-44"
        />
        <Button
          variant="outline"
          size="sm"
          disabled={!newClientIP}
          onclick={() => addClientLimit({ ip: newClientIP, device: newClientName || 'Custom Client' })}
        >
          <Plus class="w-3.5 h-3.5 mr-1" />
          <span>Add Custom IP</span>
        </Button>
      </div>
    </div>
  </Card>

  <!-- Simple Queue / Per-Client Bandwidth Limiters -->
  <Card title="Client Speed Limiters (Simple Queue)" subtitle="Granular download & upload bandwidth caps per IP" tone="ice">
    <div class="space-y-3 font-mono text-xs">
      {#if config.client_limits.length === 0}
        <div class="p-6 text-center text-muted bg-card-sub border-2 border-dashed border-border rounded-lg space-y-1">
          <p class="font-bold text-foreground">No Per-Client Rules Configured</p>
          <p class="text-[10px]">Click "+ Limit" on any active hotspot client above to assign individual speed limits.</p>
        </div>
      {:else}
        <div class="space-y-2.5">
          {#each config.client_limits as client, idx}
            <div class="p-3.5 bg-card-sub border-2 border-border shadow-[2px_2px_0px_var(--neo-shadow)] rounded-lg space-y-3">
              <!-- Top Header Row -->
              <div class="flex items-center justify-between gap-2">
                <div class="flex items-center gap-2 min-w-0">
                  <div class="w-7 h-7 rounded bg-accent/15 text-accent border border-accent/30 flex items-center justify-center shrink-0 font-bold">
                    {idx + 1}
                  </div>
                  <div class="truncate">
                    <span class="font-black text-foreground text-xs block truncate">{client.hostname || 'Client Device'}</span>
                    <span class="text-[10px] text-muted">{client.ip} {client.mac ? `• ${client.mac}` : ''}</span>
                  </div>
                </div>

                <div class="flex items-center gap-2 shrink-0">
                  <label class="flex items-center gap-1.5 cursor-pointer select-none">
                    <input
                      type="checkbox"
                      bind:checked={client.enabled}
                      class="w-4 h-4 accent-accent cursor-pointer"
                    />
                    <span class="text-[10px] font-bold uppercase {client.enabled ? 'text-accent' : 'text-muted'}">
                      {client.enabled ? 'ACTIVE' : 'OFF'}
                    </span>
                  </label>

                  <button
                    type="button"
                    onclick={() => removeClientLimit(idx)}
                    class="p-1.5 rounded text-red-400 hover:text-red-300 hover:bg-red-500/10 transition cursor-pointer"
                    title="Remove Rule"
                  >
                    <Trash2 class="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>

              <!-- Speed Controls & Presets Grid -->
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-1 border-t border-border/60">
                <!-- Download Cap -->
                <div class="space-y-1.5">
                  <div class="flex items-center justify-between text-[11px]">
                    <span class="text-muted uppercase font-bold flex items-center gap-1">
                      <ArrowDown class="w-3 h-3 text-accent" />
                      Download Cap:
                    </span>
                    <span class="font-black text-accent">{client.download_limit} Mbps</span>
                  </div>
                  <div class="flex items-center gap-1.5">
                    <input
                      type="range"
                      min="1"
                      max="100"
                      step="1"
                      bind:value={client.download_limit}
                      class="w-full accent-accent cursor-pointer"
                    />
                  </div>
                  <!-- Quick Preset Chips -->
                  <div class="flex items-center gap-1">
                    <button
                      type="button"
                      onclick={() => (client.download_limit = 2)}
                      class="px-1.5 py-0.5 rounded text-[9px] font-bold bg-card border border-border hover:border-accent"
                    >2M</button>
                    <button
                      type="button"
                      onclick={() => (client.download_limit = 5)}
                      class="px-1.5 py-0.5 rounded text-[9px] font-bold bg-card border border-border hover:border-accent"
                    >5M</button>
                    <button
                      type="button"
                      onclick={() => (client.download_limit = 10)}
                      class="px-1.5 py-0.5 rounded text-[9px] font-bold bg-card border border-border hover:border-accent"
                    >10M</button>
                    <button
                      type="button"
                      onclick={() => (client.download_limit = 25)}
                      class="px-1.5 py-0.5 rounded text-[9px] font-bold bg-card border border-border hover:border-accent"
                    >25M</button>
                  </div>
                </div>

                <!-- Upload Cap -->
                <div class="space-y-1.5">
                  <div class="flex items-center justify-between text-[11px]">
                    <span class="text-muted uppercase font-bold flex items-center gap-1">
                      <ArrowUp class="w-3 h-3 text-purple-400" />
                      Upload Cap:
                    </span>
                    <span class="font-black text-purple-400">{client.upload_limit} Mbps</span>
                  </div>
                  <div class="flex items-center gap-1.5">
                    <input
                      type="range"
                      min="1"
                      max="50"
                      step="1"
                      bind:value={client.upload_limit}
                      class="w-full accent-purple-500 cursor-pointer"
                    />
                  </div>
                  <!-- Quick Preset Chips -->
                  <div class="flex items-center gap-1">
                    <button
                      type="button"
                      onclick={() => (client.upload_limit = 1)}
                      class="px-1.5 py-0.5 rounded text-[9px] font-bold bg-card border border-border hover:border-purple-400"
                    >1M</button>
                    <button
                      type="button"
                      onclick={() => (client.upload_limit = 3)}
                      class="px-1.5 py-0.5 rounded text-[9px] font-bold bg-card border border-border hover:border-purple-400"
                    >3M</button>
                    <button
                      type="button"
                      onclick={() => (client.upload_limit = 5)}
                      class="px-1.5 py-0.5 rounded text-[9px] font-bold bg-card border border-border hover:border-purple-400"
                    >5M</button>
                    <button
                      type="button"
                      onclick={() => (client.upload_limit = 15)}
                      class="px-1.5 py-0.5 rounded text-[9px] font-bold bg-card border border-border hover:border-purple-400"
                    >15M</button>
                  </div>
                </div>
              </div>

              <!-- Priority Selector -->
              <div class="flex items-center justify-between gap-2 pt-1 border-t border-border/40 text-[10px]">
                <span class="text-muted uppercase font-bold">Traffic Class Priority:</span>
                <select
                  bind:value={client.priority}
                  class="bg-card border-2 border-border px-2 py-0.5 rounded font-mono text-foreground font-bold focus:outline-none focus:border-accent"
                >
                  <option value="gaming">🎮 High Priority / Gaming (prio 1)</option>
                  <option value="browsing">🌐 Standard Browsing (prio 3)</option>
                  <option value="strict">🛑 Strict Throttled (prio 5)</option>
                </select>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </Card>
</div>
