<script lang="ts">
  import type { Snippet } from 'svelte'

  interface Props {
    variant?: 'default' | 'success' | 'warning' | 'danger' | 'info' | 'pop-pink' | 'pop-lime' | 'pop-yellow' | 'pop-cyan' | 'pop-purple'
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
    'pop-pink': 'bg-[#ff5ca8] text-white border-[#17151a] shadow-[1px_1px_0px_#17151a]',
    'pop-lime': 'bg-[#b8f34a] text-black border-[#17151a] shadow-[1px_1px_0px_#17151a]',
    'pop-yellow': 'bg-[#ffe14a] text-black border-[#17151a] shadow-[1px_1px_0px_#17151a]',
    'pop-cyan': 'bg-[#55dfff] text-black border-[#17151a] shadow-[1px_1px_0px_#17151a]',
    'pop-purple': 'bg-[#9b7cff] text-white border-[#17151a] shadow-[1px_1px_0px_#17151a]',
  }
</script>

<span class="inline-flex items-center px-2 py-0.5 rounded text-[10px] font-mono font-bold uppercase tracking-wider border {variantStyles[variant]} {className}">
  {@render children?.()}
</span>
