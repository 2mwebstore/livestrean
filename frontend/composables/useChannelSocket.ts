import type { LiveStatus } from '~/types/channel'
import type { Comment } from '~/types/comment'
import type { ChannelEvent } from '~/types/realtime'

interface Handlers {
  onComment?: (comment: Comment) => void
  onCommentDeleted?: (id: number) => void
  /** Called after reconnecting, to catch up on anything missed. */
  onReconnect?: () => void
}

/**
 * Live status, viewer count and chat events for one channel, over
 * WebSocket. Browser-only: connects on mount, reconnects with backoff,
 * and disconnects on unmount. While connected, this page counts as a viewer.
 */
export function useChannelSocket(slug: string, handlers: Handlers = {}) {
  const status = ref<LiveStatus | null>(null)
  const viewers = ref<number | null>(null)
  const connected = ref(false)

  let socket: WebSocket | null = null
  let attempts = 0
  let everConnected = false
  let retryTimer: ReturnType<typeof setTimeout> | undefined
  let stopped = false

  function connect() {
    if (stopped) return
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    socket = new WebSocket(`${protocol}//${location.host}/ws/channels/${encodeURIComponent(slug)}`)

    socket.onopen = () => {
      connected.value = true
      attempts = 0
      if (everConnected) handlers.onReconnect?.()
      everConnected = true
    }

    socket.onmessage = (message) => {
      let event: ChannelEvent
      try {
        event = JSON.parse(message.data)
      }
      catch {
        return
      }
      switch (event.type) {
        case 'status':
          status.value = event.data
          break
        case 'viewers':
          viewers.value = event.data.count
          break
        case 'comment':
          handlers.onComment?.(event.data)
          break
        case 'comment_deleted':
          handlers.onCommentDeleted?.(event.data.id)
          break
      }
    }

    socket.onclose = () => {
      connected.value = false
      socket = null
      if (!stopped) {
        retryTimer = setTimeout(connect, Math.min(30_000, 1000 * 2 ** attempts++))
      }
    }
  }

  onMounted(connect)
  onBeforeUnmount(() => {
    stopped = true
    clearTimeout(retryTimer)
    socket?.close()
  })

  return { status, viewers, connected }
}
