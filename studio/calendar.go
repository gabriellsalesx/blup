package main

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"time"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/blupcloud"
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
	// pushed: the last list of times sent to Blup Cloud (a hash), and when.
	pushMu   sync.Mutex
	pushed   string
	pushedAt time.Time
	failedAt time.Time
	limit    int // the plan's reminder e-mails, from the last answer
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
			application.Get().Event.Emit("calendarOpen", calendarOpenData(r.Response.UserInfo))
		})
		reminders.svc, reminders.ok = svc, true
		loadSent()
		// The app's context ends when it quits, and the checks with it.
		ctx := application.Get().Context()
		go func() {
			tick := time.NewTicker(30 * time.Second)
			defer tick.Stop()
			for {
				s.remind(time.Now())
				select {
				case <-ctx.Done():
					return
				case <-tick.C:
				}
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
	if !s.Prefs().Notifications {
		// Off: nothing is read; the empty list cancels the e-mails
		// waiting in Blup Cloud, as before.
		pushMail(nil, now)
		return
	}
	g, err := config.LoadGlobal()
	if err != nil {
		return
	}
	changed := false
	var mail []blupcloud.Reminder
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
			// The reminder comes at most a week before; the e-mail list
			// goes a month ahead.
			for _, d := range ev.Occurrences(now.AddDate(0, 0, -1), now.AddDate(0, 0, 31)) {
				at, _ := ev.RemindAt(d)
				key := dir + "|" + ev.Path + "|" + d.Format("2006-01-02") + "|" + ev.Remind
				if at.After(now.Add(-12*time.Hour)) && at.Before(now.AddDate(0, 0, 31)) {
					// Only the time and an opaque id go up (an HMAC under a
					// key of this computer): never the title, path or text.
					mail = append(mail, blupcloud.Reminder{ID: app.ReminderID(key), At: at.UnixMilli()})
				}
				if at.After(now) || now.Sub(at) > 12*time.Hour {
					continue
				}
				reminders.mu.Lock()
				done := reminders.sent[key] != 0
				reminders.sent[key] = now.Unix()
				reminders.mu.Unlock()
				if done {
					continue
				}
				changed = true
				opts := notifications.NotificationOptions{
					ID:    fmt.Sprintf("blup-%x", now.UnixNano()),
					Title: ev.Title,
					Body:  s.when(ev, d, now),
					Data:  map[string]any{"dir": dir, "path": ev.Path, "date": d.Format("2006-01-02")},
				}
				// Linux shows no app icon by itself: Blup's goes as the image.
				if icon := notifyIcon(); icon != "" {
					opts.Attachments = []notifications.NotificationAttachment{{ID: "icon", Path: icon}}
				}
				reminders.svc.SendNotification(opts)
			}
		}
	}
	if changed {
		saveSent(now)
	}
	pushMail(mail, now)
}

// pushMail sends the coming reminders' times to Blup Cloud (e-mailed in
// Max and Ultra) when they changed, and every 6 hours anyway. Signed out
// or offline, it tries again later. Notifications off sends none, which
// cancels the ones waiting.
func pushMail(list []blupcloud.Reminder, now time.Time) {
	reminders.pushMu.Lock()
	defer reminders.pushMu.Unlock()
	slices.SortFunc(list, func(a, b blupcloud.Reminder) int { return cmp.Compare(a.At, b.At) })
	if len(list) > 300 {
		list = list[:300]
	}
	h := sha256.New()
	for _, r := range list {
		fmt.Fprintf(h, "%s %d\n", r.ID, r.At)
	}
	sum := hex.EncodeToString(h.Sum(nil))
	// Unchanged, or a plan without e-mails (Free, Pro): asked again only
	// every 6 hours, so nothing goes up for nothing.
	recent := now.Sub(reminders.pushedAt) < 6*time.Hour
	if recent && (sum == reminders.pushed || reminders.limit == 0) || now.Sub(reminders.failedAt) < 5*time.Minute {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	switch u, err := app.PushReminders(ctx, list); {
	case err == nil:
		reminders.pushed, reminders.pushedAt, reminders.limit = sum, now, u.Limit
	case errors.Is(err, blupcloud.ErrSignedOut):
		reminders.pushed = "" // goes as soon as the person signs in
	default:
		reminders.failedAt = now // offline: again in a few minutes
	}
}

// ReminderEmails is the month's use of reminder e-mails (zero signed out
// or on a plan without them).
func (s *Studio) ReminderEmails(ctx context.Context) blupcloud.ReminderUse {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	u, _ := app.ReminderEmails(ctx)
	return u
}

// SetNotifications turns the reminders on or off; off, the ones waiting
// in Blup Cloud are cancelled right away.
func (s *Studio) SetNotifications(on bool) error {
	p := s.Prefs()
	p.Notifications = on
	if err := savePrefs(p); err != nil {
		return err
	}
	reminders.pushMu.Lock()
	reminders.pushed, reminders.failedAt, reminders.pushedAt = "", time.Time{}, time.Time{}
	reminders.pushMu.Unlock()
	if reminders.ok {
		go s.remind(time.Now())
	} else if !on {
		go s.cancelMailUntilDone() // offline: tried again later (cloud.go)
	}
	return nil
}

// notifyIcon is Blup's icon as a file, for Linux notifications ("" on the
// other systems, which show the app's icon themselves).
func notifyIcon() string {
	if runtime.GOOS != "linux" {
		return ""
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	p := filepath.Join(dir, "blup", "notify-icon.png")
	if b, err := os.ReadFile(p); err == nil && len(b) == len(trayIcon) {
		return p
	}
	if os.MkdirAll(filepath.Dir(p), 0o755) != nil || os.WriteFile(p, trayIcon, 0o644) != nil {
		return ""
	}
	return p
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
	reminders.sent = readSent(sentFile())
}

// readSent reads the file of reminders sent. A damaged one (null, not an
// object, a time that is no time) counts as empty or loses just that entry:
// the map is never nil, which would panic when a reminder is written to it.
func readSent(f string) map[string]int64 {
	sent := map[string]int64{}
	if f == "" {
		return sent
	}
	b, err := os.ReadFile(f)
	if err != nil {
		return sent
	}
	var read map[string]int64
	if json.Unmarshal(b, &read) != nil {
		return sent
	}
	for k, at := range read {
		if k != "" && at > 0 {
			sent[k] = at
		}
	}
	return sent
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
	if err == nil && os.MkdirAll(filepath.Dir(f), 0o755) == nil {
		_ = os.WriteFile(f, b, 0o600) // best effort: lost, a reminder may come twice
	}
}

// ServiceStartup runs as the Studio starts: the reminders begin.
func (s *Studio) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	go s.startReminders()
	go s.cloudStartup()
	return nil
}
