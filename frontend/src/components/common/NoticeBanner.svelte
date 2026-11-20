<script lang="ts">
  import type { Snippet } from 'svelte'
  import { Lightbulb, AlertTriangle, Info } from '@lucide/svelte'
  import { themeStore } from '../../stores/theme.svelte'

  interface Props {
    title?: string
    message?: string
    icon?: 'bulb' | 'alert' | 'info'
    class?: string
    children?: Snippet
  }

  let {
    title = 'NOTICE',
    message,
    icon = 'bulb',
    class: className = '',
    children,
  }: Props = $props()

  const isNeobrutal = $derived(themeStore.currentStyle === 'neobrutal')
</script>

<div
  class="w-full font-mono text-xs flex items-start gap-3 transition-all select-none {isNeobrutal
      ? 'bg-amber-950/30 border-2 border-amber-500/40 text-amber-200 shadow-neobrutal-sm rounded-lg p-3.5'
      : 'bg-amber-500/10 border border-amber-500/30 text-amber-200 rounded-xl p-3.5 backdrop-blur-sm'} {className}"
  role="alert"
>
  <!-- Icon Box -->
  <div
    class="shrink-0 flex items-center justify-center w-7 h-7 rounded bg-amber-500/20 border border-amber-500/40 text-amber-400"
  >
    {#if icon === 'bulb'}
      <Lightbulb class="w-4 h-4" />
    {:else if icon === 'alert'}
      <AlertTriangle class="w-4 h-4" />
    {:else}
      <Info class="w-4 h-4" />
    {/if}
  </div>

  <!-- Content Body -->
  <div class="space-y-0.5 flex-1 min-w-0">
    <div
      class="text-[11px] font-black uppercase tracking-wider flex items-center gap-2 text-amber-300"
    >
      <span>{title}</span>
    </div>
    {#if message}
      <p class="text-xs leading-relaxed text-amber-200/90">
        {message}
      </p>
    {/if}
    {#if children}
      <div class="mt-1">
        {@render children()}
      </div>
    {/if}
  </div>
</div>
