<script lang="ts">
  import { themeStore } from '../../stores/theme.svelte'
  import SakugaInput from './SakugaInput.svelte'

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
    onkeydown?: (e: KeyboardEvent) => void
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
    onkeydown,
  }: Props = $props()

  // When Sakuga FX is active, text-like inputs render as the native
  // SakugaInput. Other types (number, date, file, ...) stay on the stock
  // input because their native pickers should not be restyled.
  const useSakuga = $derived(
    themeStore.useSakugaControls() &&
      ['text', 'email', 'search', 'url', 'tel', 'password'].includes(type),
  )
</script>

<div class="space-y-1 w-full">
  {#if useSakuga}
    <SakugaInput
      bind:value
      {type}
      label={label ?? ''}
      {placeholder}
      {disabled}
      {oninput}
      {onchange}
      {onkeydown}
    />
  {:else}
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
      {onkeydown}
      class="neo-input w-full bg-card-sub border border-border rounded px-3 py-2 text-sm font-mono text-foreground placeholder:text-muted focus:outline-none focus:border-accent disabled:opacity-50 transition-colors {className}"
    />
  {/if}
  {#if error}
    <p class="text-xs text-red-500 font-mono">{error}</p>
  {/if}
</div>
