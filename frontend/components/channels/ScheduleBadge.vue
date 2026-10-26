<script setup lang="ts">
import { useNow } from '@vueuse/core'
import { CalendarClockIcon } from '@lucide/vue'

const props = defineProps<{ at: string }>()

const now = useNow({ interval: 30_000 })
const label = computed(() =>
  Date.parse(props.at) > now.value.getTime() ? `Starts in ${formatTimeUntil(props.at, now.value.getTime())}` : 'Starting soon',
)
</script>

<template>
  <span class="inline-flex items-center gap-1 rounded-md bg-black/70 px-1.5 py-0.5 text-xs font-medium text-white">
    <CalendarClockIcon class="size-3.5" aria-hidden="true" />
    <ClientOnly>
      {{ label }}
      <template #fallback>
        Upcoming
      </template>
    </ClientOnly>
  </span>
</template>
