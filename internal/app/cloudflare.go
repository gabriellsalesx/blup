package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"

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
		knownError{cfworkers.ErrTooManyFiles, "cf.err_files"},
		knownError{cfworkers.ErrQuota, "cf.err_quota"},
		knownError{cfworkers.ErrFileTooLarge, "cf.err_large"},
		knownError{cfworkers.ErrUploadExpired, "cf.err_expired"},
	)
}

// cloudflareTokenURL opens the token form pre-filled with the only
// permission needed: Account › Workers Scripts › Edit.
func cloudflareTokenURL() string {
	q := url.Values{}
	q.Set("permissionGroupKeys", `[{"key":"workers_scripts","type":"edit"}]`)
	q.Set("accountId", "*")
	q.Set("zoneId", "all")
	q.Set("name", "blup")
	return "https://dash.cloudflare.com/profile/api-tokens?" + q.Encode()
}

func cloudflareClient(ctx context.Context) (*cfworkers.Client, error) {
	creds, err := config.LoadCredentials()
	if err != nil {
		return nil, err
	}
	for {
		if creds.CloudflareToken == "" {
			link := cloudflareTokenURL()
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

func publishCloudflare(ctx context.Context, site *config.Site) (string, error) {
	c, err := cloudflareClient(ctx)
	if err != nil {
		return "", err
	}
	d := &site.Deploy.Cloudflare

	if d.Account == "" {
		var accounts []cfworkers.Account
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

	first := d.Project == ""
	if first {
		d.Project = cfworkers.ScriptNameFromTitle(site.Title)
		if err := ui.Ask(i18n.T("cf.name_q"), i18n.T("cf.name_help"), d.Project, &d.Project, cfworkers.ValidateScriptName); err != nil {
			return "", err
		}
	}

	var target cfworkers.Site
	for {
		err = ui.Spin(ctx, i18n.T("cf.preparing"), func(ctx context.Context) error {
			target, err = c.EnsureSite(ctx, d.Account, d.Project)
			return err
		})
		if !errors.Is(err, cfworkers.ErrNoWorkersSubdomain) {
			break
		}
		// A brand-new account has no workers.dev subdomain until the
		// Workers page is opened once in the dashboard.
		link := cfworkers.WorkersOnboardingURL(d.Account)
		ui.OpenBrowser(link)
		if err := ui.Explain(i18n.T("cf.subdomain_title"), i18n.T("cf.subdomain_steps", link)); err != nil {
			return "", err
		}
	}
	if err != nil {
		return "", err
	}
	if first && target.Exists {
		ok, err := ui.Confirm(i18n.T("cf.exists_q", d.Project), i18n.T("cf.exists_help"), false)
		if err != nil || !ok {
			d.Project = ""
			if err == nil {
				err = errors.New(i18n.T("cancelled"))
			}
			return "", err
		}
	}
	d.Subdomain = target.Host

	live := siteURL(site, target.URL)
	dir, err := build(ctx, site, live)
	if err != nil {
		return "", err
	}
	var res cfworkers.Result
	if err := ui.Spin(ctx, i18n.T("pub.uploading"), func(ctx context.Context) error {
		res, err = c.Deploy(ctx, d.Account, d.Project, dir, nil)
		return err
	}); err != nil {
		return "", err
	}
	ui.OK(i18n.T("pub.uploaded", res.Uploaded, res.Skipped))
	if site.Domain != "" {
		ui.Hint(fmt.Sprintf(i18n.T("cf.domain_hint"), site.Domain, target.URL))
	}
	return live, nil
}
