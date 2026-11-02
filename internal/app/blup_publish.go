package app

import (
	"context"
	"errors"
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
	)
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
