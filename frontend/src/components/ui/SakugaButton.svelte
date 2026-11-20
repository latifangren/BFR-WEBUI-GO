<script lang="ts">
  import type { Snippet } from 'svelte'

  // Thin wrapper around the vendored <sakuga-button> custom element
  // (sakuga-vfx, MIT, loaded lazily by the theme store). Primary is the
  // default variant; pass variant="secondary" for the calmer one.
  let {
    children,
    variant = 'primary',
    onclick,
  }: {
    children: Snippet
    variant?: 'primary' | 'secondary'
    onclick?: (e: MouseEvent) => void
  } = $props()
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<sakuga-button
  variant={variant === 'secondary' ? 'secondary' : undefined}
  onclick={onclick}
>
  {@render children()}
</sakuga-button>
