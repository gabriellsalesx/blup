package main

import (
	"os"
	"strings"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/obsidian"
	"github.com/wailsapp/wails/v3/pkg/application"
	"go.yaml.in/yaml/v3"
)

// A note in the editor: its properties as form fields, its text in the
// editor. Properties the forms don't know stay as they are.

// NoteData is a note split for the editor.
type NoteData struct {
	Props map[string]any `json:"props"`
	Body  string         `json:"body"`
	Kind  string         `json:"kind"` // section of the note (Bento): docs, roadmap...
}

// PropEdit is a property changed in the form; a nil, empty or false value
// removes it.
type PropEdit struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

// ReadNote returns a note's properties and text.
func (s *Studio) ReadNote(rel string) (NoteData, error) {
	p, err := s.notePath(rel)
	if err != nil {
		return NoteData{}, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return NoteData{}, err
	}
	fm, body := splitFM(string(b))
	props := map[string]any{}
	if fm != "" {
		_ = yaml.Unmarshal([]byte(fm), &props)
	}
	return NoteData{Props: jsonSafe(props).(map[string]any), Body: body, Kind: s.kindOf(rel)}, nil
}

// SaveNote saves a note's text and the properties changed in the form.
func (s *Studio) SaveNote(rel, body string, edits []PropEdit) error {
	p, err := s.notePath(rel)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(strings.ToLower(p), ".md") {
		return os.ErrInvalid
	}
	if _, err := os.Stat(p); err == nil {
		if err := obsidian.SetBody(p, body); err != nil {
			return err
		}
	} else if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		return err
	}
	props := make([]obsidian.Prop, 0, len(edits))
	for _, e := range edits {
		props = append(props, obsidian.Prop{Key: e.Key, Value: clean(e.Value)})
	}
	if len(props) == 0 {
		return nil
	}
	return obsidian.SetProperties(p, props...)
}

// clean turns form values into property values; empty ones become nil (the
// property is removed).
func clean(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case string:
		if strings.TrimSpace(x) == "" {
			return nil
		}
		return strings.TrimSpace(x)
	case bool:
		if !x {
			return nil
		}
		return true
	case float64:
		if x == 0 {
			return nil
		}
		if x == float64(int64(x)) {
			return int64(x)
		}
		return x
	case []any:
		var out []string
		for _, i := range x {
			if s, ok := i.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	}
	return v
}

// jsonSafe makes YAML values JSON-friendly (dates become "2006-01-02").
func jsonSafe(v any) any {
	switch x := v.(type) {
	case map[string]any:
		for k, i := range x {
			x[k] = jsonSafe(i)
		}
		return x
	case []any:
		for k, i := range x {
			x[k] = jsonSafe(i)
		}
		return x
	case interface{ Format(string) string }:
		return x.Format("2006-01-02")
	}
	return v
}

func splitFM(s string) (fm, body string) {
	norm := strings.ReplaceAll(strings.TrimPrefix(s, "\ufeff"), "\r\n", "\n")
	if !strings.HasPrefix(norm, "---\n") {
		return "", s
	}
	rest := norm[4:]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		if strings.HasPrefix(rest, "---\n") {
			return "", rest[4:]
		}
		return "", s
	}
	return rest[:end+1], rest[end+5:]
}

// kindOf is the Bento section of a note, from its top folder.
func (s *Studio) kindOf(rel string) string {
	site, err := s.current()
	if err != nil || site.Model != "bento" {
		return ""
	}
	top, _, _ := strings.Cut(rel, "/")
	for kind, folder := range obsidian.Sections(site.ContentDir()) {
		if strings.EqualFold(folder, top) {
			return kind
		}
	}
	return ""
}

// PickCover asks for an image, copies it into the site and returns the
// link for the "image" property ("" when the user cancels).
func (s *Studio) PickCover() (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	path, err := application.Get().Dialog.OpenFile().SetTitle("Blup Studio").
		AddFilter("Images", "*.png;*.jpg;*.jpeg;*.webp;*.gif;*.svg").PromptForSingleSelection()
	if err != nil || path == "" {
		return "", err
	}
	return app.AddImage(site, path)
}

// LinkedImage is the image of a property ("[[cover.jpg]]") as a data: URL.
func (s *Studio) LinkedImage(link string) (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	f := app.AttachmentFile(site, link)
	if f == "" {
		return "", nil
	}
	return s.ImageData(f)
}
