package main

import "github.com/gabriellsalesx/blup/internal/app"

// The trash is the Studio's, kept beside the notes and not among them: it
// stays on this phone and does not sync.

// Trash lists what was deleted, the newest first.
func (s *NotesService) Trash() ([]app.TrashItem, error) {
	st, err := site()
	if err != nil {
		return nil, err
	}
	return app.ListTrash(st)
}

// TrashDelete deletes an item of the trash for good.
func (s *NotesService) TrashDelete(path string) error {
	st, err := site()
	if err != nil {
		return err
	}
	return app.DeleteFromTrash(st, path)
}

// TrashEmpty deletes everything in the trash for good.
func (s *NotesService) TrashEmpty() error {
	st, err := site()
	if err != nil {
		return err
	}
	return app.EmptyTrash(st)
}
