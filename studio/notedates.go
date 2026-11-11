package main

import (
	"io"
	"os"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// NoteDates is what the file list shows while the mouse rests on a note:
// when it was last changed and when it was made, as "2006-01-02 15:04"
// (Created only a date when the note's "created" property gives a day).
// Created is "" when neither the note nor the system says.
type NoteDates struct {
	Modified string `json:"modified"`
	Created  string `json:"created"`
}

const shownTime = "2006-01-02 15:04"

// NoteDates reads a note's dates: the file's change time, and its
// "created" property (written by Blup, the same on every device), else the
// file's creation time where the system keeps one.
func (s *Studio) NoteDates(rel string) (NoteDates, error) {
	p, err := s.notePath(rel)
	if err != nil {
		return NoteDates{}, err
	}
	f, err := os.Open(p)
	if err != nil {
		return NoteDates{}, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return NoteDates{}, err
	}
	d := NoteDates{Modified: st.ModTime().Local().Format(shownTime)}
	// The properties are at the top: the first 16 KB hold them.
	head, _ := io.ReadAll(io.LimitReader(f, 16<<10))
	if c := createdProp(string(head)); c != "" {
		d.Created = c
	} else if b := birthTime(p, st); !b.IsZero() {
		d.Created = b.Local().Format(shownTime)
	}
	return d, nil
}

// createdProp is the "created" property of a note's text, as a date or a
// date and time ("" when there is none).
func createdProp(text string) string {
	fm, _ := splitFM(text)
	if fm == "" {
		return ""
	}
	var props struct {
		Created any `yaml:"created"`
	}
	if yaml.Unmarshal([]byte(fm), &props) != nil {
		return ""
	}
	switch v := props.Created.(type) {
	case time.Time:
		if v.Hour() == 0 && v.Minute() == 0 && v.Second() == 0 {
			return v.Format("2006-01-02")
		}
		return v.Local().Format(shownTime)
	case string:
		v = strings.TrimSpace(v)
		for _, layout := range []string{time.RFC3339, "2006-01-02T15:04", "2006-01-02 15:04", "2006-01-02"} {
			if t, err := time.Parse(layout, v); err == nil {
				if layout == "2006-01-02" {
					return t.Format("2006-01-02")
				}
				return t.Format(shownTime)
			}
		}
	}
	return ""
}
