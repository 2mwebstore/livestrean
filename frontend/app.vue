<script setup lang="ts">
const siteName = useRuntimeConfig().public.siteName
useHead({ titleTemplate: title => (title && title !== siteName ? `${title} · ${siteName}` : siteName) })
</script>

<template>
  <NuxtLoadingIndicator color="var(--live)" />
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>
</template>
