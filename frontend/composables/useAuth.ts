import { storeToRefs } from 'pinia'

export function useAuth() {
  const store = useAuthStore()
  const { user, isAuthenticated } = storeToRefs(store)
  return { user, isAuthenticated, login: store.login, register: store.register, logout: store.logout }
}
