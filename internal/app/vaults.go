package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/obsidian"
)

// Vaults of notes, for Blup Studio: notes that are not a site. A vault is
// kept like a site (in the list, with blup.toml), so it opens in the same
// editor and syncs the same way, but it has no theme, preview or
// publishing. Its technical folder is Blup's own, out of sight; only the
// notes folder is the person's.

// DefaultVaultNotesDir is a new folder for a vault's notes, in Documents.
func DefaultVaultNotesDir(title string) string {
	home, _ := os.UserHomeDir()
	docs := filepath.Join(home, "Documents")
	if _, err := os.Stat(docs); err != nil {
		docs = home
	}
	return filepath.Join(docs, "Blup", i18n.T("vault.folder"), firstOf(strings.TrimSpace(title), "notes"))
}

// vaultDir is where a vault's blup.toml goes: Blup's config folder.
func vaultDir(title string) (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return freeDir(filepath.Join(dir, "vaults", Slug(firstOf(title, "notes")))), nil
}

// CreateVault makes a vault of notes named title over notes (a new folder,
// or one that already has notes) and adds it to the list.
func CreateVault(g *config.Global, title, notes string) (*config.Site, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, errors.New(i18n.T("required"))
	}
	if notes == "" {
		notes = freeDir(DefaultVaultNotesDir(title))
	}
	notes = expandHome(notes)
	dir, err := vaultDir(title)
	if err != nil {
		return nil, err
	}
	for _, d := range []string{notes, dir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}
	site := config.NewSite(dir)
	site.Title, site.Kind, site.Model, site.Lang = title, config.KindNotes, "mochi", string(i18n.Current())
	site.Vault, site.Folder = notes, ""
	if obsidian.CountNotes(notes) == 0 {
		welcome := filepath.Join(notes, i18n.T("vault.welcome_file"))
		if err := os.WriteFile(welcome, []byte(i18n.T("vault.welcome_text", title)), 0o644); err != nil {
			return nil, err
		}
	}
	if err := WriteAgentGuides(site); err != nil {
		return nil, err
	}
	if err := site.Save(); err != nil {
		return nil, err
	}
	g.AddSite(dir)
	return site, g.Save()
}
