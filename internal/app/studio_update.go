package app

import (
	"context"
	"os"
	"time"

	"github.com/gabriellsalesx/blup/internal/update"
)

// UpdateInfo tells the Studio about a newer Blup: Newer when the latest
// release is above this build, with its notes and page.
type UpdateInfo struct {
	Current string `json:"current"`
	Latest  string `json:"latest"`
	Newer   bool   `json:"newer"`
	Notes   string `json:"notes"`
	URL     string `json:"url"`
	// UpdatesBy: a store or the system's packages bring new versions (see
	// UpdatesBy); then nothing is looked up.
	UpdatesBy string `json:"updatesBy"`
	// SelfUpdate: this install can download and put the new version in
	// place itself (StudioSelfUpdate); else the site's download page.
	SelfUpdate bool `json:"selfUpdate"`
	// Changes: what every version after this one changed, up to the latest
	// (newest first), in the Studio's language.
	Changes []update.VersionNotes `json:"changes"`
}

// CheckUpdate looks for a newer release: from the day's cache, or from
// GitHub when the cache is older than a day or force is set (the
// "Check for updates" button). A development build never checks.
func CheckUpdate(ctx context.Context, force bool, lang string) (UpdateInfo, error) {
	info := UpdateInfo{Current: Version, UpdatesBy: UpdatesBy(), Changes: []update.VersionNotes{}}
	if Version == "dev" || info.UpdatesBy != "" || os.Getenv("BLUP_NO_UPDATE_CHECK") != "" {
		return info, nil
	}
	cache := update.LoadCache()
	if force || time.Since(cache.Checked) > 24*time.Hour {
		// The recent versions in one call: the latest, and the notes of
		// those a Studio a few versions behind skipped.
		cctx, cancel := context.WithTimeout(ctx, 8*time.Second)
		rels, err := (&update.Client{}).Releases(cctx)
		cancel()
		if err != nil {
			return info, err
		}
		if len(rels) == 0 {
			return info, nil
		}
		cache.History = cache.History[:0]
		for _, r := range rels {
			cache.History = append(cache.History, update.VersionNotes{Version: r.Version, Notes: r.Notes})
		}
		cache.Checked, cache.Latest, cache.Notes, cache.URL = time.Now(), rels[0].Version, rels[0].Notes, rels[0].URL
		_ = cache.Save()
	}
	info.Latest, info.Notes, info.URL = cache.Latest, cache.Notes, cache.URL
	info.Newer = update.Newer(cache.Latest, Version)
	if info.Newer {
		info.Changes = update.Since(cache.History, Version, cache.Latest, lang)
		// A cache from before the history was kept: the latest's notes.
		if len(info.Changes) == 0 && cache.Notes != "" {
			info.Changes = []update.VersionNotes{{Version: cache.Latest, Notes: update.NotesFor(cache.Notes, lang)}}
		}
	}
	kind, _ := StudioSelfUpdate()
	info.SelfUpdate = kind != ""
	return info, nil
}
