import type { ColorVibe, ThemeMode, UIStyle } from '../types/common'

const VALID_THEMES: ThemeMode[] = [
  'dark',
  'amoled',
  'light',
  'projectx-light',
  'dracula',
  'nord',
  'cyberpunk',
  'emerald',
  'sunset',
  'retro',
  'projectx',
]

const VALID_STYLES: UIStyle[] = ['neobrutal', 'modern', 'glass']
const VALID_VIBES: ColorVibe[] = ['default', 'synthwave', 'toxic', 'arctic', 'sunset', 'lavender']

class ThemeStore {
  currentTheme = $state<ThemeMode>('dark')
  currentStyle = $state<UIStyle>('neobrutal')
  currentColorVibe = $state<ColorVibe>('default')
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
  constructor() {
    if (typeof window !== 'undefined') {
      const storedTheme = localStorage.getItem('colorTheme') || localStorage.getItem('theme')
      const initialTheme = (
        VALID_THEMES.includes(storedTheme as ThemeMode)
          ? storedTheme
          : storedTheme === 'light'
            ? 'light'
            : 'dark'
      ) as ThemeMode

      const storedStyle = localStorage.getItem('uiStyle')
      const initialStyle = (
        VALID_STYLES.includes(storedStyle as UIStyle) ? storedStyle : 'neobrutal'
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
      const storedVibe = localStorage.getItem('colorVibe')
      const initialVibe = (VALID_VIBES.includes(storedVibe as ColorVibe) ? storedVibe : 'default') as ColorVibe


      this.setTheme(initialTheme)
      this.setStyle(initialStyle)
      this.applyGlassTokens()
      this.setColorVibe(initialVibe)
      this.applyNeobrutalTokens()
    }
  }

  openAppearanceModal() {
    this.showAppearanceModal = true
  }

  closeAppearanceModal() {
    this.showAppearanceModal = false
  }

  setTheme(theme: ThemeMode) {
    this.currentTheme = theme
    if (typeof window !== 'undefined') {
      localStorage.setItem('colorTheme', theme)
      document.documentElement.setAttribute('data-theme', theme)
      const isLightFamily = theme === 'light' || theme === 'projectx-light' || theme === 'retro'
      if (isLightFamily) {
        document.documentElement.classList.remove('dark')
        document.documentElement.classList.add('light')
      } else {
        document.documentElement.classList.add('dark')
        document.documentElement.classList.remove('light')
      }
    }
  }

  setStyle(style: UIStyle) {
    this.currentStyle = style
    if (typeof window !== 'undefined') {
      localStorage.setItem('uiStyle', style)
      document.documentElement.setAttribute('data-ui-style', style)
    }
  }
  setColorVibe(vibe: ColorVibe) {
    this.currentColorVibe = vibe
    if (typeof window !== 'undefined') {
      localStorage.setItem('colorVibe', vibe)
      if (vibe === 'default') {
        document.documentElement.removeAttribute('data-color')
      } else {
        document.documentElement.setAttribute('data-color', vibe)
      }
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
    if (this.currentTheme === 'light') {
      this.setTheme('dark')
    } else {
      this.setTheme('light')
    }
  }
}

export const themeStore = new ThemeStore()

