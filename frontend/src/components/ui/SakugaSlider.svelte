<script lang="ts">
  // Wrapper around the vendored <sakuga-slider> custom element
  // (sakuga-vfx, MIT, loaded lazily by the theme store). Only rendered
  // at callsites that already branch on themeStore.isSakugaFxActive();
  // this wrapper has no native fallback of its own.
  let {
    value = $bindable(50),
    min = 0,
    max = 100,
    step = 1,
    label = '',
    unit = '',
    disabled = false,
    oninput,
    onchange,
  }: {
    value?: number
    min?: number
    max?: number
    step?: number
    label?: string
    unit?: string
    disabled?: boolean
    oninput?: (e: Event) => void
    onchange?: (e: Event) => void
  } = $props()

  let el: any = $state(null)

  $effect(() => {
    if (el && el.valueAsNumber !== value) el.value = value
  })

  function handleInput(e: Event) {
    value = (e.target as any).valueAsNumber ?? Number((e.target as any).value)
    oninput?.(e)
  }

  function handleChange(e: Event) {
    value = (e.target as any).valueAsNumber ?? Number((e.target as any).value)
    onchange?.(e)
  }
</script>

<sakuga-slider
  bind:this={el}
  value={String(value)}
  {min}
  {max}
  {step}
  label={label || undefined}
  unit={unit || undefined}
  {disabled}
  oninput={handleInput}
  onchange={handleChange}
></sakuga-slider>
