export type ThemeMode =
  | 'dark'
  | 'light'
  | 'neobrutalism-light'
  | 'neobrutalism'
// 'neobrutal' is the locked shape of the Neobrutalism themes, not a
// user-selectable style: components branch on it, and the theme store
// forces it whenever a Neobrutalism theme is active.
export type UIStyle = 'neobrutal' | 'modern' | 'glass'

export interface WallpaperItem {
  id: string
  name: string
  size: number
  url: string
  updated_at: number
}

export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
  error?: string
}

export interface ToastMessage {
  id: string
  title: string
  message: string
  type: 'info' | 'success' | 'warning' | 'error'
  duration?: number
}
