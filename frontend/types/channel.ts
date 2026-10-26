export type ChannelStatus = 'active' | 'disabled'

export interface LiveStatus {
  live: boolean
  started_at: string | null
}

/** Public view of a channel. */
export interface Channel {
  slug: string
  title: string
  description: string
  owner: { id: number, name: string }
  live: boolean
  live_started_at: string | null
  viewers: number
  last_live_at: string | null
  hls_url: string
  whep_url: string
}

/** The owner's dashboard view. Never contains the stream key. */
export interface OwnChannel {
  id: number
  slug: string
  title: string
  description: string
  status: ChannelStatus
  live: boolean
  live_started_at: string | null
  viewers: number
  last_live_at: string | null
  created_at: string
  obs: {
    rtmp_server: string
    whip_url: string
  }
}

/** Returned once, when a channel is created or its key regenerated. */
export interface StreamKeyResult {
  channel: OwnChannel
  stream_key: string
  rtmp_stream_key: string
}

export interface AdminChannel {
  id: number
  slug: string
  title: string
  status: ChannelStatus
  owner: { id: number, name: string, email: string }
  live: boolean
  viewers: number
  created_at: string
}

export interface CreateChannelRequest {
  slug: string
  title: string
  description: string
}

export interface UpdateChannelRequest {
  title: string
  description: string
}

export interface ChannelFormValues {
  slug: string
  title: string
  description: string
}
