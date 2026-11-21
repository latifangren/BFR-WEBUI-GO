<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Terminal,
    RefreshCw,
    Play,
    Square,
    RotateCcw,
    Check,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { LL } from '../../../i18n/i18n-svelte'
  import { toastStore } from '../../../stores/toast.svelte'
  import type { SSHStatus, SSHConfig } from '../../../types/ssh'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Input from '../../ui/Input.svelte'
  import { themeStore } from '../../../stores/theme.svelte'
  import SakugaCheckbox from '../../ui/SakugaCheckbox.svelte'

  let status = $state<SSHStatus>({ running: false, port: 2222 })
  let portInput = $state('2222')
  let config = $state<SSHConfig>({
    port: 2222,
    bind: '127.0.0.1',
    key_auth_only: true,
  })

  let isLoading = $state(false)
  let isSaving = $state(false)
  let isControlling = $state(false)

  onMount(async () => {
    await fetchSSHStatus()
  })

  async function fetchSSHStatus() {
    try {
      isLoading = true
      const res = await api.get<SSHStatus>('/api/ssh/status')
      if (res) {
        status = res
        const p = res.config?.port || res.port || 2222
        portInput = String(p)
        config.port = p
        if (res.config) {
          config.bind = res.config.bind || '127.0.0.1'
          config.key_auth_only = res.config.key_auth_only ?? true
        }
      }
    } catch {
      // Keep defaults
    } finally {
      isLoading = false
    }
  }

  async function saveConfig() {
    try {
      isSaving = true
      config.port = parseInt(portInput, 10) || 2222
      const res = await api.post<SSHStatus>('/api/ssh/config', config)
      if (res) status = res
      toastStore.success($LL.ssh.toasts.configSaved())
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : $LL.ssh.toasts.configSaveFailed())
    } finally {
      isSaving = false
    }
  }

  async function controlSSH(action: 'start' | 'stop' | 'restart') {
    try {
      isControlling = true
      const res = await api.post<SSHStatus>('/api/ssh/control', { action })
      if (res) status = res
      toastStore.success(
        action === 'start'
          ? $LL.ssh.toasts.started()
          : action === 'stop'
            ? $LL.ssh.toasts.stopped()
            : $LL.ssh.toasts.restarted()
      )
      await fetchSSHStatus()
    } catch (err: unknown) {
      toastStore.error(
        err instanceof Error
          ? err.message
          : action === 'start'
            ? $LL.ssh.toasts.startFailed()
            : action === 'stop'
              ? $LL.ssh.toasts.stopFailed()
              : $LL.ssh.toasts.restartFailed()
      )
    } finally {
      isControlling = false
    }
  }
</script>

<div class="space-y-3 sm:space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Terminal class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        {$LL.ssh.title()}
      </h2>
      <Badge variant={status.running ? 'success' : 'default'}>
        {status.running ? $LL.ssh.status.running({ port: status.port }) : $LL.ssh.status.stopped()}
      </Badge>
    </div>

    <div class="flex items-center gap-2">
      <Button
        variant="secondary"
        size="sm"
        disabled={isLoading}
        onclick={fetchSSHStatus}
      >
        <RefreshCw class="w-3.5 h-3.5 mr-1.5 {isLoading ? 'animate-spin' : ''}" />
        <span>{$LL.common.refresh()}</span>
      </Button>
    </div>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-3 sm:gap-6">
    <!-- Configuration Card -->
    <Card
      title={$LL.ssh.config.title()}
      subtitle={$LL.ssh.config.subtitle()}
      tone="lavender"
      class="lg:col-span-2"
    >
      {#snippet action()}
        <Button
          variant="primary"
          size="sm"
          disabled={isSaving}
          onclick={saveConfig}
        >
          <Check class="w-3.5 h-3.5 mr-1.5" />
          <span>{$LL.ssh.config.saveConfig()}</span>
        </Button>
      {/snippet}

      <div class="space-y-4 font-mono text-xs">
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <Input
            type="number"
            label={$LL.ssh.config.portLabel()}
            placeholder="2222"
            bind:value={portInput}
            disabled={isSaving}
          />
          <div class="p-3 bg-card-sub border border-border rounded flex flex-col justify-center">
            <span class="text-[10px] text-muted uppercase font-bold">{$LL.ssh.config.pidLabel()}</span>
            <span class="text-sm font-bold text-foreground">
              {status.pid ? status.pid : $LL.ssh.config.inactive()}
            </span>
          </div>
        </div>

        <div class="space-y-3 pt-2 border-t border-border">
          <Input
            label={$LL.ssh.config.bindLabel()}
            placeholder="127.0.0.1"
            bind:value={config.bind}
            disabled={isSaving}
          />
          <p class="text-[10px] text-muted">
            {$LL.ssh.config.bindHint()}
          </p>
          <label class="flex items-center justify-between p-3 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
            <div>
              <span class="font-bold text-foreground block">{$LL.ssh.config.keyAuthOnly()}</span>
              <span class="text-[10px] text-muted">{$LL.ssh.config.keyAuthOnlyDesc()}</span>
            </div>
            {#if themeStore.useSakugaControls()}
              <SakugaCheckbox bind:checked={config.key_auth_only} />
            {:else}
            <input
              type="checkbox"
              bind:checked={config.key_auth_only}
              class="w-4 h-4 accent-accent"
            />
            {/if}
          </label>
        </div>
      </div>
    </Card>

    <!-- Service Controls Card -->
    <Card title={$LL.ssh.lifecycle.title()} subtitle={$LL.ssh.lifecycle.subtitle()} tone="lavender">
      <div class="space-y-4 font-mono text-xs">
        <div class="p-4 bg-card-sub border border-border rounded space-y-2">
          <div class="flex items-center justify-between">
            <span class="text-muted uppercase font-bold text-[10px]">{$LL.ssh.lifecycle.serviceState()}</span>
            <Badge variant={status.running ? 'success' : 'default'}>
              {status.running ? $LL.common.active() : $LL.ssh.status.offline()}
            </Badge>
          </div>
          <p class="text-[11px] text-muted">
            {$LL.ssh.lifecycle.desc()}
          </p>
        </div>

        <div class="space-y-2 pt-2">
          {#if !status.running}
            <Button
              variant="primary"
              size="md"
              class="w-full justify-center"
              disabled={isControlling}
              onclick={() => controlSSH('start')}
            >
              <Play class="w-3.5 h-3.5 mr-1.5" />
              <span>{$LL.ssh.lifecycle.startDaemon()}</span>
            </Button>
          {:else}
            <Button
              variant="danger"
              size="md"
              class="w-full justify-center"
              disabled={isControlling}
              onclick={() => controlSSH('stop')}
            >
              <Square class="w-3.5 h-3.5 mr-1.5" />
              <span>{$LL.ssh.lifecycle.stopDaemon()}</span>
            </Button>
            <Button
              variant="secondary"
              size="md"
              class="w-full justify-center"
              disabled={isControlling}
              onclick={() => controlSSH('restart')}
            >
              <RotateCcw class="w-3.5 h-3.5 mr-1.5" />
              <span>{$LL.ssh.lifecycle.restartDaemon()}</span>
            </Button>
          {/if}
        </div>
      </div>
    </Card>
  </div>
</div>
