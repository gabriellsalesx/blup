package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	// A vault's note that is a record of a database: the database and its
	// columns (its fields in Properties).
	Table    string        `json:"table,omitempty"`
	Database *app.DBSchema `json:"database,omitempty"`
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
	data := NoteData{Props: jsonSafe(props).(map[string]any), Body: body, Kind: s.kindOf(rel)}
	if site, err := s.current(); err == nil && site.IsNotes() {
		data.Table, data.Database = app.DatabaseOf(site, rel)
	}
	return data, nil
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
		props = append(props, obsidian.Prop{Key: e.Key, Value: s.propValue(e.Value)})
	}
	if len(props) == 0 {
		return nil
	}
	return obsidian.SetProperties(p, props...)
}

// propValue is clean, but in a vault of notes a number 0 is a value (a
// database's number column), not "none" as in a site's forms.
func (s *Studio) propValue(v any) any {
	if f, ok := v.(float64); ok && f == 0 {
		if site, err := s.current(); err == nil && site.IsNotes() {
			return int64(0)
		}
	}
	return clean(v)
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

// NoteImage is an image a note shows (![](src), relative to the note, or
// ![[name.png]]) as a data: URL, for the editor. "" when it isn't found.
func (s *Studio) NoteImage(note, src string) (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(src, "[[") {
		f := app.AttachmentFile(site, src)
		if f == "" {
			return "", nil
		}
		return s.ImageData(f)
	}
	if u, err := url.PathUnescape(src); err == nil {
		src = u
	}
	f := filepath.Join(site.ContentDir(), filepath.FromSlash(path.Join(path.Dir("/"+note), src)))
	if _, err := os.Stat(f); err != nil {
		return "", nil
	}
	return s.ImageData(f)
}

// CreateNote makes a note named name in folder (relative to the notes
// folder, "" for the top), with its title, and returns its path. A taken
// name gets a number.
func (s *Studio) CreateNote(folder, name string) (string, error) {
	return s.create(folder, name, false)
}

// CreateFolder makes a folder in folder and returns its path.
func (s *Studio) CreateFolder(folder, name string) (string, error) {
	return s.create(folder, name, true)
}

func (s *Studio) create(folder, name string, dir bool) (string, error) {
	if dir {
		return s.createFile(folder, name, "", "")
	}
	return s.createFile(folder, name, ".md", "")
}

// createFile makes a file named name+ext in folder with text (a note gets
// its title), or a folder when ext is "", and returns its path. A taken
// name gets a number.
func (s *Studio) createFile(folder, name, ext, text string) (string, error) {
	name = strings.TrimSpace(strings.Map(func(r rune) rune {
		if strings.ContainsRune(`/\:*?"<>|#^[]`, r) {
			return -1
		}
		return r
	}, name))
	name = boardName(strings.TrimSuffix(name, ".md"))
	if name == "" || strings.HasPrefix(name, ".") {
		return "", errors.New("invalid name")
	}
	for i := 1; ; i++ {
		base := name
		if i > 1 {
			base = fmt.Sprintf("%s %d", name, i)
		}
		rel := strings.TrimPrefix(folder+"/"+base, "/") + ext
		p, err := s.notePath(rel)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(p); err == nil {
			continue
		}
		if ext == "" {
			return rel, os.MkdirAll(p, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return "", err
		}
		if ext == ".md" {
			text = "---\ntitle: " + strconv.Quote(base) + "\n---\n\n"
			// A vault's note has no page title: it starts with its dates
			// (Properties), the name being the file's.
			if site, err := s.current(); err == nil && site.IsNotes() {
				today := time.Now().Format("2006-01-02")
				text = "---\ncreated: " + today + "\nupdated: " + today + "\n---\n\n"
				// It takes the tags its folder's notes have (or its
				// database's).
				if tags := app.FolderTags(site, folder); len(tags) > 0 {
					if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
						return "", err
					}
					return rel, obsidian.SetProperties(p, obsidian.Prop{Key: "tags", Value: tags})
				}
			}
		}
		return rel, os.WriteFile(p, []byte(text), 0o644)
	}
}

// PasteImage saves an image pasted or dropped on note (its bytes in
// base64) and returns the Markdown that shows it.
func (s *Studio) PasteImage(note, name, data string) (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	b, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}
	return app.PasteImage(site, note, name, b)
}

// PickNoteImage asks for an image on the computer and adds it to note,
// like PasteImage. "" when nothing was picked.
func (s *Studio) PickNoteImage(note string) (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	path, err := application.Get().Dialog.OpenFile().SetTitle("Blup Studio").
		AddFilter("Images", "*.png;*.jpg;*.jpeg;*.webp;*.gif;*.svg;*.avif").PromptForSingleSelection()
	if err != nil || path == "" {
		return "", err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return app.PasteImage(site, note, filepath.Base(path), b)
}

// AuthorNames are the authors with a note in the site's authors folder,
// and the site's own author, for the Authors field's suggestions.
func (s *Studio) AuthorNames() []string {
	site, err := s.current()
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	out := []string{}
	add := func(name string) {
		if name = strings.TrimSpace(name); name != "" && !seen[strings.ToLower(name)] {
			seen[strings.ToLower(name)] = true
			out = append(out, name)
		}
	}
	if folder := obsidian.Sections(site.ContentDir())["authors"]; folder != "" {
		entries, _ := os.ReadDir(filepath.Join(site.ContentDir(), filepath.FromSlash(folder)))
		for _, e := range entries {
			if name := e.Name(); !e.IsDir() && strings.HasSuffix(strings.ToLower(name), ".md") && !strings.HasPrefix(name, "_") {
				add(strings.TrimSuffix(name, filepath.Ext(name)))
			}
		}
	}
	add(site.Author)
	return out
}
