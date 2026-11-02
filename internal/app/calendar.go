package app

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/obsidian"
)

// The calendar of a vault of notes. Each event is a note in the calendar's
// folder (Agenda), its day, time and kind in the properties: it syncs and
// opens like any note (Obsidian too), and holds what one writes about it
// (the questions for an interview, what falls in an exam).
//
//	---
//	date: "2026-11-05"
//	time: "14:00"     (none: the whole day)
//	event: exam       (appointment, birthday, exam, interview, deadline, reminder)
//	repeat: yearly    (weekly, monthly, yearly; none: once)
//	remind: 1d        (0, 15m, 1h, 1d, 1w; none: no reminder)
//	---

// Event is one event of the calendar.
type Event struct {
	Path   string `json:"path"`   // the note, relative to the notes folder
	Title  string `json:"title"`  // the note's name
	Date   string `json:"date"`   // 2006-01-02
	Time   string `json:"time"`   // 15:04; "" for the whole day
	Kind   string `json:"kind"`   // see EventKinds
	Repeat string `json:"repeat"` // "", weekly, monthly or yearly
	Remind string `json:"remind"` // "", 0, 15m, 1h, 1d or 1w
}

// EventKinds are the kinds of event, each with its color and icon in the
// Studio.
var EventKinds = []string{"appointment", "birthday", "exam", "interview", "deadline", "reminder"}

var (
	repeats = []string{"", "weekly", "monthly", "yearly"}
	reminds = []string{"", "0", "15m", "1h", "1d", "1w"}
)

// calendarFolders is the calendar's folder in each language; events are
// read from all of them (a vault made in another language keeps working)
// and new ones go to the Studio's language.
var calendarFolders = map[i18n.Lang]string{i18n.PT: "Agenda", i18n.EN: "Calendar", i18n.ES: "Agenda"}

// CalendarFolder is where new events go.
func CalendarFolder() string {
	if f, ok := calendarFolders[i18n.Current()]; ok {
		return f
	}
	return calendarFolders[i18n.EN]
}

// Events lists the vault's events, by day.
func Events(site *config.Site) ([]Event, error) {
	out := []Event{}
	seen := map[string]bool{}
	for _, folder := range calendarFolders {
		if seen[folder] {
			continue
		}
		seen[folder] = true
		dir, err := notesPath(site, folder)
		if err != nil {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".md") || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			n, err := obsidian.ReadNote(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			ev, ok := eventOf(n)
			if !ok {
				continue
			}
			ev.Path = path.Join(folder, e.Name())
			ev.Title = strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
			out = append(out, ev)
		}
	}
	slices.SortFunc(out, func(a, b Event) int { return strings.Compare(a.Date+a.Time+a.Title, b.Date+b.Time+b.Title) })
	return out, nil
}

// eventOf reads an event's properties; a note without a valid date is not
// an event.
func eventOf(n obsidian.Note) (Event, bool) {
	ev := Event{Date: propDay(n.Props["date"]), Time: n.String("time"), Kind: n.String("event"), Repeat: n.String("repeat"), Remind: n.String("remind")}
	if ev.Date == "" {
		return ev, false
	}
	if _, err := time.Parse("15:04", ev.Time); err != nil {
		ev.Time = ""
	}
	if !slices.Contains(EventKinds, ev.Kind) {
		ev.Kind = "reminder"
	}
	if !slices.Contains(repeats, ev.Repeat) {
		ev.Repeat = ""
	}
	if !slices.Contains(reminds, ev.Remind) {
		ev.Remind = ""
	}
	return ev, true
}

// propDay is a date property as 2006-01-02: YAML reads an unquoted date as
// a time, a quoted one as text.
func propDay(v any) string {
	switch d := v.(type) {
	case time.Time:
		return d.Format("2006-01-02")
	case string:
		s := strings.TrimSpace(d)
		if len(s) >= 10 {
			if t, err := time.Parse("2006-01-02", s[:10]); err == nil {
				return t.Format("2006-01-02")
			}
		}
	}
	return ""
}

// SaveEvent makes an event (ev.Path "") or changes one, renaming its note
// when the title changed. It returns the note's path.
func SaveEvent(site *config.Site, ev Event) (string, error) {
	title := safeFileName(ev.Title)
	if title == "" {
		return "", errors.New(i18nInvalidName())
	}
	if _, err := time.Parse("2006-01-02", ev.Date); err != nil {
		return "", errors.New("invalid date")
	}
	if ev.Time != "" {
		if _, err := time.Parse("15:04", ev.Time); err != nil {
			return "", errors.New("invalid time")
		}
	}
	if !slices.Contains(EventKinds, ev.Kind) || !slices.Contains(repeats, ev.Repeat) || !slices.Contains(reminds, ev.Remind) {
		return "", errors.New("invalid event")
	}
	rel := ev.Path
	if rel == "" {
		folder := CalendarFolder()
		dir, err := notesPath(site, folder)
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
		file := title + ".md"
		for i := 2; ; i++ {
			if _, err := os.Stat(filepath.Join(dir, file)); errors.Is(err, os.ErrNotExist) {
				break
			}
			file = fmt.Sprintf("%s %d.md", title, i)
		}
		today := time.Now().Format("2006-01-02")
		if err := os.WriteFile(filepath.Join(dir, file), []byte("---\ncreated: "+today+"\nupdated: "+today+"\n---\n\n"), 0o644); err != nil {
			return "", err
		}
		rel = path.Join(folder, file)
	} else if old := strings.TrimSuffix(path.Base(rel), path.Ext(rel)); old != title {
		moved, err := RenameItem(site, rel, title+".md")
		if err != nil {
			return "", err
		}
		rel = moved
	}
	p, err := notesPath(site, rel)
	if err != nil {
		return "", err
	}
	opt := func(s string) any {
		if s == "" {
			return nil
		}
		return s
	}
	return rel, obsidian.SetProperties(p,
		obsidian.Prop{Key: "date", Value: ev.Date},
		obsidian.Prop{Key: "time", Value: opt(ev.Time)},
		obsidian.Prop{Key: "event", Value: ev.Kind},
		obsidian.Prop{Key: "repeat", Value: opt(ev.Repeat)},
		obsidian.Prop{Key: "remind", Value: opt(ev.Remind)},
		obsidian.Prop{Key: "updated", Value: time.Now().Format("2006-01-02")},
	)
}

// Occurrences are the days in [from, to] an event happens on: its day,
// and the repeats from it (a birthday on February 29 is on the 28th in the
// other years; the 31st of a month, on the month's last day).
func (ev Event) Occurrences(from, to time.Time) []time.Time {
	start, err := time.ParseInLocation("2006-01-02", ev.Date, from.Location())
	if err != nil {
		return nil
	}
	from, to = day(from), day(to)
	var out []time.Time
	add := func(d time.Time) {
		if !d.Before(from) && !d.After(to) {
			out = append(out, d)
		}
	}
	switch ev.Repeat {
	case "weekly":
		d := start
		if from.After(d) {
			d = d.AddDate(0, 0, int(from.Sub(d).Hours()/24/7)*7)
		}
		for ; !d.After(to); d = d.AddDate(0, 0, 7) {
			add(d)
		}
	case "monthly", "yearly":
		step := 1
		if ev.Repeat == "yearly" {
			step = 12
		}
		first := (from.Year()-start.Year())*12 + int(from.Month()) - int(start.Month())
		first = max(0, first-first%step)
		for i := first; ; i += step {
			d := monthDay(start, i)
			if d.After(to) {
				break
			}
			add(d)
		}
	default:
		add(start)
	}
	return out
}

func day(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// monthDay is start's day of the month, months later (the month's last
// day when it is shorter).
func monthDay(start time.Time, months int) time.Time {
	first := time.Date(start.Year(), start.Month()+time.Month(months), 1, 0, 0, 0, 0, start.Location())
	last := first.AddDate(0, 1, -1).Day()
	return first.AddDate(0, 0, min(start.Day(), last)-1)
}

// At is when an occurrence on day starts: its time, or 9:00 for the whole
// day (when a reminder "on the day" comes).
func (ev Event) At(d time.Time) time.Time {
	h, m := 9, 0
	if t, err := time.Parse("15:04", ev.Time); err == nil {
		h, m = t.Hour(), t.Minute()
	}
	return time.Date(d.Year(), d.Month(), d.Day(), h, m, 0, 0, d.Location())
}

// RemindAt is when the reminder of an occurrence on day comes; false when
// the event has none.
func (ev Event) RemindAt(d time.Time) (time.Time, bool) {
	at := ev.At(d)
	switch ev.Remind {
	case "0":
		return at, true
	case "15m":
		return at.Add(-15 * time.Minute), true
	case "1h":
		return at.Add(-time.Hour), true
	case "1d":
		return ev.At(d.AddDate(0, 0, -1)), true
	case "1w":
		return ev.At(d.AddDate(0, 0, -7)), true
	}
	return time.Time{}, false
}
