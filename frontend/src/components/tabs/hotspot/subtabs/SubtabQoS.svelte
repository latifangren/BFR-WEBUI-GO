<script lang="ts">
  import {
    Gauge,
    RefreshCw,
    Check,
    Gamepad2,
    PhoneCall,
    Sliders,
    Zap,
    Scale,
    Shield,
    Trash2,
    Plus,
    Activity,
    ArrowDown,
    ArrowUp,
  } from '@lucide/svelte'
  import { api } from '../../../../api/client'
  import { toastStore } from '../../../../stores/toast.svelte'
  import Card from '../../../ui/Card.svelte'
  import Button from '../../../ui/Button.svelte'
  import Badge from '../../../ui/Badge.svelte'
  import Input from '../../../ui/Input.svelte'
  import type { QoSConfig, QoSStatus, ClientLimit } from '../../../../types/qos'
  import { LL } from '../../../../i18n/i18n-svelte'
  import type { ConnectedClient } from '../../../../types/hotspot'
  import { themeStore } from '../../../../stores/theme.svelte'
  import SakugaCheckbox from '../../../ui/SakugaCheckbox.svelte'

  interface Props {
    prefillClient?: ConnectedClient | null
  }

  let { prefillClient = null }: Props = $props()

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

  let isLoading = $state(false)
  let isSaving = $state(false)
  let newClientIP = $state('')
  let newClientName = $state('')
  let newClientDown = $state('10')
  let newClientUp = $state('5')

  $effect(() => {
    if (prefillClient && prefillClient.ip) {
      newClientIP = prefillClient.ip
      newClientName = prefillClient.device || 'Client'
    }
  })

  $effect(() => {
    void fetchQoS()
  })

  async function fetchQoS() {
    try {
      isLoading = true
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
      // Ignored
    } finally {
      isLoading = false
    }
  }

  async function saveQoS(enabledState = config.enabled) {
    try {
      isSaving = true
      config.enabled = enabledState
      await api.post('/api/qos/apply', config)
      status.active = enabledState
      toastStore.success(enabledState ? $LL.hotspot.qos.applied() : $LL.hotspot.qos.cleared())
      await fetchQoS()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.hotspot.qos.applyFailed())
    } finally {
      isSaving = false
    }
  }

  function applyPreset(presetType: 'gaming' | 'fair' | 'saver' | 'custom') {
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

  function addClientRule() {
    if (!newClientIP) return
    const exists = config.client_limits.some((c) => c.ip === newClientIP)
    if (exists) {
      toastStore.warning($LL.hotspot.qos.alreadyLimited({ ip: newClientIP }))
      return
    }

    config.client_limits = [
      ...config.client_limits,
      {
        ip: newClientIP,
        mac: '',
        hostname: newClientName || 'Client',
        download_limit: Number(newClientDown) || 10,
        upload_limit: Number(newClientUp) || 5,
        priority: 'browsing',
        enabled: true,
      },
    ]

    newClientIP = ''
    newClientName = ''
    toastStore.success($LL.hotspot.qos.addedClient())
  }

  function removeClientRule(ip: string) {
    config.client_limits = config.client_limits.filter((c) => c.ip !== ip)
  }
</script>

<div class="space-y-4 font-mono text-xs">
  <!-- Presets & Master Toggle -->
  <Card title={$LL.hotspot.qos.title()} subtitle={$LL.hotspot.qos.subtitle()} tone="lavender">
    {#snippet action()}
      <div class="flex items-center gap-2">
        <Badge variant={status.active ? 'success' : 'default'}>
          {status.active ? $LL.hotspot.qos.activeEngine({ engine: status.engine_used }) : $LL.hotspot.qos.disabled()}
        </Badge>
        <Button
          variant={config.enabled ? 'danger' : 'primary'}
          size="sm"
          disabled={isSaving}
          onclick={() => saveQoS(!config.enabled)}
        >
          <span>{config.enabled ? $LL.hotspot.qos.disableQos() : $LL.hotspot.qos.enableQos()}</span>
        </Button>
      </div>
    {/snippet}

    <div class="space-y-4">
      <!-- Preset Selection Buttons -->
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
        <button
          type="button"
          class="p-3 rounded-lg border-2 text-left transition-all cursor-pointer {config.preset === 'gaming' ? 'bg-accent/10 border-accent text-accent' : 'bg-card-sub border-border text-foreground hover:border-accent'}"
          onclick={() => applyPreset('gaming')}
        >
          <div class="flex items-center gap-1.5 font-bold mb-1">
            <Gamepad2 class="w-4 h-4 text-emerald-400" />
            <span>{$LL.hotspot.qos.presets.gaming()}</span>
          </div>
          <span class="text-[10px] text-muted block">{$LL.hotspot.qos.presets.gamingDesc()}</span>
        </button>

        <button
          type="button"
          class="p-3 rounded-lg border-2 text-left transition-all cursor-pointer {config.preset === 'fair' ? 'bg-accent/10 border-accent text-accent' : 'bg-card-sub border-border text-foreground hover:border-accent'}"
          onclick={() => applyPreset('fair')}
        >
          <div class="flex items-center gap-1.5 font-bold mb-1">
            <Scale class="w-4 h-4 text-sky-400" />
            <span>{$LL.hotspot.qos.presets.fair()}</span>
          </div>
          <span class="text-[10px] text-muted block">{$LL.hotspot.qos.presets.fairDesc()}</span>
        </button>

        <button
          type="button"
          class="p-3 rounded-lg border-2 text-left transition-all cursor-pointer {config.preset === 'saver' ? 'bg-accent/10 border-accent text-accent' : 'bg-card-sub border-border text-foreground hover:border-accent'}"
          onclick={() => applyPreset('saver')}
        >
          <div class="flex items-center gap-1.5 font-bold mb-1">
            <Zap class="w-4 h-4 text-amber-400" />
            <span>{$LL.hotspot.qos.presets.saver()}</span>
          </div>
          <span class="text-[10px] text-muted block">{$LL.hotspot.qos.presets.saverDesc()}</span>
        </button>

        <button
          type="button"
          class="p-3 rounded-lg border-2 text-left transition-all cursor-pointer {config.preset === 'custom' ? 'bg-accent/10 border-accent text-accent' : 'bg-card-sub border-border text-foreground hover:border-accent'}"
          onclick={() => applyPreset('custom')}
        >
          <div class="flex items-center gap-1.5 font-bold mb-1">
            <Sliders class="w-4 h-4 text-lavender" />
            <span>{$LL.hotspot.qos.presets.custom()}</span>
          </div>
          <span class="text-[10px] text-muted block">{$LL.hotspot.qos.presets.customDesc()}</span>
        </button>
      </div>

      <!-- Priority Packet Switches -->
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-2">
        <label class="flex items-center justify-between p-3 bg-card-sub border-2 border-border rounded-lg cursor-pointer hover:border-accent transition-colors">
          <div class="flex items-center gap-2">
            <Gamepad2 class="w-4 h-4 text-emerald-400" />
            <div>
              <span class="font-bold text-foreground block">{$LL.hotspot.qos.prioritizeGaming()}</span>
              <span class="text-[10px] text-muted">{$LL.hotspot.qos.gamingDesc()}</span>
            </div>
          </div>
          {#if themeStore.useSakugaControls()}
            <SakugaCheckbox bind:checked={config.prioritize_gaming} />
          {:else}
          <input
            type="checkbox"
            bind:checked={config.prioritize_gaming}
            class="w-4 h-4 accent-accent cursor-pointer"
          />
          {/if}
        </label>

        <label class="flex items-center justify-between p-3 bg-card-sub border-2 border-border rounded-lg cursor-pointer hover:border-accent transition-colors">
          <div class="flex items-center gap-2">
            <PhoneCall class="w-4 h-4 text-sky-400" />
            <div>
              <span class="font-bold text-foreground block">{$LL.hotspot.qos.prioritizeVoip()}</span>
              <span class="text-[10px] text-muted">{$LL.hotspot.qos.voipDesc()}</span>
            </div>
          </div>
          {#if themeStore.useSakugaControls()}
            <SakugaCheckbox bind:checked={config.prioritize_voip} />
          {:else}
          <input
            type="checkbox"
            bind:checked={config.prioritize_voip}
            class="w-4 h-4 accent-accent cursor-pointer"
          />
          {/if}
        </label>
      </div>

      <!-- Global Bandwidth Rates -->
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-2 border-t border-border">
        <div>
          <label for="global-download-rate" class="text-muted block mb-1">{$LL.hotspot.qos.globalDownload()}</label>
          <input
            id="global-download-rate"
            type="number"
            min="0"
            max="1000"
            bind:value={config.global_download}
            class="w-full px-3 py-1.5 bg-card border-2 border-border rounded text-foreground font-mono"
          />
        </div>
        <div>
          <label for="global-upload-rate" class="text-muted block mb-1">{$LL.hotspot.qos.globalUpload()}</label>
          <input
            id="global-upload-rate"
            type="number"
            min="0"
            max="1000"
            bind:value={config.global_upload}
            class="w-full px-3 py-1.5 bg-card border-2 border-border rounded text-foreground font-mono"
          />
        </div>
      </div>
    </div>
  </Card>

  <!-- Per-Client Bandwidth Limiter Table -->
  <Card title={$LL.hotspot.qos.limitsTitle()} subtitle={$LL.hotspot.qos.limitsSubtitle()} tone="lavender">
    <div class="space-y-3">
      <!-- Add Client Limit Form -->
      <div class="p-3 bg-card-sub border-2 border-border rounded-lg space-y-2">
        <span class="text-muted text-[10px] uppercase font-bold">{$LL.hotspot.qos.addLabel()}</span>
        <div class="grid grid-cols-1 sm:grid-cols-4 gap-2">
          <Input placeholder="192.168.43.100" bind:value={newClientIP} />
          <Input placeholder={$LL.hotspot.qos.hostnamePlaceholder()} bind:value={newClientName} />
          <Input type="number" placeholder={$LL.hotspot.qos.downPlaceholder()} bind:value={newClientDown} />
          <Button variant="secondary" size="md" onclick={addClientRule}>
            <Plus class="w-3.5 h-3.5 mr-1" />
            <span>{$LL.hotspot.qos.addLimit()}</span>
          </Button>
        </div>
      </div>

      {#if config.client_limits.length === 0}
        <div class="p-6 text-center text-muted bg-card-sub border border-border rounded text-[11px]">
          {$LL.hotspot.qos.emptyLimits()}
        </div>
      {:else}
        <div class="overflow-x-auto">
          <table class="w-full text-left font-mono text-xs">
            <thead>
              <tr class="border-b border-border text-muted uppercase text-[10px]">
                <th class="py-2 px-3">{$LL.hotspot.qos.thClient()}</th>
                <th class="py-2 px-3">{$LL.hotspot.qos.thIp()}</th>
                <th class="py-2 px-3">{$LL.hotspot.qos.thMaxDownload()}</th>
                <th class="py-2 px-3">{$LL.hotspot.qos.thMaxUpload()}</th>
                <th class="py-2 px-3 text-right">{$LL.hotspot.qos.thAction()}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-border">
              {#each config.client_limits as client}
                <tr class="hover:bg-card-sub transition-colors">
                  <td class="py-2 px-3 font-bold text-foreground">{client.hostname || $LL.hotspot.qos.fallbackClient()}</td>
                  <td class="py-2 px-3 text-accent">{client.ip}</td>
                  <td class="py-2 px-3 text-foreground">{client.download_limit} Mbps</td>
                  <td class="py-2 px-3 text-foreground">{client.upload_limit} Mbps</td>
                  <td class="py-2 px-3 text-right">
                    <button
                      type="button"
                      class="text-red-400 hover:text-red-300 p-1 cursor-pointer"
                      onclick={() => removeClientRule(client.ip)}
                    >
                      <Trash2 class="w-3.5 h-3.5" />
                    </button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}

      <div class="flex justify-end pt-3 border-t border-border">
        <Button
          variant="primary"
          size="md"
          disabled={isSaving}
          onclick={() => saveQoS(true)}
        >
          <Check class="w-3.5 h-3.5 mr-1" />
          <span>{$LL.hotspot.qos.saveApply()}</span>
        </Button>
      </div>
    </div>
  </Card>
</div>
