export interface AuthStatus {
  authenticated: boolean
  is_default_pass: boolean
}

export interface LoginPayload {
  username?: string
  password?: string
}

export interface LoginResponse {
  success: boolean
  message: string
}

