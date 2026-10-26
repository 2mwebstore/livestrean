<script setup lang="ts">
import { useIntervalFn } from '@vueuse/core'
import { PlusIcon, VideoIcon } from '@lucide/vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { channelService } from '~/services/channel.service'
import type { ChannelFormValues, StreamKeyResult } from '~/types/channel'

definePageMeta({ middleware: ['auth', 'permission'], permission: 'channels.manage_own' })
useSeoMeta({ title: 'Your channels' })

const { can } = usePermission()
const { data: channels, status, error, refresh } = await useAsyncData('me:channels', () => channelService.listOwn().then(r => r.data), {
  default: () => [],
})
useIntervalFn(() => refresh(), 15_000)

const createOpen = ref(false)
const creating = ref(false)
const createErrors = ref<Record<string, string>>({})
const createMessage = ref('')
const keyResult = ref<StreamKeyResult | null>(null)
const keyOpen = ref(false)

function openCreate() {
  createErrors.value = {}
  createMessage.value = ''
  createOpen.value = true
}

async function create(values: ChannelFormValues) {
  creating.value = true
  createErrors.value = {}
  createMessage.value = ''
  try {
    keyResult.value = (await channelService.create(values)).data
    createOpen.value = false
    keyOpen.value = true
    await refresh()
  }
  catch (err) {
    createErrors.value = apiFieldErrors(err)
    createMessage.value = apiErrorMessage(err)
  }
  finally {
    creating.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-wrap items-end justify-between gap-4">
      <div>
        <h1 class="text-xl font-semibold">
          Your channels
        </h1>
        <p class="text-sm text-muted-foreground">
          Each channel has its own page, chat and stream key.
        </p>
      </div>
      <Button v-if="can('channels.create')" @click="openCreate">
        <PlusIcon />
        New channel
      </Button>
    </div>

    <ErrorState v-if="error" :message="apiErrorMessage(error)" @retry="refresh()" />

    <div v-else-if="status === 'pending' && !channels.length" class="grid gap-3">
      <Skeleton v-for="i in 3" :key="i" class="h-20 w-full rounded-xl" />
    </div>

    <EmptyState v-else-if="!channels.length" title="No channels yet" description="Create a channel to get your stream key and start streaming from OBS.">
      <template #icon>
        <VideoIcon class="size-6 text-muted-foreground" />
      </template>
      <Button v-if="can('channels.create')" @click="openCreate">
        <PlusIcon />
        Create your first channel
      </Button>
    </EmptyState>

    <div v-else class="grid gap-3">
      <Card v-for="ch in channels" :key="ch.id">
        <CardContent class="flex flex-wrap items-center gap-4">
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <p class="truncate font-medium">
                {{ ch.title }}
              </p>
              <LiveBadge :live="ch.live" />
              <Badge v-if="ch.status !== 'active'" variant="destructive">
                Disabled by admin
              </Badge>
            </div>
            <p class="mt-0.5 font-mono text-xs text-muted-foreground">
              /c/{{ ch.slug }}
            </p>
          </div>
          <span v-if="ch.live" class="text-sm text-muted-foreground">
            <ViewerCount :count="ch.viewers" />
          </span>
          <div class="flex gap-2">
            <Button variant="outline" size="sm" as-child>
              <NuxtLink :to="`/c/${ch.slug}`">
                Watch
              </NuxtLink>
            </Button>
            <Button size="sm" as-child>
              <NuxtLink :to="`/dashboard/channels/${ch.id}`">
                Manage
              </NuxtLink>
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>

    <Dialog v-model:open="createOpen">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>New channel</DialogTitle>
          <DialogDescription>You'll get its stream key right after creating it.</DialogDescription>
        </DialogHeader>
        <p v-if="createMessage && !Object.keys(createErrors).length" class="text-sm text-destructive" role="alert">
          {{ createMessage }}
        </p>
        <ChannelForm v-if="createOpen" mode="create" :submitting="creating" :server-errors="createErrors" @submit="create" />
      </DialogContent>
    </Dialog>

    <StreamKeyDialog v-model:open="keyOpen" :result="keyResult" />
  </div>
</template>
