<script setup lang="ts">
import { useNow } from '@vueuse/core'

const props = defineProps<{ at: string, compact?: boolean }>()

// Ticks every second in the browser. The digits depend on the viewer's
// clock, so they render client-side only (no SSR/hydration mismatch).
const now = useNow({ interval: 1000 })
const remaining = computed(() => Date.parse(props.at) - now.value.getTime())
const units = computed(() => {
  const p = countdownParts(remaining.value)
  return [
    ...(p.days ? [{ label: 'days', value: p.days }] : []),
    { label: 'hours', value: p.hours },
    { label: 'min', value: p.minutes },
    { label: 'sec', value: p.seconds },
  ]
})
</script>

<template>
  <div class="flex flex-col items-center gap-3 text-white">
    <ClientOnly>
      <template v-if="remaining > 0">
        <p class="text-xs font-semibold tracking-widest uppercase opacity-80">
          Live in
        </p>
        <div class="flex gap-2" role="timer" :aria-label="`Live in ${formatTimeUntil(at, now.getTime())}`">
          <div
            v-for="unit in units"
            :key="unit.label"
            class="rounded-lg bg-black/60 px-2 py-1.5 ring-1 ring-white/10"
            :class="compact ? 'min-w-11' : 'min-w-14 sm:min-w-16 sm:py-2'"
          >
            <div class="font-bold tabular-nums" :class="compact ? 'text-xl' : 'text-2xl sm:text-3xl'">
              {{ String(unit.value).padStart(2, '0') }}
            </div>
            <div class="text-[10px] tracking-wide uppercase opacity-70">
              {{ unit.label }}
            </div>
          </div>
        </div>
      </template>
      <p v-else class="text-lg font-semibold">
        Starting soon…
      </p>
      <p class="text-sm opacity-80">
        {{ formatScheduleDate(at) }}
      </p>
      <template #fallback>
        <p class="text-sm opacity-80">
          Live stream scheduled
        </p>
      </template>
    </ClientOnly>
  </div>
</template>
