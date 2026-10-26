import { commentService } from '~/services/comment.service'
import type { Comment } from '~/types/comment'

/** Most messages kept on screen; older ones scroll away. */
const MAX_MESSAGES = 200

/** Chat state for one channel: the message list, sending and deleting. */
export function useChannelChat(slug: string, initial: Comment[] = []) {
  const comments = ref<Comment[]>([...initial])
  const sending = ref(false)
  const error = ref('')

  // From our own POST and from the WebSocket; the same message can arrive twice.
  function add(comment: Comment) {
    if (comments.value.some(c => c.id === comment.id)) return
    comments.value.push(comment)
    if (comments.value.length > MAX_MESSAGES) {
      comments.value.splice(0, comments.value.length - MAX_MESSAGES)
    }
  }

  function remove(id: number) {
    comments.value = comments.value.filter(c => c.id !== id)
  }

  async function refresh() {
    try {
      comments.value = (await commentService.list(slug)).data
    }
    catch {
      // Keep what we have; the next reconnect tries again.
    }
  }

  async function send(body: string): Promise<boolean> {
    error.value = ''
    sending.value = true
    try {
      add((await commentService.create(slug, { body })).data)
      return true
    }
    catch (err) {
      error.value = apiErrorMessage(err)
      return false
    }
    finally {
      sending.value = false
    }
  }

  async function destroy(id: number) {
    error.value = ''
    try {
      await commentService.remove(id)
      remove(id)
    }
    catch (err) {
      error.value = apiErrorMessage(err)
    }
  }

  return { comments, sending, error, add, remove, refresh, send, destroy }
}

export type ChannelChat = ReturnType<typeof useChannelChat>
