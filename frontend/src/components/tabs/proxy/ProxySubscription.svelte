<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Download,
    RefreshCw,
    FolderOpen,
    Globe,
    ExternalLink,
    FileCode,
    Sparkles,
  } from '@lucide/svelte'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Input from '../../ui/Input.svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import { navigationStore } from '../../../stores/navigation.svelte'

  interface SubsConfig {
    subscription_urls: string[]
    auto_update: boolean
    interval_cron: string
  }

  let subURL = $state('')
  let isUpdatingSub = $state(false)
  let isUpdatingGeoX = $state(false)
  let subStatusMessage = $state('')

  onMount(async () => {
    try {
      const res = await api.get<{ subs: SubsConfig }>('/api/proxy/subs')
      if (res?.subs?.subscription_urls && res.subs.subscription_urls.length > 0) {
        subURL = res.subs.subscription_urls[0]
      }
    } catch {
      // Ignored
    }
  })

  async function updateSubscription() {
    if (!subURL.trim()) {
      toastStore.error('Please enter a remote Subscription URL first')
      return
    }

    try {
      isUpdatingSub = true
      subStatusMessage = ''
      const res = await api.post<{ success: boolean; message?: string; error?: string }>('/api/proxy/subs', {
        action: 'update_sub',
        url: subURL.trim(),
      })
      if (res && res.success) {
        toastStore.success('Subscription downloaded and updated successfully!')
        subStatusMessage = res.message || 'Updated successfully'
      } else {
        toastStore.error(res?.error || 'Failed to update subscription')
        subStatusMessage = res?.error || 'Update failed'
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Subscription download error')
    } finally {
      isUpdatingSub = false
    }
  }

  async function updateGeoX() {
    try {
      isUpdatingGeoX = true
      const res = await api.post<{ success: boolean; message?: string; error?: string }>('/api/proxy/subs', {
        action: 'update_geox',
      })
      if (res && res.success) {
        toastStore.success('GeoX database (Country.mmdb) updated successfully!')
      } else {
        toastStore.error(res?.error || 'Failed to update GeoX')
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'GeoX update failed')
    } finally {
      isUpdatingGeoX = false
    }
  }

  function openInFileManager() {
    navigationStore.activeTab = 'files'
    toastStore.info('Opening File Manager. Navigate to /data/adb/box to view config files.')
  }
</script>

<Card
  title="Subscription & Config Assets"
  subtitle="Remote subscription updater, GeoIP/GeoSite database synchronizer & File Manager shortcut"
  tone="mint"
>
  {#snippet action()}
    <Button
      variant="outline"
      size="sm"
      onclick={openInFileManager}
      title="Buka folder konfigurasi /data/adb/box di File Manager"
    >
      <FolderOpen class="w-3.5 h-3.5 mr-1.5 text-accent" />
      <span>Open in File Manager</span>
      <ExternalLink class="w-3 h-3 ml-1 text-muted" />
    </Button>
  {/snippet}

  <div class="space-y-4 font-mono text-xs">
    <!-- Subscription URL Input & Update Button -->
    <div class="space-y-2 p-3 bg-card-sub border border-border rounded-lg">
      <span class="text-[10px] text-muted uppercase font-bold block">Remote Subscription Provider URL</span>
      <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-2">
        <div class="flex-1">
          <Input
            placeholder="https://provider.example.com/api/v1/client/subscribe?token=..."
            bind:value={subURL}
          />
        </div>
        <Button
          variant="primary"
          size="md"
          disabled={isUpdatingSub || !subURL.trim()}
          onclick={updateSubscription}
        >
          <Download class="w-4 h-4 mr-1.5 {isUpdatingSub ? 'animate-bounce' : ''}" />
          <span>{isUpdatingSub ? 'Downloading...' : 'Update Subs'}</span>
        </Button>
      </div>
      {#if subStatusMessage}
        <p class="text-[11px] text-accent pt-1">{subStatusMessage}</p>
      {/if}
    </div>

    <!-- Quick Utilities Grid: GeoX & File Manager -->
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-2 border-t border-border">
      <!-- GeoX Asset Updater -->
      <div class="p-3 bg-card-sub border border-border rounded-lg space-y-2 flex flex-col justify-between">
        <div class="space-y-1">
          <div class="flex items-center gap-1.5 font-bold text-foreground">
            <Globe class="w-4 h-4 text-accent" />
            <span>GeoX Asset Database</span>
            <a
              href="https://github.com/MetaCubeX/meta-rules-dat"
              target="_blank"
              rel="noreferrer"
              class="inline-flex items-center gap-0.5 text-[10px] text-accent hover:underline ml-auto"
              title="Official MetaCubeX meta-rules-dat releases"
            >
              <span>Source</span>
              <ExternalLink class="w-2.5 h-2.5" />
            </a>
          </div>
          <p class="text-[10px] text-muted">Synchronize Country.mmdb & GeoSite databases from MetaCubeX releases.</p>
        </div>
        <Button
          variant="secondary"
          size="sm"
          disabled={isUpdatingGeoX}
          onclick={updateGeoX}
        >
          <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isUpdatingGeoX ? 'animate-spin' : ''}" />
          <span>{isUpdatingGeoX ? 'Updating GeoX...' : 'Update GeoX Database'}</span>
        </Button>
      </div>

      <!-- File Manager Direct Link -->
      <div class="p-3 bg-card-sub border border-border rounded-lg space-y-2 flex flex-col justify-between">
        <div class="space-y-1">
          <div class="flex items-center gap-1.5 font-bold text-foreground">
            <FileCode class="w-4 h-4 text-emerald-400" />
            <span>Full Config Files</span>
          </div>
          <p class="text-[10px] text-muted">Inspect and edit config.yaml, proxy-providers, or rulesets directly using BFR File Manager.</p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onclick={openInFileManager}
        >
          <FolderOpen class="w-3.5 h-3.5 mr-1.5 text-accent" />
          <span>Open /data/adb/box in Files</span>
        </Button>
      </div>
    </div>
  </div>
</Card>
