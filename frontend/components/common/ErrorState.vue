<script setup lang="ts">
import { TriangleAlertIcon } from '@lucide/vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'

withDefaults(defineProps<{ title?: string, message: string, retryable?: boolean }>(), {
  title: 'Couldn\'t load this',
  retryable: true,
})
defineEmits<{ retry: [] }>()
</script>

<template>
  <Alert variant="destructive">
    <TriangleAlertIcon />
    <AlertTitle>{{ title }}</AlertTitle>
    <AlertDescription>
      <p>{{ message }}</p>
      <Button v-if="retryable" variant="outline" size="sm" class="mt-2" @click="$emit('retry')">
        Try again
      </Button>
    </AlertDescription>
  </Alert>
</template>
