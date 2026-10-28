package app

import (
	"embed"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/obsidian"
	"github.com/gabriellsalesx/blup/internal/ui"
)

// starter is the site a new site starts with: a complete tutorial about
// Blup, written as ordinary notes, so the user learns by reading it and by
// opening the notes in Obsidian to see how each page is made.
//
//	starter/<lang>/<model>/  notes, copied into the site folder as they are
//	starter/covers/          images the notes point to ([[name.jpg]])
//
// In the bento model each top-level folder is a section (its _index has
// "blup: docs", "roadmap", ...) and only the sections the user picked are
// copied. Placeholders: {{title}}, {{date}} and {{date-N}} (N days ago).
//
//go:embed all:starter
var starter embed.FS

// docsKinds are the optional sections of the bento model, in menu order.
var docsKinds = []string{"docs", "roadmap", "changelog", "help", "blog"}

// attachmentsFolders name the folder of images, by site language.
var attachmentsFolders = map[string]string{"pt": "Anexos", "en": "Attachments"}

func attachmentsFolder(site *config.Site) string {
	if f := attachmentsFolders[site.Lang]; f != "" {
		return f
	}
	return attachmentsFolders["en"]
}

func starterRoot(site *config.Site) string {
	lang := site.Lang
	if lang != "pt" {
		lang = "en"
	}
	return path.Join("starter", lang, site.Model)
}

// starterSections maps each section kind of the starter to its folder.
func starterSections(site *config.Site) map[string]string {
	out := map[string]string{}
	entries, _ := starter.ReadDir(starterRoot(site))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		b, err := starter.ReadFile(path.Join(starterRoot(site), e.Name(), "_index.md"))
		if err != nil {
			continue
		}
		if m := sectionKindRe.FindSubmatch(b); m != nil {
			out[string(m[1])] = e.Name()
		}
	}
	return out
}

var sectionKindRe = regexp.MustCompile(`(?m)^blup:\s*["']?([a-z]+)["']?\s*$`)

// existingSections maps each section kind found in the site folder to its
// folder name.
func existingSections(site *config.Site) map[string]string {
	return obsidian.Sections(site.ContentDir())
}

// writeSamples fills an empty site folder with the starter site. For the
// bento model, sections lists the parts chosen (docs, roadmap, ...).
func writeSamples(site *config.Site, sections []string) error {
	skip := map[string]bool{}
	if site.Model == "bento" {
		for kind, folder := range starterSections(site) {
			if kind != "authors" && !contains(sections, kind) {
				skip[folder] = true
			}
		}
	}
	return copyStarter(site, "", skip)
}

// writeSection adds one section of the starter to an existing site. It does
// nothing when the site already has that section.
func writeSection(site *config.Site, kind string) error {
	if _, ok := existingSections(site)[kind]; ok {
		return nil
	}
	folder, ok := starterSections(site)[kind]
	if !ok {
		return nil
	}
	return copyStarter(site, folder, nil)
}

// sectionFolder is the folder a section gets in the starter.
func sectionFolder(site *config.Site, kind string) string {
	return starterSections(site)[kind]
}

// ensureHome creates the home page note (_index.md) when it is missing.
func ensureHome(site *config.Site) error {
	b, err := starter.ReadFile(path.Join(starterRoot(site), "_index.md"))
	if err != nil {
		return err
	}
	created, err := obsidian.WriteNoteIfMissing(filepath.Join(site.ContentDir(), "_index.md"), fillStarter(site, string(b)))
	if created {
		ui.OK(i18n.T("new.home_note"))
	}
	return err
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

var (
	daysAgoRe  = regexp.MustCompile(`\{\{date-(\d+)\}\}`)
	coverRefRe = regexp.MustCompile(`\[\[([^\]|]+\.(?:jpg|png|webp|svg))(?:\|[^\]]*)?\]\]`)
)

func fillStarter(site *config.Site, body string) string {
	now := time.Now()
	body = daysAgoRe.ReplaceAllStringFunc(body, func(m string) string {
		n, _ := strconv.Atoi(daysAgoRe.FindStringSubmatch(m)[1])
		return now.AddDate(0, 0, -n).Format("2006-01-02")
	})
	return strings.NewReplacer(
		"{{date}}", now.Format("2006-01-02"),
		"{{title}}", site.Title,
		"{{attach}}", attachmentsFolder(site),
	).Replace(body)
}

// copyStarter writes the starter notes under sub ("" = everything, except
// the top-level folders in skip) that the site folder does not have yet,
// and the covers they point to.
func copyStarter(site *config.Site, sub string, skip map[string]bool) error {
	root := starterRoot(site)
	base := path.Join(root, sub)
	images := map[string]bool{}
	err := fs.WalkDir(starter, base, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(p, root), "/")
		if d.IsDir() {
			if top, _, _ := strings.Cut(rel, "/"); rel != "" && skip[top] {
				return fs.SkipDir
			}
			return nil
		}
		b, err := starter.ReadFile(p)
		if err != nil {
			return err
		}
		body := fillStarter(site, string(b))
		for _, m := range coverRefRe.FindAllStringSubmatch(body, -1) {
			images[path.Base(m[1])] = true
		}
		for _, m := range attachRefRe.FindAllStringSubmatch(body, -1) {
			images[m[1]] = true
		}
		_, err = obsidian.WriteNoteIfMissing(filepath.Join(site.ContentDir(), filepath.FromSlash(rel)), body)
		return err
	})
	if err != nil {
		return err
	}
	for name := range images {
		b, err := starter.ReadFile(path.Join("starter", "covers", name))
		if err != nil {
			continue // an image the user is meant to add
		}
		dest := filepath.Join(site.ContentDir(), attachmentsFolder(site), name)
		if _, err := os.Stat(dest); err == nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dest, b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// attachRefRe finds images linked as Markdown inside the notes
// (![...](../Anexos/name.jpg)).
var attachRefRe = regexp.MustCompile(`\]\((?:\.\./)*(?:Anexos|Attachments)/([^)\s]+\.(?:jpg|png|webp|svg))\)`)
