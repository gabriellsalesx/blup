package main

import (
	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// The Authors screen: the site's authors as forms (app.ListAuthors). Their
// notes live in the authors folder, which the file tree hides.

// Authors lists the site's authors.
func (s *Studio) Authors() ([]app.Author, error) {
	site, err := s.current()
	if err != nil {
		return nil, err
	}
	return app.ListAuthors(site)
}

// SaveAuthor creates (old "") or updates an author.
func (s *Studio) SaveAuthor(old string, a app.Author) error {
	return s.with(func(site *config.Site) error { return app.SaveAuthor(site, old, a) })
}

// DeleteAuthor moves the author's note to the trash.
func (s *Studio) DeleteAuthor(name string) error {
	return s.with(func(site *config.Site) error { return app.DeleteAuthor(site, name) })
}

// PickAuthorPhoto asks for a picture and returns the link for the author's
// photo ("" when the user cancels).
func (s *Studio) PickAuthorPhoto(name string) (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	path, err := application.Get().Dialog.OpenFile().SetTitle("Blup Studio").
		AddFilter("Images", "*.png;*.jpg;*.jpeg;*.webp;*.gif;*.svg").PromptForSingleSelection()
	if err != nil || path == "" {
		return "", err
	}
	return app.AuthorPhoto(site, name, path)
}
