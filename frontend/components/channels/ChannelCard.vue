<script setup lang="ts">
import type { Channel } from '~/types/channel'

const props = defineProps<{ channel: Channel }>()

// A stable color per channel for the placeholder thumbnail.
const hue = computed(() => [...props.channel.slug].reduce((h, c) => (h * 31 + c.charCodeAt(0)) % 360, 7))
</script>

<template>
  <NuxtLink :to="`/c/${channel.slug}`" class="group block focus-visible:outline-none">
    <div
      class="relative aspect-video overflow-hidden rounded-xl ring-offset-background transition group-hover:ring-2 group-hover:ring-ring group-focus-visible:ring-2 group-focus-visible:ring-ring"
      :style="{ background: `linear-gradient(135deg, oklch(0.35 0.08 ${hue}), oklch(0.2 0.04 ${hue + 40}))` }"
    >
      <span class="absolute inset-0 grid place-items-center text-5xl font-bold text-white/25 uppercase select-none">
        {{ channel.title.charAt(0) }}
      </span>
      <div class="absolute top-2 left-2">
        <LiveBadge :live="channel.live" />
      </div>
      <div v-if="channel.live" class="absolute bottom-2 left-2 rounded-md bg-black/70 px-1.5 py-0.5 text-xs text-white">
        <ViewerCount :count="channel.viewers" />
      </div>
    </div>
    <div class="mt-2.5 min-w-0">
      <p class="truncate font-medium group-hover:underline">
        {{ channel.title }}
      </p>
      <p class="truncate text-sm text-muted-foreground">
        {{ channel.owner.name }}
        <template v-if="!channel.live && channel.last_live_at">
          · live {{ formatRelativeTime(channel.last_live_at) }}
        </template>
      </p>
    </div>
  </NuxtLink>
</template>
