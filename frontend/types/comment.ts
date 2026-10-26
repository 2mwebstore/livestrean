export interface Comment {
  id: number
  body: string
  created_at: string
  user: { id: number, name: string }
}

export interface CreateCommentRequest {
  body: string
}
