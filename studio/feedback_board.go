package main

import (
	"context"

	"github.com/gabriellsalesx/blup/internal/app"
)

// FeedbackPosts lists the requests of the open site's feedback board.
func (s *Studio) FeedbackPosts(ctx context.Context) ([]app.FeedbackPost, error) {
	site, err := s.current()
	if err != nil {
		return nil, err
	}
	return app.FeedbackPosts(ctx, site)
}

// SetFeedbackStatus sets a request's status (new, planned, progress, done,
// declined) on GitHub; the site shows it within a minute.
func (s *Studio) SetFeedbackStatus(ctx context.Context, number int, status string) error {
	site, err := s.current()
	if err != nil {
		return err
	}
	return app.SetFeedbackStatus(ctx, site, number, status)
}
