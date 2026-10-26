import type { ApiResponse } from '~/types/api'
import type { AuthUser, LoginRequest, RegisterRequest } from '~/types/auth'

export const authService = {
  me() {
    return useApi()<ApiResponse<AuthUser>>('/auth/me')
  },
  login(body: LoginRequest) {
    return useApi()<ApiResponse<AuthUser>>('/auth/login', { method: 'POST', body })
  },
  register(body: RegisterRequest) {
    return useApi()<ApiResponse<AuthUser>>('/auth/register', { method: 'POST', body })
  },
  logout() {
    return useApi()<ApiResponse<null>>('/auth/logout', { method: 'POST' })
  },
}
