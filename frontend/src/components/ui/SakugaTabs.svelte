<script lang="ts">
  import { fireSakugaFx } from '../../lib/sakuga-fx'

  // Native tab bar: real buttons, theme tokens for the active tab. The
  // change event carries .selected on its target, matching the old
  // custom-element contract the subtabs rely on
  // (SUBTAB_IDS[(e.target as any).selected]).
  let {
    selected = $bindable(0),
    label = '',
    options = [],
    onchange,
  }: {
    selected?: number
    label?: string
    options?: string[]
    onchange?: (e: Event) => void
  } = $props()

  function handleClick(e: MouseEvent, i: number) {
    selected = i
    fireSakugaFx(e, 'tabs')
    const btn = e.currentTarget as HTMLButtonElement & { selected?: number }
    btn.selected = i
    const ev = new Event('change', { bubbles: true })
    btn.dispatchEvent(ev)
    onchange?.(ev)
  }
</script>

<div
  class="flex items-center gap-2 overflow-x-auto pb-1"
  role="tablist"
  aria-label={label || undefined}
>
  {#each options as opt, i}
    <button
      type="button"
      role="tab"
      aria-selected={i === selected}
      onclick={(e) => handleClick(e, i)}
      class="sakuga-tab shrink-0 px-3 py-1.5 font-mono text-xs font-bold select-none cursor-pointer {i ===
      selected
        ? 'sakuga-tab-active'
        : ''}"
    >
      {opt}
    </button>
  {/each}
</div>

<style>
  .sakuga-tab {
    background-color: var(--neo-card-sub);
    color: var(--neo-text);
    border: var(--neo-border-width, 2px) solid var(--neo-border);
    border-radius: var(--neo-radius, 4px);
    box-shadow: 2px 2px 0 0 var(--neo-shadow);
    transition:
      transform 0.12s cubic-bezier(0.34, 1.56, 0.64, 1),
      box-shadow 0.12s ease,
      background-color 0.12s ease;
  }
  .sakuga-tab:hover {
    filter: brightness(1.08);
  }
  .sakuga-tab:active {
    transform: translate(2px, 2px) scale(0.97);
    box-shadow: 0 0 0 0 var(--neo-shadow);
  }
  .sakuga-tab-active {
    background-color: var(--neo-accent);
    color: var(--neo-accent-text);
    transform: translate(-1px, -1px);
    box-shadow: 3px 3px 0 0 var(--neo-shadow);
  }
</style>
