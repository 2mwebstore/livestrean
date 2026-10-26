<script setup lang="ts">
import { ExternalLinkIcon } from '@lucide/vue'
import { channelService } from '~/services/channel.service'

// Player only, for <iframe> embeds on other websites (see ShareCard).
definePageMeta({ layout: 'embed' })

const route = useRoute()
const slug = String(route.params.slug)
const siteName = useRuntimeConfig().public.siteName

const { data: channel } = await useAsyncData(`channel:${slug}`, () => channelService.get(slug).then(r => r.data))
const socket = useChannelSocket(slug)
const live = computed(() => socket.status.value?.live ?? channel.value?.live ?? false)
const viewers = computed(() => socket.viewers.value ?? channel.value?.viewers ?? 0)

useHead({
  title: () => channel.value?.title ?? 'Channel unavailable',
  meta: [{ name: 'robots', content: 'noindex' }],
})
</script>

<template>
  <div v-if="channel" class="relative">
    <LivePlayer :hls-url="channel.hls_url" :whep-url="channel.whep_url" :live="live" compact />
    <div class="pointer-events-none absolute inset-x-0 top-0 flex items-start justify-between gap-2 p-2">
      <div class="flex items-center gap-2 text-xs text-white">
        <LiveBadge :live="live" />
        <span v-if="live" class="rounded-md bg-black/70 px-1.5 py-0.5">
          <ViewerCount :count="viewers" />
        </span>
      </div>
      <a
        :href="`/c/${slug}`"
        target="_blank"
        rel="noopener"
        class="pointer-events-auto inline-flex items-center gap-1 rounded-md bg-black/70 px-2 py-1 text-xs text-white hover:bg-black/90"
      >
        Watch on {{ siteName }}
        <ExternalLinkIcon class="size-3" />
      </a>
    </div>
  </div>
  <div v-else class="grid aspect-video place-items-center text-sm text-white/70">
    This channel isn't available.
  </div>
</template>
