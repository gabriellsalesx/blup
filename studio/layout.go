package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"

	"github.com/gabriellsalesx/blup/internal/config"
)

// The home screen's layout: sites and folders of sites, in the user's
// order. It is the Studio's own (studio.json next to Blup's settings) and
// changes nothing in the sites.

// HomeItem is a site, or a folder of sites, on the home screen.
type HomeItem struct {
	Site   string   `json:"site,omitempty"`   // a site's folder
	Folder string   `json:"folder,omitempty"` // a folder's name
	Sites  []string `json:"sites,omitempty"`  // the sites in the folder
}

func layoutFile() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "studio.json"), nil
}

// Layout is the home screen's order, with every known site in it: new
// sites at the end, sites that are gone left out.
func (s *Studio) Layout() ([]HomeItem, error) {
	g, err := config.LoadGlobal()
	if err != nil {
		return nil, err
	}
	known := g.ExistingSites()
	var saved []HomeItem
	if f, err := layoutFile(); err == nil {
		if b, err := os.ReadFile(f); err == nil {
			_ = json.Unmarshal(b, &saved)
		}
	}
	placed := map[string]bool{}
	out := []HomeItem{}
	for _, it := range saved {
		if it.Folder != "" {
			var sites []string
			for _, d := range it.Sites {
				if slices.Contains(known, d) && !placed[d] {
					placed[d] = true
					sites = append(sites, d)
				}
			}
			if len(sites) > 0 {
				out = append(out, HomeItem{Folder: it.Folder, Sites: sites})
			}
		} else if slices.Contains(known, it.Site) && !placed[it.Site] {
			placed[it.Site] = true
			out = append(out, it)
		}
	}
	for _, d := range known {
		if !placed[d] {
			out = append(out, HomeItem{Site: d})
		}
	}
	return out, nil
}

// SaveLayout saves the home screen's order.
func (s *Studio) SaveLayout(items []HomeItem) error {
	for _, it := range items {
		if it.Folder == "" && it.Site == "" {
			return errors.New("empty item")
		}
	}
	f, err := layoutFile()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
		return err
	}
	return os.WriteFile(f, b, 0o644)
}
