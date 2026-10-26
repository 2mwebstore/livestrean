<script setup lang="ts">
import { KeyRoundIcon } from '@lucide/vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Separator } from '@/components/ui/separator'
import type { StreamKeyResult } from '~/types/channel'

defineProps<{ result: StreamKeyResult | null }>()
const open = defineModel<boolean>('open', { required: true })
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent v-if="result" class="sm:max-w-xl">
      <DialogHeader>
        <DialogTitle>Your stream key for “{{ result.channel.title }}”</DialogTitle>
        <DialogDescription>Paste these into OBS: Settings → Stream.</DialogDescription>
      </DialogHeader>

      <Alert>
        <KeyRoundIcon />
        <AlertTitle>Copy it now</AlertTitle>
        <AlertDescription>
          For security the key is only shown once. If you lose it, generate a new one from the channel page.
          Anyone with this key can stream to your channel.
        </AlertDescription>
      </Alert>

      <div class="grid gap-3">
        <p class="text-sm font-medium">
          Option A: RTMP (any OBS version). Service: Custom…
        </p>
        <CopyField label="Server" :value="result.channel.obs.rtmp_server" />
        <CopyField label="Stream Key" :value="result.rtmp_stream_key" secret />
      </div>

      <Separator />

      <div class="grid gap-3">
        <p class="text-sm font-medium">
          Option B: WHIP (OBS 30+, under 1 second delay). Service: WHIP
        </p>
        <CopyField label="Server" :value="result.channel.obs.whip_url" />
        <CopyField label="Bearer Token" :value="result.stream_key" secret />
      </div>

      <DialogFooter>
        <Button @click="open = false">
          I've copied it
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
