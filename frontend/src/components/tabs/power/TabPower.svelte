<script lang="ts">
  import {
    Power,
    RotateCcw,
    AlertTriangle,
    ShieldAlert,
    Smartphone,
  } from '@lucide/svelte'
  import { api } from '../../../api/client'
  import { toastStore } from '../../../stores/toast.svelte'
  import type { PowerAction } from '../../../types/network'
  import Card from '../../ui/Card.svelte'
  import Button from '../../ui/Button.svelte'
  import Modal from '../../ui/Modal.svelte'

  let confirmAction = $state<PowerAction | null>(null)
  let isExecuting = $state(false)

  const actionLabels: Record<PowerAction, { title: string; desc: string; danger: boolean }> = {
    reboot: {
      title: 'Reboot Device',
      desc: 'Performs standard Android OS system reboot via root execution.',
      danger: false,
    },
    recovery: {
      title: 'Reboot to Recovery',
      desc: 'Reboots into TWRP / OrangeFox / Custom Recovery environment.',
      danger: true,
    },
    bootloader: {
      title: 'Reboot to Bootloader',
      desc: 'Reboots into Fastboot / Download mode for flashing.',
      danger: true,
    },
    poweroff: {
      title: 'System Power Off',
      desc: 'Completely shuts down the Android device hardware.',
      danger: true,
    },
  }

  async function executePowerAction() {
    if (!confirmAction) return

    try {
      isExecuting = true
      await api.post('/api/power', { action: confirmAction })
      toastStore.success(`Power command '${confirmAction}' dispatched to Android kernel.`)
    } catch (err: unknown) {
      toastStore.error(err instanceof Error ? err.message : 'Failed to execute power action')
    } finally {
      isExecuting = false
      confirmAction = null
    }
  }
</script>

<div class="space-y-6">
  <!-- Top Bar Controls -->
  <div class="flex items-center justify-between">
    <div class="flex items-center gap-2">
      <Power class="w-5 h-5 text-red-500" />
      <h2 class="text-base sm:text-lg font-mono font-bold uppercase tracking-wider text-foreground">
        Power & System Actions
      </h2>
    </div>
  </div>

  <!-- Safety Warning Banner -->
  <div class="bg-amber-950/50 border-2 border-amber-800 text-amber-300 p-4 rounded font-mono text-xs flex items-center gap-3">
    <AlertTriangle class="w-5 h-5 text-amber-400 shrink-0" />
    <div>
      <p class="font-bold uppercase">Root Execution Warning</p>
      <p>
        Power actions execute system-level commands with superuser privileges (`su -c`).
        The WebUI daemon and active services will disconnect immediately upon reboot.
      </p>
    </div>
  </div>

  <!-- Power Actions Grid -->
  <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
    <!-- Standard Reboot -->
    <Card title="Standard Reboot" subtitle="Restart system services & Android framework">
      <div class="space-y-3 font-mono text-xs">
        <p class="text-muted">
          Reboots Android OS normally. Modules and Magisk/KernelSU daemons will reload automatically on boot.
        </p>
        <Button
          variant="secondary"
          size="md"
          fullWidth={true}
          onclick={() => (confirmAction = 'reboot')}
        >
          <RotateCcw class="w-4 h-4 mr-2 text-accent" />
          <span>Reboot System</span>
        </Button>
      </div>
    </Card>

    <!-- Recovery Reboot -->
    <Card title="Recovery Mode" subtitle="Enter custom recovery environment">
      <div class="space-y-3 font-mono text-xs">
        <p class="text-muted">
          Reboots directly to custom recovery partition (TWRP, OrangeFox, PBRP) for flashing zips or backups.
        </p>
        <Button
          variant="outline"
          size="md"
          fullWidth={true}
          onclick={() => (confirmAction = 'recovery')}
        >
          <Smartphone class="w-4 h-4 mr-2 text-amber-400" />
          <span>Reboot Recovery</span>
        </Button>
      </div>
    </Card>

    <!-- Bootloader / Fastboot -->
    <Card title="Bootloader / Fastboot" subtitle="Enter bootloader partition">
      <div class="space-y-3 font-mono text-xs">
        <p class="text-muted">
          Enters Fastboot mode for PC USB flashing, firmware upgrades, or kernel flashing via fastboot command.
        </p>
        <Button
          variant="outline"
          size="md"
          fullWidth={true}
          onclick={() => (confirmAction = 'bootloader')}
        >
          <ShieldAlert class="w-4 h-4 mr-2 text-purple-400" />
          <span>Reboot Bootloader</span>
        </Button>
      </div>
    </Card>

    <!-- Power Off -->
    <Card title="Power Off Hardware" subtitle="Halt kernel and turn off device">
      <div class="space-y-3 font-mono text-xs">
        <p class="text-muted">
          Shuts down Android completely. Physical power button will be required to turn the device back on.
        </p>
        <Button
          variant="danger"
          size="md"
          fullWidth={true}
          onclick={() => (confirmAction = 'poweroff')}
        >
          <Power class="w-4 h-4 mr-2" />
          <span>Shut Down Device</span>
        </Button>
      </div>
    </Card>
  </div>

  <!-- Explicit Confirmation Modal -->
  {#if confirmAction}
    <Modal
      open={true}
      title="Confirm Power Action"
      onclose={() => (confirmAction = null)}
    >
      <div class="space-y-4 font-mono text-xs">
        <div class="p-3 bg-red-950/50 border border-red-800 rounded text-red-200">
          <p class="font-bold uppercase text-sm mb-1">
            {actionLabels[confirmAction].title}
          </p>
          <p>{actionLabels[confirmAction].desc}</p>
        </div>

        <p class="text-muted">
          Are you sure you want to proceed? Any unsaved changes in running processes may be lost.
        </p>

        <div class="flex items-center justify-end gap-2 pt-2 border-t border-border">
          <Button
            variant="ghost"
            size="md"
            disabled={isExecuting}
            onclick={() => (confirmAction = null)}
          >
            Cancel
          </Button>
          <Button
            variant="danger"
            size="md"
            disabled={isExecuting}
            onclick={executePowerAction}
          >
            {isExecuting ? 'Executing...' : 'Yes, Proceed'}
          </Button>
        </div>
      </div>
    </Modal>
  {/if}
</div>
