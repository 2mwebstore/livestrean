import type { LiveStatus } from './channel'
import type { Comment } from './comment'

/** Events pushed on /ws/channels/:slug (backend/internal/realtime). */
export type ChannelEvent =
  | { type: 'status', data: LiveStatus }
  | { type: 'viewers', data: { count: number } }
  | { type: 'comment', data: Comment }
  | { type: 'comment_deleted', data: { id: number } }
