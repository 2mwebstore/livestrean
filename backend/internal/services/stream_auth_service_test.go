package services

import (
	"slices"
	"testing"

	"livestream/internal/dto"
)

func TestPresentedKeys(t *testing.T) {
	cases := []struct {
		name string
		req  dto.MediaMTXAuthRequest
		want string
	}{
		{"rtmp query credentials", dto.MediaMTXAuthRequest{Password: "sk_1"}, "sk_1"},
		{"whip bearer", dto.MediaMTXAuthRequest{Token: "sk_2"}, "sk_2"},
		{"whip bearer user:pass", dto.MediaMTXAuthRequest{Token: "obs:sk_3"}, "sk_3"},
		{"query key", dto.MediaMTXAuthRequest{Query: "key=sk_4"}, "sk_4"},
	}
	for _, tc := range cases {
		if got := PresentedKeys(tc.req); !slices.Contains(got, tc.want) {
			t.Errorf("%s: %v does not contain %q", tc.name, got, tc.want)
		}
	}
	if got := PresentedKeys(dto.MediaMTXAuthRequest{}); len(got) != 0 {
		t.Errorf("empty request gave %v", got)
	}
}

func TestCleanCommentBody(t *testing.T) {
	cases := map[string]string{
		"  hello  ":           "hello",
		"line1\nline2":        "line1 line2",
		"bell\x07char":        "bellchar",
		"\x00\x01":            "",
		"<b>kept as text</b>": "<b>kept as text</b>", // escaped when rendered, not here
		"សួស្តី":              "សួស្តី",
	}
	for in, want := range cases {
		if got := cleanCommentBody(in); got != want {
			t.Errorf("cleanCommentBody(%q) = %q, want %q", in, got, want)
		}
	}
}
