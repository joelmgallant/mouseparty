package main

// Suggestions: links viewers put forward from their pages (the suggest box by the chat), for the host
// to add to the list as Party > Add Download Link… would (AddURL) or to dismiss. Nothing is fetched
// until the host adds one, and none is kept: they last as long as the app's memory of them.

import (
	"net/url"
	"slices"
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

// Suggestion: one waiting in the window
type Suggestion struct {
	ID   int
	Name string // who suggested it
	URL  string
	Why  string // what AddURL said when Add was pressed and it wouldn't
}

// SuggestionRow: a suggestion as the window draws it
type SuggestionRow struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Link string `json:"link"` // its site and path, for reading
	Why  string `json:"why"`
}

// suggested: a viewer's link, on the app's goroutine: kept unless it's waiting already or in the list,
// and refused (its viewer told, only them) past suggestsMax
func (a *App) suggested(name, link string, viewer int) {
	for _, s := range a.suggestions {
		if s.URL == link {
			return
		}
	}
	for _, it := range a.cfg.Queue {
		if it.URL == link {
			return
		}
	}
	if len(a.suggestions) >= suggestsMax {
		if a.host != nil {
			a.host.SayTo(viewer, "Too many suggestions waiting: try again once some are added or dismissed")
		}
		return
	}
	a.lastSuggestion++
	a.suggestions = append(a.suggestions, Suggestion{ID: a.lastSuggestion, Name: name, URL: link})
	a.notify(name+" suggested a video", link)
}

func (a *App) suggestionAt(id int) int {
	return slices.IndexFunc(a.suggestions, func(s Suggestion) bool { return s.ID == id })
}

func (a *App) suggestionRows() []SuggestionRow {
	rows := []SuggestionRow{}
	for _, s := range a.suggestions {
		link := s.URL
		if u, err := url.Parse(s.URL); err == nil {
			link = u.Host + u.EscapedPath()
		}
		rows = append(rows, SuggestionRow{ID: s.ID, Name: s.Name, Link: link, Why: s.Why})
	}
	return rows
}

// AcceptSuggestion: the window's Add: the link as Party > Add Download Link… adds one, with no login;
// "" or why not, which then stays on its row
func (a *App) AcceptSuggestion(id int) string {
	var s Suggestion
	found := false
	a.wait(func() {
		if i := a.suggestionAt(id); i >= 0 {
			s, found = a.suggestions[i], true
		}
	})
	if !found {
		return ""
	}
	why := a.AddURL(s.URL, "", "", false) // waits on the app's goroutine itself
	a.wait(func() {
		i := a.suggestionAt(id)
		if i < 0 {
			return
		}
		if why != "" {
			a.suggestions[i].Why = why
			return
		}
		a.suggestions = slices.Delete(a.suggestions, i, i+1)
		if a.host != nil {
			a.host.Say(s.Name+"'s suggestion was added", 0)
		}
	})
	return why
}

// DismissSuggestion: the window's Dismiss
func (a *App) DismissSuggestion(id int) {
	a.wait(func() {
		if i := a.suggestionAt(id); i >= 0 {
			name := a.suggestions[i].Name
			a.suggestions = slices.Delete(a.suggestions, i, i+1)
			if a.host != nil {
				a.host.Say(name+"'s suggestion was dismissed", 0)
			}
		}
	})
}
