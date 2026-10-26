<script setup lang="ts">
import { LogOutIcon } from '@lucide/vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'

const { user, isAuthenticated, logout } = useAuth()
const { can } = usePermission()
const route = useRoute()
const siteName = useRuntimeConfig().public.siteName

const links = computed(() => mainNav.filter(item => !item.permission || can(item.permission)))
const loginHref = computed(() => ({ path: '/login', query: route.path === '/' ? {} : { redirect: route.fullPath } }))

function isActive(to: string) {
  return to === '/' ? route.path === '/' : route.path.startsWith(to)
}

async function onLogout() {
  await logout()
  await navigateTo('/')
}
</script>

<template>
  <header class="sticky top-0 z-40 border-b bg-background/90 backdrop-blur">
    <div class="mx-auto flex h-14 max-w-7xl items-center gap-4 px-4">
      <NuxtLink to="/" class="flex items-center gap-2 font-semibold">
        <span class="size-2.5 rounded-full bg-live" aria-hidden="true" />
        {{ siteName }}
      </NuxtLink>

      <nav class="flex items-center gap-1 text-sm" aria-label="Main">
        <NuxtLink
          v-for="item in links"
          :key="item.to"
          :to="item.to"
          class="rounded-md px-2.5 py-1.5 text-muted-foreground transition-colors hover:text-foreground"
          :class="{ 'bg-muted text-foreground': isActive(item.to) }"
        >
          {{ item.label }}
        </NuxtLink>
      </nav>

      <div class="ml-auto flex items-center gap-2">
        <template v-if="isAuthenticated && user">
          <span class="hidden max-w-40 truncate text-sm sm:inline">{{ user.name }}</span>
          <Badge variant="secondary" class="hidden capitalize sm:inline-flex">
            {{ user.role }}
          </Badge>
          <Button variant="ghost" size="sm" @click="onLogout">
            <LogOutIcon />
            <span class="hidden sm:inline">Log out</span>
          </Button>
        </template>
        <template v-else>
          <Button variant="ghost" size="sm" as-child>
            <NuxtLink :to="loginHref">
              Log in
            </NuxtLink>
          </Button>
          <Button size="sm" as-child>
            <NuxtLink to="/register">
              Sign up
            </NuxtLink>
          </Button>
        </template>
      </div>
    </div>
  </header>
</template>
