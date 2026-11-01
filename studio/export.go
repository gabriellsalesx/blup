package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// SaveExport asks where to save a note exported as HTML (name is the
// suggested file name) and writes html there. It returns the file's path,
// "" when the person cancels.
func (s *Studio) SaveExport(name, html string) (string, error) {
	name = strings.Map(func(r rune) rune {
		if strings.ContainsRune(`/\:*?"<>|`, r) {
			return -1
		}
		return r
	}, strings.TrimSpace(name))
	if name == "" {
		name = "note"
	}
	home, _ := os.UserHomeDir()
	p, err := application.Get().Dialog.SaveFile().SetFilename(name+".html").SetDirectory(home).
		AddFilter("HTML", "*.html").CanCreateDirectories(true).PromptForSingleSelection()
	if err != nil || p == "" {
		return "", err
	}
	if !strings.EqualFold(filepath.Ext(p), ".html") && !strings.EqualFold(filepath.Ext(p), ".htm") {
		p += ".html"
	}
	return p, os.WriteFile(p, []byte(html), 0o644)
}
