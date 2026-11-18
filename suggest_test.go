package main

import (
	"fmt"
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
		"http://example.com:8080/a":                            "example.com",
		"http://[::1]:8080/a":                                  "::1",
		"http://[::1]/a":                                       "::1",
	} {
		link, site, ok := suggestedLink(in)
		if !ok || site != want || link != strings.ToLower(strings.TrimSpace(strings.Replace(in, "HTTPS", "https", 1))) {
			t.Errorf("%q: %q %q %v", in, link, site, ok)
		}
	}
	if link, _, ok := suggestedLink("http://example.com:8080/a"); !ok || link != "http://example.com:8080/a" {
		t.Errorf("a port was lost: %q", link)
	}
	if link, site, ok := suggestedLink("https://\u0430rchive.org/x"); !ok || !strings.HasPrefix(site, "xn--") || !strings.Contains(link, "xn--") {
		t.Errorf("a look-alike host should come out as punycode: %q %q %v", link, site, ok)
	}
	if link, site, ok := suggestedLink("https://exam\u202eple.com/x"); strings.Contains(link, "\u202e") || strings.Contains(site, "\u202e") {
		t.Errorf("a bidi override came through: %q %q %v", link, site, ok)
	}
	if link, _, ok := suggestedLink("https://example.com/a.mkv#t=10"); !ok || strings.Contains(link, "#") {
		t.Errorf("the fragment stayed: %q", link)
	}
	exact := "https://example.com/" + strings.Repeat("a", suggestURLMax-len("https://example.com/"))
	if _, _, ok := suggestedLink(exact); !ok || len(exact) != suggestURLMax {
		t.Errorf("a link of exactly %d bytes (%d) was refused", suggestURLMax, len(exact))
	}
	if _, _, ok := suggestedLink(exact + "a"); ok {
		t.Error("a link of one byte over was taken")
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

// Suggestions in the window: one row each, a link already waiting or in the list not twice, no more
// than suggestsMax; Add is AddURL's (a row downloading), and what it refuses stays with its reason;
// Dismiss takes one away; an id gone is nothing
func TestSuggestions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a := NewApp(standby)
	a.hostSetup = func(h *Host) { // never the default ports (CLAUDE.md)
		h.HTTPPort, h.ICEPort, h.MapPorts, h.CleanLeftovers, h.Cloudflared = freePort(), freeICEPort(), false, false, ""
	}
	go a.run()
	defer a.shutdown(nil)
	suggest := func(name, link string) { a.wait(func() { a.suggested(name, link, 1) }) }

	suggest("Ann", "http://127.0.0.1:1/one.mkv")
	suggest("Bob", "http://127.0.0.1:1/one.mkv") // waiting already
	v := a.GetView()
	if len(v.Suggestions) != 1 || v.Suggestions[0].Name != "Ann" || v.Suggestions[0].Link != "127.0.0.1:1/one.mkv" {
		t.Fatalf("%+v", v.Suggestions)
	}
	for i := 0; i < suggestsMax+5; i++ {
		suggest("Cat", fmt.Sprintf("http://127.0.0.1:1/%d.mkv", i))
	}
	if n := len(a.GetView().Suggestions); n != suggestsMax {
		t.Fatal(n, "waiting")
	}

	first := a.GetView().Suggestions[0].ID
	if why := a.AcceptSuggestion(first); why != "" {
		t.Fatal(why)
	}
	v = a.GetView()
	if len(v.Suggestions) != suggestsMax-1 || v.Suggestions[0].ID == first {
		t.Fatalf("Add left it: %+v", v.Suggestions[0])
	}
	found := false
	for _, r := range v.Rows {
		found = found || r.Icon == "download"
	}
	if !found {
		t.Fatalf("Add made no download row: %+v", v.Rows)
	}
	suggest("Dan", "http://127.0.0.1:1/one.mkv") // in the list now
	if n := len(a.GetView().Suggestions); n != suggestsMax-1 {
		t.Fatal("a link in the list was suggested again:", n)
	}

	var id int
	a.wait(func() { a.locked, id = true, a.suggestions[0].ID }) // a stand-in to make AddURL refuse (locked); the view shows nothing then, so read the state
	if why := a.AcceptSuggestion(id); why == "" {
		t.Fatal("added while sealed")
	}
	a.wait(func() {
		if a.suggestions[0].ID != id || a.suggestions[0].Why == "" {
			t.Errorf("a refused one should stay with its reason: %+v", a.suggestions[0])
		}
		a.locked = false
	})

	a.DismissSuggestion(id)
	a.DismissSuggestion(id) // gone already
	if why := a.AcceptSuggestion(id); why != "" {
		t.Fatal("an id gone:", why)
	}
	if n := len(a.GetView().Suggestions); n != suggestsMax-2 {
		t.Fatal(n)
	}
}
