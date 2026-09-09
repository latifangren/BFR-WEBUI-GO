<script lang="ts">
  import type { Snippet } from 'svelte'

  interface Props {
    open?: boolean
    title?: string
    class?: string
    onclose?: () => void
    children?: Snippet
    footer?: Snippet
  }

  let {
    open = false,
    title,
    class: className = '',
    onclose,
    children,
    footer,
  }: Props = $props()

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && open) {
      onclose?.()
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4">
    <!-- Backdrop -->
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="fixed inset-0 bg-black/80 backdrop-blur-sm transition-opacity"
      onclick={() => onclose?.()}
    ></div>

    <!-- Modal Box -->
    <div
      class="relative w-full max-w-lg bg-card border-2 border-border shadow-neobrutal rounded-lg p-5 z-10 space-y-4 max-h-[90vh] overflow-y-auto {className}"
    >
      {#if title}
        <div class="flex items-center justify-between border-b border-border pb-3">
          <h3 class="text-sm font-bold font-mono tracking-wider uppercase text-foreground">
            {title}
          </h3>
          <button
            type="button"
            class="text-muted hover:text-foreground font-mono text-lg leading-none cursor-pointer"
            onclick={() => onclose?.()}
          >
            &times;
          </button>
        </div>
      {/if}

      <div>
        {@render children?.()}
      </div>

      {#if footer}
        <div class="border-t border-border pt-3 flex items-center justify-end gap-2">
          {@render footer()}
        </div>
      {/if}
    </div>
  </div>
{/if}
