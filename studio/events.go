package main

import "github.com/wailsapp/wails/v3/pkg/application"

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
