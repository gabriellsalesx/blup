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
