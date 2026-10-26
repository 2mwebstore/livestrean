package auth

import "sort"

// Role slugs, seeded in migrations/001_create_roles_permissions.sql.
const (
	RoleAdmin    = "admin"
	RoleStreamer = "streamer"
	RoleViewer   = "viewer"
)

// Permission slugs. The frontend uses the same identifiers for UI checks.
const (
	PermCommentsCreate    = "comments.create"
	PermCommentsModerate  = "comments.moderate"
	PermChannelsCreate    = "channels.create"
	PermChannelsManageOwn = "channels.manage_own"
	PermChannelsViewAll   = "channels.view_all"
	PermChannelsUpdateAll = "channels.update_all"
	PermUsersView         = "users.view"
	PermUsersUpdate       = "users.update"
)

// Principal is the authenticated user attached to a request.
type Principal struct {
	UserID      uint64
	Name        string
	Email       string
	Role        string
	SessionID   uint64
	permissions map[string]struct{}
}

func NewPrincipal(userID uint64, name, email, role string, sessionID uint64, permissions []string) *Principal {
	set := make(map[string]struct{}, len(permissions))
	for _, p := range permissions {
		set[p] = struct{}{}
	}
	return &Principal{UserID: userID, Name: name, Email: email, Role: role, SessionID: sessionID, permissions: set}
}

func (p *Principal) Can(permission string) bool {
	if p == nil {
		return false
	}
	_, ok := p.permissions[permission]
	return ok
}

func (p *Principal) Permissions() []string {
	out := make([]string, 0, len(p.permissions))
	for perm := range p.permissions {
		out = append(out, perm)
	}
	sort.Strings(out)
	return out
}
