import type { ApiResponse, PaginatedResponse } from '~/types/api'
import type { Channel, CreateChannelRequest, OwnChannel, StreamKeyResult, UpdateChannelRequest } from '~/types/channel'

export const channelService = {
  // Public. live: true = only live channels, false = only offline ones.
  list(params: { live?: boolean, page?: number, limit?: number } = {}) {
    return useApi()<PaginatedResponse<Channel>>('/channels', {
      query: { live: params.live === undefined ? undefined : String(params.live), page: params.page, limit: params.limit },
    })
  },
  get(slug: string) {
    return useApi()<ApiResponse<Channel>>(`/channels/${encodeURIComponent(slug)}`)
  },

  // The logged-in streamer's own channels
  listOwn() {
    return useApi()<ApiResponse<OwnChannel[]>>('/me/channels')
  },
  getOwn(id: number) {
    return useApi()<ApiResponse<OwnChannel>>(`/me/channels/${id}`)
  },
  create(body: CreateChannelRequest) {
    return useApi()<ApiResponse<StreamKeyResult>>('/me/channels', { method: 'POST', body })
  },
  update(id: number, body: UpdateChannelRequest) {
    return useApi()<ApiResponse<OwnChannel>>(`/me/channels/${id}`, { method: 'PUT', body })
  },
  regenerateKey(id: number) {
    return useApi()<ApiResponse<StreamKeyResult>>(`/me/channels/${id}/stream-key`, { method: 'POST' })
  },
  remove(id: number) {
    return useApi()<ApiResponse<null>>(`/me/channels/${id}`, { method: 'DELETE' })
  },
}
