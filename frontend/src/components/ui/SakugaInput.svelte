<script lang="ts">
  // Native text input dressed as a neo field. Same props as the old
  // custom-element wrapper; the input itself is a real <input>, so all
  // types, events and focus behavior are standard.
  let {
    value = $bindable(''),
    type = 'text',
    label = '',
    placeholder = '',
    disabled = false,
    oninput,
    onchange,
    onkeydown,
  }: {
    value?: string
    type?: string
    label?: string
    placeholder?: string
    disabled?: boolean
    oninput?: (e: Event) => void
    onchange?: (e: Event) => void
    onkeydown?: (e: KeyboardEvent) => void
  } = $props()
</script>

<div class="w-full space-y-1">
  {#if label}
    <span class="block text-xs font-mono font-bold uppercase text-muted tracking-wider"
      >{label}</span
    >
  {/if}
  <input
    {type}
    bind:value
    {placeholder}
    {disabled}
    {oninput}
    {onchange}
    {onkeydown}
    class="sakuga-input w-full bg-card-sub px-3 py-2 font-mono text-sm text-foreground placeholder:text-muted focus:outline-none disabled:opacity-50"
  />
</div>

<style>
  .sakuga-input {
    border: var(--neo-border-width, 2px) solid var(--neo-border);
    border-radius: var(--neo-radius, 4px);
    box-shadow: var(--neo-shadow-size, 4px) var(--neo-shadow-size, 4px) 0 0 var(--neo-shadow);
    transition:
      box-shadow 0.12s ease,
      transform 0.12s ease,
      border-color 0.12s ease;
  }
  .sakuga-input:focus {
    border-color: var(--neo-accent);
    transform: translate(-1px, -1px);
    box-shadow: calc(var(--neo-shadow-size, 4px) + 1px) calc(var(--neo-shadow-size, 4px) + 1px) 0 0
      var(--neo-shadow);
  }
</style>
