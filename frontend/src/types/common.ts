export type ThemeMode = 'dark' | 'amoled' | 'light' | 'dracula' | 'nord' | 'cyberpunk' | 'emerald' | 'sunset' | 'retro'
export type UIStyle = 'neobrutal' | 'modern'

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
