<script setup lang="ts">
defineProps<{ live: boolean }>()
</script>

<template>
  <span
    v-if="live"
    class="inline-flex items-center gap-1.5 rounded-md bg-live px-2 py-0.5 text-[11px] font-bold tracking-wider text-white uppercase"
  >
    <span class="size-1.5 animate-pulse rounded-full bg-white" aria-hidden="true" />
    Live
  </span>
  <span
    v-else
    class="inline-flex items-center rounded-md bg-muted px-2 py-0.5 text-[11px] font-semibold tracking-wider text-muted-foreground uppercase"
  >
    Offline
  </span>
</template>
