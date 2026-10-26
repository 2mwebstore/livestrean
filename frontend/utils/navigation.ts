import type { Permission } from '~/types/auth'

export interface NavItem {
  label: string
  to: string
  /** Hidden unless the user has this permission (UX only). */
  permission?: Permission
}

export const mainNav: NavItem[] = [
  { label: 'Browse', to: '/' },
  { label: 'Dashboard', to: '/dashboard', permission: 'channels.manage_own' },
  { label: 'Admin', to: '/admin', permission: 'users.view' },
]

export const adminNav: NavItem[] = [
  { label: 'Users', to: '/admin/users', permission: 'users.view' },
  { label: 'Channels', to: '/admin/channels', permission: 'channels.view_all' },
]
