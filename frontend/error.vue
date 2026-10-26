<script setup lang="ts">
import type { NuxtError } from '#app'
import { Button } from '@/components/ui/button'

const props = defineProps<{ error: NuxtError }>()

const code = computed(() => props.error.statusCode ?? 500)

const title = computed(() => ({
  403: 'No access',
  404: 'Page not found',
}[code.value] ?? 'Something went wrong'))

// Only show messages we wrote ourselves; never internal error details.
const message = computed(() => [403, 404].includes(code.value) ? props.error.statusMessage : 'Please try again in a moment.')

useHead({ title })
</script>

<template>
  <NuxtLayout name="default">
    <div class="grid place-items-center py-24 text-center">
      <p class="font-mono text-sm text-muted-foreground">
        {{ code }}
      </p>
      <h1 class="mt-2 text-2xl font-semibold">
        {{ title }}
      </h1>
      <p class="mt-2 max-w-md text-muted-foreground">
        {{ message }}
      </p>
      <Button class="mt-6" @click="clearError({ redirect: '/' })">
        Go to the homepage
      </Button>
    </div>
  </NuxtLayout>
</template>
