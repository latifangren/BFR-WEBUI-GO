<script lang="ts">
  import { fireSakugaFx } from '../../lib/sakuga-fx'

  // Native toggle: a real checkbox input (correct change events, keyboard
  // support) dressed as a chunky neo switch. The track/knob read the theme
  // tokens, so dark mode and the theme accent follow automatically. No
  // element sync: bind:checked is the single source of truth.
  let {
    checked = $bindable(false),
    label = '',
    disabled = false,
    onchange,
  }: {
    checked?: boolean
    label?: string
    disabled?: boolean
    onchange?: (e: Event) => void
  } = $props()

  let labelEl: HTMLLabelElement | null = $state(null)

  function handleChange(e: Event) {
    if (labelEl) fireSakugaFx(labelEl, 'toggle')
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
  <span class="sakuga-toggle-track relative inline-block h-7 w-12 shrink-0">
    <span class="sakuga-toggle-knob absolute top-0.5 left-0.5 h-5 w-5"></span>
  </span>
  {#if label}<span class="text-sm font-mono font-bold text-foreground">{label}</span>{/if}
</label>

<style>
  .sakuga-toggle-track {
    background-color: var(--neo-card-sub);
    border: var(--neo-border-width, 2px) solid var(--neo-border);
    border-radius: 9999px;
    box-shadow: 2px 2px 0 0 var(--neo-shadow);
    transition: background-color 0.15s ease;
  }
  .sakuga-toggle-knob {
    background-color: var(--neo-text);
    border: 2px solid var(--neo-border);
    border-radius: 9999px;
    transition:
      transform 0.18s cubic-bezier(0.34, 1.56, 0.64, 1),
      background-color 0.15s ease;
  }
  .peer:checked + .sakuga-toggle-track {
    background-color: var(--neo-accent);
  }
  .peer:checked + .sakuga-toggle-track .sakuga-toggle-knob {
    transform: translateX(20px);
    background-color: var(--neo-accent-text);
  }
  .peer:focus-visible + .sakuga-toggle-track {
    outline: 2px solid var(--neo-accent);
    outline-offset: 2px;
  }
</style>
