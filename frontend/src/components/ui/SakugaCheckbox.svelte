<script lang="ts">
  import type { Snippet } from 'svelte'
  import { fireSakugaFx } from '../../lib/sakuga-fx'

  // Native checkbox: a real input (correct events, keyboard support) with a
  // chunky neo box + hand-drawn check. 24px box: fits dense tables, unlike
  // the 38px vendored one. bind:checked is the single source of truth.
  let {
    checked = $bindable(false),
    disabled = false,
    onchange,
    children,
  }: {
    checked?: boolean
    disabled?: boolean
    onchange?: (e: Event) => void
    children?: Snippet
  } = $props()

  let labelEl: HTMLLabelElement | null = $state(null)

  function handleChange(e: Event) {
    if (labelEl) fireSakugaFx(labelEl, 'checkbox')
    onchange?.(e)
  }
</script>

<label
  bind:this={labelEl}
  class="inline-flex items-center gap-2 cursor-pointer select-none {disabled
    ? 'opacity-50 cursor-not-allowed'
    : ''}"
>
  <input type="checkbox" bind:checked {disabled} onchange={handleChange} class="sr-only peer" />
  <span class="sakuga-check-box relative inline-flex h-6 w-6 shrink-0 items-center justify-center">
    <svg class="sakuga-check-mark h-4 w-4" viewBox="0 0 24 24" fill="none" aria-hidden="true">
      <path
        d="M4 12.5 9.5 18 20 6.5"
        stroke="currentColor"
        stroke-width="4"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
  </span>
  {#if children}<span class="text-sm text-foreground">{@render children()}</span>{/if}
</label>

<style>
  .sakuga-check-box {
    background-color: var(--neo-card);
    border: var(--neo-border-width, 2px) solid var(--neo-border);
    border-radius: var(--neo-radius, 4px);
    box-shadow: 2px 2px 0 0 var(--neo-shadow);
    color: var(--neo-accent-text);
    transition:
      background-color 0.15s ease,
      transform 0.12s cubic-bezier(0.34, 1.56, 0.64, 1);
  }
  .sakuga-check-mark {
    opacity: 0;
    transform: scale(0.5) rotate(-10deg);
    transition:
      opacity 0.12s ease,
      transform 0.18s cubic-bezier(0.34, 1.56, 0.64, 1);
  }
  .peer:checked + .sakuga-check-box {
    background-color: var(--neo-accent);
  }
  .peer:checked + .sakuga-check-box .sakuga-check-mark {
    opacity: 1;
    transform: scale(1) rotate(0deg);
  }
  .peer:focus-visible + .sakuga-check-box {
    outline: 2px solid var(--neo-accent);
    outline-offset: 2px;
  }
  label:active .sakuga-check-box {
    transform: scale(0.92);
  }
</style>
