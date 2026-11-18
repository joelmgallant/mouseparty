package main

import (
	"strings"
	"testing"

	"github.com/ftrain/mouseparty/apps"
)

// What the site says a viewer suggested is checked again by the app, as a stranger's would be
func TestMacCallsSuggest(t *testing.T) {
	var got []string
	h := &Host{urls: map[string]string{}, Suggested: func(name, link string, viewer int) {
		got = append(got, name+"|"+link+"|"+string(rune('0'+viewer)))
	}}
	m := &MacCalls{h: h}
	if err := m.Event(SiteEvent{Kind: "suggest", Text: "Ann\nhttps://archive.org/details/a", N: 3}, &None{}); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		"Ann", "\nhttps://archive.org/details/a", "Ann\nftp://x/y", "Ann\nhttps://u:p@x/y",
		strings.Repeat("n", nameMax+1) + "\nhttps://x/y", "A\x07nn\nhttps://x/y",
	} {
		if m.Event(SiteEvent{Kind: "suggest", Text: bad, N: 1}, &None{}) == nil {
			t.Errorf("%q taken", bad)
		}
	}
	if len(got) != 1 || got[0] != "Ann|https://archive.org/details/a|3" {
		t.Fatal(got)
	}
}

// The app's say and sayto, told to the site, reach its chat
func TestSiteToldToSay(t *testing.T) {
	var lines []string
	chat := &Chat{hello: func(apps.Viewer) map[string]any { return nil }, conns: map[*chatConn]bool{}}
	c := &SiteCalls{s: &siteSide{h: &Host{urls: map[string]string{}, chat: chat}}}
	if err := c.Tell(SiteTell{Do: "say", Text: "Ann's suggestion was added"}, &None{}); err != nil {
		t.Fatal(err)
	}
	if err := c.Tell(SiteTell{Do: "sayto", Viewer: 99, Text: "nobody's"}, &None{}); err != nil { // no such socket: nothing
		t.Fatal(err)
	}
	for _, m := range chat.history {
		lines = append(lines, m["text"].(string))
	}
	if strings.Join(lines, ",") != "Ann's suggestion was added" {
		t.Fatal(lines)
	}
}
