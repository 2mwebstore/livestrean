<script setup lang="ts">
import { ImageIcon, LoaderCircleIcon, Trash2Icon, UploadIcon } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { channelService } from '~/services/channel.service'
import type { OwnChannel } from '~/types/channel'

const props = defineProps<{ channel: OwnChannel }>()
const emit = defineEmits<{ updated: [channel: OwnChannel] }>()

// Same limits as the API (backend/internal/services/thumbnail.go).
const ACCEPTED = ['image/jpeg', 'image/png', 'image/webp', 'image/gif']
const MAX_BYTES = 5 * 1024 * 1024

const input = ref<HTMLInputElement | null>(null)
const busy = ref(false)
const error = ref('')

async function onFile(event: Event) {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  target.value = '' // so choosing the same file again still triggers change
  if (!file) return
  error.value = ''
  if (!ACCEPTED.includes(file.type)) {
    error.value = 'Choose a JPEG, PNG, WebP or GIF image.'
    return
  }
  if (file.size > MAX_BYTES) {
    error.value = 'The image must be 5 MB or smaller.'
    return
  }
  busy.value = true
  try {
    emit('updated', (await channelService.uploadThumbnail(props.channel.id, file)).data)
  }
  catch (err) {
    error.value = apiFieldErrors(err).thumbnail ?? apiErrorMessage(err)
  }
  finally {
    busy.value = false
  }
}

async function remove() {
  busy.value = true
  error.value = ''
  try {
    emit('updated', (await channelService.removeThumbnail(props.channel.id)).data)
  }
  catch (err) {
    error.value = apiErrorMessage(err)
  }
  finally {
    busy.value = false
  }
}
</script>

<template>
  <Card>
    <CardHeader>
      <CardTitle>Thumbnail</CardTitle>
      <CardDescription>
        Shown on channel cards, behind the player while you're offline, and when your link is shared.
        Use a 16:9 image; it's cropped to 1280×720.
      </CardDescription>
    </CardHeader>
    <CardContent class="grid gap-3">
      <div class="relative aspect-video overflow-hidden rounded-lg border bg-muted">
        <img v-if="channel.thumbnail_url" :src="channel.thumbnail_url" alt="Channel thumbnail" class="size-full object-cover">
        <div v-else class="grid size-full place-items-center text-sm text-muted-foreground">
          <div class="flex flex-col items-center gap-2">
            <ImageIcon class="size-6" />
            No thumbnail yet
          </div>
        </div>
        <div v-if="busy" class="absolute inset-0 grid place-items-center bg-black/50">
          <LoaderCircleIcon class="size-6 animate-spin text-white" />
        </div>
      </div>
      <p v-if="error" class="text-sm text-destructive" role="alert">
        {{ error }}
      </p>
      <div class="flex flex-wrap gap-2">
        <input ref="input" type="file" :accept="ACCEPTED.join(',')" class="hidden" aria-label="Thumbnail image" @change="onFile">
        <Button :disabled="busy" @click="input?.click()">
          <UploadIcon />
          {{ channel.thumbnail_url ? 'Replace image' : 'Upload image' }}
        </Button>
        <Button v-if="channel.thumbnail_url" variant="outline" :disabled="busy" @click="remove">
          <Trash2Icon />
          Remove
        </Button>
      </div>
    </CardContent>
  </Card>
</template>
