export type ThemeMode = 'dark' | 'amoled' | 'light' | 'dracula' | 'nord' | 'cyberpunk' | 'emerald' | 'sunset' | 'retro' | 'projectx'
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
