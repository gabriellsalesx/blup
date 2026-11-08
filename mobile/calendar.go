package main

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/config"
)

// CalendarService is the Studio's calendar: each event is a note in the
// Agenda folder, read and written by the same code as on the computer, so
// it travels with the notes.
type CalendarService struct{}

// site is the notes of this phone as the Blup packages see them. It only
// lives here: a phone has no site to publish.
func site() (*config.Site, error) {
	root, err := currentRoot()
	if err != nil {
		return nil, err
	}
	return siteAt(root)
}

// siteAt is a vault of this phone, by its folder.
func siteAt(root string) (*config.Site, error) {
	lang := appLang() // the Blup packages speak it too (lang.go)
	notes, err := notesAt(root)
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(root, "site")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	s := config.NewSite(dir)
	s.Title, s.Kind, s.Lang, s.Vault, s.Folder = "Blup no celular", "notes", lang, notes, ""
	return s, nil
}

// Events lists the events.
func (c *CalendarService) Events() ([]app.Event, error) {
	s, err := site()
	if err != nil {
		return nil, err
	}
	return app.Events(s)
}

// Save makes or changes an event and returns its note.
func (c *CalendarService) Save(ev app.Event) (string, error) {
	s, err := site()
	if err != nil {
		return "", err
	}
	rel, err := app.SaveEvent(s, ev)
	if err != nil {
		return "", err
	}
	return rel, writeReminders()
}

// Delete sends a note or an event to the trash and returns where, for undoing.
func (c *CalendarService) Delete(rel string) (string, error) {
	s, err := site()
	if err != nil {
		return "", err
	}
	trashed, err := app.DeleteItem(s, rel)
	if err != nil {
		return "", err
	}
	return trashed, writeReminders()
}

// Restore brings a deleted note or event back.
func (c *CalendarService) Restore(trashed string) error {
	s, err := site()
	if err != nil {
		return err
	}
	if err := app.RestoreSection(s, trashed); err != nil {
		return err
	}
	return writeReminders()
}

// DailyNote opens today's note, with the time at its end to write under.
func (c *CalendarService) DailyNote() (string, error) {
	s, err := site()
	if err != nil {
		return "", err
	}
	daily := dailyFolders[appLang()]
	return app.DailyNote(s, daily[0], daily[1], time.Now())
}

// reminder is one alarm for the system, read by ReminderScheduler.java.
type reminder struct {
	ID    int32  `json:"id"`
	At    int64  `json:"at"`
	Title string `json:"title"`
	Body  string `json:"body"`
	Alarm bool   `json:"alarm,omitempty"` // rings until stopped
	// The system's texts in the app's language: the alarm's buttons and
	// the names of the two notification channels in Android's settings.
	Stop    string `json:"stop"`
	Snooze  string `json:"snooze"`
	Channel string `json:"channel"`
	Alarms  string `json:"alarms"`
}

// The system keeps at most 500 alarms per app; the nearest ones are enough,
// as the list is written again every time the app runs.
const maxReminders = 100

// writeReminders lists the coming reminders in a file. Android fires them
// by itself, with the app minimized or closed, which is what the tray does
// on the computer.
func writeReminders() error {
	root, err := dataDir()
	if err != nil {
		return err
	}
	spaceLock.Lock()
	f, err := readSpaces()
	spaceLock.Unlock()
	if err != nil {
		return err
	}
	now := time.Now()
	list := []reminder{}
	// Every vault of the phone: an event warns whichever one is open.
	for _, e := range allSpaces(f) {
		home, err := spaceRoot(e.ID)
		if err != nil {
			return err
		}
		s, err := siteAt(home)
		if err != nil {
			return err
		}
		events, err := app.Events(s)
		if err != nil {
			return err
		}
		for _, ev := range events {
			if ev.Remind == "" && !ev.Alarm {
				continue
			}
			for _, d := range ev.Occurrences(now.AddDate(0, 0, -1), now.AddDate(0, 0, 31)) {
				add := func(at time.Time, kind string, alarm bool) {
					if !at.After(now) {
						return
					}
					key := fnv.New32a()
					fmt.Fprint(key, e.ID, "|", ev.Path, "|", d.Format("2006-01-02"), "|", kind)
					list = append(list, reminder{ID: int32(key.Sum32() >> 1), At: at.UnixMilli(), Title: ev.Title, Body: whenText(ev, d, at), Alarm: alarm,
						Stop: tr("remStop"), Snooze: tr("remSnooze"), Channel: tr("remChannel"), Alarms: tr("remAlarms")})
				}
				// The alarm rings at the event's time; a notice at that
				// same time would only repeat it.
				if ev.Alarm {
					add(ev.At(d), "alarm", true)
				}
				if at, ok := ev.RemindAt(d); ok && !(ev.Alarm && ev.Remind == "0") {
					add(at, ev.Remind, false)
				}
			}
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].At < list[j].At })
	list = list[:min(len(list), maxReminders)]
	if st, err := readSettings(); err != nil {
		return err
	} else if st.Quiet {
		list = list[:0] // the phone cancels what no longer is on the list
	}
	data, err := json.Marshal(list)
	if err != nil {
		return err
	}
	file := filepath.Join(root, "reminders.json")
	return writeSafe(file, data)
}

// whenText says when the event is, seen from the moment the reminder comes.
func whenText(ev app.Event, d, at time.Time) string {
	today := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, at.Location())
	var out string
	switch days := int(d.Sub(today).Hours()/24 + 0.5); days {
	case 0:
		out = tr("today")
	case 1:
		out = tr("tomorrow")
	default:
		out = tr("inDays", days, d.Format(tr("dateFormat")))
	}
	if ev.Time != "" {
		return tr("atTime", out, ev.Time)
	}
	return tr("allDay", out)
}
