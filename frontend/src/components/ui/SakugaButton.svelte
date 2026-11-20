<script lang="ts">
  import type { Snippet } from 'svelte'
  import { fireSakugaFx } from '../../lib/sakuga-fx'

  // Native Svelte re-implementation of the Sakuga button look (paper card,
  // thick border, hard shadow, italic shout) driven by the theme tokens, so
  // dark/light Neobrutalism stay in sync. A real <button>: type="submit"
  // and sizes work, unlike the old <sakuga-button> custom element.
  let {
    children,
    variant = 'primary',
    size = 'md',
    type = 'button',
    disabled = false,
    title,
    class: className = '',
    onclick,
  }: {
    children: Snippet
    variant?: 'primary' | 'secondary'
    size?: 'sm' | 'md' | 'lg'
    type?: 'button' | 'submit' | 'reset'
    disabled?: boolean
    title?: string
    class?: string
    onclick?: (e: MouseEvent) => void
  } = $props()

  const sizeClasses = {
    sm: 'px-3 py-1 text-xs',
    md: 'px-4 py-2 text-sm',
    lg: 'px-6 py-3 text-base',
  }

  function handleClick(e: MouseEvent) {
    if (disabled) return
    fireSakugaFx(e, 'button')
    onclick?.(e)
  }
</script>

<button
  {type}
  {disabled}
  {title}
  onclick={handleClick}
  class="sakuga-btn inline-flex items-center justify-center font-mono font-black italic select-none cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed {variant === 'secondary'
    ? 'sakuga-btn-secondary'
    : 'sakuga-btn-primary'} {sizeClasses[size]} {className}"
>
  {@render children()}
</button>

<style>
  .sakuga-btn {
    border: var(--neo-border-width, 2px) solid var(--neo-border);
    border-radius: var(--neo-radius, 4px);
    box-shadow: var(--neo-shadow-size, 4px) var(--neo-shadow-size, 4px) 0 0 var(--neo-shadow);
    transition:
      transform 0.12s cubic-bezier(0.34, 1.56, 0.64, 1),
      box-shadow 0.12s cubic-bezier(0.34, 1.56, 0.64, 1),
      filter 0.12s ease;
  }
  .sakuga-btn-primary {
    background-color: var(--neo-accent);
    color: var(--neo-accent-text);
  }
  .sakuga-btn-secondary {
    background-color: var(--neo-card-sub);
    color: var(--neo-text);
  }
  .sakuga-btn:hover:not(:disabled) {
    filter: brightness(1.08);
  }
  .sakuga-btn:active:not(:disabled) {
    transform: translate(var(--neo-press-depth, 2px), var(--neo-press-depth, 2px)) scale(0.97);
    box-shadow: 1px 1px 0 0 var(--neo-shadow);
  }
</style>
