package main

import "github.com/gabriellsalesx/blup/internal/app"

// Tags are the tags used in the notes' properties (tags: [a, b] or
// tags: a), the most used first, for the tags field's suggestions.
func (s *Studio) Tags() ([]string, error) {
	site, err := s.current()
	if err != nil {
		return nil, err
	}
	return app.VaultTags(site)
}
