package dto

import "time"

type CreateCommentRequest struct {
	Body string `json:"body" binding:"required,max=500"`
}

type CommentUser struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

type CommentResponse struct {
	ID        uint64      `json:"id"`
	Body      string      `json:"body"`
	CreatedAt time.Time   `json:"created_at"`
	User      CommentUser `json:"user"`
}

type CommentDeleted struct {
	ID uint64 `json:"id"`
}
