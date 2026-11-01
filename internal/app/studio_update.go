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
}

// CheckUpdate looks for a newer release: from the day's cache, or from
// GitHub when the cache is older than a day or force is set (the
// "Check for updates" button). A development build never checks.
func CheckUpdate(ctx context.Context, force bool) (UpdateInfo, error) {
	info := UpdateInfo{Current: Version}
	if Version == "dev" || os.Getenv("BLUP_NO_UPDATE_CHECK") != "" {
		return info, nil
	}
	cache := update.LoadCache()
	if force || time.Since(cache.Checked) > 24*time.Hour {
		cctx, cancel := context.WithTimeout(ctx, 8*time.Second)
		rel, err := (&update.Client{}).Latest(cctx)
		cancel()
		if err != nil {
			return info, err
		}
		cache.Checked, cache.Latest, cache.Notes, cache.URL = time.Now(), rel.Version, rel.Notes, rel.URL
		_ = cache.Save()
	}
	info.Latest, info.Notes, info.URL = cache.Latest, cache.Notes, cache.URL
	info.Newer = update.Newer(cache.Latest, Version)
	return info, nil
}
