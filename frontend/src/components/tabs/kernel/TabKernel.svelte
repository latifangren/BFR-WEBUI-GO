<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Cpu,
    Zap,
    Battery,
    Flame,
    Gauge,
    HardDrive,
    Network,
    RefreshCw,
    Sliders,
    Check,
    AlertCircle,
    SlidersHorizontal,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import type { KernelStatusResponse, CPUCluster } from '../../../types/kernel'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'

  let isLoading = $state(true)
  let status = $state<KernelStatusResponse | null>(null)
  let applyingProfile = $state<string | null>(null)
  let updatingCluster = $state<number | null>(null)
  let updatingVM = $state(false)

  // Local form state for clusters
  let clusterGovs = $state<Record<number, string>>({})
  let clusterMinFreqs = $state<Record<number, number>>({})
  let clusterMaxFreqs = $state<Record<number, number>>({})

  // Local form state for VM
  let vmSwappiness = $state<number>(60)
  let vmVfsPressure = $state<number>(100)
  let vmDirtyRatio = $state<number>(20)
  let vmDirtyBgRatio = $state<number>(5)
  let vmTcp = $state<string>('cubic')

  onMount(() => {
    fetchStatus()
  })

  async function fetchStatus() {
    try {
      isLoading = true
      const data = await api.get<KernelStatusResponse>('/api/kernel/status')
      if (data) {
        status = data
        // Populate local states
        for (const c of data.clusters) {
          clusterGovs[c.policy_id] = c.governor
          clusterMinFreqs[c.policy_id] = c.min_freq
          clusterMaxFreqs[c.policy_id] = c.max_freq
        }
        if (data.vm) {
          vmSwappiness = data.vm.swappiness
          vmVfsPressure = data.vm.vfs_cache_pressure
          vmDirtyRatio = data.vm.dirty_ratio
          vmDirtyBgRatio = data.vm.dirty_background_ratio
          vmTcp = data.vm.tcp_congestion
        }
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to fetch kernel status')
    } finally {
      isLoading = false
    }
  }

  async function selectProfile(profile: string) {
    try {
      applyingProfile = profile
      const res = await api.post<{ success: boolean; error?: string }>('/api/kernel/profile', { profile })
      if (res && res.success) {
        toastStore.success(`Applied ${profile.toUpperCase()} profile successfully`)
        await fetchStatus()
      } else {
        toastStore.error(res?.error || 'Failed to apply profile')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Error applying profile')
    } finally {
      applyingProfile = null
    }
  }

  async function applyCluster(cluster: CPUCluster) {
    const pID = cluster.policy_id
    try {
      updatingCluster = pID
      const res = await api.post<{ success: boolean; error?: string }>('/api/kernel/cluster', {
        policy_id: pID,
        governor: clusterGovs[pID] || cluster.governor,
        min_freq: clusterMinFreqs[pID] || cluster.min_freq,
        max_freq: clusterMaxFreqs[pID] || cluster.max_freq,
      })
      if (res && res.success) {
        toastStore.success(`Updated CPU cluster ${cluster.label} successfully`)
        await fetchStatus()
      } else {
        toastStore.error(res?.error || 'Failed to update cluster')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Error updating cluster')
    } finally {
      updatingCluster = null
    }
  }

  async function applyVMSettings() {
    try {
      updatingVM = true
      const res = await api.post<{ success: boolean; error?: string }>('/api/kernel/vm', {
        swappiness: vmSwappiness,
        vfs_cache_pressure: vmVfsPressure,
        dirty_ratio: vmDirtyRatio,
        dirty_background_ratio: vmDirtyBgRatio,
        tcp_congestion: vmTcp,
      })
      if (res && res.success) {
        toastStore.success('Kernel VM parameters updated successfully')
        await fetchStatus()
      } else {
        toastStore.error(res?.error || 'Failed to update VM settings')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Error updating VM settings')
    } finally {
      updatingVM = false
    }
  }

  function formatFreq(khz: number): string {
    if (!khz || khz <= 0) return 'N/A'
    if (khz >= 1000000) {
      return `${(khz / 1000000).toFixed(2)} GHz`
    }
    return `${Math.round(khz / 1000)} MHz`
  }
</script>

<div class="space-y-4 sm:space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between flex-wrap gap-2">
    <div class="flex items-center gap-2">
      <Cpu class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        Kernel Manager & SoC Tuner
      </h2>
      {#if status?.kernel_release}
        <Badge variant="default" class="hidden sm:inline-flex text-[10px] font-mono">
          {status.kernel_release}
        </Badge>
      {/if}
    </div>

    <div class="flex items-center gap-2">
      {#if status?.active_profile}
        <span class="px-2 py-0.5 rounded bg-accent/15 text-accent text-xs font-mono font-bold uppercase border border-accent/30 flex items-center gap-1">
          <Check class="w-3 h-3" />
          Profile: {status.active_profile}
        </span>
      {/if}
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoading}
        onclick={fetchStatus}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoading ? 'animate-spin' : ''}" />
        <span>Refresh</span>
      </Button>
    </div>
  </div>

  {#if isLoading && !status}
    <div class="neo-card bg-card p-12 text-center font-mono text-sm text-muted">
      <RefreshCw class="w-6 h-6 animate-spin mx-auto mb-3 text-accent" />
      Reading CPU clusters, cpufreq policies, and virtual memory parameters...
    </div>
  {:else if status}
    <!-- Section 1: Quick Tuning Profiles -->
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-2.5 sm:gap-4">
      <!-- Battery Saver -->
      <button
        type="button"
        onclick={() => selectProfile('powersave')}
        disabled={applyingProfile !== null}
        class="neo-card bg-card p-3 sm:p-4 rounded border-2 text-left select-none transition-all cursor-pointer relative overflow-hidden group
          {status.active_profile === 'powersave' ? 'border-emerald-500 bg-emerald-500/5' : 'border-border hover:border-emerald-500/50'}"
      >
        <div class="flex items-center justify-between mb-2">
          <div class="flex items-center gap-2 font-mono text-xs font-bold uppercase text-foreground">
            <Battery class="w-4 h-4 text-emerald-400" />
            <span>Battery Saver</span>
          </div>
          {#if status.active_profile === 'powersave'}
            <span class="w-2 h-2 rounded-full bg-emerald-400 beacon-pulse"></span>
          {/if}
        </div>
        <p class="text-[11px] text-muted font-mono leading-relaxed mb-3">
          Caps frequency ceiling & limits scheduler bursts. Swappiness 30.
        </p>
        <div class="text-[10px] font-mono font-bold {status.active_profile === 'powersave' ? 'text-emerald-400' : 'text-muted group-hover:text-foreground'}">
          {applyingProfile === 'powersave' ? 'Applying...' : status.active_profile === 'powersave' ? '✓ Active Profile' : 'Click to Activate'}
        </div>
      </button>

      <!-- Balanced -->
      <button
        type="button"
        onclick={() => selectProfile('balanced')}
        disabled={applyingProfile !== null}
        class="neo-card bg-card p-3 sm:p-4 rounded border-2 text-left select-none transition-all cursor-pointer relative overflow-hidden group
          {status.active_profile === 'balanced' ? 'border-accent bg-accent/5' : 'border-border hover:border-accent/50'}"
      >
        <div class="flex items-center justify-between mb-2">
          <div class="flex items-center gap-2 font-mono text-xs font-bold uppercase text-foreground">
            <Gauge class="w-4 h-4 text-accent" />
            <span>Balanced</span>
          </div>
          {#if status.active_profile === 'balanced'}
            <span class="w-2 h-2 rounded-full bg-accent beacon-pulse"></span>
          {/if}
        </div>
        <p class="text-[11px] text-muted font-mono leading-relaxed mb-3">
          Energy-Aware EAS (schedutil). Dynamic clock scaling. Swappiness 60.
        </p>
        <div class="text-[10px] font-mono font-bold {status.active_profile === 'balanced' ? 'text-accent' : 'text-muted group-hover:text-foreground'}">
          {applyingProfile === 'balanced' ? 'Applying...' : status.active_profile === 'balanced' ? '✓ Active Profile' : 'Click to Activate'}
        </div>
      </button>

      <!-- Maximum Performance -->
      <button
        type="button"
        onclick={() => selectProfile('performance')}
        disabled={applyingProfile !== null}
        class="neo-card bg-card p-3 sm:p-4 rounded border-2 text-left select-none transition-all cursor-pointer relative overflow-hidden group
          {status.active_profile === 'performance' ? 'border-red-500 bg-red-500/5' : 'border-border hover:border-red-500/50'}"
      >
        <div class="flex items-center justify-between mb-2">
          <div class="flex items-center gap-2 font-mono text-xs font-bold uppercase text-foreground">
            <Flame class="w-4 h-4 text-red-500" />
            <span>Performance</span>
          </div>
          {#if status.active_profile === 'performance'}
            <span class="w-2 h-2 rounded-full bg-red-500 beacon-pulse"></span>
          {/if}
        </div>
        <p class="text-[11px] text-muted font-mono leading-relaxed mb-3">
          Peak CPU clock lock, zero throttle latency. Swappiness 100.
        </p>
        <div class="text-[10px] font-mono font-bold {status.active_profile === 'performance' ? 'text-red-400' : 'text-muted group-hover:text-foreground'}">
          {applyingProfile === 'performance' ? 'Applying...' : status.active_profile === 'performance' ? '✓ Active Profile' : 'Click to Activate'}
        </div>
      </button>
    </div>

    <!-- Section 2: CPU Clusters & Frequencies -->
    <div class="space-y-3">
      <div class="flex items-center justify-between">
        <h3 class="text-xs font-mono font-bold uppercase tracking-wider text-muted flex items-center gap-1.5">
          <Sliders class="w-3.5 h-3.5 text-accent" />
          <span>CPU Frequency Scaling & Clusters ({status.clusters.length} Policies)</span>
        </h3>
        <span class="text-[10px] text-muted font-mono">
          Persistent in /data/adb/bfr_webui_go/data/kernel_config.json
        </span>
      </div>

      {#if status.clusters.length === 0}
        <div class="p-6 bg-card border-2 border-border rounded text-center text-xs font-mono text-muted">
          No cpufreq policies found on this device.
        </div>
      {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
          {#each status.clusters as cluster (cluster.policy_id)}
            <Card
              title={cluster.label}
            >
              {#snippet action()}
                <Badge variant="default" class="text-[9px] font-mono">
                  {cluster.cores}
                </Badge>
              {/snippet}

              <div class="space-y-3 font-mono text-xs">
                <!-- Live Frequency Status -->
                <div class="flex items-center justify-between p-2 rounded bg-card-sub border border-border">
                  <span class="text-muted text-[11px]">Current Clock</span>
                  <span class="font-bold text-sm text-foreground">
                    {formatFreq(cluster.cur_freq)}
                  </span>
                </div>

                <!-- Governor Selector -->
                <div class="space-y-1">
                  <label for="gov-{cluster.policy_id}" class="text-[10px] text-muted uppercase font-bold block">
                    Scaling Governor
                  </label>
                  <select
                    id="gov-{cluster.policy_id}"
                    bind:value={clusterGovs[cluster.policy_id]}
                    class="neo-input w-full bg-card-sub border border-border rounded px-2.5 py-1.5 text-xs font-mono text-foreground focus:outline-none focus:border-accent"
                  >
                    {#each cluster.available_governors as gov}
                      <option value={gov}>{gov} {gov === cluster.governor ? '(Active)' : ''}</option>
                    {/each}
                  </select>
                </div>

                <!-- Frequencies Selector (Min / Max) -->
                {#if cluster.available_frequencies && cluster.available_frequencies.length > 0}
                  <div class="grid grid-cols-2 gap-2">
                    <div class="space-y-1">
                      <label for="min-freq-{cluster.policy_id}" class="text-[10px] text-muted uppercase font-bold block">
                        Min Clock
                      </label>
                      <select
                        id="min-freq-{cluster.policy_id}"
                        bind:value={clusterMinFreqs[cluster.policy_id]}
                        class="neo-input w-full bg-card-sub border border-border rounded px-2 py-1.5 text-[11px] font-mono text-foreground focus:outline-none focus:border-accent"
                      >
                        {#each cluster.available_frequencies as freq}
                          <option value={freq}>{formatFreq(freq)}</option>
                        {/each}
                      </select>
                    </div>

                    <div class="space-y-1">
                      <label for="max-freq-{cluster.policy_id}" class="text-[10px] text-muted uppercase font-bold block">
                        Max Clock
                      </label>
                      <select
                        id="max-freq-{cluster.policy_id}"
                        bind:value={clusterMaxFreqs[cluster.policy_id]}
                        class="neo-input w-full bg-card-sub border border-border rounded px-2 py-1.5 text-[11px] font-mono text-foreground focus:outline-none focus:border-accent"
                      >
                        {#each cluster.available_frequencies as freq}
                          <option value={freq}>{formatFreq(freq)}</option>
                        {/each}
                      </select>
                    </div>
                  </div>
                {/if}

                <!-- Save Action Button -->
                <Button
                  variant="primary"
                  size="sm"
                  fullWidth
                  disabled={updatingCluster === cluster.policy_id}
                  onclick={() => applyCluster(cluster)}
                >
                  <Check class="w-3.5 h-3.5 mr-1" />
                  <span>{updatingCluster === cluster.policy_id ? 'Saving...' : 'Apply Cluster'}</span>
                </Button>
              </div>
            </Card>
          {/each}
        </div>
      {/if}
    </div>

    <!-- Section 3: Virtual Memory & I/O Tuning -->
    <Card title="Virtual Memory & Network Subsystem Tuning">
      {#snippet action()}
        <div class="neo-icon-box">
          <HardDrive class="w-4 h-4 text-accent" />
        </div>
      {/snippet}

      <div class="space-y-4 font-mono text-xs">
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
          <!-- Swappiness -->
          <div class="p-3 rounded bg-card-sub border border-border space-y-1.5">
            <div class="flex items-center justify-between">
              <span class="text-[10px] uppercase font-bold text-muted">VM Swappiness</span>
              <span class="text-xs font-bold text-accent">{vmSwappiness}</span>
            </div>
            <input
              type="range"
              min="0"
              max="200"
              step="5"
              bind:value={vmSwappiness}
              class="w-full accent-accent cursor-pointer"
            />
            <p class="text-[10px] text-muted leading-tight">
              Aggressiveness of kernel swapping memory into ZRAM (0=min, 100=std, 200=max).
            </p>
          </div>

          <!-- VFS Cache Pressure -->
          <div class="p-3 rounded bg-card-sub border border-border space-y-1.5">
            <div class="flex items-center justify-between">
              <span class="text-[10px] uppercase font-bold text-muted">VFS Cache Pressure</span>
              <span class="text-xs font-bold text-accent">{vmVfsPressure}</span>
            </div>
            <input
              type="range"
              min="10"
              max="250"
              step="10"
              bind:value={vmVfsPressure}
              class="w-full accent-accent cursor-pointer"
            />
            <p class="text-[10px] text-muted leading-tight">
              Tendency of the kernel to reclaim memory used for directory & inode caches.
            </p>
          </div>

          <!-- Dirty Ratio -->
          <div class="p-3 rounded bg-card-sub border border-border space-y-1.5">
            <div class="flex items-center justify-between">
              <span class="text-[10px] uppercase font-bold text-muted">Dirty Ratio</span>
              <span class="text-xs font-bold text-accent">{vmDirtyRatio}%</span>
            </div>
            <input
              type="range"
              min="5"
              max="60"
              step="5"
              bind:value={vmDirtyRatio}
              class="w-full accent-accent cursor-pointer"
            />
            <p class="text-[10px] text-muted leading-tight">
              Max percentage of system memory filled with dirty pages before writeback.
            </p>
          </div>

          <!-- Dirty Background Ratio -->
          <div class="p-3 rounded bg-card-sub border border-border space-y-1.5">
            <div class="flex items-center justify-between">
              <span class="text-[10px] uppercase font-bold text-muted">Dirty Background Ratio</span>
              <span class="text-xs font-bold text-accent">{vmDirtyBgRatio}%</span>
            </div>
            <input
              type="range"
              min="1"
              max="30"
              step="1"
              bind:value={vmDirtyBgRatio}
              class="w-full accent-accent cursor-pointer"
            />
            <p class="text-[10px] text-muted leading-tight">
              Percentage threshold where pdflush/kswapd begins background disk flushing.
            </p>
          </div>

          <!-- TCP Congestion Control -->
          <div class="p-3 rounded bg-card-sub border border-border space-y-1.5 sm:col-span-2 lg:col-span-2">
            <div class="flex items-center justify-between">
              <span class="text-[10px] uppercase font-bold text-muted">TCP Congestion Control</span>
              <span class="text-xs font-bold text-emerald-400 uppercase">{vmTcp}</span>
            </div>
            <select
              bind:value={vmTcp}
              class="neo-input w-full bg-card border border-border rounded px-3 py-1.5 text-xs font-mono text-foreground focus:outline-none focus:border-accent"
            >
              {#each status.vm.available_tcp_congestion as cc}
                <option value={cc}>{cc} {cc === status.vm.tcp_congestion ? '(Active)' : ''}</option>
              {/each}
            </select>
            <p class="text-[10px] text-muted leading-tight">
              Network throughput algorithm (BBR reduces bufferbloat; Cubic is default TCP standard).
            </p>
          </div>
        </div>

        <div class="flex justify-end pt-2">
          <Button
            variant="primary"
            size="md"
            disabled={updatingVM}
            onclick={applyVMSettings}
          >
            <Check class="w-4 h-4 mr-1.5" />
            <span>{updatingVM ? 'Applying Changes...' : 'Save & Apply VM Parameters'}</span>
          </Button>
        </div>
      </div>
    </Card>
  {/if}
</div>
