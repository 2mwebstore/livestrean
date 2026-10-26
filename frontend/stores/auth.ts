import { defineStore } from 'pinia'
import { authService } from '~/services/auth.service'
import type { AuthUser, LoginRequest, Permission, RegisterRequest } from '~/types/auth'

// Who is logged in. The API is the source of truth: this is only used to
// render the right UI, and every request is checked again server-side.
export const useAuthStore = defineStore('auth', () => {
  const user = ref<AuthUser | null>(null)
  const loaded = ref(false)
  const isAuthenticated = computed(() => user.value !== null)

  async function fetchMe() {
    try {
      user.value = (await authService.me()).data
    }
    catch {
      user.value = null
    }
    finally {
      loaded.value = true
    }
  }

  function markGuest() {
    user.value = null
    loaded.value = true
  }

  async function login(body: LoginRequest) {
    user.value = (await authService.login(body)).data
  }

  async function register(body: RegisterRequest) {
    user.value = (await authService.register(body)).data
  }

  async function logout() {
    try {
      await authService.logout()
    }
    finally {
      user.value = null
    }
  }

  function clear() {
    user.value = null
  }

  function can(permission: Permission): boolean {
    return user.value?.permissions.includes(permission) ?? false
  }

  return { user, loaded, isAuthenticated, fetchMe, markGuest, login, register, logout, clear, can }
})
