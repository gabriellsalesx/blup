package main

import (
	"context"

	"github.com/gabriellsalesx/blup/internal/app"
)

// CheckUpdate tells whether a newer Blup Studio is out (force asks GitHub
// now instead of using the day's answer).
func (s *Studio) CheckUpdate(force bool) (app.UpdateInfo, error) {
	return app.CheckUpdate(context.Background(), force)
}
