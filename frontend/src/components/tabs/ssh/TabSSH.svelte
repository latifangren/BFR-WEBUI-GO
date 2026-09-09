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
  import { toastStore } from '../../../stores/toast.svelte'
  import type { SSHStatus, SSHConfig } from '../../../types/ssh'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Badge from '../../ui/Badge.svelte'
  import Input from '../../ui/Input.svelte'

  let status = $state<SSHStatus>({ running: false, port: 2222 })
  let portInput = $state('2222')
  let config = $state<SSHConfig>({
    port: 2222,
    root_login: true,
    password_auth: true,
    authorized_keys: '',
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
          config.root_login = res.config.root_login ?? true
          config.password_auth = res.config.password_auth ?? true
          config.authorized_keys = res.config.authorized_keys || ''
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
      toastStore.success('SSH configuration saved successfully.')
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to save SSH configuration')
    } finally {
      isSaving = false
    }
  }

  async function controlSSH(action: 'start' | 'stop' | 'restart') {
    try {
      isControlling = true
      const res = await api.post<SSHStatus>('/api/ssh/control', { action })
      if (res) status = res
      toastStore.success(`SSH service ${action}ed successfully.`)
      await fetchSSHStatus()
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : `Failed to ${action} SSH service`)
    } finally {
      isControlling = false
    }
  }
</script>

<div class="space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Terminal class="w-5 h-5 text-accent" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        Dropbear SSH Daemon
      </h2>
      <Badge variant={status.running ? 'success' : 'default'}>
        {status.running ? `Running (Port ${status.port})` : 'Stopped'}
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
        <span>Refresh</span>
      </Button>
    </div>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <!-- Configuration Card -->
    <Card
      title="SSH Server Settings"
      subtitle="Configure port, authentication, and keys"
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
          <span>Save Config</span>
        </Button>
      {/snippet}

      <div class="space-y-4 font-mono text-xs">
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <Input
            type="number"
            label="SSH Listening Port"
            placeholder="2222"
            bind:value={portInput}
            disabled={isSaving}
          />
          <div class="p-3 bg-card-sub border border-border rounded flex flex-col justify-center">
            <span class="text-[10px] text-muted uppercase font-bold">Process PID</span>
            <span class="text-sm font-bold text-foreground">
              {status.pid ? status.pid : 'Inactive'}
            </span>
          </div>
        </div>

        <div class="space-y-3 pt-2 border-t border-border">
          <label class="flex items-center justify-between p-3 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
            <div>
              <span class="font-bold text-foreground block">Allow Root Login</span>
              <span class="text-[10px] text-muted">Permit root user access via SSH</span>
            </div>
            <input
              type="checkbox"
              bind:checked={config.root_login}
              class="w-4 h-4 accent-accent"
            />
          </label>

          <label class="flex items-center justify-between p-3 bg-card-sub border border-border rounded cursor-pointer hover:border-accent transition-colors">
            <div>
              <span class="font-bold text-foreground block">Password Authentication</span>
              <span class="text-[10px] text-muted">Allow password-based logins</span>
            </div>
            <input
              type="checkbox"
              bind:checked={config.password_auth}
              class="w-4 h-4 accent-accent"
            />
          </label>
        </div>

        <div class="space-y-1.5 pt-2 border-t border-border">
          <label class="block text-muted uppercase font-bold text-[11px]" for="auth-keys">
            Authorized Public Keys (`authorized_keys`)
          </label>
          <textarea
            id="auth-keys"
            bind:value={config.authorized_keys}
            rows="4"
            placeholder="ssh-rsa AAAAB3NzaC1yc2E... user@machine"
            class="w-full bg-card-sub border border-border rounded p-2.5 font-mono text-xs text-foreground focus:outline-none focus:border-accent resize-y"
          ></textarea>
        </div>
      </div>
    </Card>

    <!-- Service Controls Card -->
    <Card title="Daemon Lifecycle" subtitle="Direct process control">
      <div class="space-y-4 font-mono text-xs">
        <div class="p-4 bg-card-sub border border-border rounded space-y-2">
          <div class="flex items-center justify-between">
            <span class="text-muted uppercase font-bold text-[10px]">Service State</span>
            <Badge variant={status.running ? 'success' : 'default'}>
              {status.running ? 'Active' : 'Offline'}
            </Badge>
          </div>
          <p class="text-[11px] text-muted">
            Dropbear provides lightweight secure shell access with minimal memory footprint on Android.
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
              <span>Start SSH Daemon</span>
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
              <span>Stop SSH Daemon</span>
            </Button>
            <Button
              variant="secondary"
              size="md"
              class="w-full justify-center"
              disabled={isControlling}
              onclick={() => controlSSH('restart')}
            >
              <RotateCcw class="w-3.5 h-3.5 mr-1.5" />
              <span>Restart Daemon</span>
            </Button>
          {/if}
        </div>
      </div>
    </Card>
  </div>
</div>
