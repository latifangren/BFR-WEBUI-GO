<script lang="ts">
  import type { Snippet } from 'svelte'
  import { themeStore } from '../../stores/theme.svelte'
  import SakugaButton from './SakugaButton.svelte'

  interface Props {
    type?: 'button' | 'submit' | 'reset'
    variant?: 'primary' | 'secondary' | 'danger' | 'outline' | 'ghost' | 'start' | 'stop' | 'restart' | 'cyan' | 'purple'
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
    start: 'bg-[#b8f34a] text-[#000000] border-2 border-[#17151a] hover:brightness-105 shadow-[2px_2px_0px_#17151a]',
    stop: 'bg-[#ff5ca8] text-[#ffffff] border-2 border-[#17151a] hover:brightness-105 shadow-[2px_2px_0px_#17151a]',
    restart: 'bg-[#ffe14a] text-[#000000] border-2 border-[#17151a] hover:brightness-105 shadow-[2px_2px_0px_#17151a]',
    cyan: 'bg-[#55dfff] text-[#000000] border-2 border-[#17151a] hover:brightness-105 shadow-[2px_2px_0px_#17151a]',
    purple: 'bg-[#9b7cff] text-[#ffffff] border-2 border-[#17151a] hover:brightness-105 shadow-[2px_2px_0px_#17151a]',
  }

  const sizeClasses = {
    sm: 'px-2.5 py-1 text-xs',
    md: 'px-3.5 py-1.5 text-sm',
    lg: 'px-5 py-2.5 text-base',
  }

  function handleClick(e: MouseEvent) {
    if (!disabled && typeof navigator !== 'undefined' && 'vibrate' in navigator) {
      try {
        navigator.vibrate(8)
      } catch {}
    }
    onclick?.(e)
  }

  // When Sakuga FX is active, loud variants render as the native
  // SakugaButton (Svelte re-implementation, theme-driven). Semantic colors
  // stay native: danger keeps its red warning and ghost keeps its subtlety.
  // Submit buttons work too: unlike the old <sakuga-button> custom element,
  // the native button is a real <button> and form-associated.
  const useSakuga = $derived(
    themeStore.isSakugaFxActive() && variant !== 'danger' && variant !== 'ghost',
  )
  const sakugaVariant: 'primary' | 'secondary' = $derived(
    variant === 'secondary' || variant === 'outline' || variant === 'stop'
      ? 'secondary'
      : 'primary',
  )
</script>

{#if useSakuga}
  <SakugaButton
    variant={sakugaVariant}
    {type}
    size={size}
    {disabled}
    {title}
    class={fullWidth ? 'w-full' : ''}
    onclick={handleClick}
  >
    {@render children?.()}
  </SakugaButton>
{:else}
  <button
    {type}
    {disabled}
    {title}
    onclick={handleClick}
    class="neo-button inline-flex items-center justify-center font-mono font-bold select-none cursor-pointer active:scale-[0.98] active:translate-y-0.5 transition-transform disabled:opacity-50 disabled:cursor-not-allowed {variantClasses[variant]} {sizeClasses[size]} {fullWidth ? 'w-full' : ''} {className}"
  >
    {@render children?.()}
  </button>
{/if}
