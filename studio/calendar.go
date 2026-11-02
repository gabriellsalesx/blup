package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

// The calendar of a vault of notes (app/calendar.go): its events are
// notes, and their reminders come as the computer's notifications while
// the Studio is open (or in the tray), from every vault Blup knows.

// CalendarEvents lists the open vault's events.
func (s *Studio) CalendarEvents() ([]app.Event, error) {
	site, err := s.current()
	if err != nil {
		return nil, err
	}
	return app.Events(site)
}

// SaveEvent makes or changes an event and returns its note's path.
func (s *Studio) SaveEvent(ev app.Event) (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	rel, err := app.SaveEvent(site, ev)
	if err == nil && ev.Remind != "" {
		go askNotifications()
	}
	return rel, err
}

// reminders sends the notifications; it starts with the window.
var reminders struct {
	once sync.Once
	svc  *notifications.NotificationService
	ok   bool
	mu   sync.Mutex
	sent map[string]int64 // event path|day|remind -> when it was sent (unix)
}

// startReminders checks the events every half minute. A reminder missed
// while the Studio was closed still comes when it opens, up to 12 hours
// late (after that it no longer helps).
func (s *Studio) startReminders() {
	reminders.once.Do(func() {
		svc := notifications.New()
		if err := svc.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
			return // no notifications on this computer
		}
		svc.OnNotificationResponse(func(r notifications.NotificationResult) {
			if r.Error != nil {
				return
			}
			showWindow()
			application.Get().Event.Emit("calendarOpen", r.Response.UserInfo)
		})
		reminders.svc, reminders.ok = svc, true
		loadSent()
		go func() {
			for {
				s.remind(time.Now())
				time.Sleep(30 * time.Second)
			}
		}()
	})
}

// askNotifications asks the system to allow them (macOS asks once; the
// others don't ask).
func askNotifications() {
	if reminders.ok {
		reminders.svc.RequestNotificationAuthorization()
	}
}

func (s *Studio) remind(now time.Time) {
	g, err := config.LoadGlobal()
	if err != nil {
		return
	}
	changed := false
	for _, dir := range g.ExistingSites() {
		site, err := config.LoadSite(dir)
		if err != nil || site.Kind != "notes" {
			continue
		}
		evs, _ := app.Events(site)
		for _, ev := range evs {
			if ev.Remind == "" {
				continue
			}
			// The reminder comes at most a week before.
			for _, d := range ev.Occurrences(now.AddDate(0, 0, -1), now.AddDate(0, 0, 8)) {
				at, _ := ev.RemindAt(d)
				if at.After(now) || now.Sub(at) > 12*time.Hour {
					continue
				}
				key := dir + "|" + ev.Path + "|" + d.Format("2006-01-02") + "|" + ev.Remind
				reminders.mu.Lock()
				done := reminders.sent[key] != 0
				reminders.sent[key] = now.Unix()
				reminders.mu.Unlock()
				if done {
					continue
				}
				changed = true
				reminders.svc.SendNotification(notifications.NotificationOptions{
					ID:    fmt.Sprintf("blup-%x", now.UnixNano()),
					Title: ev.Title,
					Body:  s.when(ev, d, now),
					Data:  map[string]any{"dir": dir, "path": ev.Path, "date": d.Format("2006-01-02")},
				})
			}
		}
	}
	if changed {
		saveSent(now)
	}
}

// when says when an event is, from now: "Today at 14:00", "Tomorrow",
// "In 7 days (Thu, 12/11)".
func (s *Studio) when(ev app.Event, d, now time.Time) string {
	w := map[string][5]string{
		"pt": {"Hoje", "Amanhã", "Em %d dias", "às %s", "o dia todo"},
		"es": {"Hoy", "Mañana", "En %d días", "a las %s", "todo el día"},
		"en": {"Today", "Tomorrow", "In %d days", "at %s", "all day"},
	}[s.lang]
	if w[0] == "" {
		w = [5]string{"Today", "Tomorrow", "In %d days", "at %s", "all day"}
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	days := int(d.Sub(today).Hours()/24 + 0.5)
	var out string
	switch days {
	case 0:
		out = w[0]
	case 1:
		out = w[1]
	default:
		out = fmt.Sprintf(w[2], days)
		if s.lang == "en" {
			out += d.Format(" (Jan 2)")
		} else {
			out += d.Format(" (02/01)")
		}
	}
	if ev.Time != "" {
		return out + " " + fmt.Sprintf(w[3], ev.Time)
	}
	return out + ", " + w[4]
}

// The reminders already sent, kept so a restart doesn't repeat them.
func sentFile() string {
	dir, err := config.Dir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "reminders-sent.json")
}

func loadSent() {
	reminders.sent = map[string]int64{}
	if f := sentFile(); f != "" {
		if b, err := os.ReadFile(f); err == nil {
			json.Unmarshal(b, &reminders.sent)
		}
	}
}

// saveSent writes them, forgetting the ones over a month old.
func saveSent(now time.Time) {
	f := sentFile()
	if f == "" {
		return
	}
	reminders.mu.Lock()
	for k, at := range reminders.sent {
		if now.Unix()-at > 31*24*3600 {
			delete(reminders.sent, k)
		}
	}
	b, err := json.Marshal(reminders.sent)
	reminders.mu.Unlock()
	if err == nil {
		os.MkdirAll(filepath.Dir(f), 0o755)
		os.WriteFile(f, b, 0o600)
	}
}

// ServiceStartup runs as the Studio starts: the reminders begin.
func (s *Studio) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	go s.startReminders()
	return nil
}
