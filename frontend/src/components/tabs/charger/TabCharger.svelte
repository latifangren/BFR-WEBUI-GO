<script lang="ts">
  import { onMount } from 'svelte'
  import {
    BatteryCharging,
    RefreshCw,
    Sliders,
    Power,
    Check,
    AlertCircle,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import { sysinfoStore } from '../../../stores/sysinfo.svelte'
  import type { ChargerConfig } from '../../../types/network'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Input from '../../ui/Input.svelte'

  let config = $state<ChargerConfig>({
    enabled: false,
    start_percent: 70,
    stop_percent: 80,
    custom_path: '',
  })
  let isLoading = $state(false)
  let isSaving = $state(false)
  let isToggling = $state(false)

  onMount(async () => {
    await fetchChargerConfig()
  })

  async function fetchChargerConfig() {
    try {
      isLoading = true
      const res = await api.get<{ config?: ChargerConfig } | ChargerConfig>('/api/charger/config')
      if ('config' in res && res.config) {
        config = { ...config, ...res.config }
      } else if ('enabled' in res) {
        config = { ...config, ...res }
      }
    } catch {
      // Keep defaults if unconfigured
    } finally {
      isLoading = false
    }
  }

  async function saveConfig() {
    try {
      isSaving = true
      await api.post('/api/charger/toggle', config)
      toastStore.success('Charger threshold configuration saved.')
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to save charger config')
    } finally {
      isSaving = false
    }
  }

  async function toggleCharging() {
    try {
      isToggling = true
      config.enabled = !config.enabled
      await api.post('/api/charger/toggle', config)
      toastStore.success(`Charging limiter ${config.enabled ? 'enabled' : 'disabled'}.`)
      await sysinfoStore.refresh()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to toggle charging')
    } finally {
      isToggling = false
    }
  }
</script>

<div class="space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <BatteryCharging class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        Smart Charging Limiter
      </h2>
      <Badge variant="warning">Experimental</Badge>
    </div>
    <div class="flex items-center gap-2">
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoading}
        onclick={fetchChargerConfig}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoading ? 'animate-spin' : ''}" />
        <span>Refresh</span>
      </Button>
    </div>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <!-- Main Configuration Card -->
    <Card
      title="Hardware Charging Protection"
      subtitle="Bypass charging & battery cycle optimization"
      tone="peach"
      class="lg:col-span-2"
    >
      {#snippet action()}
        <div class="flex items-center gap-2">
          <Badge variant={config.enabled ? 'success' : 'default'}>
            {config.enabled ? 'Limiter Active' : 'Limiter Inactive'}
          </Badge>
        </div>
      {/snippet}

      <div class="space-y-5 font-mono text-xs">
        <!-- Limiter Master Toggle -->
        <label class="flex items-center justify-between p-3 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
          <div>
            <span class="font-bold text-foreground block">Automatic Threshold Limiter</span>
            <span class="text-[10px] text-muted">
              Auto-cuts charging power when upper threshold is reached
            </span>
          </div>
          <input
            type="checkbox"
            bind:checked={config.enabled}
            class="w-4 h-4 accent-accent cursor-pointer"
          />
        </label>

        <!-- Sliders -->
        <div class="space-y-4 pt-2">
          <div>
            <div class="flex items-center justify-between mb-1.5">
              <span class="text-muted uppercase font-bold">Stop Charging Threshold</span>
              <span class="text-sm font-black text-accent">{config.stop_percent}%</span>
            </div>
            <input
              type="range"
              min="50"
              max="95"
              step="1"
              bind:value={config.stop_percent}
              class="w-full accent-accent cursor-pointer"
            />
            <p class="text-[10px] text-muted mt-1">
              Charging will halt once battery level reaches this percentage.
            </p>
          </div>

          <div>
            <div class="flex items-center justify-between mb-1.5">
              <span class="text-muted uppercase font-bold">Resume Charging Threshold</span>
              <span class="text-sm font-black text-purple-400">{config.start_percent}%</span>
            </div>
            <input
              type="range"
              min="30"
              max="90"
              step="1"
              bind:value={config.start_percent}
              class="w-full accent-purple-500 cursor-pointer"
            />
            <p class="text-[10px] text-muted mt-1">
              Charging resumes when battery discharges down to this percentage.
            </p>
          </div>
        </div>

        <!-- Custom sysfs Path (Optional) -->
        <div class="pt-2 border-t border-border">
          <Input
            label="Custom Charging Switch sysfs Path (Optional)"
            placeholder="/sys/class/power_supply/battery/charging_enabled"
            bind:value={config.custom_path}
          />
        </div>

        <div class="flex justify-end pt-2 border-t border-border">
          <Button
            variant="primary"
            size="md"
            disabled={isSaving}
            onclick={saveConfig}
          >
            <Check class="w-3.5 h-3.5 mr-1.5" />
            <span>Save Configuration</span>
          </Button>
        </div>
      </div>
    </Card>

    <!-- Manual Control & Current State -->
    <div class="space-y-6">
      <Card title="Instant Bypass Switch" subtitle="Immediate kernel charge toggle" tone="peach">
        <div class="space-y-4 font-mono text-xs">
          <p class="text-muted">
            Immediately cuts or restores power to battery charging IC without modifying threshold profiles.
          </p>
          <Button
            variant="secondary"
            size="md"
            fullWidth={true}
            disabled={isToggling}
            onclick={toggleCharging}
          >
            <Power class="w-4 h-4 mr-2 text-amber-400" />
            <span>Toggle Charging Switch</span>
          </Button>
        </div>
      </Card>

      <Card title="Active Battery Status" tone="peach">
        {#if sysinfoStore.stats}
          <div class="space-y-3 font-mono text-xs">
            <div class="flex items-center justify-between">
              <span class="text-muted">Level</span>
              <span class="font-bold text-foreground">{sysinfoStore.stats.battery_level}%</span>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-muted">Status</span>
              <Badge variant={sysinfoStore.stats.battery_status === 'Charging' ? 'success' : 'default'}>
                {sysinfoStore.stats.battery_status}
              </Badge>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-muted">Temperature</span>
              <span class="font-bold text-amber-400">
                {sysinfoStore.stats.battery_temp ? sysinfoStore.stats.battery_temp.toFixed(1) + '°C' : 'N/A'}
              </span>
            </div>
          </div>
        {:else}
          <p class="text-xs font-mono text-muted">Telemetry unavailable.</p>
        {/if}
      </Card>
    </div>
  </div>
</div>
