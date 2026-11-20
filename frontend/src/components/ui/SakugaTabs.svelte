<script lang="ts">
  // Wrapper around the vendored <sakuga-tabs> custom element
  // (sakuga-vfx, MIT, loaded lazily by the theme store). One wrapper
  // replaces a static tab bar: pass the tab labels as options.
  // Only rendered at callsites that already branch on
  // themeStore.isSakugaFxActive(); no native fallback of its own.
  // Main navigation (TopNav / Sidebar / BottomPopoverNav) is out of
  // scope for this wrapper and must keep its native implementation.
  let {
    selected = $bindable(0),
    label = '',
    options = [],
    onchange,
  }: {
    selected?: number
    label?: string
    options?: string[]
    onchange?: (e: Event) => void
  } = $props()

  let el: any = $state(null)

  $effect(() => {
    if (el && el.selected !== selected) el.selected = selected
  })

  function handleChange(e: Event) {
    selected = (e.target as any).selected
    onchange?.(e)
  }
</script>

<sakuga-tabs
  bind:this={el}
  label={label || undefined}
  onchange={handleChange}
>
  {#each options as opt, i}
    <option selected={i === selected}>{opt}</option>
  {/each}
</sakuga-tabs>
