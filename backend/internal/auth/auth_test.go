package auth

import (
	"strings"
	"testing"
)

func TestTokenRoundTrip(t *testing.T) {
	token, hash := NewToken(StreamKeyPrefix)
	if !strings.HasPrefix(token, StreamKeyPrefix) {
		t.Fatalf("token %q missing prefix", token)
	}
	if len(hash) != 64 {
		t.Fatalf("hash length = %d, want 64", len(hash))
	}
	if !TokenMatches(token, hash) {
		t.Fatal("token should match its own hash")
	}
	if TokenMatches(token+"x", hash) {
		t.Fatal("altered token must not match")
	}
	other, _ := NewToken(StreamKeyPrefix)
	if other == token {
		t.Fatal("tokens must be unique")
	}
}

func TestPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(hash, "correct horse battery") {
		t.Fatal("correct password rejected")
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("wrong password accepted")
	}
}

func TestPrincipalCan(t *testing.T) {
	p := NewPrincipal(1, "A", "a@example.com", RoleViewer, 1, []string{PermCommentsCreate})
	if !p.Can(PermCommentsCreate) || p.Can(PermUsersView) {
		t.Fatal("permission check wrong")
	}
	var nobody *Principal
	if nobody.Can(PermCommentsCreate) {
		t.Fatal("nil principal must have no permissions")
	}
}
