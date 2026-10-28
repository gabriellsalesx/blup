package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/obsidian"
)

// The site's settings live in Obsidian: properties of the home note (the
// _index of the site folder) and one note per author in the authors folder.
// The theme reads them (theme/layouts/_partials/site/); the CLI only writes
// the same properties the user could type in Obsidian.

func homeNotePath(site *config.Site) string {
	return filepath.Join(site.ContentDir(), "_index.md")
}

// authorsFolders are the names of a new authors folder, by site language.
// Not "Authors": /authors/ is where the theme lists each author's posts.
var authorsFolders = map[string]string{"pt": "Autores", "en": "Writers"}

// authorsDir is the folder of author notes, created (with its _index note)
// when the site has none.
func authorsDir(site *config.Site) (string, error) {
	if f := obsidian.Sections(site.ContentDir())["authors"]; f != "" {
		return filepath.Join(site.ContentDir(), filepath.FromSlash(f)), nil
	}
	name := authorsFolders[site.Lang]
	if name == "" {
		name = authorsFolders["en"]
	}
	dir := filepath.Join(site.ContentDir(), name)
	body := fmt.Sprintf("---\ntitle: %s\nblup: authors\n---\n%s\n", name, authorsAbout[site.Lang])
	if _, err := obsidian.WriteNoteIfMissing(filepath.Join(dir, "_index.md"), body); err != nil {
		return "", err
	}
	return dir, nil
}

var authorsAbout = map[string]string{
	"pt": "Uma nota por autor, com o nome da pessoa. Propriedades: **photo** (imagem), **role** (cargo) e **social** (links dos perfis). O texto da nota é a bio. Nos posts, escreva `authors: [Nome]`.\n\nEsta pasta não vira página do site.",
	"en": "One note per author, named after the person. Properties: **photo** (image), **role** and **social** (profile links). The note's text is the bio. In posts, write `authors: [Name]`.\n\nThis folder is not published as pages.",
}

// authorNotePath is the note of the author called name (created when
// missing, with empty properties to fill in).
func authorNotePath(site *config.Site, name string) (string, error) {
	dir, err := authorsDir(site)
	if err != nil {
		return "", err
	}
	if entries, err := os.ReadDir(dir); err == nil {
		for _, e := range entries {
			if strings.EqualFold(strings.TrimSuffix(e.Name(), ".md"), name) && strings.HasSuffix(e.Name(), ".md") {
				return filepath.Join(dir, e.Name()), nil
			}
		}
	}
	p := filepath.Join(dir, safeFileName(name)+".md")
	_, err = obsidian.WriteNoteIfMissing(p, "---\nphoto:\nrole:\nsocial: []\n---\n")
	return p, err
}

// safeFileName drops the characters Obsidian does not allow in note names.
func safeFileName(s string) string {
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if strings.ContainsRune(`*"\/<>:|?#^[]`, r) {
			return -1
		}
		return r
	}, s))
}

// copyAttachment copies an image the user picked on the computer into the
// site's attachments folder and returns the link for a property ([[name]]).
func copyAttachment(site *config.Site, src, name string) (string, error) {
	b, err := os.ReadFile(src)
	if err != nil {
		return "", err
	}
	file := name + strings.ToLower(filepath.Ext(src))
	dest := filepath.Join(site.ContentDir(), attachmentsFolder(site), file)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(dest, b, 0o644); err != nil {
		return "", err
	}
	return "[[" + file + "]]", nil
}

// textProp is a property that is removed when empty.
func textProp(key, v string) obsidian.Prop {
	if v = strings.TrimSpace(v); v == "" {
		return obsidian.Prop{Key: key}
	}
	return obsidian.Prop{Key: key, Value: v}
}

// listProp is a list property that is removed when empty.
func listProp(key string, v []string) obsidian.Prop {
	if len(v) == 0 {
		return obsidian.Prop{Key: key}
	}
	return obsidian.Prop{Key: key, Value: v}
}

// writeSiteNotes records what the wizard asked: name and description on the
// home note, and the author's note.
func writeSiteNotes(site *config.Site) error {
	if err := obsidian.SetProperties(homeNotePath(site),
		textProp("site_name", site.Title), textProp("site_description", site.Description)); err != nil {
		return err
	}
	if site.Author == "" {
		return nil
	}
	_, err := authorNotePath(site, site.Author)
	return err
}
