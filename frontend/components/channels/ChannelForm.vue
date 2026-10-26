<script setup lang="ts">
import { LoaderCircleIcon } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import type { ChannelFormValues } from '~/types/channel'

const props = defineProps<{
  mode: 'create' | 'edit'
  initial?: Partial<ChannelFormValues>
  submitting?: boolean
  /** Field errors from the API, e.g. { slug: "This channel URL is already taken" }. */
  serverErrors?: Record<string, string>
}>()
const emit = defineEmits<{ submit: [values: ChannelFormValues] }>()

const form = reactive<ChannelFormValues>({
  slug: props.initial?.slug ?? '',
  title: props.initial?.title ?? '',
  description: props.initial?.description ?? '',
})
const touched = ref(false)
// Suggest a URL from the title until the user edits the URL themselves.
const slugEdited = ref(props.mode === 'edit')

watch(() => form.title, (title) => {
  if (!slugEdited.value) form.slug = slugify(title)
})

const origin = useRequestURL().origin

const errors = computed(() => {
  const local: Record<string, string | undefined> = {
    slug: props.mode === 'create' ? slugError(form.slug) : undefined,
    title: lengthError(form.title, { required: true, max: 100 }),
    description: lengthError(form.description, { max: 1000 }),
  }
  return {
    slug: (touched.value && local.slug) || props.serverErrors?.slug,
    title: (touched.value && local.title) || props.serverErrors?.title,
    description: (touched.value && local.description) || props.serverErrors?.description,
    valid: !local.slug && !local.title && !local.description,
  }
})

function submit() {
  touched.value = true
  if (!errors.value.valid) return
  emit('submit', { slug: form.slug, title: form.title.trim(), description: form.description.trim() })
}
</script>

<template>
  <form class="grid gap-4" novalidate @submit.prevent="submit">
    <FormField id="channel-title" label="Title" :error="errors.title">
      <Input id="channel-title" v-model="form.title" maxlength="100" placeholder="Friday night music" autocomplete="off" />
    </FormField>

    <FormField
      id="channel-slug"
      label="Channel URL"
      :error="errors.slug"
      :hint="mode === 'create' ? `${origin}/c/${form.slug || 'your-channel'} (can't be changed later)` : `${origin}/c/${form.slug}`"
    >
      <Input
        id="channel-slug"
        v-model="form.slug"
        :disabled="mode === 'edit'"
        maxlength="40"
        placeholder="friday-music"
        autocomplete="off"
        class="font-mono"
        @input="slugEdited = true"
      />
    </FormField>

    <FormField id="channel-description" label="Description" :error="errors.description" hint="Optional. Shown under the video.">
      <Textarea id="channel-description" v-model="form.description" maxlength="1000" rows="3" />
    </FormField>

    <div class="flex justify-end gap-2">
      <slot name="actions" />
      <Button type="submit" :disabled="submitting">
        <LoaderCircleIcon v-if="submitting" class="animate-spin" />
        {{ mode === 'create' ? 'Create channel' : 'Save changes' }}
      </Button>
    </div>
  </form>
</template>
