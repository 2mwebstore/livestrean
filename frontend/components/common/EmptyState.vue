<script setup lang="ts">
defineProps<{ title: string, description?: string }>()
</script>

<template>
  <div class="flex flex-col items-center justify-center gap-2 rounded-xl border border-dashed px-6 py-12 text-center">
    <slot name="icon" />
    <p class="font-medium">
      {{ title }}
    </p>
    <p v-if="description" class="max-w-md text-sm text-muted-foreground">
      {{ description }}
    </p>
    <div v-if="$slots.default" class="mt-3">
      <slot />
    </div>
  </div>
</template>
