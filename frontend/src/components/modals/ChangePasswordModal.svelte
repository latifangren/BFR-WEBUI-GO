<script lang="ts">
  import { Lock, KeyRound, Check, AlertCircle } from '@lucide/svelte'
  import { api } from '../../api/client'
  import { toastStore } from '../../stores/toast.svelte'
  import Modal from '../ui/Modal.svelte'
  import Button from '../ui/Button.svelte'
  import Input from '../ui/Input.svelte'

  interface Props {
    open?: boolean
    onclose?: () => void
  }

  let { open = $bindable(false), onclose }: Props = $props()

  let currentPassword = $state('')
  let newPassword = $state('')
  let confirmPassword = $state('')
  let isLoading = $state(false)
  let errorMessage = $state('')

  function handleClose() {
    currentPassword = ''
    newPassword = ''
    confirmPassword = ''
    errorMessage = ''
    open = false
    onclose?.()
  }

  async function handleSubmit() {
    errorMessage = ''

    if (!currentPassword) {
      errorMessage = 'Current password is required.'
      return
    }

    if (newPassword.length < 3) {
      errorMessage = 'New password must be at least 3 characters.'
      return
    }

    if (newPassword !== confirmPassword) {
      errorMessage = 'New passwords do not match.'
      return
    }

    try {
      isLoading = true
      const res = await api.post<{ success: boolean; message?: string; error?: string }>(
        '/api/auth/change-password',
        {
          current_password: currentPassword,
          new_password: newPassword,
        }
      )

      if (res && res.success) {
        toastStore.success('Admin password updated successfully')
        handleClose()
      } else {
        errorMessage = res?.error || 'Failed to change password'
        toastStore.error(errorMessage)
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to change password'
      errorMessage = msg
      toastStore.error(msg)
    } finally {
      isLoading = false
    }
  }
</script>

<Modal
  open={open}
  title="Change Admin Password"
  class="max-w-md"
  onclose={handleClose}
>
  <form onsubmit={(e) => { e.preventDefault(); handleSubmit() }} class="space-y-4 font-mono text-xs">
    <div class="flex items-center gap-2.5 p-3 bg-card-sub border border-border rounded text-muted">
      <KeyRound class="w-4 h-4 text-accent shrink-0" />
      <span class="text-[11px] leading-relaxed">
        Update WebUI authentication credentials. Default initial password is <code class="text-accent font-bold">admin</code>.
      </span>
    </div>

    {#if errorMessage}
      <div class="p-2.5 bg-red-950/40 border border-red-800 rounded flex items-center gap-2 text-red-400 text-xs">
        <AlertCircle class="w-4 h-4 shrink-0" />
        <span>{errorMessage}</span>
      </div>
    {/if}

    <div class="space-y-3">
      <Input
        label="Current Password"
        type="password"
        placeholder="Enter active password"
        bind:value={currentPassword}
        disabled={isLoading}
      />

      <Input
        label="New Password (min. 3 chars)"
        type="password"
        placeholder="Enter new password"
        bind:value={newPassword}
        disabled={isLoading}
      />

      <Input
        label="Confirm New Password"
        type="password"
        placeholder="Re-enter new password"
        bind:value={confirmPassword}
        disabled={isLoading}
      />
    </div>

    <div class="flex justify-end gap-2 pt-3 border-t border-border">
      <Button
        variant="ghost"
        size="md"
        type="button"
        disabled={isLoading}
        onclick={handleClose}
      >
        Cancel
      </Button>
      <Button
        variant="primary"
        size="md"
        type="submit"
        disabled={isLoading || !currentPassword || !newPassword || !confirmPassword}
      >
        <Check class="w-3.5 h-3.5 mr-1" />
        <span>{isLoading ? 'Updating...' : 'Update Password'}</span>
      </Button>
    </div>
  </form>
</Modal>
