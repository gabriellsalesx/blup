package app

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/gabriellsalesx/blup/internal/obsidian"
)

// A blog offers the podcast in its sections (the Studio's only); adding it
// brings the folder with its note and puts it in a chosen menu.
func TestPodcastSection(t *testing.T) {
	site := settingsSite(t)
	write(t, site.ContentDir(), "_index.md", "---\ntitle: Início\nmenu:\n  - Posts\n---\nOi\n")
	s, err := LoadSettings(site)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Sections) != 1 || s.Sections[0].Kind != "podcast" || s.Sections[0].Present {
		t.Fatalf("sections = %+v", s.Sections)
	}
	if err := AddSection(site, "docs"); err == nil {
		t.Fatal("a blog can't add Bento's docs")
	}
	if err := AddSection(site, "podcast"); err != nil {
		t.Fatal(err)
	}
	if obsidian.Sections(site.ContentDir())["podcast"] != "Podcast" {
		t.Fatal("no Podcast folder")
	}
	home, _ := obsidian.ReadNote(filepath.Join(site.ContentDir(), "_index.md"))
	if !slices.Contains(home.List("menu"), "Podcast") {
		t.Fatalf("menu = %v", home.List("menu"))
	}
	// A new blog doesn't get it by itself.
	fresh := settingsSite(t)
	os.RemoveAll(fresh.ContentDir())
	if err := writeSamples(fresh, nil); err != nil {
		t.Fatal(err)
	}
	if _, ok := obsidian.Sections(fresh.ContentDir())["podcast"]; ok {
		t.Fatal("a new blog came with the podcast")
	}
}

// The Podcast screen: the show's details, the episodes (posts with a
// narration the podcast lists) and the feed of each language.
func TestPodcastInfo(t *testing.T) {
	site := settingsSite(t)
	site.Domain = "blog.exemplo.com"
	site.Translations = []string{"en"}
	if err := AddSection(site, "podcast"); err != nil {
		t.Fatal(err)
	}
	root := site.ContentDir()
	write(t, root, "Posts/Com audio.md", "---\naudio: \"[[a.mp3]]\"\n---\nTexto\n")
	write(t, root, "Posts/Escondido.md", "---\naudio: \"[[b.mp3]]\"\nhidden: true\n---\n")
	write(t, root, "Posts/Senha.md", "---\naudio: \"[[c.mp3]]\"\npassword: sim\n---\n")
	write(t, root, "Posts/en/With audio.md", "---\naudio: \"[[d.mp3]]\"\n---\n")
	info, err := Podcast(site)
	if err != nil {
		t.Fatal(err)
	}
	if info.Episodes != 1 {
		t.Fatalf("episodes = %d", info.Episodes)
	}
	if !info.HasURL || len(info.Feeds) != 2 || info.Feeds[0].URL != "https://blog.exemplo.com/podcast/feed.xml" || info.Feeds[1].URL != "https://blog.exemplo.com/en/podcast/feed.xml" {
		t.Fatalf("feeds = %+v", info.Feeds)
	}
	if info.Category != "Technology" || info.Explicit {
		t.Fatalf("defaults = %+v", info)
	}
	if err := SavePodcast(site, []obsidian.Prop{{Key: "podcast_email", Value: "dono@exemplo.com"}, {Key: "podcast_explicit", Value: true}, {Key: "podcast_category", Value: "Education"}}); err != nil {
		t.Fatal(err)
	}
	if info, _ = Podcast(site); info.Email != "dono@exemplo.com" || !info.Explicit || info.Category != "Education" {
		t.Fatalf("saved = %+v", info)
	}
	if err := SavePodcast(site, []obsidian.Prop{{Key: "password", Value: "x"}}); err == nil {
		t.Fatal("the podcast screen changed another property")
	}
	if err := SavePodcast(site, []obsidian.Prop{{Key: "podcast_email", Value: "sem-arroba"}}); err == nil || !strings.Contains(err.Error(), "e-mail") {
		t.Fatalf("bad e-mail: %v", err)
	}
}
