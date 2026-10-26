<script setup lang="ts">
import { LoaderCircleIcon, RefreshCwIcon } from '@lucide/vue'
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription,
  AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import type { OwnChannel } from '~/types/channel'

defineProps<{ channel: OwnChannel, regenerating?: boolean }>()
defineEmits<{ regenerate: [] }>()
</script>

<template>
  <Card>
    <CardHeader>
      <CardTitle>Stream settings</CardTitle>
      <CardDescription>In OBS: Settings → Stream. Use either RTMP or WHIP.</CardDescription>
    </CardHeader>
    <CardContent class="grid gap-4">
      <CopyField label="RTMP server (Service: Custom…)" :value="channel.obs.rtmp_server" />
      <CopyField label="WHIP server (Service: WHIP, OBS 30+)" :value="channel.obs.whip_url" />

      <div class="flex flex-wrap items-center justify-between gap-3 rounded-lg border p-3">
        <div class="text-sm">
          <p class="font-medium">
            Stream key
          </p>
          <p class="text-muted-foreground">
            Shown only once, when it's created. Lost it? Generate a new one.
          </p>
        </div>
        <AlertDialog>
          <AlertDialogTrigger as-child>
            <Button variant="outline" :disabled="regenerating">
              <LoaderCircleIcon v-if="regenerating" class="animate-spin" />
              <RefreshCwIcon v-else />
              New stream key
            </Button>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Generate a new stream key?</AlertDialogTitle>
              <AlertDialogDescription>
                The current key stops working immediately, and a stream using it will be disconnected.
                You'll need to paste the new key into OBS.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Cancel</AlertDialogCancel>
              <AlertDialogAction @click="$emit('regenerate')">
                Generate new key
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </div>

      <details class="text-sm text-muted-foreground">
        <summary class="cursor-pointer font-medium text-foreground">
          Recommended OBS output settings
        </summary>
        <ul class="mt-2 list-disc space-y-1 pl-5">
          <li>Settings → Output → Output Mode: Advanced</li>
          <li>Encoder: x264, or Apple VT H264 on a Mac</li>
          <li>Rate control CBR, 2500–6000 Kbps</li>
          <li>Keyframe interval: 1 s</li>
          <li>RTMP sends AAC audio, which only plays in Standard mode. Use WHIP for sound in Low latency mode.</li>
        </ul>
      </details>
    </CardContent>
  </Card>
</template>
