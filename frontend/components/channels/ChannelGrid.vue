<script setup lang="ts">
import { Skeleton } from '@/components/ui/skeleton'
import type { Channel } from '~/types/channel'

defineProps<{ channels: Channel[], loading?: boolean, skeletons?: number }>()
</script>

<template>
  <div class="grid grid-cols-1 gap-x-4 gap-y-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
    <template v-if="loading && !channels.length">
      <div v-for="i in skeletons ?? 4" :key="i">
        <Skeleton class="aspect-video w-full rounded-xl" />
        <Skeleton class="mt-2.5 h-4 w-3/4" />
        <Skeleton class="mt-1.5 h-3 w-1/2" />
      </div>
    </template>
    <ChannelCard v-for="channel in channels" :key="channel.slug" :channel="channel" />
  </div>
</template>
