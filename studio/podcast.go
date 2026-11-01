package main

import (
	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/obsidian"
)

// Podcast is the open blog's podcast (the Podcast screen).
func (s *Studio) Podcast() (app.PodcastInfo, error) {
	site, err := s.current()
	if err != nil {
		return app.PodcastInfo{}, err
	}
	return app.Podcast(site)
}

// SavePodcast sets the podcast's details from the Podcast screen (an empty
// value takes the property off: the site's name, author or the newest
// episode's cover stand in).
func (s *Studio) SavePodcast(edits []PropEdit) (app.PodcastInfo, error) {
	site, err := s.current()
	if err != nil {
		return app.PodcastInfo{}, err
	}
	props := make([]obsidian.Prop, 0, len(edits))
	for _, e := range edits {
		v := clean(e.Value)
		// Explicit is a yes or no the feed reads: false is kept, not removed.
		if e.Key == "podcast_explicit" {
			b, _ := e.Value.(bool)
			v = b
		}
		props = append(props, obsidian.Prop{Key: e.Key, Value: v})
	}
	if err := app.SavePodcast(site, props); err != nil {
		return app.PodcastInfo{}, err
	}
	return app.Podcast(site)
}
