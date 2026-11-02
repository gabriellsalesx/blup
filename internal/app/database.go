package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/obsidian"
	"go.yaml.in/yaml/v3"
)

// Databases of a vault of notes, made for people who never used one: a
// list of things (clients, tasks, books) where each thing is a note, so it
// has room for writing, links to and from other notes, and shows in the
// search and the map of notes.
//
//	Bancos de dados/Clientes.table   what the database has: its columns
//	                                 (each a property of the notes) and
//	                                 its views (table, board)
//	Bancos de dados/Clientes/        one note per record, its fields in
//	                                 its properties
//
// The .table file is JSON; the Studio's own format.

// DBOption is a choice of a select column, and its color.
type DBOption struct {
	Value string `json:"value"`
	Color string `json:"color,omitempty"`
}

// DBColumn is a field of the records: Key is the property's name in the
// notes; Type is text, number, phone, email, url, date, select, multi,
// check or link (a note).
type DBColumn struct {
	Key     string     `json:"key"`
	Name    string     `json:"name"`
	Type    string     `json:"type"`
	Options []DBOption `json:"options,omitempty"`
}

// DBView is a way of seeing the records: a table or a board (one column
// per option of GroupBy), sorted and filtered.
type DBView struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"` // table, board
	GroupBy string `json:"groupBy,omitempty"`
	Sort    string `json:"sort,omitempty"` // a column's key ("" the name), "-" first for descending
	Filter  string `json:"filter,omitempty"`
	Match   string `json:"match,omitempty"`
}

// DBSchema is a .table file.
type DBSchema struct {
	Version int        `json:"version"`
	Records string     `json:"records"` // the records' folder, next to the .table file
	Tag     string     `json:"tag"`     // every record has it
	Columns []DBColumn `json:"columns"`
	Views   []DBView   `json:"views"`
}

// DBRecord is a record: a note of the records' folder.
type DBRecord struct {
	Path  string         `json:"path"`
	Name  string         `json:"name"`
	Props map[string]any `json:"props"`
	Text  string         `json:"text"` // the note's text (the start), for the search
}

// Database is a database and its records.
type Database struct {
	Schema  DBSchema   `json:"schema"`
	Records []DBRecord `json:"records"`
}

// DatabaseFolder is where new databases go when no folder is given (the
// Studio gives one in its language).
const DatabaseFolder = "Bancos de dados"

func dbPath(site *config.Site, rel string) (string, error) {
	if !strings.EqualFold(path.Ext(rel), ".table") {
		return "", errors.New("not a database")
	}
	return notesPath(site, rel)
}

// CreateDatabase makes a database named title in folder ("" for the
// databases' folder) with schema (its columns and views, from the
// Studio's templates), and returns its .table path. The records' folder
// takes the same name; a taken name gets a number.
func CreateDatabase(site *config.Site, folder, title string, schema DBSchema) (string, error) {
	title = safeFileName(title)
	if title == "" {
		return "", errors.New(i18nInvalidName())
	}
	if folder == "" {
		folder = DatabaseFolder
	}
	for i := 1; ; i++ {
		name := title
		if i > 1 {
			name = fmt.Sprintf("%s %d", title, i)
		}
		rel := path.Join(folder, name+".table")
		p, err := notesPath(site, rel)
		if err != nil {
			return "", err
		}
		records := filepath.Join(filepath.Dir(p), name)
		if _, err := os.Stat(p); err == nil {
			continue
		}
		if _, err := os.Stat(records); err == nil {
			continue
		}
		if err := os.MkdirAll(records, 0o755); err != nil {
			return "", err
		}
		schema.Version, schema.Records = 1, name
		if schema.Tag == "" {
			schema.Tag = tagName(name)
		}
		return rel, writeSchema(p, schema)
	}
}

// tagName is a tag from a name: lower case, spaces as dashes.
func tagName(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), "-")
}

func writeSchema(p string, s DBSchema) error {
	if s.Columns == nil {
		s.Columns = []DBColumn{}
	}
	if s.Views == nil {
		s.Views = []DBView{}
	}
	b, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(b, '\n'), 0o644)
}

func readSchema(p string) (DBSchema, error) {
	var s DBSchema
	b, err := os.ReadFile(p)
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return s, fmt.Errorf("the database file can't be read: %w", err)
	}
	// The records' folder is a name next to the .table: a path in a file
	// that came from elsewhere ("../../.config/autostart") would read and
	// write notes outside the vault.
	if !recordsName(s.Records) {
		s.Records = strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
	}
	return s, nil
}

// recordsName tells whether name is a folder's name, not a path.
func recordsName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, `/\`) && !strings.HasPrefix(name, ".") && filepath.VolumeName(name) == ""
}

// ReadDatabase reads a database and its records (the notes of its
// records' folder, by name).
func ReadDatabase(site *config.Site, rel string) (Database, error) {
	p, err := dbPath(site, rel)
	if err != nil {
		return Database{}, err
	}
	s, err := readSchema(p)
	if err != nil {
		return Database{}, err
	}
	db := Database{Schema: s, Records: []DBRecord{}}
	dir := path.Join(path.Dir(rel), s.Records)
	if _, err := notesPath(site, dir); err != nil {
		return db, err
	}
	entries, err := os.ReadDir(filepath.Join(filepath.Dir(p), s.Records))
	if errors.Is(err, os.ErrNotExist) {
		return db, nil
	}
	if err != nil {
		return db, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".md") || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		recRel := path.Join(dir, e.Name())
		b, err := os.ReadFile(filepath.Join(filepath.Dir(p), s.Records, e.Name()))
		if err != nil {
			continue
		}
		props, body := splitProps(string(b))
		if r := []rune(body); len(r) > 2000 {
			body = string(r[:2000])
		}
		db.Records = append(db.Records, DBRecord{Path: recRel, Name: strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())), Props: props, Text: body})
	}
	slices.SortFunc(db.Records, func(a, b DBRecord) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
	return db, nil
}

// splitProps reads a note's properties (JSON-friendly: dates as text)
// and its text.
func splitProps(text string) (map[string]any, string) {
	props := map[string]any{}
	fm, body := splitFront(text)
	_ = yaml.Unmarshal([]byte(fm), &props)
	for k, v := range props {
		props[k] = plainValue(v)
	}
	return props, strings.TrimLeft(body, "\n")
}

func plainValue(v any) any {
	switch x := v.(type) {
	case interface{ Format(string) string }:
		return x.Format("2006-01-02")
	case []any:
		for i, e := range x {
			x[i] = plainValue(e)
		}
	}
	return v
}

// SaveDatabaseSchema saves a database's columns and views.
func SaveDatabaseSchema(site *config.Site, rel string, s DBSchema) error {
	p, err := dbPath(site, rel)
	if err != nil {
		return err
	}
	old, err := readSchema(p)
	if err != nil {
		return err
	}
	s.Version, s.Records = 1, old.Records // the records stay where they are
	if s.Tag == "" {
		s.Tag = old.Tag
	}
	return writeSchema(p, s)
}

// AddRecord makes a record (a note named name in the records' folder,
// with the database's tag, its dates and props) and returns its path.
func AddRecord(site *config.Site, rel, name string, props map[string]any) (string, error) {
	p, err := dbPath(site, rel)
	if err != nil {
		return "", err
	}
	s, err := readSchema(p)
	if err != nil {
		return "", err
	}
	name = safeFileName(name)
	if name == "" {
		return "", errors.New(i18nInvalidName())
	}
	dir, err := notesPath(site, path.Join(path.Dir(rel), s.Records))
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	file := name + ".md"
	for i := 2; ; i++ {
		if _, err := os.Lstat(filepath.Join(dir, file)); errors.Is(err, os.ErrNotExist) {
			break
		}
		file = fmt.Sprintf("%s %d.md", name, i)
	}
	today := time.Now().Format("2006-01-02")
	full := filepath.Join(dir, file)
	if err := writeNew(full, []byte("---\ncreated: "+today+"\nupdated: "+today+"\n---\n\n")); err != nil {
		return "", err
	}
	list := []obsidian.Prop{}
	if s.Tag != "" {
		list = append(list, obsidian.Prop{Key: "tags", Value: []string{s.Tag}})
	}
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		if props[k] != nil && props[k] != "" {
			list = append(list, obsidian.Prop{Key: k, Value: props[k]})
		}
	}
	if err := obsidian.SetProperties(full, list...); err != nil {
		return "", err
	}
	return path.Join(path.Dir(rel), s.Records, file), nil
}

// followDatabase keeps a database whole after a move or rename (from was
// moved to to): a .table takes its records' folder along (renamed like
// it), and a renamed records' folder is written in its .table.
func followDatabase(site *config.Site, from, to string, dir bool) error {
	if !dir && strings.EqualFold(path.Ext(from), ".table") {
		p, err := notesPath(site, to)
		if err != nil {
			return err
		}
		s, err := readSchema(p)
		if err != nil {
			return nil // not a database we can read: leave it
		}
		oldDir, err := notesPath(site, path.Join(path.Dir(from), s.Records))
		if err != nil {
			return err
		}
		if _, err := os.Stat(oldDir); err != nil {
			return nil
		}
		name := strings.TrimSuffix(path.Base(to), path.Ext(to))
		newRel := path.Join(path.Dir(to), name)
		newDir, err := notesPath(site, newRel)
		if err != nil {
			return err
		}
		if newDir != oldDir {
			if _, err := os.Stat(newDir); err == nil {
				return errors.New(name + " already exists there")
			}
			if err := os.Rename(oldDir, newDir); err != nil {
				return err
			}
			if err := fixLinks(site, path.Join(path.Dir(from), s.Records), newRel, true); err != nil {
				return err
			}
		}
		s.Records = name
		return writeSchema(p, s)
	}
	if dir && path.Dir(from) == path.Dir(to) {
		// The records' folder renamed by itself: its .table follows the name.
		p, err := notesPath(site, path.Join(path.Dir(from), path.Base(from)+".table"))
		if err != nil {
			return nil
		}
		if s, err := readSchema(p); err == nil && s.Records == path.Base(from) {
			s.Records = path.Base(to)
			return writeSchema(p, s)
		}
	}
	return nil
}

// DatabaseOf is the database a note is a record of (its .table path and
// columns), or "" when it is not one: the note is in a folder that a
// .table next to that folder names as its records' folder.
func DatabaseOf(site *config.Site, rel string) (string, *DBSchema) {
	dir := path.Dir(rel)
	if dir == "." || dir == "" {
		return "", nil
	}
	parent := path.Dir(dir)
	if parent == "." {
		parent = ""
	}
	// Usually the .table has the folder's name; else any .table next to
	// the folder that names it (the folder renamed by itself).
	tables := []string{path.Join(parent, path.Base(dir)+".table")}
	pdir := filepath.Clean(site.ContentDir())
	if parent != "" {
		pdir, _ = notesPath(site, parent)
	}
	if pdir != "" {
		if entries, err := os.ReadDir(pdir); err == nil {
			for _, e := range entries {
				if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".table") {
					tables = append(tables, path.Join(parent, e.Name()))
				}
			}
		}
	}
	for _, table := range tables {
		p, err := dbPath(site, table)
		if err != nil {
			continue
		}
		if s, err := readSchema(p); err == nil && s.Records == path.Base(dir) {
			return table, &s
		}
	}
	return "", nil
}

// RecordsFolder is the name of a database's records' folder (next to its
// .table), or "" when the file can't be read.
func RecordsFolder(site *config.Site, rel string) string {
	p, err := dbPath(site, rel)
	if err != nil {
		return ""
	}
	s, err := readSchema(p)
	if err != nil {
		return ""
	}
	return s.Records
}
