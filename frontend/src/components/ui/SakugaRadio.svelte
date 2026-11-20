<script lang="ts">
  // Wrapper around the vendored <sakuga-radio> custom element
  // (sakuga-vfx, MIT, loaded lazily by the theme store). One wrapper
  // replaces a whole radio group: pass the choices as options.
  // Only rendered at callsites that already branch on
  // themeStore.isSakugaFxActive(); no native fallback of its own.
  export interface SakugaRadioOption {
    value: string
    label: string
  }

  let {
    value = $bindable(''),
    label = '',
    options = [],
    disabled = false,
    onchange,
  }: {
    value?: string
    label?: string
    options?: SakugaRadioOption[]
    disabled?: boolean
    onchange?: (e: Event) => void
  } = $props()

  let el: any = $state(null)

  $effect(() => {
    if (el && el.value !== value) el.value = value
  })

  function handleChange(e: Event) {
    value = (e.target as any).value
    onchange?.(e)
  }
</script>

<sakuga-radio
  bind:this={el}
  label={label || undefined}
  {disabled}
  onchange={handleChange}
>
  {#each options as opt}
    <option value={opt.value} selected={opt.value === value}>{opt.label}</option>
  {/each}
</sakuga-radio>
