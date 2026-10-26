<script setup lang="ts">
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

const props = defineProps<{ slug: string }>()

const origin = useRequestURL().origin
const watchUrl = computed(() => `${origin}/c/${props.slug}`)
const embedCode = computed(() =>
  `<iframe src="${origin}/embed/${props.slug}" width="960" height="540" style="border:0;max-width:100%;aspect-ratio:16/9;height:auto" allow="autoplay; fullscreen; picture-in-picture" allowfullscreen></iframe>`,
)
</script>

<template>
  <Card>
    <CardHeader>
      <CardTitle>Share</CardTitle>
      <CardDescription>Link to the channel page, or put the player on any other website.</CardDescription>
    </CardHeader>
    <CardContent class="grid gap-4">
      <CopyField label="Watch link" :value="watchUrl" />
      <CopyField label="Embed code" :value="embedCode" />
    </CardContent>
  </Card>
</template>
