<script lang="ts">
  import { fireSakugaFx } from '../../lib/sakuga-fx'
  import { LL } from '../../i18n/i18n-svelte'

  // Native range input dressed as a neo slider. The fill is a gradient
  // driven by the value, so it tracks live without any element sync.
  // FX fires on release (change), not on every input tick.
  let {
    value = $bindable(50),
    min = 0,
    max = 100,
    step = 1,
    label = '',
    unit = '',
    disabled = false,
    oninput,
    onchange,
  }: {
    value?: number
    min?: number
    max?: number
    step?: number
    label?: string
    unit?: string
    disabled?: boolean
    oninput?: (e: Event) => void
    onchange?: (e: Event) => void
  } = $props()

  let inputEl: HTMLInputElement | null = $state(null)

  const fillPct = $derived(max > min ? ((value - min) / (max - min)) * 100 : 0)

  function handleInput(e: Event) {
    oninput?.(e)
  }

  function handleChange(e: Event) {
    if (inputEl) fireSakugaFx(inputEl, 'slider')
    onchange?.(e)
  }
</script>

<div class="w-full">
  {#if label}
    <div class="mb-1 flex items-baseline justify-between font-mono">
      <span class="text-xs font-bold uppercase tracking-wider text-muted">{label}</span>
      <span class="text-xs font-black text-foreground"
        >{value}{unit ? ` ${unit}` : ''}</span
      >
    </div>
  {/if}
  <input
    bind:this={inputEl}
    type="range"
    bind:value
    {min}
    {max}
    {step}
    {disabled}
    oninput={handleInput}
    onchange={handleChange}
    class="sakuga-slider w-full"
    style="--sakuga-fill: {fillPct}%"
    aria-label={label || $LL.common.slider()}
  />
</div>

<style>
  .sakuga-slider {
    -webkit-appearance: none;
    appearance: none;
    height: 14px;
    background: linear-gradient(
      to right,
      var(--neo-accent) 0%,
      var(--neo-accent) var(--sakuga-fill, 50%),
      var(--neo-card-sub) var(--sakuga-fill, 50%),
      var(--neo-card-sub) 100%
    );
    border: var(--neo-border-width, 2px) solid var(--neo-border);
    border-radius: 9999px;
    box-shadow: 2px 2px 0 0 var(--neo-shadow);
    outline-offset: 2px;
    cursor: pointer;
  }
  .sakuga-slider:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .sakuga-slider::-webkit-slider-thumb {
    -webkit-appearance: none;
    appearance: none;
    width: 22px;
    height: 22px;
    background: var(--neo-card);
    border: var(--neo-border-width, 2px) solid var(--neo-border);
    border-radius: 6px;
    box-shadow: 2px 2px 0 0 var(--neo-shadow);
    transition: transform 0.12s cubic-bezier(0.34, 1.56, 0.64, 1);
  }
  .sakuga-slider::-webkit-slider-thumb:active {
    transform: scale(1.15) rotate(-6deg);
  }
  .sakuga-slider::-moz-range-thumb {
    width: 22px;
    height: 22px;
    background: var(--neo-card);
    border: var(--neo-border-width, 2px) solid var(--neo-border);
    border-radius: 6px;
    box-shadow: 2px 2px 0 0 var(--neo-shadow);
  }
  .sakuga-slider:focus-visible {
    outline: 2px solid var(--neo-accent);
  }
</style>
