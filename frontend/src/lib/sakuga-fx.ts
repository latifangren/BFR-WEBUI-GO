// Native Sakuga FX engine: canvas burst particles + manga SFX text, fired
// on interaction with the native Sakuga components. No vendored scripts, no
// custom elements: one fixed canvas overlay plus absolutely-positioned SFX
// labels, both colored from the active theme tokens so dark/light
// Neobrutalism stay in sync. Respects prefers-reduced-motion and only runs
// while Sakuga FX is active (the components only render in that mode).
import { themeStore } from '../stores/theme.svelte'

export type SakugaFxKind =
  | 'button'
  | 'toggle'
  | 'checkbox'
  | 'radio'
  | 'slider'
  | 'tabs'
  | 'error'
  | 'complete'
  | 'charge'
  | 'sparkle'

const SFX_WORDS: Record<SakugaFxKind, string[]> = {
  button: ['ドン!', 'バン!'],
  toggle: ['カチッ!'],
  checkbox: ['ポン!'],
  radio: ['ポン!'],
  slider: ['シュッ!'],
  tabs: ['パッ!'],
  error: ['ガーン!'],
  complete: ['完了!'],
  charge: ['チャージ!'],
  sparkle: ['キラッ!'],
}

interface Particle {
  x: number
  y: number
  vx: number
  vy: number
  size: number
  rot: number
  vr: number
  life: number
  maxLife: number
  color: string
  shape: 'chip' | 'dot' | 'slash'
}

let canvas: HTMLCanvasElement | null = null
let ctx: CanvasRenderingContext2D | null = null
let particles: Particle[] = []
let rafId: number | null = null
let lastTick = 0

function themeColors(): string[] {
  const cs = getComputedStyle(document.documentElement)
  const pick = (name: string, fallback: string) => cs.getPropertyValue(name).trim() || fallback
  // Pop colors only exist in the Neobrutalism themes; elsewhere fall back
  // to the theme's own semantic colors so bursts stay theme-native
  // (accent + success/warning/danger/info exist in every theme).
  return [
    pick('--neo-accent', '#ffe14a'),
    pick('--color-pink', pick('--neo-danger', '#ff5ca8')),
    pick('--color-cyan', pick('--neo-info', '#55dfff')),
    pick('--color-lime', pick('--neo-success', '#b8f34a')),
    pick('--color-purple', pick('--neo-accent', '#9b7cff')),
    pick('--neo-text', '#fffaf4'),
  ]
}

function ensureCanvas(): boolean {
  if (typeof document === 'undefined') return false
  if (!canvas) {
    canvas = document.createElement('canvas')
    canvas.style.cssText =
      'position:fixed;inset:0;width:100vw;height:100vh;pointer-events:none;z-index:2147483000'
    document.body.appendChild(canvas)
    ctx = canvas.getContext('2d')
  }
  const dpr = Math.min(window.devicePixelRatio || 1, 2)
  const w = Math.floor(window.innerWidth * dpr)
  const h = Math.floor(window.innerHeight * dpr)
  if (canvas.width !== w || canvas.height !== h) {
    canvas.width = w
    canvas.height = h
  }
  ctx?.setTransform(dpr, 0, 0, dpr, 0, 0)
  return !!ctx
}

function spawnBurst(x: number, y: number): void {
  const colors = themeColors()
  for (let i = 0; i < 16; i++) {
    const angle = Math.random() * Math.PI * 2
    const speed = 90 + Math.random() * 260
    const roll = Math.random()
    const shape: Particle['shape'] = roll < 0.55 ? 'chip' : roll < 0.8 ? 'slash' : 'dot'
    particles.push({
      x,
      y,
      vx: Math.cos(angle) * speed,
      vy: Math.sin(angle) * speed - 60,
      size: shape === 'slash' ? 14 + Math.random() * 14 : 4 + Math.random() * 7,
      rot: Math.random() * Math.PI,
      vr: (Math.random() - 0.5) * 12,
      life: 0,
      maxLife: 0.45 + Math.random() * 0.25,
      color: colors[Math.floor(Math.random() * colors.length)],
      shape,
    })
  }
  if (rafId === null) {
    lastTick = 0
    rafId = requestAnimationFrame(tick)
  }
}

function tick(t: number): void {
  if (lastTick === 0) lastTick = t
  const dt = Math.min((t - lastTick) / 1000, 0.05)
  lastTick = t
  if (!ctx) {
    rafId = null
    return
  }
  ctx.clearRect(0, 0, window.innerWidth, window.innerHeight)
  particles = particles.filter((p) => {
    p.life += dt
    if (p.life >= p.maxLife) return false
    p.vy += 320 * dt
    p.x += p.vx * dt
    p.y += p.vy * dt
    p.vx *= 0.98
    p.rot += p.vr * dt
    const alpha = 1 - p.life / p.maxLife
    ctx!.save()
    ctx!.globalAlpha = alpha
    ctx!.translate(p.x, p.y)
    ctx!.rotate(p.rot)
    ctx!.fillStyle = p.color
    if (p.shape === 'chip') {
      ctx!.fillRect(-p.size / 2, -p.size / 4, p.size, p.size / 2)
    } else if (p.shape === 'slash') {
      ctx!.fillRect(-p.size / 2, -1.5, p.size, 3)
    } else {
      ctx!.beginPath()
      ctx!.arc(0, 0, p.size / 2, 0, Math.PI * 2)
      ctx!.fill()
    }
    ctx!.restore()
    return true
  })
  if (particles.length > 0) {
    rafId = requestAnimationFrame(tick)
  } else {
    ctx.clearRect(0, 0, window.innerWidth, window.innerHeight)
    rafId = null
  }
}

function spawnSfx(x: number, y: number, kind: SakugaFxKind): void {
  const words = SFX_WORDS[kind]
  const text = words[Math.floor(Math.random() * words.length)]
  const el = document.createElement('div')
  el.className = 'sakuga-sfx'
  el.textContent = text
  el.style.left = `${x}px`
  el.style.top = `${y}px`
  document.body.appendChild(el)
  el.addEventListener('animationend', () => el.remove(), { once: true })
  // Fallback removal in case animationend never fires.
  window.setTimeout(() => el.remove(), 900)
}

// Fire a burst + SFX word at a point. Pass the MouseEvent for a precise
// click position, an Element to burst at its center (for change events,
// which carry no coordinates), or raw viewport { x, y } coordinates when
// the anchor may unmount before the fire (e.g. a login button after the
// awaited login flips the view). No-op when Sakuga FX is off or the user
// prefers reduced motion.
export function fireSakugaFx(
  source: MouseEvent | Element | { x: number; y: number },
  kind: SakugaFxKind,
): void {
  if (typeof window === 'undefined' || typeof document === 'undefined') return
  if (!themeStore.isSakugaFxActive()) return
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return
  let x: number
  let y: number
  if (source instanceof MouseEvent) {
    x = source.clientX
    y = source.clientY
  } else if (source instanceof Element) {
    const rect = source.getBoundingClientRect()
    if (!rect) return
    x = rect.left + rect.width / 2
    y = rect.top + rect.height / 2
  } else {
    x = source.x
    y = source.y
  }
  if (!ensureCanvas()) return
  spawnBurst(x, y)
  spawnSfx(x, y - 16, kind)
}
