package app

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gabriellsalesx/blup/internal/config"
)

// FindSite finds the Blup site of a folder the user picked: the site's own
// folder (with blup.toml), or its notes (a vault or a folder of it). It
// looks in the folder, a few levels around it and the usual places for
// sites. "" when none matches.
func FindSite(dir string) string {
	dir = filepath.Clean(expandHome(dir))
	if fileExists(filepath.Join(dir, config.SiteFile)) {
		return dir
	}
	home, _ := os.UserHomeDir()
	roots := []string{dir}
	for p, i := filepath.Dir(dir), 0; i < 3 && p != filepath.Dir(p); p, i = filepath.Dir(p), i+1 {
		roots = append(roots, p)
	}
	for _, name := range []string{"Sites", "sites", "Blup"} {
		roots = append(roots, filepath.Join(home, name))
	}
	if g, err := config.LoadGlobal(); err == nil {
		roots = append(roots, g.Sites...)
	}
	best, bestScore := "", 0
	seen := map[string]bool{}
	for _, root := range roots {
		for _, site := range siteFiles(root, 3) {
			if seen[site] {
				continue
			}
			seen[site] = true
			s, err := config.LoadSite(site)
			if err != nil {
				continue
			}
			content := filepath.Clean(s.ContentDir())
			score := 0
			switch {
			case content == dir:
				score = 3 // the notes of the site
			case strings.HasPrefix(dir, content+string(filepath.Separator)):
				score = 2 // a folder inside its notes
			case strings.HasPrefix(content, dir+string(filepath.Separator)), strings.HasPrefix(site, dir+string(filepath.Separator)):
				score = 1 // a folder that holds the site or its notes
			}
			if score > bestScore {
				best, bestScore = site, score
			}
		}
	}
	return best
}

// siteFiles lists the folders with a blup.toml under root, depth levels down
// (skipping hidden folders and the site's generated ones).
func siteFiles(root string, depth int) []string {
	out := []string{}
	base := strings.Count(filepath.Clean(root), string(filepath.Separator))
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if p != root && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "public" || name == "resources" || name == "themes") {
				return filepath.SkipDir
			}
			if strings.Count(p, string(filepath.Separator))-base > depth {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == config.SiteFile {
			out = append(out, filepath.Dir(p))
		}
		return nil
	})
	return out
}

// RememberSite adds a site folder to Blup's list of sites.
func RememberSite(dir string) error {
	g, err := config.LoadGlobal()
	if err != nil {
		return err
	}
	g.AddSite(dir)
	return g.Save()
}
