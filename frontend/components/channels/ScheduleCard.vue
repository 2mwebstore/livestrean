<script setup lang="ts">
import { LoaderCircleIcon } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { channelService } from '~/services/channel.service'
import type { OwnChannel } from '~/types/channel'

const props = defineProps<{ channel: OwnChannel }>()
const emit = defineEmits<{ updated: [channel: OwnChannel] }>()

const value = ref(toDateTimeLocal(props.channel.scheduled_at))
const minValue = toDateTimeLocal(new Date(Date.now() + 60_000).toISOString())
const busy = ref(false)
const error = ref('')
const saved = ref(false)

const active = computed(() => isScheduleActive(props.channel.scheduled_at))

async function update(scheduledAt: string | null) {
  busy.value = true
  error.value = ''
  try {
    const updated = (await channelService.setSchedule(props.channel.id, scheduledAt)).data
    value.value = toDateTimeLocal(updated.scheduled_at)
    emit('updated', updated)
    saved.value = true
    setTimeout(() => (saved.value = false), 2500)
  }
  catch (err) {
    error.value = apiFieldErrors(err).scheduled_at ?? apiErrorMessage(err)
  }
  finally {
    busy.value = false
  }
}

function save() {
  error.value = ''
  // datetime-local has no time zone: it's the streamer's local time.
  const at = new Date(value.value)
  if (!value.value || Number.isNaN(at.getTime())) {
    error.value = 'Pick a date and time.'
    return
  }
  if (at.getTime() <= Date.now()) {
    error.value = 'Pick a time in the future.'
    return
  }
  update(at.toISOString())
}
</script>

<template>
  <Card>
    <CardHeader>
      <CardTitle>Next live stream</CardTitle>
      <CardDescription>
        Viewers see a countdown on your channel page. It disappears when you go live.
      </CardDescription>
    </CardHeader>
    <CardContent class="grid gap-3">
      <p v-if="active && channel.scheduled_at" class="text-sm">
        Scheduled for <strong>{{ formatScheduleDate(channel.scheduled_at) }}</strong>{{ ' ' }}<span class="text-muted-foreground">({{ Date.parse(channel.scheduled_at) > Date.now() ? `in ${formatTimeUntil(channel.scheduled_at)}` : 'starting soon' }})</span>
      </p>
      <p v-else class="text-sm text-muted-foreground">
        No stream scheduled.
      </p>

      <form class="flex flex-wrap items-center gap-2" @submit.prevent="save">
        <Input
          v-model="value"
          type="datetime-local"
          :min="minValue"
          class="w-auto"
          aria-label="Next live stream date and time"
        />
        <Button type="submit" :disabled="busy">
          <LoaderCircleIcon v-if="busy" class="animate-spin" />
          Save
        </Button>
        <Button v-if="channel.scheduled_at" type="button" variant="outline" :disabled="busy" @click="update(null)">
          Clear
        </Button>
        <span v-if="saved" class="text-sm text-emerald-500">Saved</span>
      </form>
      <p v-if="error" class="text-sm text-destructive" role="alert">
        {{ error }}
      </p>
    </CardContent>
  </Card>
</template>
