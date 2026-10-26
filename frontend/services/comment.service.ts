import type { ApiResponse } from '~/types/api'
import type { Comment, CreateCommentRequest } from '~/types/comment'

export const commentService = {
  list(slug: string, params: { beforeId?: number, limit?: number } = {}) {
    return useApi()<ApiResponse<Comment[]>>(`/channels/${encodeURIComponent(slug)}/comments`, {
      query: { before_id: params.beforeId, limit: params.limit },
    })
  },
  create(slug: string, body: CreateCommentRequest) {
    return useApi()<ApiResponse<Comment>>(`/channels/${encodeURIComponent(slug)}/comments`, { method: 'POST', body })
  },
  remove(id: number) {
    return useApi()<ApiResponse<null>>(`/comments/${id}`, { method: 'DELETE' })
  },
}
