package validators

import "testing"

func TestValidSlug(t *testing.T) {
	cases := map[string]bool{
		"my-show":    true,
		"abc":        true,
		"a1b2c3":     true,
		"ab":         false, // too short
		"-show":      false,
		"show-":      false,
		"My-Show":    false, // uppercase
		"my_show":    false,
		"my show":    false,
		"admin":      false, // reserved
		"live":       false,
		"../etc":     false,
		"x?pass=abc": false,
		"abcdefghijabcdefghijabcdefghijabcdefghij":  true, // 40
		"abcdefghijabcdefghijabcdefghijabcdefghijk": false,
	}
	for slug, want := range cases {
		if got := ValidSlug(slug); got != want {
			t.Errorf("ValidSlug(%q) = %v, want %v", slug, got, want)
		}
	}
}
