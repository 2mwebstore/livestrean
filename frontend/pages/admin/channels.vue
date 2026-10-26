<script setup lang="ts">
import { refDebounced, useIntervalFn } from '@vueuse/core'
import { SearchIcon, VideoIcon } from '@lucide/vue'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { adminService } from '~/services/admin.service'
import type { ChannelStatus } from '~/types/channel'

definePageMeta({ layout: 'admin', middleware: ['auth', 'permission'], permission: 'channels.view_all' })
useSeoMeta({ title: 'Channels' })

const { can } = usePermission()

const search = ref('')
const query = refDebounced(search, 300)
const page = ref(1)
watch(query, () => (page.value = 1))

const { data, status, error, refresh } = await useAsyncData(
  'admin:channels',
  () => adminService.listChannels({ q: query.value || undefined, page: page.value, limit: 20 }),
  { watch: [query, page] },
)
useIntervalFn(() => refresh(), 15_000)

const busyId = ref<number | null>(null)
const actionError = ref('')

async function changeStatus(id: number, next: ChannelStatus) {
  busyId.value = id
  actionError.value = ''
  try {
    const updated = (await adminService.updateChannelStatus(id, next)).data
    if (data.value) data.value.data = data.value.data.map(c => (c.id === updated.id ? updated : c))
  }
  catch (err) {
    actionError.value = apiErrorMessage(err)
    await refresh()
  }
  finally {
    busyId.value = null
  }
}
</script>

<template>
  <div class="space-y-5">
    <div>
      <h1 class="text-xl font-semibold">
        Channels
      </h1>
      <p class="text-sm text-muted-foreground">
        Disabling a channel stops its stream and hides it from viewers.
      </p>
    </div>

    <div class="relative max-w-sm">
      <SearchIcon class="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
      <Input v-model="search" placeholder="Search title or URL" class="pl-8" aria-label="Search channels" />
    </div>

    <p v-if="actionError" class="text-sm text-destructive" role="alert">
      {{ actionError }}
    </p>

    <ErrorState v-if="error" :message="apiErrorMessage(error)" @retry="refresh()" />
    <div v-else-if="status === 'pending' && !data" class="grid gap-2">
      <Skeleton v-for="i in 6" :key="i" class="h-14 w-full" />
    </div>
    <EmptyState v-else-if="!data?.data.length" :title="query ? 'No channels match your search' : 'No channels yet'">
      <template #icon>
        <VideoIcon class="size-6 text-muted-foreground" />
      </template>
    </EmptyState>
    <template v-else>
      <AdminChannelTable :channels="data.data" :can-update="can('channels.update_all')" :busy-id="busyId" @status-change="changeStatus" />
      <PaginationBar :meta="data.meta" @change="(p) => (page = p)" />
    </template>
  </div>
</template>
