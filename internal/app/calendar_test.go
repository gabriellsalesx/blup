package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gabriellsalesx/blup/internal/i18n"
)

func TestEvents(t *testing.T) {
	i18n.Set(i18n.PT)
	t.Cleanup(func() { i18n.Set(i18n.EN) })
	site := settingsSite(t)
	root := site.ContentDir()
	rel, err := SaveEvent(site, Event{Title: "Prova de cálculo", Date: "2026-11-05", Time: "14:00", Kind: "exam", Remind: "1d"})
	if err != nil || rel != "Agenda/Prova de cálculo.md" {
		t.Fatalf("save = %q, %v", rel, err)
	}
	// One made in another language, with an unquoted date, and a note that
	// is not an event.
	write(t, root, "Calendar/Ana.md", "---\ndate: 2000-02-29\nevent: birthday\nrepeat: yearly\n---\n")
	write(t, root, "Agenda/Ideias.md", "sem data")
	evs, err := Events(site)
	if err != nil || len(evs) != 2 {
		t.Fatalf("events = %+v, %v", evs, err)
	}
	if evs[0].Title != "Ana" || evs[0].Date != "2000-02-29" || evs[0].Repeat != "yearly" || evs[1].Time != "14:00" || evs[1].Remind != "1d" {
		t.Fatalf("events = %+v", evs)
	}

	// Renamed with its title; the text written in it stays.
	write(t, root, rel, strings.Replace(string(must(os.ReadFile(filepath.Join(root, rel)))), "---\n\n", "---\n\nCapítulos 3 e 4\n", 1))
	ev := evs[1]
	ev.Title, ev.Time, ev.Remind = "Prova final", "", ""
	rel, err = SaveEvent(site, ev)
	if err != nil || rel != "Agenda/Prova final.md" {
		t.Fatalf("rename = %q, %v", rel, err)
	}
	s := string(must(os.ReadFile(filepath.Join(root, rel))))
	if !strings.Contains(s, "Capítulos 3 e 4") || strings.Contains(s, "time:") || strings.Contains(s, "remind:") || !strings.Contains(s, "event: exam") {
		t.Fatalf("note = %q", s)
	}
	if _, err := SaveEvent(site, Event{Title: "x", Date: "amanhã", Kind: "exam"}); err == nil {
		t.Fatal("a bad date was saved")
	}
}

func TestOccurrences(t *testing.T) {
	d := func(s string) time.Time { return must(time.ParseInLocation("2006-01-02", s, time.Local)) }
	days := func(ev Event, from, to string) string {
		var out []string
		for _, x := range ev.Occurrences(d(from), d(to)) {
			out = append(out, x.Format("2006-01-02"))
		}
		return strings.Join(out, " ")
	}
	cases := []struct {
		ev       Event
		from, to string
		want     string
	}{
		{Event{Date: "2026-11-05"}, "2026-11-01", "2026-11-30", "2026-11-05"},
		{Event{Date: "2026-11-05"}, "2026-12-01", "2026-12-31", ""},
		{Event{Date: "2000-02-29", Repeat: "yearly"}, "2027-02-01", "2028-03-01", "2027-02-28 2028-02-29"},
		{Event{Date: "2026-01-31", Repeat: "monthly"}, "2026-02-01", "2026-04-30", "2026-02-28 2026-03-31 2026-04-30"},
		{Event{Date: "2026-11-02", Repeat: "weekly"}, "2026-11-10", "2026-11-24", "2026-11-16 2026-11-23"},
		{Event{Date: "2026-12-01", Repeat: "weekly"}, "2026-11-01", "2026-11-30", ""},
	}
	for _, c := range cases {
		if got := days(c.ev, c.from, c.to); got != c.want {
			t.Errorf("%+v in %s..%s = %q, want %q", c.ev, c.from, c.to, got, c.want)
		}
	}

	ev := Event{Date: "2026-11-05", Time: "14:00", Remind: "1h"}
	if at, ok := ev.RemindAt(d("2026-11-05")); !ok || at.Format("2006-01-02 15:04") != "2026-11-05 13:00" {
		t.Fatalf("1h before = %v", at)
	}
	ev = Event{Date: "2026-11-05", Remind: "1d"}
	if at, _ := ev.RemindAt(d("2026-11-05")); at.Format("2006-01-02 15:04") != "2026-11-04 09:00" {
		t.Fatalf("a day before (whole day) = %v", at)
	}
	if _, ok := (Event{Date: "2026-11-05"}).RemindAt(d("2026-11-05")); ok {
		t.Fatal("no reminder set, yet one came")
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
