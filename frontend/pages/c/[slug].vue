<script setup lang="ts">
import { useNow } from '@vueuse/core'
import { channelService } from '~/services/channel.service'
import { commentService } from '~/services/comment.service'

const route = useRoute()
const slug = String(route.params.slug)

// Server-rendered: channel details and recent chat are in the first HTML.
const { data: channel, error } = await useAsyncData(`channel:${slug}`, () => channelService.get(slug).then(r => r.data))
if (error.value || !channel.value) {
  const notFound = !error.value || error.value.statusCode === 404
  throw createError({
    statusCode: notFound ? 404 : 500,
    statusMessage: notFound ? 'This channel doesn\'t exist or is no longer available.' : 'Couldn\'t load this channel.',
    fatal: true,
  })
}
const { data: initialComments } = await useAsyncData(`comments:${slug}`, () => commentService.list(slug).then(r => r.data), {
  default: () => [],
})

// Browser-only from here: chat state and the live connection.
const chat = reactive(useChannelChat(slug, initialComments.value))
const socket = useChannelSocket(slug, {
  onComment: chat.add,
  onCommentDeleted: chat.remove,
  onReconnect: chat.refresh,
})

const live = computed(() => socket.status.value?.live ?? channel.value?.live ?? false)
const startedAt = computed(() => (socket.status.value ? socket.status.value.started_at : channel.value?.live_started_at) ?? null)
const viewers = computed(() => socket.viewers.value ?? channel.value?.viewers ?? 0)
const now = useNow({ interval: 30_000 })

useSeoMeta({
  title: () => channel.value?.title,
  description: () => channel.value?.description || `Watch ${channel.value?.owner.name} live.`,
  ogTitle: () => channel.value?.title,
  ogDescription: () => channel.value?.description || `Watch ${channel.value?.owner.name} live.`,
  ogType: 'video.other',
})
</script>

<template>
  <div v-if="channel" class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_360px]">
    <div class="min-w-0">
      <LivePlayer :hls-url="channel.hls_url" :whep-url="channel.whep_url" :live="live" />

      <div class="mt-5 flex flex-wrap items-start justify-between gap-3">
        <div class="min-w-0">
          <h1 class="text-xl font-semibold break-words">
            {{ channel.title }}
          </h1>
          <p class="text-sm text-muted-foreground">
            {{ channel.owner.name }}
          </p>
        </div>
        <div class="flex items-center gap-3 text-sm text-muted-foreground">
          <LiveBadge :live="live" />
          <ViewerCount :count="viewers" />
          <ClientOnly>
            <span v-if="live && startedAt">Live for {{ formatDuration(startedAt, now.getTime()) }}</span>
          </ClientOnly>
        </div>
      </div>

      <p v-if="channel.description" class="mt-3 text-sm whitespace-pre-line text-muted-foreground">
        {{ channel.description }}
      </p>
    </div>

    <ChatPanel
      :chat="chat"
      :slug="slug"
      :host-id="channel.owner.id"
      class="h-[480px] lg:sticky lg:top-20 lg:h-[calc(100dvh-7rem)]"
    />
  </div>
</template>
