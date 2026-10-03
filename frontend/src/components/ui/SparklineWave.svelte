<script lang="ts">
  interface Props {
    value?: number // percentage 0 - 100
    history?: number[] // series of historical data points
    color?: string // CSS color or hex (default: 'var(--color-accent, var(--neo-accent, #3b82f6))')
    height?: number // px (default: 28)
    animated?: boolean
    showLeadingDot?: boolean
    class?: string
  }

  let {
    value = 50,
    history,
    color = 'var(--color-accent, var(--neo-accent, #3b82f6))',
    height = 28,
    animated = true,
    showLeadingDot = true,
    class: className = '',
  }: Props = $props()

  const uid = Math.random().toString(36).substring(2, 9)
  const gradId = `spark-grad-${uid}`

  // 1. If history is supplied with at least 2 points, render authentic time-series spline
  const points = $derived.by(() => {
    if (!history || history.length < 2) return null
    const len = history.length
    const maxVal = Math.max(...history, 1)
    return history.map((val, i) => {
      const x = Number(((i / (len - 1)) * 100).toFixed(1))
      const normalized = Math.min(Math.max(val / maxVal, 0), 1)
      const y = Number((25 - normalized * 21).toFixed(1))
      return { x, y }
    })
  })

  const historyStroke = $derived.by(() => {
    if (!points || points.length < 2) return null
    let path = `M ${points[0].x},${points[0].y}`
    for (let i = 0; i < points.length - 1; i++) {
      const p0 = points[i === 0 ? i : i - 1]
      const p1 = points[i]
      const p2 = points[i + 1]
      const p3 = points[i + 2 < points.length ? i + 2 : i + 1]

      const cp1x = Number((p1.x + (p2.x - p0.x) / 6).toFixed(1))
      const cp1y = Number((p1.y + (p2.y - p0.y) / 6).toFixed(1))
      const cp2x = Number((p2.x - (p3.x - p1.x) / 6).toFixed(1))
      const cp2y = Number((p2.y - (p3.y - p1.y) / 6).toFixed(1))

      path += ` C ${cp1x},${cp1y} ${cp2x},${cp2y} ${p2.x},${p2.y}`
    }
    return path
  })

  const historyArea = $derived.by(() => {
    if (!historyStroke || !points || points.length === 0) return null
    return `${historyStroke} L 100,28 L 0,28 Z`
  })

  const leadingPoint = $derived.by(() => {
    if (!points || points.length === 0) return null
    return points[points.length - 1]
  })

  // 2. Fallback scalar formula when history is not provided
  const clamped = $derived(Math.min(Math.max(Number(value) || 0, 0), 100))
  const amp = $derived((clamped / 100) * 16 + 4)
  const y1 = $derived((22 - amp * 0.35).toFixed(1))
  const y2 = $derived((22 - amp * 0.95).toFixed(1))
  const y3 = $derived((22 - amp * 0.55).toFixed(1))
  const y4 = $derived((22 - amp * 1.0).toFixed(1))
  const y5 = $derived((22 - amp * 0.45).toFixed(1))

  const strokePath = $derived(
    historyStroke ?? `M 0,22 Q 15,${y1} 30,${y2} T 60,${y3} T 85,${y4} T 100,${y5}`
  )

  const areaPath = $derived(
    historyArea ?? `M 0,22 Q 15,${y1} 30,${y2} T 60,${y3} T 85,${y4} T 100,${y5} L 100,28 L 0,28 Z`
  )
</script>

<div
  class="w-full overflow-hidden select-none pointer-events-none {className}"
  style="height: {height}px;"
>
  <svg
    viewBox="0 0 100 28"
    preserveAspectRatio="none"
    class="w-full h-full block {animated && !history ? 'sparkline-wave-anim' : ''}"
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

    <!-- Smooth Spline stroke curve -->
    <path
      d={strokePath}
      fill="none"
      stroke={color}
      stroke-width="1.75"
      stroke-linecap="round"
      stroke-linejoin="round"
      opacity="0.85"
    />

    <!-- Leading glowing point for history -->
    {#if showLeadingDot && leadingPoint}
      <circle cx={leadingPoint.x} cy={leadingPoint.y} r="2.5" fill={color} />
      <circle cx={leadingPoint.x} cy={leadingPoint.y} r="5" fill={color} opacity="0.35" class="animate-ping" />
    {/if}
  </svg>
</div>

<style>
  @keyframes sparkWaveShift {
    0% {
      transform: translateX(0);
    }
    50% {
      transform: translateX(-1.5%);
    }
    100% {
      transform: translateX(0);
    }
  }

  .sparkline-wave-anim {
    animation: sparkWaveShift 4s ease-in-out infinite;
  }
</style>
