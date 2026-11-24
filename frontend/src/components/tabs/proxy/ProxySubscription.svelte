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
  import { LL } from '../../../i18n/i18n-svelte'
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
      toastStore.error($LL.proxy.subscription.toasts.enterUrl())
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
        toastStore.success($LL.proxy.subscription.toasts.subUpdated())
        subStatusMessage = res.message || $LL.proxy.subscription.updatedSuccessfully()
      } else {
        toastStore.error(res?.error || $LL.proxy.subscription.toasts.subFailed())
        subStatusMessage = res?.error || $LL.proxy.subscription.toasts.updateFailed()
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.proxy.subscription.toasts.downloadError())
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
        toastStore.success($LL.proxy.subscription.toasts.geoXUpdated())
      } else {
        toastStore.error(res?.error || $LL.proxy.subscription.toasts.geoXFailed())
      }
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.proxy.subscription.toasts.geoXError())
    } finally {
      isUpdatingGeoX = false
    }
  }

  function openInFileManager() {
    navigationStore.activeTab = 'files'
    toastStore.info($LL.proxy.subscription.toasts.openingFiles())
  }
</script>

<Card
  title={$LL.proxy.subscription.title()}
  subtitle={$LL.proxy.subscription.subtitle()}
  tone="mint"
>
  {#snippet action()}
    <Button
      variant="outline"
      size="sm"
      onclick={openInFileManager}
      title={$LL.proxy.subscription.openFolderTitle()}
    >
      <FolderOpen class="w-3.5 h-3.5 mr-1.5 text-accent" />
      <span>{$LL.proxy.subscription.openInFiles()}</span>
      <ExternalLink class="w-3 h-3 ml-1 text-muted" />
    </Button>
  {/snippet}

  <div class="space-y-4 font-mono text-xs">
    <!-- Subscription URL Input & Update Button -->
    <div class="space-y-2 p-3 bg-card-sub border border-border rounded-lg">
      <span class="text-[10px] text-muted uppercase font-bold block">{$LL.proxy.subscription.providerUrl()}</span>
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
          <span>{isUpdatingSub ? $LL.proxy.subscription.downloading() : $LL.proxy.subscription.updateSubs()}</span>
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
            <span>{$LL.proxy.subscription.geoXTitle()}</span>
            <a
              href="https://github.com/MetaCubeX/meta-rules-dat"
              target="_blank"
              rel="noreferrer"
              class="inline-flex items-center gap-0.5 text-[10px] text-accent hover:underline ml-auto"
              title={$LL.proxy.subscription.sourceTitle()}
            >
              <span>{$LL.proxy.subscription.source()}</span>
              <ExternalLink class="w-2.5 h-2.5" />
            </a>
          </div>
          <p class="text-[10px] text-muted">{$LL.proxy.subscription.geoXDesc()}</p>
        </div>
        <Button
          variant="secondary"
          size="sm"
          disabled={isUpdatingGeoX}
          onclick={updateGeoX}
        >
          <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isUpdatingGeoX ? 'animate-spin' : ''}" />
          <span>{isUpdatingGeoX ? $LL.proxy.subscription.updatingGeoX() : $LL.proxy.subscription.updateGeoX()}</span>
        </Button>
      </div>

      <!-- File Manager Direct Link -->
      <div class="p-3 bg-card-sub border border-border rounded-lg space-y-2 flex flex-col justify-between">
        <div class="space-y-1">
          <div class="flex items-center gap-1.5 font-bold text-foreground">
            <FileCode class="w-4 h-4 text-emerald-400" />
            <span>{$LL.proxy.subscription.configFilesTitle()}</span>
          </div>
          <p class="text-[10px] text-muted">{$LL.proxy.subscription.configFilesDesc()}</p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onclick={openInFileManager}
        >
          <FolderOpen class="w-3.5 h-3.5 mr-1.5 text-accent" />
          <span>{$LL.proxy.subscription.openBoxFolder()}</span>
        </Button>
      </div>
    </div>
  </div>
</Card>
