<script setup lang="ts">
const { can } = usePermission()
const route = useRoute()
const items = computed(() => adminNav.filter(item => !item.permission || can(item.permission)))
</script>

<template>
  <div class="flex min-h-dvh flex-col">
    <AppHeader />
    <div class="mx-auto flex w-full max-w-7xl flex-1 flex-col gap-6 px-4 py-6 md:flex-row">
      <nav class="flex gap-1 md:w-44 md:flex-col" aria-label="Admin">
        <NuxtLink
          v-for="item in items"
          :key="item.to"
          :to="item.to"
          class="rounded-md px-3 py-2 text-sm text-muted-foreground hover:bg-muted hover:text-foreground"
          :class="{ 'bg-muted text-foreground': route.path.startsWith(item.to) }"
        >
          {{ item.label }}
        </NuxtLink>
      </nav>
      <main class="min-w-0 flex-1">
        <slot />
      </main>
    </div>
  </div>
</template>
