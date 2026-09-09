<script lang="ts">
  interface Props {
    value?: number // percentage 0 - 100
    color?: string // CSS color or hex (default: 'var(--color-accent, var(--neo-accent, #3b82f6))')
    height?: number // px (default: 28)
    animated?: boolean
    class?: string
  }

  let {
    value = 50,
    color = 'var(--color-accent, var(--neo-accent, #3b82f6))',
    height = 28,
    animated = true,
    class: className = '',
  }: Props = $props()

  const uid = Math.random().toString(36).substring(2, 9)
  const gradId = `spark-grad-${uid}`

  // Calculate wave curve based on normalized value (0 - 100)
  const clamped = $derived(Math.min(Math.max(Number(value) || 0, 0), 100))

  // Amplitude ranges dynamically relative to 28px height viewBox
  const amp = $derived((clamped / 100) * 16 + 4)
  const y1 = $derived((22 - amp * 0.35).toFixed(1))
  const y2 = $derived((22 - amp * 0.95).toFixed(1))
  const y3 = $derived((22 - amp * 0.55).toFixed(1))
  const y4 = $derived((22 - amp * 1.0).toFixed(1))
  const y5 = $derived((22 - amp * 0.45).toFixed(1))

  const strokePath = $derived(
    `M 0,22 Q 15,${y1} 30,${y2} T 60,${y3} T 85,${y4} T 100,${y5}`
  )

  const areaPath = $derived(
    `M 0,22 Q 15,${y1} 30,${y2} T 60,${y3} T 85,${y4} T 100,${y5} L 100,28 L 0,28 Z`
  )
</script>

<div
  class="w-full overflow-hidden select-none pointer-events-none {className}"
  style="height: {height}px;"
>
  <svg
    viewBox="0 0 100 28"
    preserveAspectRatio="none"
    class="w-full h-full block {animated ? 'sparkline-wave-anim' : ''}"
    xmlns="http://www.w3.org/2000/svg"
  >
    <defs>
      <linearGradient id={gradId} x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" stop-color={color} stop-opacity="0.32" />
        <stop offset="100%" stop-color={color} stop-opacity="0.0" />
      </linearGradient>
    </defs>

    <!-- Closed area gradient fill -->
    <path d={areaPath} fill="url(#{gradId})" />

    <!-- Cubic Bézier stroke curve -->
    <path
      d={strokePath}
      fill="none"
      stroke={color}
      stroke-width="1.75"
      stroke-linecap="round"
      stroke-linejoin="round"
      opacity="0.85"
    />
  </svg>
</div>

<style>
  .sparkline-wave-anim {
    transform-origin: bottom center;
    animation: wave-pulse 3.5s ease-in-out infinite;
  }

  @keyframes wave-pulse {
    0%,
    100% {
      transform: scaleY(1);
      opacity: 0.85;
    }
    50% {
      transform: scaleY(1.08);
      opacity: 1;
    }
  }
</style>
