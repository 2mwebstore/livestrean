<script setup lang="ts">
import { ChevronLeftIcon, ChevronRightIcon } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import type { PaginationMeta } from '~/types/api'

defineProps<{ meta: PaginationMeta }>()
defineEmits<{ change: [page: number] }>()
</script>

<template>
  <div v-if="meta.total_pages > 1" class="flex items-center justify-between gap-4 text-sm text-muted-foreground">
    <span>Page {{ meta.page }} of {{ meta.total_pages }} · {{ meta.total }} total</span>
    <div class="flex gap-1.5">
      <Button variant="outline" size="sm" :disabled="meta.page <= 1" @click="$emit('change', meta.page - 1)">
        <ChevronLeftIcon />
        Previous
      </Button>
      <Button variant="outline" size="sm" :disabled="meta.page >= meta.total_pages" @click="$emit('change', meta.page + 1)">
        Next
        <ChevronRightIcon />
      </Button>
    </div>
  </div>
</template>
