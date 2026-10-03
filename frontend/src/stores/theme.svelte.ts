import type { ThemeMode, UIStyle } from '../types/common'

const VALID_THEMES: ThemeMode[] = [
  'dark',
  'amoled',
  'light',
  'dracula',
  'nord',
  'cyberpunk',
  'emerald',
  'sunset',
  'retro',
  'projectx',
]

const VALID_STYLES: UIStyle[] = ['neobrutal', 'modern', 'glass']

class ThemeStore {
  currentTheme = $state<ThemeMode>('dark')
  currentStyle = $state<UIStyle>('neobrutal')
  showAppearanceModal = $state<boolean>(false)
  glassBlur = $state<number>(16)
  glassOpacity = $state<number>(0.65)
  glassDim = $state<number>(0.75)
  activeWallpaper = $state<string>('')
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

      this.setTheme(initialTheme)
      this.setStyle(initialStyle)
      this.applyGlassTokens()
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
      if (theme === 'light') {
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

