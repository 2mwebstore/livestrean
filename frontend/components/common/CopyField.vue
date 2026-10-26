<script setup lang="ts">
import { CheckIcon, CopyIcon, EyeIcon, EyeOffIcon } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

const props = defineProps<{ label: string, value: string, secret?: boolean }>()

const id = useId()
const revealed = ref(!props.secret)
const copied = ref(false)
const input = ref<{ $el: HTMLInputElement } | null>(null)

async function copy() {
  try {
    await navigator.clipboard.writeText(props.value)
  }
  catch {
    // Clipboard API needs HTTPS or localhost; fall back to selecting the text.
    const el = input.value?.$el
    if (el) {
      el.type = 'text'
      el.select()
      document.execCommand('copy')
      el.type = revealed.value ? 'text' : 'password'
    }
  }
  copied.value = true
  setTimeout(() => (copied.value = false), 1500)
}
</script>

<template>
  <div class="grid gap-1.5">
    <Label :for="id">{{ label }}</Label>
    <div class="flex gap-1.5">
      <Input
        :id="id"
        ref="input"
        :model-value="value"
        :type="revealed ? 'text' : 'password'"
        readonly
        class="font-mono text-xs"
        @focus="($event.target as HTMLInputElement).select()"
      />
      <Button
        v-if="secret"
        variant="outline"
        size="icon"
        :aria-label="revealed ? 'Hide' : 'Show'"
        @click="revealed = !revealed"
      >
        <EyeOffIcon v-if="revealed" />
        <EyeIcon v-else />
      </Button>
      <Button variant="outline" size="icon" :aria-label="`Copy ${label}`" @click="copy">
        <CheckIcon v-if="copied" class="text-emerald-500" />
        <CopyIcon v-else />
      </Button>
    </div>
  </div>
</template>
