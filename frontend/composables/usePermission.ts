import type { Permission } from '~/types/auth'

/** Permission checks for showing or hiding UI. Not a security boundary. */
export function usePermission() {
  const store = useAuthStore()
  return { can: (permission: Permission) => store.can(permission) }
}
