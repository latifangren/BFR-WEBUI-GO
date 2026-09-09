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
]

const VALID_STYLES: UIStyle[] = ['neobrutal', 'modern']

class ThemeStore {
  currentTheme = $state<ThemeMode>('dark')
  currentStyle = $state<UIStyle>('neobrutal')
  showAppearanceModal = $state<boolean>(false)

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

      this.setTheme(initialTheme)
      this.setStyle(initialStyle)
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

  toggleDarkMode() {
    if (this.currentTheme === 'light') {
      this.setTheme('dark')
    } else {
      this.setTheme('light')
    }
  }
}

export const themeStore = new ThemeStore()

