package app

import (
	"context"
	"errors"

	"charm.land/huh/v2"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/deploy/cfpages"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/ui"
)

func init() {
	registerErrors(
		knownError{cfpages.ErrInvalidToken, "cf.err_token"},
		knownError{cfpages.ErrMissingPermission, "cf.err_permission"},
		knownError{cfpages.ErrProjectNameTaken, "cf.err_taken"},
		knownError{cfpages.ErrTooManyFiles, "cf.err_files"},
		knownError{cfpages.ErrQuota, "cf.err_quota"},
		knownError{cfpages.ErrFileTooLarge, "cf.err_large"},
	)
}

func cloudflareClient(ctx context.Context) (*cfpages.Client, error) {
	creds, err := config.LoadCredentials()
	if err != nil {
		return nil, err
	}
	for {
		if creds.CloudflareToken == "" {
			link := cfpages.TokenTemplateURL
			if err := ui.Explain(i18n.T("cf.token_title"), i18n.T("cf.token_steps")); err != nil {
				return nil, err
			}
			ui.OpenBrowser(link)
			ui.Hint(i18n.T("browser_fallback", link))
			if err := ui.AskSecret(i18n.T("cf.token_q"), i18n.T("cf.token_help"), &creds.CloudflareToken); err != nil {
				return nil, err
			}
		}
		c := &cfpages.Client{Token: creds.CloudflareToken}
		err := ui.Spin(ctx, i18n.T("cf.checking"), c.VerifyToken)
		if err == nil {
			return c, creds.Save()
		}
		ui.Error(describe(err))
		creds.CloudflareToken = ""
		if ok, _ := ui.Confirm(i18n.T("cf.retry"), "", true); !ok {
			return nil, err
		}
	}
}

// publishCloudflare sends the site to Cloudflare Pages (free, site.pages.dev).
func publishCloudflare(ctx context.Context, site *config.Site) (string, error) {
	c, err := cloudflareClient(ctx)
	if err != nil {
		return "", err
	}
	d := &site.Deploy.Cloudflare

	if d.Account == "" {
		var accounts []cfpages.Account
		if err := ui.Spin(ctx, i18n.T("cf.accounts"), func(ctx context.Context) error {
			accounts, err = c.Accounts(ctx)
			return err
		}); err != nil {
			return "", err
		}
		switch len(accounts) {
		case 0:
			return "", errors.New(i18n.T("cf.err_permission"))
		case 1:
			d.Account = accounts[0].ID
		default:
			d.Account = accounts[0].ID
			var opts []huh.Option[string]
			for _, a := range accounts {
				opts = append(opts, ui.Option(a.Name, a.ID))
			}
			if err := ui.Choose(i18n.T("cf.account_q"), "", &d.Account, opts...); err != nil {
				return "", err
			}
		}
	}

	if d.Project == "" {
		d.Project = cfpages.ProjectNameFromTitle(site.Title)
		if err := ui.Ask(i18n.T("cf.name_q"), i18n.T("cf.name_help"), d.Project, &d.Project, cfpages.ValidateProjectName); err != nil {
			return "", err
		}
	}

	// The real address can differ from the name: when "name.pages.dev" is
	// taken elsewhere, Cloudflare adds a suffix. It is needed before building.
	var project cfpages.Project
	if err := ui.Spin(ctx, i18n.T("cf.preparing"), func(ctx context.Context) error {
		project, err = c.EnsureProject(ctx, d.Account, d.Project)
		return err
	}); err != nil {
		return "", err
	}
	d.Subdomain = project.Subdomain

	live := siteURL(site, project.URL())
	dir, err := build(ctx, site, live)
	if err != nil {
		return "", err
	}
	var res cfpages.Result
	if err := ui.Spin(ctx, i18n.T("pub.uploading"), func(ctx context.Context) error {
		res, err = c.Deploy(ctx, d.Account, d.Project, dir, nil)
		return err
	}); err != nil {
		return "", err
	}
	ui.OK(i18n.T("pub.uploaded", res.Uploaded, res.Skipped))
	return live, nil
}
