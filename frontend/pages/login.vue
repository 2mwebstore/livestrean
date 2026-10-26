<script setup lang="ts">
import { LoaderCircleIcon } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'

definePageMeta({ layout: 'auth', middleware: ['guest'] })
useSeoMeta({ title: 'Log in' })

const route = useRoute()
const { login } = useAuth()

const form = reactive({ email: '', password: '' })
const errors = ref<Record<string, string>>({})
const message = ref('')
const submitting = ref(false)

async function submit() {
  errors.value = {}
  message.value = ''
  if (!form.email.trim()) errors.value.email = 'This field is required'
  if (!form.password) errors.value.password = 'This field is required'
  if (Object.keys(errors.value).length) return

  submitting.value = true
  try {
    await login({ email: form.email.trim(), password: form.password })
    await navigateTo(safeRedirect(route.query.redirect))
  }
  catch (err) {
    message.value = apiErrorMessage(err)
    errors.value = apiFieldErrors(err)
  }
  finally {
    submitting.value = false
  }
}
</script>

<template>
  <Card>
    <CardHeader>
      <CardTitle>Log in</CardTitle>
      <CardDescription>Welcome back. Log in to chat and manage your channels.</CardDescription>
    </CardHeader>
    <CardContent>
      <form class="grid gap-4" novalidate @submit.prevent="submit">
        <FormField id="email" label="Email" :error="errors.email">
          <Input id="email" v-model="form.email" type="email" autocomplete="email" required />
        </FormField>
        <FormField id="password" label="Password" :error="errors.password">
          <Input id="password" v-model="form.password" type="password" autocomplete="current-password" required />
        </FormField>
        <p v-if="message" class="text-sm text-destructive" role="alert">
          {{ message }}
        </p>
        <Button type="submit" class="w-full" :disabled="submitting">
          <LoaderCircleIcon v-if="submitting" class="animate-spin" />
          Log in
        </Button>
      </form>
    </CardContent>
    <CardFooter class="justify-center text-sm text-muted-foreground">
      No account?
      <NuxtLink :to="{ path: '/register', query: route.query }" class="ml-1 text-foreground underline-offset-4 hover:underline">
        Sign up
      </NuxtLink>
    </CardFooter>
  </Card>
</template>
