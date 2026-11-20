<script lang="ts">
  // Wrapper around the vendored <sakuga-progress> custom element
  // (sakuga-vfx, MIT, loaded lazily by the theme store). Display-only:
  // pass progress as 0..1. Only rendered at callsites that already
  // branch on themeStore.isSakugaFxActive(); no native fallback.
  let {
    value = 0,
    label = '',
  }: {
    value?: number
    label?: string
  } = $props()

  let el: any = $state(null)

  $effect(() => {
    if (el) el.set(Math.max(0, Math.min(1, value)))
  })
</script>

<sakuga-progress
  bind:this={el}
  label={label || undefined}
></sakuga-progress>
