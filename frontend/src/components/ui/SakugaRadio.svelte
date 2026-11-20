<script lang="ts">
  import { fireSakugaFx } from '../../lib/sakuga-fx'

  // Native radio group: real inputs with bind:group (correct change events,
  // keyboard support), dressed as chunky neo radios. One component replaces
  // a whole group: pass the choices as options.
  export interface SakugaRadioOption {
    value: string
    label: string
  }

  let {
    value = $bindable(''),
    label = '',
    options = [],
    disabled = false,
    onchange,
  }: {
    value?: string
    label?: string
    options?: SakugaRadioOption[]
    disabled?: boolean
    onchange?: (e: Event) => void
  } = $props()

  const groupName = `sakuga-radio-${Math.random().toString(36).slice(2, 9)}`
  let groupEl: HTMLDivElement | null = $state(null)

  function handleChange(e: Event) {
    if (groupEl) fireSakugaFx(groupEl, 'radio')
    onchange?.(e)
  }
</script>

<div bind:this={groupEl} class="space-y-1.5" role="radiogroup" aria-label={label || undefined}>
  {#if label}
    <div class="text-xs font-mono font-bold uppercase tracking-wider text-muted">{label}</div>
  {/if}
  {#each options as opt}
    <label
      class="flex cursor-pointer items-center gap-2 select-none {disabled
        ? 'opacity-50 cursor-not-allowed'
        : ''}"
    >
      <input
        type="radio"
        name={groupName}
        value={opt.value}
        bind:group={value}
        {disabled}
        onchange={handleChange}
        class="sr-only peer"
      />
      <span class="sakuga-radio-dot relative inline-flex h-5 w-5 shrink-0 items-center justify-center"
      ></span>
      <span class="text-sm text-foreground">{opt.label}</span>
    </label>
  {/each}
</div>

<style>
  .sakuga-radio-dot {
    background-color: var(--neo-card);
    border: var(--neo-border-width, 2px) solid var(--neo-border);
    border-radius: 9999px;
    box-shadow: 2px 2px 0 0 var(--neo-shadow);
    transition: transform 0.12s cubic-bezier(0.34, 1.56, 0.64, 1);
  }
  .sakuga-radio-dot::after {
    content: '';
    width: 10px;
    height: 10px;
    border-radius: 9999px;
    background-color: var(--neo-accent-text);
    transform: scale(0);
    transition: transform 0.18s cubic-bezier(0.34, 1.56, 0.64, 1);
  }
  .peer:checked + .sakuga-radio-dot {
    background-color: var(--neo-accent);
  }
  .peer:checked + .sakuga-radio-dot::after {
    transform: scale(1);
  }
  .peer:focus-visible + .sakuga-radio-dot {
    outline: 2px solid var(--neo-accent);
    outline-offset: 2px;
  }
  label:active .sakuga-radio-dot {
    transform: scale(0.9);
  }
</style>
