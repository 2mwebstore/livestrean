<script setup lang="ts">
import { useIntervalFn } from '@vueuse/core'
import { ChevronLeftIcon, LoaderCircleIcon, Trash2Icon } from '@lucide/vue'
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription,
  AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { channelService } from '~/services/channel.service'
import type { ChannelFormValues, StreamKeyResult } from '~/types/channel'

definePageMeta({ middleware: ['auth', 'permission'], permission: 'channels.manage_own' })

const route = useRoute()
const id = Number(route.params.id)

const { data: channel, status, error, refresh } = await useAsyncData(`me:channel:${id}`, () => channelService.getOwn(id).then(r => r.data))
const notFound = computed(() => apiErrorStatus(error.value) === 404 || error.value?.statusCode === 404)
useSeoMeta({ title: () => channel.value?.title ?? 'Channel' })

// Refresh live status and viewers. Polling instead of the channel's
// WebSocket, so the streamer's own dashboard isn't counted as a viewer.
useIntervalFn(() => {
  if (channel.value) refresh()
}, 10_000)
const live = computed(() => channel.value?.live ?? false)
const viewers = computed(() => channel.value?.viewers ?? 0)

const actionError = ref('')

// Details
const saving = ref(false)
const saved = ref(false)
const saveErrors = ref<Record<string, string>>({})
async function save(values: ChannelFormValues) {
  saving.value = true
  saveErrors.value = {}
  actionError.value = ''
  try {
    channel.value = (await channelService.update(id, { title: values.title, description: values.description })).data
    saved.value = true
    setTimeout(() => (saved.value = false), 2500)
  }
  catch (err) {
    saveErrors.value = apiFieldErrors(err)
    actionError.value = apiErrorMessage(err)
  }
  finally {
    saving.value = false
  }
}

// Stream key
const regenerating = ref(false)
const keyResult = ref<StreamKeyResult | null>(null)
const keyOpen = ref(false)
async function regenerate() {
  regenerating.value = true
  actionError.value = ''
  try {
    keyResult.value = (await channelService.regenerateKey(id)).data
    keyOpen.value = true
  }
  catch (err) {
    actionError.value = apiErrorMessage(err)
  }
  finally {
    regenerating.value = false
  }
}

// Delete
const deleting = ref(false)
async function remove() {
  deleting.value = true
  actionError.value = ''
  try {
    await channelService.remove(id)
    await navigateTo('/dashboard')
  }
  catch (err) {
    actionError.value = apiErrorMessage(err)
    deleting.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <NuxtLink to="/dashboard" class="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
      <ChevronLeftIcon class="size-4" />
      Your channels
    </NuxtLink>

    <EmptyState v-if="notFound" title="Channel not found" description="It may have been deleted, or it belongs to someone else." />
    <ErrorState v-else-if="error" :message="apiErrorMessage(error)" @retry="refresh()" />
    <div v-else-if="status === 'pending' && !channel" class="grid gap-4">
      <Skeleton class="h-8 w-64" />
      <Skeleton class="h-64 w-full rounded-xl" />
    </div>

    <template v-else-if="channel">
      <div class="flex flex-wrap items-center justify-between gap-4">
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2">
            <h1 class="truncate text-xl font-semibold">
              {{ channel.title }}
            </h1>
            <LiveBadge :live="live" />
            <Badge v-if="channel.status !== 'active'" variant="destructive">
              Disabled by admin
            </Badge>
          </div>
          <p class="text-sm text-muted-foreground">
            <ViewerCount v-if="live" :count="viewers" />
            <template v-else-if="channel.last_live_at">
              Last live {{ formatRelativeTime(channel.last_live_at) }}
            </template>
            <template v-else>
              Never streamed yet
            </template>
          </p>
        </div>
        <Button variant="outline" as-child>
          <NuxtLink :to="`/c/${channel.slug}`">
            Open channel page
          </NuxtLink>
        </Button>
      </div>

      <p v-if="actionError" class="text-sm text-destructive" role="alert">
        {{ actionError }}
      </p>

      <div class="grid gap-6 lg:grid-cols-2">
        <ObsSettingsCard :channel="channel" :regenerating="regenerating" @regenerate="regenerate" />

        <div class="grid content-start gap-6">
          <ScheduleCard :channel="channel" @updated="(c) => (channel = c)" />
          <ThumbnailCard :channel="channel" @updated="(c) => (channel = c)" />
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Details</CardTitle>
            <CardDescription>Shown to viewers on the channel page.</CardDescription>
          </CardHeader>
          <CardContent>
            <ChannelForm mode="edit" :initial="channel" :submitting="saving" :server-errors="saveErrors" @submit="save">
              <template #actions>
                <span v-if="saved" class="self-center text-sm text-emerald-500">Saved</span>
              </template>
            </ChannelForm>
          </CardContent>
        </Card>

        <ShareCard :slug="channel.slug" />

        <Card class="border-destructive/40">
          <CardHeader>
            <CardTitle>Delete channel</CardTitle>
            <CardDescription>Removes the channel, its chat history and its stream key. This can't be undone.</CardDescription>
          </CardHeader>
          <CardContent>
            <AlertDialog>
              <AlertDialogTrigger as-child>
                <Button variant="destructive" :disabled="deleting">
                  <LoaderCircleIcon v-if="deleting" class="animate-spin" />
                  <Trash2Icon v-else />
                  Delete channel
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Delete “{{ channel.title }}”?</AlertDialogTitle>
                  <AlertDialogDescription>
                    The channel page, chat history and stream key are deleted, and a live stream is stopped.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction @click="remove">
                    Delete
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </CardContent>
        </Card>
      </div>

      <StreamKeyDialog v-model:open="keyOpen" :result="keyResult" />
    </template>
  </div>
</template>
