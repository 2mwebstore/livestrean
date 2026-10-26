<script setup lang="ts">
import type { UnwrapNestedRefs } from 'vue'
import { LoaderCircleIcon, MessageSquareIcon, SendIcon } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import type { ChannelChat } from '~/composables/useChannelChat'
import type { Comment } from '~/types/comment'

const props = defineProps<{
  chat: UnwrapNestedRefs<ChannelChat>
  slug: string
  hostId: number
}>()

const MAX_LENGTH = 500
const { user, isAuthenticated } = useAuth()
const { can } = usePermission()

const draft = ref('')
const list = ref<HTMLElement | null>(null)
const redirect = computed(() => ({ redirect: `/c/${props.slug}` }))

// UX mirror of the API's rule; the API decides.
function canDelete(c: Comment) {
  if (!user.value) return false
  return can('comments.moderate') || c.user.id === user.value.id || props.hostId === user.value.id
}

// Keep the newest message in view, unless the reader has scrolled up.
let stickToBottom = true
function onScroll() {
  const el = list.value
  if (el) stickToBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 48
}
function scrollToBottom() {
  list.value?.scrollTo({ top: list.value.scrollHeight })
}
watch(() => props.chat.comments.length, async () => {
  if (!stickToBottom) return
  await nextTick()
  scrollToBottom()
})
onMounted(scrollToBottom)

async function send() {
  const body = draft.value.trim()
  if (!body || props.chat.sending) return
  if (await props.chat.send(body)) {
    draft.value = ''
    stickToBottom = true
  }
}
</script>

<template>
  <section class="flex min-h-0 flex-col rounded-xl border bg-card" aria-label="Live chat">
    <header class="flex items-center gap-2 border-b px-4 py-3 text-sm font-medium">
      <MessageSquareIcon class="size-4" />
      Live chat
    </header>

    <div ref="list" class="min-h-0 flex-1 overflow-y-auto px-2 py-2" role="log" aria-live="polite" @scroll="onScroll">
      <p v-if="!chat.comments.length" class="px-4 py-10 text-center text-sm text-muted-foreground">
        No messages yet. Say hi!
      </p>
      <ChatMessage
        v-for="comment in chat.comments"
        :key="comment.id"
        :comment="comment"
        :is-host="comment.user.id === hostId"
        :can-delete="canDelete(comment)"
        @delete="chat.destroy(comment.id)"
      />
    </div>

    <footer class="border-t p-3">
      <p v-if="chat.error" class="mb-2 text-sm text-destructive" role="alert">
        {{ chat.error }}
      </p>

      <form v-if="isAuthenticated && can('comments.create')" class="flex gap-2" @submit.prevent="send">
        <Input v-model="draft" :maxlength="MAX_LENGTH" placeholder="Send a message" aria-label="Chat message" autocomplete="off" />
        <Button type="submit" size="icon" :disabled="!draft.trim() || chat.sending" aria-label="Send message">
          <LoaderCircleIcon v-if="chat.sending" class="animate-spin" />
          <SendIcon v-else />
        </Button>
      </form>
      <p v-else-if="isAuthenticated" class="text-sm text-muted-foreground">
        Your account can't post in chat.
      </p>
      <div v-else class="flex flex-wrap items-center justify-between gap-2 text-sm">
        <span class="text-muted-foreground">Log in to join the chat.</span>
        <div class="flex gap-2">
          <Button size="sm" variant="outline" as-child>
            <NuxtLink :to="{ path: '/login', query: redirect }">
              Log in
            </NuxtLink>
          </Button>
          <Button size="sm" as-child>
            <NuxtLink :to="{ path: '/register', query: redirect }">
              Sign up
            </NuxtLink>
          </Button>
        </div>
      </div>
    </footer>
  </section>
</template>
