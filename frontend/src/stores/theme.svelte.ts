import type { ThemeMode, UIStyle } from '../types/common'

class ThemeStore {
  currentTheme = $state<ThemeMode>('dark')
  currentStyle = $state<UIStyle>('neobrutal')

  constructor() {
    if (typeof window !== 'undefined') {
      const savedTheme = (localStorage.getItem('colorTheme') ||
        (localStorage.getItem('theme') === 'light' ? 'light' : 'dark')) as ThemeMode
      const savedStyle = (localStorage.getItem('uiStyle') || 'neobrutal') as UIStyle

      this.setTheme(savedTheme)
      this.setStyle(savedStyle)
    }
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
