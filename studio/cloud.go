package main

import (
	"context"

	"github.com/gabriellsalesx/blup/internal/app"
)

// CloudSites lists the Blup sites found on the user's GitHub (their notes
// synced to a private repository).
func (s *Studio) CloudSites() ([]app.CloudSite, error) {
	return app.FindCloudSites(context.Background())
}

// BringSite makes a site found on GitHub on this computer and opens it.
func (s *Studio) BringSite(owner, repo string) (SiteInfo, error) {
	site, err := app.BringSite(context.Background(), owner, repo)
	if err != nil {
		return SiteInfo{}, err
	}
	return info(site), nil
}

// GeneralPassword tells whether the password of protected posts for every
// site is set.
func (s *Studio) GeneralPassword() bool { return app.HasGeneralPassword() }

// SetGeneralPassword sets the password of protected posts for every site
// without its own ("" removes it).
func (s *Studio) SetGeneralPassword(password string) error { return app.SetGeneralPassword(password) }
