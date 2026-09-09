<script lang="ts">
  import type { Snippet } from 'svelte'

  interface Props {
    variant?: 'default' | 'success' | 'warning' | 'danger' | 'info'
    class?: string
    children?: Snippet
  }

  let {
    variant = 'default',
    class: className = '',
    children,
  }: Props = $props()

  const variantStyles = {
    default: 'bg-card-sub text-muted border-border',
    success: 'bg-emerald-950/60 text-emerald-400 border-emerald-800',
    warning: 'bg-amber-950/60 text-amber-400 border-amber-800',
    danger: 'bg-red-950/60 text-red-400 border-red-800',
    info: 'bg-cyan-950/60 text-cyan-400 border-cyan-800',
  }
</script>

<span class="inline-flex items-center px-2 py-0.5 rounded text-[10px] font-mono font-bold uppercase tracking-wider border {variantStyles[variant]} {className}">
  {@render children?.()}
</span>
