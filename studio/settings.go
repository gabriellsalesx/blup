package main

import (
	"encoding/base64"
	"errors"
	"mime"
	"os"
	"path/filepath"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// The settings screens: the same changes as blup customize, from forms.

// Settings reads the open site's settings.
func (s *Studio) Settings() (app.Settings, error) {
	site, err := s.current()
	if err != nil {
		return app.Settings{}, err
	}
	return app.LoadSettings(site)
}

// SaveAbout changes the site's name, description and author.
func (s *Studio) SaveAbout(name, description, author string) error {
	return s.with(func(site *config.Site) error { return app.SaveAbout(site, name, description, author) })
}

// SaveMenu sets the top menu and the top right button.
func (s *Studio) SaveMenu(menu []app.MenuItem, buttonText, buttonLink string) error {
	return s.with(func(site *config.Site) error { return app.SaveMenu(site, menu, buttonText, buttonLink) })
}

// SaveSocial sets the footer's social links.
func (s *Studio) SaveSocial(links []string) error {
	return s.with(func(site *config.Site) error { return app.SaveSocial(site, links) })
}

// PickImage asks for an image file and makes it the logo or favicon. It
// returns the image's file in the site, "" when the user cancels.
func (s *Studio) PickImage(key string) (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	path, err := application.Get().Dialog.OpenFile().
		SetTitle("Blup Studio").
		AddFilter("Images", "*.png;*.jpg;*.jpeg;*.webp;*.gif;*.svg").
		PromptForSingleSelection()
	if err != nil || path == "" {
		return "", err
	}
	return app.SetImage(site, key, path)
}

// RemoveImage removes the logo or favicon.
func (s *Studio) RemoveImage(key string) error {
	return s.with(func(site *config.Site) error { _, err := app.SetImage(site, key, ""); return err })
}

// ImageData is an image of the site as a data: URL, for previews in forms.
func (s *Studio) ImageData(file string) (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	root := filepath.Clean(site.ContentDir()) + string(filepath.Separator)
	if len(file) < len(root) || file[:len(root)] != root {
		return "", errors.New("outside the notes folder")
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	kind := mime.TypeByExtension(filepath.Ext(file))
	return "data:" + kind + ";base64," + base64.StdEncoding.EncodeToString(b), nil
}

// AddSection adds a Bento section with example notes, and returns the new
// tree of notes.
func (s *Studio) AddSection(kind string) (Node, error) {
	if err := s.with(func(site *config.Site) error { return app.AddSection(site, kind) }); err != nil {
		return Node{}, err
	}
	return s.Tree()
}

// CheckCookieTool tells whether an ID fits a tool ("" when it does).
func (s *Studio) CheckCookieTool(key, value string) string {
	if err := app.CheckCookieTool(key, value); err != nil {
		return err.Error()
	}
	return ""
}

// SaveCookies sets the tools, the banner and the privacy policy; it
// returns the policy note when the site has one.
func (s *Studio) SaveCookies(c config.Cookies, privacyPage bool, contact string) (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	return app.SaveCookies(site, c, privacyPage, contact)
}

// SetPassword sets the general password of protected posts.
func (s *Studio) SetPassword(password string) error {
	return s.with(func(site *config.Site) error { return app.SetPassword(site, password) })
}

// SetFeedback turns comments and reactions on or off.
func (s *Studio) SetFeedback(on bool) error {
	return s.with(func(site *config.Site) error { return app.SetFeedback(site, on) })
}

func (s *Studio) with(fn func(*config.Site) error) error {
	site, err := s.current()
	if err != nil {
		return err
	}
	return fn(site)
}
