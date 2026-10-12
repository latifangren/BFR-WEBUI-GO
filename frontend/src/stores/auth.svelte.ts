import { api } from '../api/client'
import type { AuthStatus, LoginPayload, LoginResponse } from '../types/auth'

class AuthStore {
  // null = status unknown (initial page load); avoids flashing the dashboard
  // before /api/auth/status responds.
  authenticated = $state<boolean | null>(null)
  isDefaultPass = $state<boolean>(false)
  isLoading = $state<boolean>(false)
  loginError = $state<string | null>(null)

  constructor() {
    if (typeof window !== 'undefined') {
      window.addEventListener('bfr:unauthorized', () => {
        this.authenticated = false
      })
    }
  }

  async checkStatus(): Promise<void> {
    try {
      this.isLoading = true
      const res = await api.get<AuthStatus>('/api/auth/status')
      this.authenticated = res.authenticated
      this.isDefaultPass = res.is_default_pass
    } catch (err) {
      console.error('[AuthStore] Failed to check status:', err)
      // Don't hang on the splash screen forever on network errors.
      this.authenticated = false
    } finally {
      this.isLoading = false
    }
  }

  async login(payload: LoginPayload): Promise<boolean> {
    try {
      this.isLoading = true
      this.loginError = null
      const res = await api.post<LoginResponse>('/api/auth/login', payload)
      if (res.success) {
        this.authenticated = true
        if (typeof window !== 'undefined') {
          // Lets background stores (e.g. sysinfo polling) resume.
          window.dispatchEvent(new CustomEvent('bfr:authenticated'))
        }
        return true
      }
      this.loginError = res.message || 'Login failed'
      return false
    } catch (err: unknown) {
      this.loginError = err instanceof Error ? err.message : 'Invalid credentials'
      return false
    } finally {
      this.isLoading = false
    }
  }

  async logout(): Promise<void> {
    try {
      this.isLoading = true
      await api.post('/api/auth/logout')
      this.authenticated = false
    } catch (err) {
      console.error('[AuthStore] Logout failed:', err)
    } finally {
      this.isLoading = false
    }
  }
}

export const authStore = new AuthStore()
