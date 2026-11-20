<script lang="ts">
  // Wrapper around the vendored <sakuga-toggle> custom element
  // (sakuga-vfx, MIT, loaded lazily by the theme store). Only rendered
  // at callsites that already branch on themeStore.isSakugaFxActive();
  // this wrapper has no native fallback of its own.
  let {
    checked = $bindable(false),
    label = '',
    disabled = false,
    onchange,
  }: {
    checked?: boolean
    label?: string
    disabled?: boolean
    onchange?: (e: Event) => void
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

<sakuga-toggle
  bind:this={el}
  label={label || undefined}
  {disabled}
  onchange={handleChange}
></sakuga-toggle>
