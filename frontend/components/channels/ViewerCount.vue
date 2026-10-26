<script setup lang="ts">
import { EyeIcon } from '@lucide/vue'

defineProps<{ count: number }>()
</script>

<template>
  <span class="inline-flex items-center gap-1 tabular-nums" :title="`${count} watching now`">
    <EyeIcon class="size-3.5" aria-hidden="true" />
    {{ formatCount(count) }}
    <span class="sr-only">watching now</span>
  </span>
</template>
