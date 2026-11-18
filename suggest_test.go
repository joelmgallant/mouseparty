package main

import (
	"strings"
	"testing"
)

// A suggestion is a plain web link and nothing else: what a stranger types goes to the host's
// window and, on Add, to AddURL
func TestSuggestedLink(t *testing.T) {
	for in, want := range map[string]string{
		"https://archive.org/details/night_of_the_living_dead": "archive.org",
		"  http://example.com/a.mkv  ":                         "example.com",
		"HTTPS://Example.com/x":                                "example.com",
	} {
		link, site, ok := suggestedLink(in)
		if !ok || site != want || link != strings.TrimSpace(strings.Replace(in, "HTTPS", "https", 1)) {
			t.Errorf("%q: %q %q %v", in, link, site, ok)
		}
	}
	for _, bad := range []string{
		"", "   ", "archive.org/details/x", "ftp://example.com/a.mkv", "file:///etc/passwd",
		"javascript:alert(1)", "https://", "https://user:pass@example.com/a.mkv", "https://user@example.com/",
		"https://example.com/a\nb", "https://example.com/\x00", "https://example.com/" + strings.Repeat("a", suggestURLMax),
		"https://example.com/\xff",
	} {
		if link, _, ok := suggestedLink(bad); ok {
			t.Errorf("%q taken as %q", bad, link)
		}
	}
}
