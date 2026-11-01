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

// DatabaseFolder is where new databases go.
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
	if s.Records == "" {
		s.Records = strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
	}
	return s, nil
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
		if len(body) > 2000 {
			body = body[:2000]
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
	norm := strings.ReplaceAll(strings.TrimPrefix(text, "\ufeff"), "\r\n", "\n")
	if !strings.HasPrefix(norm, "---\n") {
		return props, norm
	}
	end := strings.Index(norm[4:], "\n---")
	if end < 0 {
		return props, norm
	}
	_ = yaml.Unmarshal([]byte(norm[4:4+end]), &props)
	for k, v := range props {
		props[k] = plainValue(v)
	}
	body := norm[4+end+4:]
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
	dir := filepath.Join(filepath.Dir(p), s.Records)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	file := name + ".md"
	for i := 2; ; i++ {
		if _, err := os.Stat(filepath.Join(dir, file)); errors.Is(err, os.ErrNotExist) {
			break
		}
		file = fmt.Sprintf("%s %d.md", name, i)
	}
	today := time.Now().Format("2006-01-02")
	full := filepath.Join(dir, file)
	if err := os.WriteFile(full, []byte("---\ncreated: "+today+"\nupdated: "+today+"\n---\n\n"), 0o644); err != nil {
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

// DatabaseOf is the database a note is a record of (its .table path and
// columns), or "" when it is not one: the note is in a folder that a
// .table next to that folder names as its records' folder.
func DatabaseOf(site *config.Site, rel string) (string, *DBSchema) {
	dir := path.Dir(rel)
	if dir == "." || dir == "" {
		return "", nil
	}
	table := path.Join(path.Dir(dir), path.Base(dir)+".table")
	if path.Dir(dir) == "." {
		table = path.Base(dir) + ".table"
	}
	p, err := dbPath(site, table)
	if err != nil {
		return "", nil
	}
	s, err := readSchema(p)
	if err != nil || s.Records != path.Base(dir) {
		return "", nil
	}
	return table, &s
}
