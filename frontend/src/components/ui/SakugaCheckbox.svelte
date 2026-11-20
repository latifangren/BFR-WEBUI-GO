<script lang="ts">
  import type { Snippet } from 'svelte'

  // Wrapper around the vendored <sakuga-checkbox> custom element
  // (sakuga-vfx, MIT, loaded lazily by the theme store). Only rendered
  // at callsites that already branch on themeStore.isSakugaFxActive();
  // this wrapper has no native fallback of its own.
  let {
    checked = $bindable(false),
    disabled = false,
    onchange,
    children,
  }: {
    checked?: boolean
    disabled?: boolean
    onchange?: (e: Event) => void
    children?: Snippet
  } = $props()

  let el: any = $state(null)

  $effect(() => {
    if (el && el.checked !== checked) el.checked = checked
  })

  function handleChange(e: Event) {
    checked = (e.target as any).checked
    onchange?.(e)
  }
</script>

<sakuga-checkbox
  bind:this={el}
  {disabled}
  onchange={handleChange}
>
  {@render children?.()}
</sakuga-checkbox>
