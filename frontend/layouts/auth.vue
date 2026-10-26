<script setup lang="ts">
const siteName = useRuntimeConfig().public.siteName
</script>

<template>
  <div class="grid min-h-dvh place-items-center px-4 py-10">
    <div class="w-full max-w-sm">
      <NuxtLink to="/" class="mb-6 flex items-center justify-center gap-2 text-lg font-semibold">
        <span class="size-3 rounded-full bg-live" aria-hidden="true" />
        {{ siteName }}
      </NuxtLink>
      <slot />
    </div>
  </div>
</template>
