package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/obsidian"
	"github.com/gabriellsalesx/blup/internal/scaffold"
)

// The vault as the Studio shows it: folders, notes, canvases, drawings and
// databases, with the same files, so what is made here opens there.

// Node is an item of the tree. A database shows as its records' folder, with
// the path of its .table in Table.
type Node struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Dir      bool   `json:"dir"`
	Kind     string `json:"kind"` // "", canvas, drawing, database
	Table    string `json:"table"`
	Children []Node `json:"children"`
}

// PropEdit is a change to a property of a record.
type PropEdit struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

// BoardService reads and writes canvases, drawings and databases.
type BoardService struct{}

const (
	kindCanvas   = "canvas"
	kindDrawing  = "drawing"
	kindDatabase = "database"

	emptyCanvas  = "{\n\t\"nodes\":[],\n\t\"edges\":[]\n}"
	emptyDrawing = "{\n\t\"version\":1,\n\t\"elements\":[]\n}"
)

func boardKind(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.HasSuffix(n, ".canvas"):
		return kindCanvas
	case strings.HasSuffix(n, ".draw"):
		return kindDrawing
	case strings.HasSuffix(n, ".table"):
		return kindDatabase
	}
	return ""
}

func boardName(name string) string {
	return strings.TrimSuffix(name, filepath.Ext(name))
}

// Tree is the open vault, folders first, by name. The calendar's folder is
// always there, as it is what the events are kept in.
func (b *BoardService) Tree() ([]Node, error) {
	st, err := site()
	if err != nil {
		return nil, err
	}
	var walk func(dir, rel string) ([]Node, error)
	walk = func(dir, rel string) ([]Node, error) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		dirs, files := []Node{}, []Node{}
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, ".") || (rel == "" && (name == attachments || slices.Contains(scaffold.AgentGuides, name))) {
				continue
			}
			p := strings.TrimPrefix(rel+"/"+name, "/")
			if e.IsDir() {
				kids, err := walk(filepath.Join(dir, name), p)
				if err != nil {
					return nil, err
				}
				dirs = append(dirs, Node{Name: name, Path: p, Dir: true, Children: kids})
			} else if kind := boardKind(name); kind != "" {
				files = append(files, Node{Name: boardName(name), Path: p, Kind: kind, Children: []Node{}})
			} else if strings.EqualFold(filepath.Ext(name), ".md") {
				files = append(files, Node{Name: boardName(name), Path: p, Children: []Node{}})
			}
		}
		kept := files[:0]
		for _, n := range files {
			if n.Kind == kindDatabase {
				records := app.RecordsFolder(st, n.Path)
				if i := slices.IndexFunc(dirs, func(d Node) bool { return d.Name == records && d.Kind == "" }); records != "" && i >= 0 {
					dirs[i].Kind, dirs[i].Table = kindDatabase, n.Path
					continue
				}
			}
			kept = append(kept, n)
		}
		byName := func(x, y Node) int { return strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name)) }
		if rel == "" && !slices.ContainsFunc(dirs, func(d Node) bool { return d.Name == app.CalendarFolder() }) {
			dirs = append(dirs, Node{Name: app.CalendarFolder(), Path: app.CalendarFolder(), Dir: true, Children: []Node{}})
		}
		slices.SortFunc(dirs, byName)
		slices.SortFunc(kept, byName)
		return append(dirs, kept...), nil
	}
	return walk(st.ContentDir(), "")
}

// file is an item of the open vault, refusing anything outside it.
func file(st *config.Site, rel string) (string, error) {
	root := filepath.Clean(st.ContentDir())
	p := filepath.Join(root, filepath.FromSlash(rel))
	if !filepath.IsLocal(filepath.FromSlash(rel)) || !app.Within(root, p) {
		return "", fail("notInVault")
	}
	return p, nil
}

func board(rel string) (string, error) {
	if boardKind(rel) == "" {
		return "", fail("notBoard")
	}
	st, err := site()
	if err != nil {
		return "", err
	}
	return file(st, rel)
}

// Read returns a canvas's or a drawing's file as it is.
func (b *BoardService) Read(rel string) (string, error) {
	p, err := board(rel)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(p)
	return string(data), err
}

// Save writes a canvas or a drawing. One that is gone is not made again.
func (b *BoardService) Save(rel, text string) error {
	p, err := board(rel)
	if err != nil {
		return err
	}
	if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return writeSafe(p, []byte(text))
}

// create makes a board named name in folder and returns its path.
func create(folder, name, ext, text string) (string, error) {
	st, err := site()
	if err != nil {
		return "", err
	}
	for i := 1; ; i++ {
		base := name
		if i > 1 {
			base = fmt.Sprintf("%s %d", name, i)
		}
		rel := strings.TrimPrefix(folder+"/"+base, "/") + ext
		p, err := file(st, rel)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(p); err == nil {
			continue
		}
		if err := app.MkdirInside(st.ContentDir(), filepath.Dir(p)); err != nil {
			return "", err
		}
		return rel, writeSafe(p, []byte(text))
	}
}

// CreateCanvas makes an empty canvas.
func (b *BoardService) CreateCanvas(folder, name string) (string, error) {
	return create(folder, name, ".canvas", emptyCanvas)
}

// CreateDrawing makes an empty drawing.
func (b *BoardService) CreateDrawing(folder, name string) (string, error) {
	return create(folder, name, ".draw", emptyDrawing)
}

var mdLink = regexp.MustCompile(`\]\(([^)]+)\)$`)

// SaveImage keeps an image put on a board in the attachments, its bytes in
// base64, and returns its path in the vault.
func (b *BoardService) SaveImage(rel, name, data string) (string, error) {
	st, err := site()
	if err != nil {
		return "", err
	}
	bytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}
	md, err := app.PasteImage(st, rel, name, bytes)
	if err != nil {
		return "", err
	}
	link := mdLink.FindStringSubmatch(md)
	if link == nil {
		return "", errors.New("unexpected image link")
	}
	to, err := url.PathUnescape(link[1])
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(path.Join(path.Dir("/"+rel), to), "/"), nil
}

// File is an image of the vault as a data: URL, for a board.
func (b *BoardService) File(rel string) (string, error) {
	st, err := site()
	if err != nil {
		return "", err
	}
	p, err := file(st, rel)
	if err != nil {
		return "", err
	}
	kind := mime.TypeByExtension(strings.ToLower(filepath.Ext(p)))
	if !strings.HasPrefix(kind, "image/") {
		return "", errors.New("not an image")
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return "data:" + kind + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

// ReadNote is the text of a note on a canvas's card, without its properties.
func (b *BoardService) ReadNote(rel string) (string, error) {
	st, err := site()
	if err != nil {
		return "", err
	}
	p, err := file(st, rel)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(p)
	return frontmatter.ReplaceAllString(string(data), ""), err
}

// CreateDatabase makes a database in folder and returns its path.
func (b *BoardService) CreateDatabase(folder, title string, schema app.DBSchema) (string, error) {
	st, err := site()
	if err != nil {
		return "", err
	}
	return app.CreateDatabase(st, folder, title, schema)
}

// ReadDatabase is a database and its records.
func (b *BoardService) ReadDatabase(rel string) (app.Database, error) {
	st, err := site()
	if err != nil {
		return app.Database{}, err
	}
	return app.ReadDatabase(st, rel)
}

// SaveDatabaseSchema saves a database's columns and views.
func (b *BoardService) SaveDatabaseSchema(rel string, schema app.DBSchema) error {
	st, err := site()
	if err != nil {
		return err
	}
	return app.SaveDatabaseSchema(st, rel, schema)
}

// AddRecord makes a record and returns its path.
func (b *BoardService) AddRecord(rel, name string, props map[string]any) (string, error) {
	st, err := site()
	if err != nil {
		return "", err
	}
	return app.AddRecord(st, rel, name, props)
}

// propValue is a value from a form, as the Studio takes it: an empty one
// removes the property, and 0 is a number.
func propValue(v any) any {
	switch x := v.(type) {
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
		if x == float64(int64(x)) {
			return int64(x)
		}
		return x
	case []any:
		out := []string{}
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

// SetProps changes properties of a record: a cell of a table or a board.
func (b *BoardService) SetProps(rel string, edits []PropEdit) error {
	st, err := site()
	if err != nil {
		return err
	}
	p, err := file(st, rel)
	if err != nil {
		return err
	}
	props := make([]obsidian.Prop, 0, len(edits))
	for _, e := range edits {
		props = append(props, obsidian.Prop{Key: e.Key, Value: propValue(e.Value)})
	}
	return setPropsSafe(p, props...)
}
