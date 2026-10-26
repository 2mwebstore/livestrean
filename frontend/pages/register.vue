<script setup lang="ts">
import { LoaderCircleIcon } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'

definePageMeta({ layout: 'auth', middleware: ['guest'] })
useSeoMeta({ title: 'Sign up' })

const route = useRoute()
const { register } = useAuth()

const form = reactive({ name: '', email: '', password: '' })
const errors = ref<Record<string, string>>({})
const message = ref('')
const submitting = ref(false)

function validate() {
  const e: Record<string, string> = {}
  const nameError = lengthError(form.name, { required: true, max: 80 })
  if (nameError) e.name = nameError
  else if (form.name.trim().length < 2) e.name = 'Must be at least 2 characters'
  if (!/^\S+@\S+\.\S+$/.test(form.email.trim())) e.email = 'Enter a valid email address'
  if (form.password.length < 8) e.password = 'Must be at least 8 characters'
  return e
}

async function submit() {
  message.value = ''
  errors.value = validate()
  if (Object.keys(errors.value).length) return

  submitting.value = true
  try {
    await register({ name: form.name.trim(), email: form.email.trim(), password: form.password })
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
      <CardTitle>Create an account</CardTitle>
      <CardDescription>Join the chat on any channel.</CardDescription>
    </CardHeader>
    <CardContent>
      <form class="grid gap-4" novalidate @submit.prevent="submit">
        <FormField id="name" label="Display name" :error="errors.name" hint="Shown next to your chat messages.">
          <Input id="name" v-model="form.name" maxlength="80" autocomplete="nickname" required />
        </FormField>
        <FormField id="email" label="Email" :error="errors.email">
          <Input id="email" v-model="form.email" type="email" autocomplete="email" required />
        </FormField>
        <FormField id="password" label="Password" :error="errors.password" hint="At least 8 characters.">
          <Input id="password" v-model="form.password" type="password" autocomplete="new-password" required />
        </FormField>
        <p v-if="message" class="text-sm text-destructive" role="alert">
          {{ message }}
        </p>
        <Button type="submit" class="w-full" :disabled="submitting">
          <LoaderCircleIcon v-if="submitting" class="animate-spin" />
          Sign up
        </Button>
      </form>
    </CardContent>
    <CardFooter class="justify-center text-sm text-muted-foreground">
      Already have an account?
      <NuxtLink :to="{ path: '/login', query: route.query }" class="ml-1 text-foreground underline-offset-4 hover:underline">
        Log in
      </NuxtLink>
    </CardFooter>
  </Card>
</template>
