package app

import (
	"context"
	"errors"

	"charm.land/huh/v2"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/deploy/cfworkers"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/ui"
)

func init() {
	registerErrors(
		knownError{cfworkers.ErrInvalidToken, "cf.err_token"},
		knownError{cfworkers.ErrMissingPermission, "cf.err_permission"},
		knownError{cfworkers.ErrNameTaken, "cf.err_taken"},
		knownError{cfworkers.ErrNoSubdomain, "cf.err_no_subdomain"},
		knownError{cfworkers.ErrSubdomainTaken, "cf.err_subdomain_taken"},
		knownError{cfworkers.ErrTooManyFiles, "cf.err_files"},
		knownError{cfworkers.ErrQuota, "cf.err_quota"},
		knownError{cfworkers.ErrFileTooLarge, "cf.err_large"},
	)
}

// cloudflareClient is the Cloudflare client of the saved key, asking for
// one in the terminal when there is none or it stopped working. Without a
// terminal (Blup Studio) it never asks: savedCloudflare's clear error
// instead.
func cloudflareClient(ctx context.Context) (*cfworkers.Client, error) {
	if !ui.Interactive() {
		return savedCloudflare()
	}
	creds, err := config.LoadCredentials()
	if err != nil {
		return nil, err
	}
	for {
		if creds.CloudflareToken == "" {
			link := cfworkers.TokenTemplateURL
			if err := ui.Explain(i18n.T("cf.token_title"), i18n.T("cf.token_steps")); err != nil {
				return nil, err
			}
			ui.OpenBrowser(link)
			ui.Hint(i18n.T("browser_fallback", link))
			if err := ui.AskSecret(i18n.T("cf.token_q"), i18n.T("cf.token_help"), &creds.CloudflareToken); err != nil {
				return nil, err
			}
		}
		c := &cfworkers.Client{Token: creds.CloudflareToken}
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

// publishCloudflare sends the site to Cloudflare Workers (free,
// site.account.workers.dev), asking what is still missing.
func publishCloudflare(ctx context.Context, site *config.Site) (string, error) {
	c, err := cloudflareClient(ctx)
	if err != nil {
		return "", err
	}
	d := &site.Deploy.Cloudflare

	var accounts []cfworkers.Account
	if err := ui.Spin(ctx, i18n.T("cf.accounts"), func(ctx context.Context) error {
		accounts, err = c.Accounts(ctx)
		return err
	}); err != nil {
		return "", err
	}
	if d.Account == "" {
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

	// A new Cloudflare account has no workers.dev address yet: it is
	// chosen once, for every Worker of the account.
	var sub string
	if err := ui.Spin(ctx, i18n.T("cf.preparing"), func(ctx context.Context) error {
		sub, err = c.Subdomain(ctx, d.Account)
		return err
	}); errors.Is(err, cfworkers.ErrNoSubdomain) {
		accountName := ""
		for _, a := range accounts {
			if a.ID == d.Account {
				accountName = a.Name
			}
		}
		name := cfworkers.NameFromTitle(accountName)
		for {
			if err := ui.Ask(i18n.T("cf.subdomain_q"), i18n.T("cf.subdomain_help"), name, &name, cfworkers.ValidateSubdomain); err != nil {
				return "", err
			}
			sub, err = c.RegisterSubdomain(ctx, d.Account, name)
			if !errors.Is(err, cfworkers.ErrSubdomainTaken) {
				break
			}
			ui.Error(describe(err))
		}
		if err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}
	d.Subdomain = sub

	if d.Worker == "" {
		d.Worker = cfworkers.NameFromTitle(site.Title)
		if err := ui.Ask(i18n.T("cf.name_q"), i18n.T("cf.name_help", d.Subdomain), d.Worker, &d.Worker, cfworkers.ValidateName); err != nil {
			return "", err
		}
	}

	return deployCloudflare(ctx, site, c)
}

// deployCloudflare builds the site and sends it to the Worker already
// chosen (site.Deploy.Cloudflare), without questions.
func deployCloudflare(ctx context.Context, site *config.Site, c *cfworkers.Client) (string, error) {
	d := &site.Deploy.Cloudflare
	if d.Account == "" || d.Worker == "" {
		return "", errors.New(i18n.T("pub.not_set"))
	}
	// The address is needed before building: it has the account's
	// workers.dev subdomain, which the person can change in Cloudflare.
	var sub string
	var err error
	if err := ui.Spin(ctx, i18n.T("cf.preparing"), func(ctx context.Context) error {
		if sub, err = c.Subdomain(ctx, d.Account); err != nil {
			return err
		}
		// Worker names are per account, and publishing replaces the Worker
		// with the name: before the first publish, it must be free.
		if d.URL == "" {
			exists, err := c.Exists(ctx, d.Account, d.Worker)
			if err != nil {
				return err
			}
			if exists {
				return cfworkers.ErrNameTaken
			}
		}
		return nil
	}); err != nil {
		return "", err
	}
	d.Subdomain = sub

	live := cfworkers.URL(d.Worker, sub)
	dir, err := build(ctx, site, live)
	if err != nil {
		return "", err
	}
	var res cfworkers.Result
	if err := ui.Spin(ctx, i18n.T("pub.uploading"), func(ctx context.Context) error {
		res, err = c.Deploy(ctx, d.Account, d.Worker, dir, nil)
		return err
	}); err != nil {
		return "", err
	}
	d.URL = live
	ui.OK(i18n.T("pub.uploaded", res.Uploaded, res.Skipped))
	return live, nil
}

// CloudflareSubdomain is the workers.dev subdomain of a Cloudflare account
// ("" when it has none yet), for the Studio's Publish screen.
func CloudflareSubdomain(ctx context.Context, account string) (string, error) {
	c, err := savedCloudflare()
	if err != nil {
		return "", err
	}
	sub, err := c.Subdomain(ctx, account)
	if errors.Is(err, cfworkers.ErrNoSubdomain) {
		return "", nil
	}
	if err != nil {
		return "", errors.New(describe(err))
	}
	return sub, nil
}

// SetCloudflareSubdomain gives a Cloudflare account its workers.dev
// subdomain (only when it has none).
func SetCloudflareSubdomain(ctx context.Context, account, name string) (string, error) {
	c, err := savedCloudflare()
	if err != nil {
		return "", err
	}
	if sub, err := c.Subdomain(ctx, account); err == nil {
		return sub, nil
	} else if !errors.Is(err, cfworkers.ErrNoSubdomain) {
		return "", errors.New(describe(err))
	}
	sub, err := c.RegisterSubdomain(ctx, account, name)
	if err != nil {
		return "", errors.New(describe(err))
	}
	return sub, nil
}
