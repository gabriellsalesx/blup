package main

import "github.com/gabriellsalesx/blup/internal/app"

// NoteGraph is the map of the notes: every note, canvas and drawing, and
// the links between them (the map of notes, and "mentioned in").
func (s *Studio) NoteGraph() (app.Graph, error) {
	site, err := s.current()
	if err != nil {
		return app.Graph{}, err
	}
	return app.NoteGraph(site)
}

// Mentions are the notes that link to rel, with the lines where they do
// ("mentioned in").
func (s *Studio) Mentions(rel string) ([]app.Mention, error) {
	site, err := s.current()
	if err != nil {
		return nil, err
	}
	return app.Mentions(site, rel)
}
