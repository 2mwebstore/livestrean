// Same identifiers as backend/internal/auth/principal.go.
export type Permission =
  | 'comments.create'
  | 'comments.moderate'
  | 'channels.create'
  | 'channels.manage_own'
  | 'channels.view_all'
  | 'channels.update_all'
  | 'users.view'
  | 'users.update'

export type RoleSlug = 'admin' | 'streamer' | 'viewer'

export interface AuthUser {
  id: number
  name: string
  email: string
  role: RoleSlug
  permissions: Permission[]
}

export interface LoginRequest {
  email: string
  password: string
}

export interface RegisterRequest {
  name: string
  email: string
  password: string
}
