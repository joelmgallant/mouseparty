package main

// Suggestions: links viewers put forward from their pages (the suggest box by the chat), for the host
// to add to the list as Party > Add Download Link… would (AddURL) or to dismiss. Nothing is fetched
// until the host adds one, and none is kept: they last as long as the app's memory of them.

import (
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	suggestURLMax = 2048             // bytes of a suggested link
	suggestsMax   = 20               // waiting at once; one more is refused, and its viewer told
	suggestEvery  = 10 * time.Second // from one socket
)

// suggestedLink: the link as it'll be added, and its site for chat; ok only for a plain web link:
// http or https, a host, no login written into it, no longer than suggestURLMax, no control characters
func suggestedLink(s string) (link, site string, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" || len(s) > suggestURLMax || !utf8.ValidString(s) || strings.IndexFunc(s, unicode.IsControl) >= 0 {
		return "", "", false
	}
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
		return "", "", false
	}
	return u.String(), strings.ToLower(u.Hostname()), true
}
