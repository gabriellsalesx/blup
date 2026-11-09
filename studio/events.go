package main

import (
	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// The events the window hears that no other file registers. A registered
// event gets its data type in the bindings (eventdata.d.ts), so the
// window's Events.On callbacks are typed; Wails also checks, on Emit,
// that the data is of exactly this type (a mismatch cancels the event).
func init() {
	// The site's preview rebuilt a page: the window reloads it. No data
	// (an empty text).
	application.RegisterEvent[string]("previewChanged")
	// A click on a calendar reminder: the window opens that day.
	application.RegisterEvent[CalendarOpen]("calendarOpen")
	// The Blup Cloud account changed: signed in, signed out, the notes
	// password made or typed, or the plan changed in the browser. The
	// account as it is now (SignedIn false when signed out).
	application.RegisterEvent[app.BlupAccount]("blupAccount")
	// The open site's settings changed outside its screens (they came
	// with a sync from another computer, or hugo.toml was edited).
	application.RegisterEvent[SiteSettingsChanged]("siteSettingsChanged")
	// A GitHub, Cloudflare or OpenRouter key was connected or forgotten:
	// the screens that show the connections read them again.
	application.RegisterEvent[ConnectionsChanged]("connectionsChanged")
	// Notes changed on disk (an agent, another editor, the sync).
	application.RegisterEvent[NotesChanged]("notesChanged")
}

// SiteSettingsChanged is heard as "siteSettingsChanged": the site's folder.
type SiteSettingsChanged struct {
	Dir string `json:"dir"`
}

// ConnectionsChanged is heard as "connectionsChanged": "github",
// "cloudflare" or "openrouter".
type ConnectionsChanged struct {
	Account string `json:"account"`
}

// NotesChanged is heard as "notesChanged": the site's folder and the
// notes that changed, relative to its notes folder, with "/".
type NotesChanged struct {
	Dir   string   `json:"dir"`
	Paths []string `json:"paths"`
}

// emit sends an event to the window (nothing when there is no app, as in
// the tests).
func emit[T any](name string, data T) {
	if a := application.Get(); a != nil {
		a.Event.Emit(name, data)
	}
}

// CalendarOpen is the reminder the person clicked, heard by the window as
// "calendarOpen": the site's folder, the event's note and the day
// (YYYY-MM-DD).
type CalendarOpen struct {
	Dir  string `json:"dir"`
	Path string `json:"path"`
	Date string `json:"date"`
}

// calendarOpenData reads a notification's data (what the reminder carried,
// back from the system) into a CalendarOpen. Anything not text is left
// empty: the system hands back what it kept, which may not be what went.
func calendarOpenData(info map[string]any) CalendarOpen {
	text := func(k string) string {
		s, _ := info[k].(string)
		return s
	}
	return CalendarOpen{Dir: text("dir"), Path: text("path"), Date: text("date")}
}
