import type { ThemeMode, UIStyle } from '../types/common'

const VALID_THEMES: ThemeMode[] = [
  'dark',
  'light',
  'neobrutalism-light',
  'neobrutalism',
]

// Only modern and glass are user-selectable styles. 'neobrutal' is the
// locked shape of the Neobrutalism themes and is forced by setTheme.
const VALID_STYLES: UIStyle[] = ['modern', 'glass']

export function isNeobrutalTheme(theme: ThemeMode): boolean {
  return theme === 'neobrutalism' || theme === 'neobrutalism-light'
}

// Themes removed in the 4-theme prune: amoled, dracula, nord, cyberpunk,
// emerald, sunset, retro. Stored values map to the closest kept theme:
// retro (light) -> neobrutalism-light, all removed dark themes -> dark.
// The ProjectX rename: projectx -> neobrutalism,
// projectx-light -> neobrutalism-light.
const REMOVED_THEME_FALLBACK: Record<string, ThemeMode> = {
  retro: 'neobrutalism-light',
  projectx: 'neobrutalism',
  'projectx-light': 'neobrutalism-light',
}

class ThemeStore {
  currentTheme = $state<ThemeMode>('neobrutalism')
  currentStyle = $state<UIStyle>('neobrutal')
  showAppearanceModal = $state<boolean>(false)
  glassBlur = $state<number>(16)
  glassOpacity = $state<number>(0.65)
  glassDim = $state<number>(0.75)
  activeWallpaper = $state<string>('')
  loginWallpaper = $state<string>('')
  loginWallpaperDim = $state<number>(0.4)
  loginWallpaperBlur = $state<number>(4)
  neoShadowSize = $state<number>(4)
  neoBorderWidth = $state<number>(2)
  neoRadius = $state<number>(4)
  neoPressDepth = $state<number>(2)
  neoPattern = $state<'dots' | 'grid' | 'none'>('dots')
  neoPatternOpacity = $state<number>(0.04)
  // Sakuga FX: anime-style controls + burst effects, re-implemented
  // natively in Svelte (src/components/ui/Sakuga*.svelte,
  // src/lib/sakuga-fx.ts) and driven by the theme tokens. The FX burst
  // layer is global — it follows the sakugaFx flag in every theme — while
  // the Sakuga controls keep the Neobrutalism look and only render in
  // Neobrutalism themes (see useSakugaControls). No scripts are loaded:
  // the components are plain Svelte, so there is no custom-element
  // upgrade race.
  sakugaFx = $state<boolean>(false)

  // Custom accent color (accent picker): a user-chosen accent that
  // overrides the active theme's accent on every theme. Empty means
  // "use the theme's own accent". Persisted in localStorage; it survives
  // theme switches because the override lives in inline properties on
  // documentElement, which setTheme never touches.
  customAccent = $state<string>('')
  constructor() {
    if (typeof window !== 'undefined') {
      const storedTheme = localStorage.getItem('colorTheme') || localStorage.getItem('theme')
      const migratedTheme = REMOVED_THEME_FALLBACK[storedTheme ?? ''] ?? storedTheme
      // Fresh install (nothing stored) opens on Neobrutalism, the new
      // default; a stored but unknown/removed theme falls back to dark.
      const initialTheme = (
        VALID_THEMES.includes(migratedTheme as ThemeMode)
          ? migratedTheme
          : storedTheme == null
            ? 'neobrutalism'
            : 'dark'
      ) as ThemeMode
      // Drop the removed color-vibe setting so a stale value can never
      // override a kept theme's accent again.
      localStorage.removeItem('colorVibe')
      document.documentElement.removeAttribute('data-color')

      const storedStyle = localStorage.getItem('uiStyle')
      // Neobrutalism themes lock the neobrutal shape; neutral themes keep
      // the user's style, falling back to modern if the locked style or
      // an unknown value was stored.
      const initialStyle = (
        isNeobrutalTheme(initialTheme)
          ? 'neobrutal'
          : VALID_STYLES.includes(storedStyle as UIStyle)
            ? storedStyle
            : 'modern'
      ) as UIStyle

      const storedBlur = localStorage.getItem('glassBlur')
      if (storedBlur) this.glassBlur = Math.max(4, Math.min(32, Number(storedBlur) || 16))

      const storedOpacity = localStorage.getItem('glassOpacity')
      if (storedOpacity) this.glassOpacity = Math.max(0.2, Math.min(0.95, Number(storedOpacity) || 0.65))

      const storedDim = localStorage.getItem('glassDim')
      if (storedDim) this.glassDim = Math.max(0.2, Math.min(1.0, Number(storedDim) || 0.75))

      const storedWallpaper = localStorage.getItem('activeWallpaper')
      if (storedWallpaper) this.activeWallpaper = storedWallpaper
      const storedLoginWp = localStorage.getItem('loginWallpaper')
      if (storedLoginWp) this.loginWallpaper = storedLoginWp

      const storedLoginDim = localStorage.getItem('loginWallpaperDim')
      if (storedLoginDim) this.loginWallpaperDim = Math.max(0, Math.min(0.9, Number(storedLoginDim) || 0.4))

      const storedLoginBlur = localStorage.getItem('loginWallpaperBlur')
      if (storedLoginBlur) this.loginWallpaperBlur = Math.max(0, Math.min(24, Number(storedLoginBlur) || 4))
      const storedNeoShadow = localStorage.getItem('neoShadowSize')
      if (storedNeoShadow) this.neoShadowSize = Math.max(0, Math.min(8, Number(storedNeoShadow) || 4))
      const storedNeoBorder = localStorage.getItem('neoBorderWidth')
      if (storedNeoBorder) this.neoBorderWidth = Math.max(1, Math.min(4, Number(storedNeoBorder) || 2))

      const storedNeoRadius = localStorage.getItem('neoRadius')
      if (storedNeoRadius) this.neoRadius = Math.max(0, Math.min(14, Number(storedNeoRadius) || 4))

      const storedNeoPress = localStorage.getItem('neoPressDepth')
      if (storedNeoPress) this.neoPressDepth = Math.max(1, Math.min(4, Number(storedNeoPress) || 2))

      const storedNeoPattern = localStorage.getItem('neoPattern')
      if (storedNeoPattern === 'dots' || storedNeoPattern === 'grid' || storedNeoPattern === 'none') {
        this.neoPattern = storedNeoPattern
      }

      const storedNeoPatOp = localStorage.getItem('neoPatternOpacity')
      if (storedNeoPatOp) this.neoPatternOpacity = Math.max(0, Math.min(0.15, Number(storedNeoPatOp) || 0.04))

      this.sakugaFx = localStorage.getItem('sakugaFx') === 'true'

      const storedAccent = localStorage.getItem('customAccent')
      if (storedAccent && /^#[0-9a-fA-F]{6}$/.test(storedAccent)) {
        this.customAccent = storedAccent.toLowerCase()
      }

      this.setTheme(initialTheme)
      this.setStyle(initialStyle)
      this.applyGlassTokens()
      this.applyNeobrutalTokens()
      this.applyAccentOverride()
    }
  }

  openAppearanceModal() {
    this.showAppearanceModal = true
  }

  closeAppearanceModal() {
    this.showAppearanceModal = false
  }

  isNeobrutalTheme(): boolean {
    return isNeobrutalTheme(this.currentTheme)
  }

  // The FX burst layer (canvas bursts + manga SFX): global, follows the
  // sakugaFx flag in every theme and style.
  isSakugaFxActive(): boolean {
    return this.sakugaFx
  }

  // The Sakuga controls (the native Sakuga* components): they keep the
  // Neobrutalism look (thick borders, hard shadows) and only render in
  // Neobrutalism themes. Every callsite that branches between a Sakuga
  // component and its native fallback uses this gate.
  useSakugaControls(): boolean {
    return this.sakugaFx && isNeobrutalTheme(this.currentTheme)
  }

  setSakugaFx(enabled: boolean) {
    this.sakugaFx = enabled
    if (typeof window !== 'undefined') {
      localStorage.setItem('sakugaFx', String(enabled))
    }
  }

  // Readable text color for an accent: black on bright accents, white on
  // dark ones, by perceived brightness. The 0.6 cut matches every theme
  // default (yellow/lime/cyan/orange -> black; pink/purple/blue -> white).
  private static accentTextFor(hex: string): string {
    const r = parseInt(hex.slice(1, 3), 16)
    const g = parseInt(hex.slice(3, 5), 16)
    const b = parseInt(hex.slice(5, 7), 16)
    const brightness = (0.299 * r + 0.587 * g + 0.114 * b) / 255
    return brightness > 0.6 ? '#000000' : '#ffffff'
  }

  // Apply (or clear) the custom-accent override on the document root.
  // Inline properties beat the [data-theme] stylesheet rules, and they
  // persist across theme switches, so the accent follows the user while
  // background, cards and borders keep following the theme.
  applyAccentOverride() {
    if (typeof window === 'undefined') return
    const root = document.documentElement
    if (this.customAccent) {
      const text = ThemeStore.accentTextFor(this.customAccent)
      root.style.setProperty('--neo-accent', this.customAccent)
      root.style.setProperty('--color-accent', this.customAccent)
      root.style.setProperty('--neo-accent-text', text)
      root.style.setProperty('--color-accent-text', text)
    } else {
      root.style.removeProperty('--neo-accent')
      root.style.removeProperty('--color-accent')
      root.style.removeProperty('--neo-accent-text')
      root.style.removeProperty('--color-accent-text')
    }
  }

  setCustomAccent(hex: string | null) {
    const next = hex && /^#[0-9a-fA-F]{6}$/.test(hex) ? hex.toLowerCase() : ''
    this.customAccent = next
    if (typeof window !== 'undefined') {
      if (next) localStorage.setItem('customAccent', next)
      else localStorage.removeItem('customAccent')
      this.applyAccentOverride()
    }
  }

  // The accent actually in effect: the custom override when set, otherwise
  // the active theme's own accent.
  effectiveAccent(themeAccent: string): string {
    return this.customAccent || themeAccent
  }

  setTheme(theme: ThemeMode) {
    this.currentTheme = theme
    if (typeof window !== 'undefined') {
      localStorage.setItem('colorTheme', theme)
      document.documentElement.setAttribute('data-theme', theme)
      const isLightFamily = theme === 'light' || theme === 'neobrutalism-light'
      if (isLightFamily) {
        document.documentElement.classList.remove('dark')
        document.documentElement.classList.add('light')
      } else {
        document.documentElement.classList.add('dark')
        document.documentElement.classList.remove('light')
      }
      // Neobrutalism themes lock the neobrutal shape; leaving them
      // restores the user's style, defaulting to modern.
      if (isNeobrutalTheme(theme)) {
        this.setStyle('neobrutal')
      } else if (this.currentStyle === 'neobrutal') {
        this.setStyle('modern')
      }
    }
  }

  setStyle(style: UIStyle) {
    // The style picker never offers 'neobrutal'; ignore attempts to
    // change the shape while a Neobrutalism theme locks it.
    if (isNeobrutalTheme(this.currentTheme) && style !== 'neobrutal') return
    this.currentStyle = style
    if (typeof window !== 'undefined') {
      localStorage.setItem('uiStyle', style)
      document.documentElement.setAttribute('data-ui-style', style)
    }
  }

  setGlassBlur(blur: number) {
    this.glassBlur = blur
    if (typeof window !== 'undefined') {
      localStorage.setItem('glassBlur', String(blur))
      this.applyGlassTokens()
    }
  }

  setGlassOpacity(opacity: number) {
    this.glassOpacity = opacity
    if (typeof window !== 'undefined') {
      localStorage.setItem('glassOpacity', String(opacity))
      this.applyGlassTokens()
    }
  }

  setGlassDim(dim: number) {
    this.glassDim = dim
    if (typeof window !== 'undefined') {
      localStorage.setItem('glassDim', String(dim))
      this.applyGlassTokens()
    }
  }

  setActiveWallpaper(wallpaper: string) {
    this.activeWallpaper = wallpaper
    if (typeof window !== 'undefined') {
      localStorage.setItem('activeWallpaper', wallpaper)
    }
  }

  setLoginWallpaper(wallpaper: string) {
    this.loginWallpaper = wallpaper
    if (typeof window !== 'undefined') {
      localStorage.setItem('loginWallpaper', wallpaper)
    }
  }

  setLoginWallpaperDim(dim: number) {
    this.loginWallpaperDim = dim
    if (typeof window !== 'undefined') {
      localStorage.setItem('loginWallpaperDim', String(dim))
    }
  }

  setLoginWallpaperBlur(blur: number) {
    this.loginWallpaperBlur = blur
    if (typeof window !== 'undefined') {
      localStorage.setItem('loginWallpaperBlur', String(blur))
    }
  }

  getLoginWallpaperBackground(): string {
    if (!this.loginWallpaper || this.loginWallpaper === 'preset:none') return 'none'
    if (this.loginWallpaper === 'preset:cyber-neon') {
      return 'linear-gradient(135deg, #0d0f18 0%, #1a102f 40%, #0d273a 70%, #05141b 100%)'
    }
    if (this.loginWallpaper === 'preset:aurora-mesh') {
      return 'radial-gradient(at 0% 0%, #120b29 0, transparent 50%), radial-gradient(at 50% 0%, #172554 0, transparent 50%), radial-gradient(at 100% 0%, #3b0764 0, transparent 50%), radial-gradient(at 0% 100%, #042f2e 0, transparent 50%), radial-gradient(at 100% 100%, #1e1b4b 0, transparent 50%), #090d16'
    }
    if (this.loginWallpaper === 'preset:sunset-glow') {
      return 'linear-gradient(135deg, #1b0a2a 0%, #3d1445 35%, #5e1c43 70%, #200a18 100%)'
    }
    if (this.loginWallpaper === 'preset:dark-velvet') {
      return 'linear-gradient(160deg, #18151e 0%, #2b1f38 50%, #110e16 100%)'
    }
    return `url('${this.loginWallpaper}')`
  }

  applyGlassTokens() {
    if (typeof window !== 'undefined') {
      document.documentElement.style.setProperty('--glass-blur', `${this.glassBlur}px`)
      document.documentElement.style.setProperty('--glass-opacity', String(this.glassOpacity))
      document.documentElement.style.setProperty('--glass-dim', String(this.glassDim))
    }
  }

  setNeoShadowSize(v: number) {
    this.neoShadowSize = v
    if (typeof window !== 'undefined') {
      localStorage.setItem('neoShadowSize', String(v))
      this.applyNeobrutalTokens()
    }
  }

  setNeoBorderWidth(v: number) {
    this.neoBorderWidth = v
    if (typeof window !== 'undefined') {
      localStorage.setItem('neoBorderWidth', String(v))
      this.applyNeobrutalTokens()
    }
  }

  setNeoRadius(v: number) {
    this.neoRadius = v
    if (typeof window !== 'undefined') {
      localStorage.setItem('neoRadius', String(v))
      this.applyNeobrutalTokens()
    }
  }

  setNeoPressDepth(v: number) {
    this.neoPressDepth = v
    if (typeof window !== 'undefined') {
      localStorage.setItem('neoPressDepth', String(v))
      this.applyNeobrutalTokens()
    }
  }

  setNeoPattern(p: 'dots' | 'grid' | 'none') {
    this.neoPattern = p
    if (typeof window !== 'undefined') {
      localStorage.setItem('neoPattern', p)
      this.applyNeobrutalTokens()
    }
  }

  setNeoPatternOpacity(v: number) {
    this.neoPatternOpacity = v
    if (typeof window !== 'undefined') {
      localStorage.setItem('neoPatternOpacity', String(v))
      this.applyNeobrutalTokens()
    }
  }

  resetNeobrutalSettings() {
    this.neoShadowSize = 4
    this.neoBorderWidth = 2
    this.neoRadius = 4
    this.neoPressDepth = 2
    this.neoPattern = 'dots'
    this.neoPatternOpacity = 0.04
    if (typeof window !== 'undefined') {
      localStorage.removeItem('neoShadowSize')
      localStorage.removeItem('neoBorderWidth')
      localStorage.removeItem('neoRadius')
      localStorage.removeItem('neoPressDepth')
      localStorage.removeItem('neoPattern')
      localStorage.removeItem('neoPatternOpacity')
      this.applyNeobrutalTokens()
    }
  }

  applyNeobrutalTokens() {
    if (typeof window !== 'undefined') {
      document.documentElement.style.setProperty('--neo-shadow-size', `${this.neoShadowSize}px`)
      document.documentElement.style.setProperty('--neo-border-width', `${this.neoBorderWidth}px`)
      document.documentElement.style.setProperty('--neo-radius', `${this.neoRadius}px`)
      document.documentElement.style.setProperty('--neo-press-depth', `${this.neoPressDepth}px`)
      document.documentElement.style.setProperty('--neo-pattern-opacity', String(this.neoPatternOpacity))
      document.documentElement.setAttribute('data-pattern', this.neoPattern)
    }
  }

  toggleDarkMode() {
    if (this.currentTheme === 'neobrutalism-light') {
      this.setTheme('neobrutalism')
    } else if (this.currentTheme === 'neobrutalism') {
      this.setTheme('neobrutalism-light')
    } else if (this.currentTheme === 'light') {
      this.setTheme('dark')
    } else {
      this.setTheme('light')
    }
  }
}

export const themeStore = new ThemeStore()

