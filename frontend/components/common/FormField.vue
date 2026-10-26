<script setup lang="ts">
import { Label } from '@/components/ui/label'

defineProps<{ id: string, label: string, error?: string, hint?: string }>()
</script>

<template>
  <div class="grid gap-1.5">
    <Label :for="id">{{ label }}</Label>
    <slot />
    <p v-if="error" :id="`${id}-error`" class="text-sm text-destructive">
      {{ error }}
    </p>
    <p v-else-if="hint" class="text-xs text-muted-foreground">
      {{ hint }}
    </p>
  </div>
</template>
