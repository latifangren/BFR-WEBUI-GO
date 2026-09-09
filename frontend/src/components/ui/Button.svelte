<script lang="ts">
  import type { Snippet } from 'svelte'

  interface Props {
    type?: 'button' | 'submit' | 'reset'
    variant?: 'primary' | 'secondary' | 'danger' | 'outline' | 'ghost'
    size?: 'sm' | 'md' | 'lg'
    disabled?: boolean
    fullWidth?: boolean
    title?: string
    class?: string
    onclick?: (e: MouseEvent) => void
    children?: Snippet
  }

  let {
    type = 'button',
    variant = 'primary',
    size = 'md',
    disabled = false,
    fullWidth = false,
    title,
    class: className = '',
    onclick,
    children,
  }: Props = $props()

  const variantClasses = {
    primary: 'bg-accent text-accent-text hover:brightness-110 border-border',
    secondary: 'bg-card text-foreground hover:bg-card-sub border-border',
    danger: 'bg-red-600 text-white hover:bg-red-700 border-red-800',
    outline: 'bg-transparent text-foreground hover:bg-card border-border',
    ghost: 'bg-transparent text-muted hover:text-foreground border-transparent shadow-none',
  }

  const sizeClasses = {
    sm: 'px-2.5 py-1 text-xs',
    md: 'px-3.5 py-1.5 text-sm',
    lg: 'px-5 py-2.5 text-base',
  }
</script>

<button
  {type}
  {disabled}
  {title}
  {onclick}
  class="neo-button inline-flex items-center justify-center font-mono font-bold select-none cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed {variantClasses[variant]} {sizeClasses[size]} {fullWidth ? 'w-full' : ''} {className}"
>
  {@render children?.()}
</button>
