<script setup lang="ts">
import { refDebounced } from '@vueuse/core'
import { SearchIcon, UsersIcon } from '@lucide/vue'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { adminService } from '~/services/admin.service'
import type { RoleSlug } from '~/types/auth'
import type { AdminUser, UserStatus } from '~/types/user'

definePageMeta({ layout: 'admin', middleware: ['auth', 'permission'], permission: 'users.view' })
useSeoMeta({ title: 'Users' })

const { user } = useAuth()
const { can } = usePermission()

const search = ref('')
const query = refDebounced(search, 300)
const page = ref(1)
watch(query, () => (page.value = 1))

const { data, status, error, refresh } = await useAsyncData(
  'admin:users',
  () => adminService.listUsers({ q: query.value || undefined, page: page.value, limit: 20 }),
  { watch: [query, page] },
)
const { data: roles } = await useAsyncData('admin:roles', () => adminService.listRoles().then(r => r.data), { default: () => [] })

const busyId = ref<number | null>(null)
const actionError = ref('')

function replace(updated: AdminUser) {
  if (data.value) data.value.data = data.value.data.map(u => (u.id === updated.id ? updated : u))
}

async function run(id: number, action: () => Promise<{ data: AdminUser }>) {
  busyId.value = id
  actionError.value = ''
  try {
    replace((await action()).data)
  }
  catch (err) {
    actionError.value = apiErrorMessage(err)
    await refresh()
  }
  finally {
    busyId.value = null
  }
}

const changeRole = (id: number, role: RoleSlug) => run(id, () => adminService.updateUserRole(id, role))
const changeStatus = (id: number, status: UserStatus) => run(id, () => adminService.updateUserStatus(id, status))
</script>

<template>
  <div class="space-y-5">
    <div>
      <h1 class="text-xl font-semibold">
        Users
      </h1>
      <p class="text-sm text-muted-foreground">
        New accounts are viewers (chat only). Make someone a streamer to let them create channels.
      </p>
    </div>

    <div class="relative max-w-sm">
      <SearchIcon class="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
      <Input v-model="search" placeholder="Search name or email" class="pl-8" aria-label="Search users" />
    </div>

    <p v-if="actionError" class="text-sm text-destructive" role="alert">
      {{ actionError }}
    </p>

    <ErrorState v-if="error" :message="apiErrorMessage(error)" @retry="refresh()" />
    <div v-else-if="status === 'pending' && !data" class="grid gap-2">
      <Skeleton v-for="i in 6" :key="i" class="h-12 w-full" />
    </div>
    <EmptyState v-else-if="!data?.data.length" :title="query ? 'No users match your search' : 'No users yet'">
      <template #icon>
        <UsersIcon class="size-6 text-muted-foreground" />
      </template>
    </EmptyState>
    <template v-else>
      <AdminUserTable
        :users="data.data"
        :roles="roles"
        :current-user-id="user?.id ?? 0"
        :can-update="can('users.update')"
        :busy-id="busyId"
        @role-change="changeRole"
        @status-change="changeStatus"
      />
      <PaginationBar :meta="data.meta" @change="(p) => (page = p)" />
    </template>
  </div>
</template>
