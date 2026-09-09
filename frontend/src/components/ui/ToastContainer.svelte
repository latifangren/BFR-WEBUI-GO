<script lang="ts">
  import { toastStore } from '../../stores/toast.svelte'

  const borderVariants = {
    success: 'border-emerald-600 bg-emerald-950/90 text-emerald-100',
    error: 'border-red-600 bg-red-950/90 text-red-100',
    warning: 'border-amber-600 bg-amber-950/90 text-amber-100',
    info: 'border-blue-600 bg-blue-950/90 text-blue-100',
  }
</script>

<div class="fixed bottom-4 right-4 z-50 flex flex-col gap-2 max-w-sm w-full pointer-events-none p-2 sm:p-0">
  {#each toastStore.toasts as toast (toast.id)}
    <div
      class="pointer-events-auto flex items-start justify-between border-2 shadow-neobrutal p-3 rounded font-mono text-xs transition-all {borderVariants[toast.type]}"
    >
      <div class="space-y-0.5">
        <p class="font-bold tracking-wider uppercase">{toast.title}</p>
        <p class="opacity-90 break-words">{toast.message}</p>
      </div>
      <button
        type="button"
        class="ml-3 text-sm opacity-60 hover:opacity-100 cursor-pointer"
        onclick={() => toastStore.dismiss(toast.id)}
      >
        &times;
      </button>
    </div>
  {/each}
</div>
