<script setup lang="ts">
import { Trash2Icon } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import type { Comment } from '~/types/comment'

const props = defineProps<{ comment: Comment, isHost: boolean, canDelete: boolean }>()
defineEmits<{ delete: [] }>()

// A stable name color per user.
const nameColor = computed(() => `oklch(0.78 0.13 ${(props.comment.user.id * 67) % 360})`)
</script>

<template>
  <div class="group flex items-start gap-2 rounded-md px-1.5 py-1 text-sm hover:bg-muted/50">
    <p class="min-w-0 flex-1 break-words">
      <span class="mr-1.5 font-semibold" :style="{ color: nameColor }">{{ comment.user.name }}</span>
      <span v-if="isHost" class="mr-1.5 rounded bg-live/15 px-1 py-px text-[10px] font-bold text-live uppercase">Host</span>
      <!-- Plain text interpolation: Vue escapes it, so messages can't inject HTML. -->
      <span>{{ comment.body }}</span>
    </p>
    <ClientOnly>
      <time class="mt-0.5 shrink-0 text-xs text-muted-foreground opacity-0 group-hover:opacity-100" :datetime="comment.created_at">
        {{ formatTime(comment.created_at) }}
      </time>
    </ClientOnly>
    <Button
      v-if="canDelete"
      variant="ghost"
      size="icon-xs"
      class="shrink-0 opacity-0 group-hover:opacity-100 focus-visible:opacity-100"
      aria-label="Delete message"
      @click="$emit('delete')"
    >
      <Trash2Icon />
    </Button>
  </div>
</template>
