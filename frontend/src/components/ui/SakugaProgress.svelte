<script lang="ts">
  // Native progress bar: plain Svelte reactivity, no element sync, so the
  // value can never get stuck at 0% the way the custom-element wrapper did
  // (calling el.set on an un-upgraded element threw and killed the effect).
  // Display-only: pass progress as 0..1.
  let {
    value = 0,
    label = '',
  }: {
    value?: number
    label?: string
  } = $props()

  const pct = $derived(Math.round(Math.max(0, Math.min(1, value)) * 100))
</script>

<div class="w-full">
  {#if label}
    <div class="mb-1 flex items-baseline justify-between font-mono">
      <span class="text-[10px] font-bold uppercase tracking-wider text-muted">{label}</span>
      <span class="text-[11px] font-black text-foreground">{pct}%</span>
    </div>
  {/if}
  <div
    class="sakuga-progress-track h-3.5 w-full overflow-hidden"
    role="progressbar"
    aria-valuenow={pct}
    aria-valuemin={0}
    aria-valuemax={100}
    aria-label={label || 'Progress'}
  >
    <div class="sakuga-progress-fill h-full" style="width: {pct}%"></div>
  </div>
</div>

<style>
  .sakuga-progress-track {
    background-color: var(--neo-card-sub);
    border: var(--neo-border-width, 2px) solid var(--neo-border);
    border-radius: 9999px;
    box-shadow: 2px 2px 0 0 var(--neo-shadow);
  }
  .sakuga-progress-fill {
    background-color: var(--neo-accent);
    border-radius: 9999px;
    transition: width 0.35s cubic-bezier(0.4, 0, 0.2, 1);
  }
</style>
