package main

import "github.com/gabriellsalesx/blup/internal/app"

// SearchNotes finds words in the open site's notes: their names first,
// then their texts, with the lines where they are.
func (s *Studio) SearchNotes(query string) ([]app.SearchHit, error) {
	site, err := s.current()
	if err != nil {
		return nil, err
	}
	return app.SearchNotes(site, query)
}
