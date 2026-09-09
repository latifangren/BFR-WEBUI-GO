export type ThemeMode = 'dark' | 'light' | 'amoled' | 'dracula' | 'nord' | 'cyberpunk' | 'emerald' | 'sunset'
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
