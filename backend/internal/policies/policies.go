// Package policies holds resource-level authorization rules: checks that
// depend on who owns a record, on top of role permissions.
package policies

import (
	"livestream/internal/auth"
	"livestream/internal/models"
)

// CanManageOwnChannel covers the owner's dashboard actions: viewing OBS
// settings, editing, regenerating the stream key and deleting.
func CanManageOwnChannel(p *auth.Principal, ch *models.Channel) bool {
	return p.Can(auth.PermChannelsManageOwn) && ch.OwnerID == p.UserID
}

// CanDeleteComment allows moderators, the comment's author, and the owner
// of the channel it was posted in.
func CanDeleteComment(p *auth.Principal, c *models.Comment, ch *models.Channel) bool {
	if p == nil {
		return false
	}
	return p.Can(auth.PermCommentsModerate) || c.UserID == p.UserID || ch.OwnerID == p.UserID
}

// IsPublic reports whether a channel may be watched and streamed to.
func IsPublic(ch *models.Channel) bool {
	return ch.Status == models.StatusActive && ch.Owner.Status == models.StatusActive
}
