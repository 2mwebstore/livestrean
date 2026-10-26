import type { Permission, RoleSlug } from './auth'

export type UserStatus = 'active' | 'disabled'

export interface AdminUser {
  id: number
  name: string
  email: string
  role: RoleSlug
  status: UserStatus
  created_at: string
}

export interface Role {
  slug: RoleSlug
  name: string
  permissions: Permission[]
}
