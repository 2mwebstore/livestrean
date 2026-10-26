package policies

import (
	"testing"

	"livestream/internal/auth"
	"livestream/internal/models"
)

func principal(id uint64, perms ...string) *auth.Principal {
	return auth.NewPrincipal(id, "u", "u@example.com", "", 1, perms)
}

func TestCanManageOwnChannel(t *testing.T) {
	ch := &models.Channel{OwnerID: 1}
	if !CanManageOwnChannel(principal(1, auth.PermChannelsManageOwn), ch) {
		t.Error("owner with permission should manage")
	}
	if CanManageOwnChannel(principal(2, auth.PermChannelsManageOwn), ch) {
		t.Error("another streamer must not manage")
	}
	if CanManageOwnChannel(principal(1), ch) {
		t.Error("owner without permission (demoted) must not manage")
	}
	if CanManageOwnChannel(nil, ch) {
		t.Error("anonymous must not manage")
	}
}

func TestCanDeleteComment(t *testing.T) {
	ch := &models.Channel{OwnerID: 10}
	c := &models.Comment{UserID: 20}
	cases := []struct {
		name string
		p    *auth.Principal
		want bool
	}{
		{"author", principal(20), true},
		{"channel owner", principal(10), true},
		{"moderator", principal(30, auth.PermCommentsModerate), true},
		{"other viewer", principal(30, auth.PermCommentsCreate), false},
		{"anonymous", nil, false},
	}
	for _, tc := range cases {
		if got := CanDeleteComment(tc.p, c, ch); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestIsPublic(t *testing.T) {
	active := models.User{Status: models.StatusActive}
	disabled := models.User{Status: models.StatusDisabled}
	if !IsPublic(&models.Channel{Status: models.StatusActive, Owner: active}) {
		t.Error("active channel should be public")
	}
	if IsPublic(&models.Channel{Status: models.StatusDisabled, Owner: active}) {
		t.Error("disabled channel must not be public")
	}
	if IsPublic(&models.Channel{Status: models.StatusActive, Owner: disabled}) {
		t.Error("channel of a disabled user must not be public")
	}
}
