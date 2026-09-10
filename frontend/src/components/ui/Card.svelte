<script lang="ts">
  import type { Snippet } from 'svelte'

  interface Props {
    title?: string
    subtitle?: string
    tone?: 'ice' | 'lavender' | 'peach' | 'mint' | 'butter' | 'none'
    class?: string
    headerClass?: string
    bodyClass?: string
    action?: Snippet
    children?: Snippet
  }

  let {
    title,
    subtitle,
    tone = 'none',
    class: className = '',
    headerClass = '',
    bodyClass = '',
    action,
    children,
  }: Props = $props()
</script>

<div class="neo-card bg-card p-4 sm:p-5 text-foreground {tone && tone !== 'none' ? 'neo-tone-' + tone : ''} {className}">
  {#if title || action}
    <div class="flex items-center justify-between border-b border-border pb-3 mb-4 {headerClass}">
      <div>
        {#if title}
          <h3 class="text-xs sm:text-sm font-bold font-mono tracking-wider uppercase text-foreground">
            {title}
          </h3>
        {/if}
        {#if subtitle}
          <p class="text-xs text-muted mt-0.5">{subtitle}</p>
        {/if}
      </div>
      {#if action}
        <div class="flex items-center gap-2">
          {@render action()}
        </div>
      {/if}
    </div>
  {/if}

  <div class={bodyClass}>
    {@render children?.()}
  </div>
</div>
