<script setup lang="ts">
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import type { AdminChannel, ChannelStatus } from '~/types/channel'

defineProps<{ channels: AdminChannel[], canUpdate: boolean, busyId: number | null }>()
defineEmits<{ statusChange: [id: number, status: ChannelStatus] }>()
</script>

<template>
  <div class="overflow-x-auto rounded-xl border">
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Channel</TableHead>
          <TableHead>Owner</TableHead>
          <TableHead>Now</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Created</TableHead>
          <TableHead class="text-right">
            <span class="sr-only">Actions</span>
          </TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        <TableRow v-for="ch in channels" :key="ch.id">
          <TableCell>
            <p class="font-medium">
              {{ ch.title }}
            </p>
            <NuxtLink :to="`/c/${ch.slug}`" class="font-mono text-xs text-muted-foreground hover:underline">
              /c/{{ ch.slug }}
            </NuxtLink>
          </TableCell>
          <TableCell>
            <p>{{ ch.owner.name }}</p>
            <p class="text-xs text-muted-foreground">
              {{ ch.owner.email }}
            </p>
          </TableCell>
          <TableCell>
            <div class="flex items-center gap-2 text-sm">
              <LiveBadge :live="ch.live" />
              <ViewerCount v-if="ch.live" :count="ch.viewers" />
            </div>
          </TableCell>
          <TableCell>
            <Badge :variant="ch.status === 'active' ? 'secondary' : 'destructive'" class="capitalize">
              {{ ch.status }}
            </Badge>
          </TableCell>
          <TableCell class="whitespace-nowrap text-muted-foreground">
            {{ formatDate(ch.created_at) }}
          </TableCell>
          <TableCell class="text-right">
            <Button
              v-if="canUpdate"
              variant="outline"
              size="sm"
              :disabled="busyId === ch.id"
              @click="$emit('statusChange', ch.id, ch.status === 'active' ? 'disabled' : 'active')"
            >
              {{ ch.status === 'active' ? 'Disable' : 'Enable' }}
            </Button>
          </TableCell>
        </TableRow>
      </TableBody>
    </Table>
  </div>
</template>
