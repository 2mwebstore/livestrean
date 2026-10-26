import type { ApiResponse, PageQuery, PaginatedResponse } from '~/types/api'
import type { RoleSlug } from '~/types/auth'
import type { AdminChannel, ChannelStatus } from '~/types/channel'
import type { AdminUser, Role, UserStatus } from '~/types/user'

export const adminService = {
  listUsers(query: PageQuery) {
    return useApi()<PaginatedResponse<AdminUser>>('/admin/users', { query })
  },
  listRoles() {
    return useApi()<ApiResponse<Role[]>>('/admin/roles')
  },
  updateUserStatus(id: number, status: UserStatus) {
    return useApi()<ApiResponse<AdminUser>>(`/admin/users/${id}/status`, { method: 'PATCH', body: { status } })
  },
  updateUserRole(id: number, role: RoleSlug) {
    return useApi()<ApiResponse<AdminUser>>(`/admin/users/${id}/role`, { method: 'PATCH', body: { role } })
  },
  listChannels(query: PageQuery) {
    return useApi()<PaginatedResponse<AdminChannel>>('/admin/channels', { query })
  },
  updateChannelStatus(id: number, status: ChannelStatus) {
    return useApi()<ApiResponse<AdminChannel>>(`/admin/channels/${id}/status`, { method: 'PATCH', body: { status } })
  },
}
