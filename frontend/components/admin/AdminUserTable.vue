<script setup lang="ts">
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import type { RoleSlug } from '~/types/auth'
import type { AdminUser, Role, UserStatus } from '~/types/user'

defineProps<{
  users: AdminUser[]
  roles: Role[]
  currentUserId: number
  canUpdate: boolean
  busyId: number | null
}>()
defineEmits<{
  roleChange: [id: number, role: RoleSlug]
  statusChange: [id: number, status: UserStatus]
}>()
</script>

<template>
  <div class="overflow-x-auto rounded-xl border">
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Name</TableHead>
          <TableHead>Email</TableHead>
          <TableHead>Role</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Joined</TableHead>
          <TableHead class="text-right">
            <span class="sr-only">Actions</span>
          </TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        <TableRow v-for="u in users" :key="u.id">
          <TableCell class="font-medium">
            {{ u.name }}
            <span v-if="u.id === currentUserId" class="text-muted-foreground">(you)</span>
          </TableCell>
          <TableCell class="text-muted-foreground">
            {{ u.email }}
          </TableCell>
          <TableCell>
            <Select
              v-if="canUpdate && u.id !== currentUserId"
              :model-value="u.role"
              :disabled="busyId === u.id"
              @update:model-value="(v) => $emit('roleChange', u.id, v as RoleSlug)"
            >
              <SelectTrigger class="w-32" :aria-label="`Role for ${u.name}`">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="r in roles" :key="r.slug" :value="r.slug">
                  {{ r.name }}
                </SelectItem>
              </SelectContent>
            </Select>
            <span v-else class="capitalize">{{ u.role }}</span>
          </TableCell>
          <TableCell>
            <Badge :variant="u.status === 'active' ? 'secondary' : 'destructive'" class="capitalize">
              {{ u.status }}
            </Badge>
          </TableCell>
          <TableCell class="whitespace-nowrap text-muted-foreground">
            {{ formatDate(u.created_at) }}
          </TableCell>
          <TableCell class="text-right">
            <Button
              v-if="canUpdate && u.id !== currentUserId"
              variant="outline"
              size="sm"
              :disabled="busyId === u.id"
              @click="$emit('statusChange', u.id, u.status === 'active' ? 'disabled' : 'active')"
            >
              {{ u.status === 'active' ? 'Disable' : 'Enable' }}
            </Button>
          </TableCell>
        </TableRow>
      </TableBody>
    </Table>
  </div>
</template>
