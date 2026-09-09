<script lang="ts">
  interface Props {
    id?: string
    type?: string
    value?: string
    placeholder?: string
    disabled?: boolean
    label?: string
    error?: string
    class?: string
    oninput?: (e: Event) => void
    onchange?: (e: Event) => void
  }

  let {
    id = 'input-' + Math.random().toString(36).substring(2, 9),
    type = 'text',
    value = $bindable(''),
    placeholder = '',
    disabled = false,
    label,
    error,
    class: className = '',
    oninput,
    onchange,
  }: Props = $props()
</script>

<div class="space-y-1 w-full">
  {#if label}
    <label for={id} class="block text-xs font-mono font-bold uppercase text-muted tracking-wider">
      {label}
    </label>
  {/if}
  <input
    {id}
    {type}
    bind:value
    {placeholder}
    {disabled}
    {oninput}
    {onchange}
    class="w-full bg-card-sub border border-border rounded px-3 py-2 text-sm font-mono text-foreground placeholder:text-muted focus:outline-none focus:border-accent disabled:opacity-50 transition-colors {className}"
  />
  {#if error}
    <p class="text-xs text-red-500 font-mono">{error}</p>
  {/if}
</div>
