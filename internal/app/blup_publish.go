package app

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/ui"
)

// Publishing through Blup Cloud (paid plans): the site goes up minified and
// gzipped and lives at <slug>.getblup.site, with nothing to set up.

func init() {
	registerErrors(
		knownError{blupcloud.ErrPlan, "pub.blup_plan"},
		knownError{blupcloud.ErrSitesLimit, "pub.blup_limit"},
		knownError{blupcloud.ErrSlugTaken, "pub.blup_taken"},
		knownError{blupcloud.ErrFileTooLarge, "pub.blup_too_large"},
		knownError{blupcloud.ErrDeployBusy, "pub.blup_busy"},
		knownError{ErrBlupSiteGone, "pub.blup_gone"},
	)
}

// ErrBlupSiteGone: the site was taken down in the Blup Cloud panel. The
// Studio forgot it; its address is free to choose again before the next
// publish.
var ErrBlupSiteGone = errors.New("app: the site was taken down on Blup Cloud")

// blupSiteGone tells whether the account no longer has the site with id
// (taken down in the panel). An error listing the sites says nothing.
func blupSiteGone(ctx context.Context, c *blupcloud.Client, id string) bool {
	sites, _, err := c.Sites(ctx)
	return err == nil && !slices.ContainsFunc(sites, func(s blupcloud.Site) bool { return s.ID == id })
}

// forgetBlupSite drops the site's Blup Cloud site, keeping its old address
// as the suggestion: the address field is free again.
func forgetBlupSite(site *config.Site) error {
	site.Deploy.Blup = config.BlupSite{Slug: site.Deploy.Blup.Slug}
	return site.Save()
}

// RefreshBlupSite forgets the site's Blup Cloud site when the panel took it
// down, so the publishing screen lets the person choose an address again
// instead of failing at the next publish. Offline or signed out, nothing
// changes. It tells whether the site was forgotten.
func RefreshBlupSite(ctx context.Context, site *config.Site) bool {
	id := site.Deploy.Blup.ID
	if id == "" {
		return false
	}
	c, _, err := blupClient()
	if err != nil || !blupSiteGone(ctx, c, id) {
		return false
	}
	return forgetBlupSite(site) == nil
}

// blupSlug suggests an address from the site's title: "Revista Nori" ->
// "revista-nori".
func blupSlug(title string) string {
	s := Slug(title)
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case b.Len() > 0 && !strings.HasSuffix(b.String(), "-"):
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 40 {
		out = strings.Trim(out[:40], "-")
	}
	if len(out) < 3 {
		out = strings.Trim("site-"+out, "-")
	}
	return out
}

// CheckBlupSlug says whether slug.getblup.site is free for this site
// (a site already on Blup Cloud keeps its own: "yours").
func CheckBlupSlug(ctx context.Context, site *config.Site, slug string) (blupcloud.SlugCheck, error) {
	if site.Deploy.Blup.ID != "" && site.Deploy.Blup.Slug == slug {
		return blupcloud.SlugCheck{Status: "yours"}, nil
	}
	c, _, err := blupClient()
	if err != nil {
		return blupcloud.SlugCheck{}, errors.New(describe(err))
	}
	r, err := c.CheckSlug(ctx, slug)
	if err != nil {
		return r, errors.New(describe(err))
	}
	return r, nil
}

// deployBlup builds the site for its Blup Cloud address and sends it.
func deployBlup(ctx context.Context, site *config.Site) (string, error) {
	c, _, err := blupClient()
	if err != nil {
		return "", err
	}
	d := &site.Deploy.Blup
	if d.ID == "" {
		if d.Slug == "" {
			d.Slug = blupSlug(site.Title)
		}
		s, err := c.CreateSite(ctx, d.Slug, site.Title)
		if err != nil {
			return "", err
		}
		d.ID, d.URL = s.ID, s.URL
		// Kept at once: the address is the person's now, even if the
		// upload fails.
		if err := site.Save(); err != nil {
			return "", err
		}
	}
	if d.URL == "" {
		return "", errors.New(i18n.T("pub.not_set"))
	}
	// Built for its own domain once that is active, else for its
	// getblup.site address (both serve the same files).
	base, own := blupBase(ctx, c, site)
	dir, err := build(ctx, site, base)
	if err != nil {
		return "", err
	}
	var res blupcloud.DeployResult
	if err := ui.Spin(ctx, i18n.T("pub.uploading"), func(ctx context.Context) error {
		files, err := blupcloud.PackSite(dir)
		if err != nil {
			return err
		}
		res, err = c.DeploySite(ctx, d.ID, files)
		return err
	}); err != nil {
		// Taken down in the panel since the last publish: forget it, so the
		// person picks the address (the old one, or another) and publishes
		// again, instead of failing every time.
		if errors.Is(err, blupcloud.ErrSiteMissing) && blupSiteGone(ctx, c, d.ID) {
			if ferr := forgetBlupSite(site); ferr != nil {
				return "", ferr
			}
			return "", ErrBlupSiteGone
		}
		return "", err
	}
	ui.OK(i18n.T("pub.uploaded", res.Uploaded, res.Skipped))
	if res.URL != "" {
		d.URL = res.URL
	}
	d.Base = base
	if own {
		return strings.TrimSuffix(base, "/"), nil
	}
	return d.URL, nil
}
