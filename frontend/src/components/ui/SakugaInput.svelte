<script lang="ts">
  // Wrapper around the vendored <sakuga-input> custom element
  // (sakuga-vfx, MIT, loaded lazily by the theme store). Only rendered
  // at callsites that already branch on themeStore.isSakugaFxActive();
  // this wrapper has no native fallback of its own.
  let {
    value = $bindable(''),
    type = 'text',
    label = '',
    placeholder = '',
    disabled = false,
    oninput,
    onchange,
    onkeydown,
  }: {
    value?: string
    type?: string
    label?: string
    placeholder?: string
    disabled?: boolean
    oninput?: (e: Event) => void
    onchange?: (e: Event) => void
    onkeydown?: (e: KeyboardEvent) => void
  } = $props()

  let el: any = $state(null)

  $effect(() => {
    if (el && el.value !== value) el.value = value
  })

  function handleInput(e: Event) {
    value = (e.target as any).value
    oninput?.(e)
  }

  function handleChange(e: Event) {
    value = (e.target as any).value
    onchange?.(e)
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<sakuga-input
  bind:this={el}
  {type}
  label={label || undefined}
  placeholder={placeholder || undefined}
  {disabled}
  oninput={handleInput}
  onchange={handleChange}
  onkeydown={onkeydown}
></sakuga-input>
