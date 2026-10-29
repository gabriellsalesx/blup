package app

import (
	"embed"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/obsidian"
)

// The authors of a site, for Blup Studio's Authors screen: each one is a
// note in the authors folder (photo, role and social as properties, the
// bio as the text), which is what the theme reads. Renaming an author also
// renames it in the posts' "authors" lists.

// An author saved without a photo gets a Blup as one, taking turns: the
// first author the first Blup, the next one the second, and so on. Sending
// a picture of their own replaces it.
//
//go:embed avatars/*.svg
var avatarFiles embed.FS

var avatars = []string{"blup.svg", "turma-3.svg", "turma-4.svg", "turma-1.svg", "turma-6.svg",
	"turma-2.svg", "turma-5.svg", "turma-8.svg", "turma-7.svg"}

// NextAvatar is the Blup the next author without a photo gets.
func NextAvatar(site *config.Site) string {
	n := 0
	if folder := obsidian.Sections(site.ContentDir())["authors"]; folder != "" {
		entries, _ := os.ReadDir(filepath.Join(site.ContentDir(), filepath.FromSlash(folder)))
		for _, e := range entries {
			if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".md") && !strings.HasPrefix(e.Name(), "_") {
				n++
			}
		}
	}
	return avatars[n%len(avatars)]
}

// blupAvatar writes the Blup into the attachments, named after the author,
// and returns the link for the photo property.
func blupAvatar(site *config.Site, name, avatar string) (string, error) {
	b, err := avatarFiles.ReadFile("avatars/" + avatar)
	if err != nil {
		return "", err
	}
	file := safeFileName(name) + ".svg"
	dir := filepath.Join(site.ContentDir(), attachmentsFolder(site))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, file), b, 0o644); err != nil {
		return "", err
	}
	return "[[" + file + "]]", nil
}

// Author is an author of the site.
type Author struct {
	Name   string   `json:"name"`
	Role   string   `json:"role"`
	Photo  string   `json:"photo"`  // the property's link ([[ana.jpg]])
	Social []string `json:"social"` // profile URLs
	Bio    string   `json:"bio"`
	Posts  int      `json:"posts"`  // posts that list this author
	Main   bool     `json:"main"`   // the site's own author (blup.toml)
	Avatar string   `json:"avatar"` // without a photo: the Blup it gets when saved
}

// ListAuthors lists the authors with a note, and the site's author.
func ListAuthors(site *config.Site) ([]Author, error) {
	count := authorPosts(site)
	out := []Author{}
	if folder := obsidian.Sections(site.ContentDir())["authors"]; folder != "" {
		dir := filepath.Join(site.ContentDir(), filepath.FromSlash(folder))
		entries, err := os.ReadDir(dir)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.EqualFold(filepath.Ext(name), ".md") || strings.HasPrefix(name, "_") || strings.HasPrefix(name, ".") {
				continue
			}
			n, err := obsidian.ReadNote(filepath.Join(dir, name))
			if err != nil {
				return nil, err
			}
			a := Author{Name: strings.TrimSuffix(name, filepath.Ext(name)), Role: n.String("role"), Photo: n.String("photo"),
				Social: n.List("social"), Bio: strings.TrimSpace(n.Body)}
			a.Posts = count[strings.ToLower(a.Name)]
			out = append(out, a)
		}
	}
	if main := strings.TrimSpace(site.Author); main != "" {
		i := slices.IndexFunc(out, func(a Author) bool { return strings.EqualFold(a.Name, main) })
		if i < 0 {
			out = append(out, Author{Name: main, Posts: count[strings.ToLower(main)]})
			i = len(out) - 1
		}
		out[i].Main = true
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Main != out[j].Main {
			return out[i].Main
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	for i := range out {
		if out[i].Social == nil {
			out[i].Social = []string{}
		}
		if out[i].Photo == "" {
			out[i].Avatar = avatars[i%len(avatars)]
		}
	}
	return out, nil
}

// authorPosts counts the notes that list each author (lowercase names).
func authorPosts(site *config.Site) map[string]int {
	out := map[string]int{}
	eachNote(site, func(p string, n obsidian.Note) {
		for _, a := range n.List("authors") {
			out[strings.ToLower(strings.TrimSpace(a))]++
		}
	})
	return out
}

// eachNote reads every note of the site (skipping hidden folders).
func eachNote(site *config.Site, fn func(path string, n obsidian.Note)) {
	root := site.ContentDir()
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(p), ".md") {
			if n, err := obsidian.ReadNote(p); err == nil {
				fn(p, n)
			}
		}
		return nil
	})
}

// SaveAuthor creates or updates an author. old is its current name ("" for
// a new one); a new name renames the note, the posts' lists and, for the
// site's author, blup.toml.
func SaveAuthor(site *config.Site, old string, a Author) error {
	name, err := cleanName(a.Name)
	if err != nil {
		return err
	}
	if old == "" || !strings.EqualFold(old, name) {
		if list, _ := ListAuthors(site); slices.ContainsFunc(list, func(x Author) bool { return strings.EqualFold(x.Name, name) && !strings.EqualFold(x.Name, old) }) {
			return errors.New(name + " already exists")
		}
	}
	if strings.TrimSpace(a.Photo) == "" {
		avatar := NextAvatar(site)
		if old != "" { // an existing author keeps the Blup it was shown with
			list, _ := ListAuthors(site)
			if i := slices.IndexFunc(list, func(x Author) bool { return strings.EqualFold(x.Name, old) }); i >= 0 {
				avatar = list[i].Avatar
			}
		}
		if a.Photo, err = blupAvatar(site, name, avatar); err != nil {
			return err
		}
	}
	p, err := authorNotePath(site, firstOf(old, name))
	if err != nil {
		return err
	}
	if old != "" && old != name {
		np := filepath.Join(filepath.Dir(p), safeFileName(name)+".md")
		if err := os.Rename(p, np); err != nil {
			return err
		}
		p = np
		renameInPosts(site, old, name)
		if strings.EqualFold(site.Author, old) {
			site.Author = name
			if err := saved(site); err != nil {
				return err
			}
		}
	}
	social := []string{}
	for _, s := range a.Social {
		if s = strings.TrimSpace(s); s != "" {
			social = append(social, s)
		}
	}
	if err := obsidian.SetProperties(p,
		textProp("photo", a.Photo),
		textProp("role", a.Role),
		obsidian.Prop{Key: "social", Value: social}); err != nil {
		return err
	}
	return obsidian.SetBody(p, strings.TrimSpace(a.Bio)+"\n")
}

// renameInPosts swaps old for name in every note's "authors" list.
func renameInPosts(site *config.Site, old, name string) {
	eachNote(site, func(p string, n obsidian.Note) {
		list := n.List("authors")
		changed := false
		for i, a := range list {
			if strings.EqualFold(strings.TrimSpace(a), old) {
				list[i], changed = name, true
			}
		}
		if changed {
			obsidian.SetProperties(p, obsidian.Prop{Key: "authors", Value: list})
		}
	})
}

// DeleteAuthor moves the author's note to the Blup trash. The posts keep
// the name (they show it without a photo).
func DeleteAuthor(site *config.Site, name string) error {
	folder := obsidian.Sections(site.ContentDir())["authors"]
	if folder == "" {
		return nil
	}
	entries, _ := os.ReadDir(filepath.Join(site.ContentDir(), filepath.FromSlash(folder)))
	for _, e := range entries {
		if strings.EqualFold(strings.TrimSuffix(e.Name(), ".md"), name) && strings.HasSuffix(e.Name(), ".md") {
			_, err := toTrash(site, folder+"/"+e.Name())
			return err
		}
	}
	return nil
}

// AuthorPhoto copies a picture into the attachments, named after the
// author, and returns the link for the photo property.
func AuthorPhoto(site *config.Site, name, src string) (string, error) {
	if err := validImage(src); err != nil {
		return "", err
	}
	n := safeFileName(name)
	if n == "" {
		n = "author"
	}
	return copyAttachment(site, src, n)
}

// AuthorsFolder is the notes folder of the authors ("" when none yet).
func AuthorsFolder(site *config.Site) string {
	return obsidian.Sections(site.ContentDir())["authors"]
}
