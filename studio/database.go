package main

import (
	"time"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/obsidian"
)

// Databases of a vault of notes (see app/database.go): a .table file with
// the columns and views, the records as notes in a folder next to it.

// CreateDatabase makes a database named title in folder ("" for the
// databases' folder) with the columns and views of a template, and
// returns its path.
func (s *Studio) CreateDatabase(folder, title string, schema app.DBSchema) (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	return app.CreateDatabase(site, folder, title, schema)
}

// ReadDatabase is a database and its records.
func (s *Studio) ReadDatabase(rel string) (app.Database, error) {
	site, err := s.current()
	if err != nil {
		return app.Database{}, err
	}
	return app.ReadDatabase(site, rel)
}

// SaveDatabaseSchema saves a database's columns and views.
func (s *Studio) SaveDatabaseSchema(rel string, schema app.DBSchema) error {
	return s.with(func(site *config.Site) error { return app.SaveDatabaseSchema(site, rel, schema) })
}

// AddRecord makes a record of a database and returns its path.
func (s *Studio) AddRecord(rel, name string, props map[string]any) (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	return app.AddRecord(site, rel, name, props)
}

// SetProps changes properties of a note (a database's cell, from its
// table or board): a nil, empty or false value removes it.
func (s *Studio) SetProps(rel string, edits []PropEdit) error {
	p, err := s.notePath(rel)
	if err != nil {
		return err
	}
	props := make([]obsidian.Prop, 0, len(edits))
	for _, e := range edits {
		props = append(props, obsidian.Prop{Key: e.Key, Value: s.propValue(e.Value)})
	}
	return obsidian.SetProperties(p, props...)
}

// DailyNote opens the note of the day in folder (made with tag the first
// time) with the time added at its end, and returns its path.
func (s *Studio) DailyNote(folder, tag string) (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	return app.DailyNote(site, folder, tag, time.Now())
}
