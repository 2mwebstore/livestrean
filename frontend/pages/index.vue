<script setup lang="ts">
import { useIntervalFn } from '@vueuse/core'
import { RadioIcon, VideoIcon } from '@lucide/vue'
import { channelService } from '~/services/channel.service'

useSeoMeta({ title: 'Live channels', description: 'Watch live streams and join the chat.' })

const route = useRoute()
const router = useRouter()
const page = computed(() => Math.max(1, Number(route.query.page) || 1))

const {
  data: liveData,
  status: liveStatus,
  error: liveError,
  refresh: refreshLive,
} = await useAsyncData('channels:live', () => channelService.list({ live: true, limit: 24 }))

// Channels that aren't live, so a live channel isn't shown twice.
const {
  data: allData,
  status: allStatus,
  error: allError,
  refresh: refreshAll,
} = await useAsyncData('channels:offline', () => channelService.list({ live: false, page: page.value, limit: 12 }), { watch: [page] })

const anyLive = computed(() => (liveData.value?.data.length ?? 0) > 0)
const showOthers = computed(() => allStatus.value === 'pending' || !!allError.value || (allData.value?.meta.total ?? 0) > 0 || !anyLive.value)

// Keep both lists fresh while the page is open: channels move between them.
useIntervalFn(() => {
  if (document.visibilityState === 'visible') {
    refreshLive()
    refreshAll()
  }
}, 20_000)

function goToPage(p: number) {
  router.push({ query: { ...route.query, page: p > 1 ? p : undefined } })
}
</script>

<template>
  <div class="space-y-12">
    <section aria-labelledby="live-heading">
      <h1 id="live-heading" class="flex items-center gap-2 text-xl font-semibold">
        <span class="size-2 animate-pulse rounded-full bg-live" aria-hidden="true" />
        Live now
      </h1>
      <div class="mt-4">
        <ErrorState v-if="liveError" :message="apiErrorMessage(liveError)" @retry="refreshLive()" />
        <ChannelGrid
          v-else-if="liveStatus === 'pending' || liveData?.data.length"
          :channels="liveData?.data ?? []"
          :loading="liveStatus === 'pending'"
        />
        <EmptyState v-else title="Nobody is live right now" description="Channels show up here the moment they start streaming.">
          <template #icon>
            <RadioIcon class="size-6 text-muted-foreground" />
          </template>
        </EmptyState>
      </div>
    </section>

    <section v-if="showOthers" aria-labelledby="all-heading">
      <h2 id="all-heading" class="text-xl font-semibold">
        {{ anyLive ? 'Other channels' : 'All channels' }}
      </h2>
      <div class="mt-4 space-y-6">
        <ErrorState v-if="allError" :message="apiErrorMessage(allError)" @retry="refreshAll()" />
        <template v-else-if="allStatus === 'pending' || allData?.data.length">
          <ChannelGrid :channels="allData?.data ?? []" :loading="allStatus === 'pending'" :skeletons="8" />
          <PaginationBar v-if="allData" :meta="allData.meta" @change="goToPage" />
        </template>
        <EmptyState v-else title="No channels yet" description="Once a streamer creates a channel, it appears here.">
          <template #icon>
            <VideoIcon class="size-6 text-muted-foreground" />
          </template>
        </EmptyState>
      </div>
    </section>
  </div>
</template>
