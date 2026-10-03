<script lang="ts">
  interface Props {
    value?: number
    decimals?: number
    duration?: number
    suffix?: string
    class?: string
  }

  let {
    value = 0,
    decimals = 0,
    duration = 350,
    suffix = '',
    class: className = '',
  }: Props = $props()

  let current = $state(0)
  let initialized = false
  let animationId: number | null = null

  $effect(() => {
    const target = Number(value) || 0
    if (!initialized) {
      current = target
      initialized = true
      return
    }

    const start = current
    if (Math.abs(target - start) < 0.01) {
      current = target
      return
    }

    const startTime = performance.now()

    function step(now: number) {
      const elapsed = now - startTime
      const progress = Math.min(elapsed / duration, 1)
      const ease = 1 - Math.pow(1 - progress, 3)
      current = start + (target - start) * ease

      if (progress < 1) {
        animationId = requestAnimationFrame(step)
      } else {
        current = target
      }
    }

    if (animationId !== null) {
      cancelAnimationFrame(animationId)
    }
    animationId = requestAnimationFrame(step)

    return () => {
      if (animationId !== null) {
        cancelAnimationFrame(animationId)
      }
    }
  })

  const formatted = $derived(
    decimals > 0 ? current.toFixed(decimals) : Math.round(current).toString()
  )
</script>

<span class={className}>{formatted}{suffix}</span>
