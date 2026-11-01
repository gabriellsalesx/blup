package app

import (
	"errors"
	"image"
	_ "image/jpeg" // the podcast's cover: its size
	_ "image/png"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/obsidian"
	_ "golang.org/x/image/webp"
)

// A blog's podcast (mochi and nori, added in Blup Studio): the Podcast
// section's note holds the show's details as properties, read by the
// theme's feed (theme/layouts/section.podcast.xml). The episodes are the
// posts with a narration.

// PodcastProps are the note's properties the Studio's Podcast screen edits.
var PodcastProps = []string{"podcast_title", "description", "podcast_author", "podcast_email", "podcast_category", "podcast_explicit", "image"}

// PodcastInfo is the podcast as the Studio shows it.
type PodcastInfo struct {
	Folder   string `json:"folder"`
	Note     string `json:"note"` // the section's note, relative to the notes folder
	Title    string `json:"title"`
	SiteName string `json:"siteName"` // the show's name when Title is empty
	Desc     string `json:"description"`
	Author   string `json:"author"`
	Email    string `json:"email"`
	Category string `json:"category"`
	Explicit bool   `json:"explicit"`
	Image    string `json:"image"` // "[[cover.jpg]]", or "" for the newest episode's cover
	// The cover's size, to say when Apple would refuse it (it wants a
	// square of 1400 to 3000 px). 0 when unknown.
	ImageW int `json:"imageW"`
	ImageH int `json:"imageH"`
	// The feeds (one per language of the site), full addresses when the
	// site has a domain, else paths from the site's root.
	Feeds    []PodcastFeed `json:"feeds"`
	HasURL   bool          `json:"hasURL"`
	Episodes int           `json:"episodes"` // posts with a narration, in the site's language
}

// PodcastFeed is the feed of one language.
type PodcastFeed struct {
	Lang string `json:"lang"`
	URL  string `json:"url"`
}

// Podcast reads the site's podcast; an error when it has none.
func Podcast(site *config.Site) (PodcastInfo, error) {
	folder, ok := existingSections(site)["podcast"]
	if !ok {
		return PodcastInfo{}, errors.New("the site has no podcast")
	}
	rel := path.Join(folder, "_index.md")
	n, err := obsidian.ReadNote(filepath.Join(site.ContentDir(), filepath.FromSlash(rel)))
	if err != nil {
		return PodcastInfo{}, err
	}
	info := PodcastInfo{
		Folder:   folder,
		Note:     rel,
		Title:    n.String("podcast_title"),
		SiteName: firstOf(homeProp(site, "site_name"), site.Title),
		Desc:     n.String("description"),
		Author:   n.String("podcast_author"),
		Email:    n.String("podcast_email"),
		Category: firstOf(n.String("podcast_category"), "Technology"),
		Image:    n.String("image"),
		HasURL:   site.Domain != "",
		Episodes: countEpisodes(site),
	}
	switch v := n.Props["podcast_explicit"].(type) {
	case bool:
		info.Explicit = v
	case string:
		info.Explicit = strings.EqualFold(v, "true") || strings.EqualFold(v, "yes") || strings.EqualFold(v, "sim")
	}
	if f := attachmentFile(site, info.Image); f != "" {
		if file, err := os.Open(f); err == nil {
			if c, _, err := image.DecodeConfig(file); err == nil {
				info.ImageW, info.ImageH = c.Width, c.Height
			}
			file.Close()
		}
	}
	// The section's address is its folder, as Hugo writes it; the main
	// language at the root, the others under their code.
	slug := urlize(folder)
	base := "/"
	if site.Domain != "" {
		base = "https://" + site.Domain + "/"
	}
	info.Feeds = append(info.Feeds, PodcastFeed{Lang: site.Lang, URL: base + slug + "/feed.xml"})
	for _, l := range site.OtherLangs() {
		info.Feeds = append(info.Feeds, PodcastFeed{Lang: l, URL: base + l + "/" + slug + "/feed.xml"})
	}
	return info, nil
}

// SavePodcast sets the podcast's details (PodcastProps only).
func SavePodcast(site *config.Site, props []obsidian.Prop) error {
	info, err := Podcast(site)
	if err != nil {
		return err
	}
	var set []obsidian.Prop
	for _, p := range props {
		known := false
		for _, k := range PodcastProps {
			known = known || p.Key == k
		}
		if !known {
			return errors.New("unknown podcast property " + p.Key)
		}
		if p.Key == "podcast_email" {
			if s, _ := p.Value.(string); s != "" && !strings.Contains(s, "@") {
				return errors.New("invalid e-mail")
			}
		}
		set = append(set, p)
	}
	if len(set) == 0 {
		return nil
	}
	return obsidian.SetProperties(filepath.Join(site.ContentDir(), filepath.FromSlash(info.Note)), set...)
}

// countEpisodes counts the posts of the site's language with a narration
// that the podcast lists (not hidden, not drafts, not protected).
func countEpisodes(site *config.Site) int {
	root := site.ContentDir()
	langs := map[string]bool{}
	for _, l := range site.OtherLangs() {
		langs[l] = true
	}
	n := 0
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if p != root && (strings.HasPrefix(name, ".") || langs[name]) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(name), ".md") || strings.HasPrefix(name, "_") {
			return nil
		}
		note, err := obsidian.ReadNote(p)
		if err != nil || note.String("audio") == "" || note.Props["password"] != nil || note.Props["hidden"] == true || note.Props["draft"] == true {
			return nil
		}
		n++
		return nil
	})
	return n
}

// homeProp is a property of the site's home note ("" when missing).
func homeProp(site *config.Site, key string) string {
	n, err := obsidian.ReadNote(homeNotePath(site))
	if err != nil {
		return ""
	}
	return n.String(key)
}

// urlize is Hugo's path for a folder name: lower case, spaces as dashes.
func urlize(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), "-"))
}
