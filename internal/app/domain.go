package app

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/idna"
	"golang.org/x/net/publicsuffix"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/deploy/github"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/ui"
)

// GitHub Pages addresses for a root domain (A records).
var githubPagesIPs = []string{"185.199.108.153", "185.199.109.153", "185.199.110.153", "185.199.111.153"}

var hostRe = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z][a-z0-9-]{0,61}[a-z0-9]$`)

// normalizeDomain accepts what people paste ("https://www.MeuSite.com.br/")
// and returns the bare host, in ASCII.
func normalizeDomain(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(strings.TrimPrefix(s, "https://"), "http://")
	s, _, _ = strings.Cut(s, "/")
	s = strings.TrimSuffix(s, ".")
	if a, err := idna.Lookup.ToASCII(s); err == nil {
		s = a
	}
	if !hostRe.MatchString(s) || strings.HasSuffix(s, ".pages.dev") || strings.HasSuffix(s, ".github.io") {
		return "", errors.New(i18n.T("domain.bad"))
	}
	if _, icann := publicsuffix.PublicSuffix(s); !icann {
		return "", errors.New(i18n.T("domain.bad"))
	}
	return s, nil
}

// isApex reports whether d is a root domain (meusite.com.br), not a
// subdomain (blog.meusite.com.br).
func isApex(d string) bool {
	root, err := publicsuffix.EffectiveTLDPlusOne(d)
	return err == nil && root == d
}

// subLabel is the part of d before its root domain ("blog" in
// blog.meusite.com.br), the "name" of its DNS record.
func subLabel(d string) string {
	root, err := publicsuffix.EffectiveTLDPlusOne(d)
	if err != nil || root == d {
		return "@"
	}
	return strings.TrimSuffix(d, "."+root)
}

// editDomain is "Custom domain" in customize: connect, check or remove.
func editDomain(ctx context.Context, site *config.Site) error {
	action := "set"
	if site.Domain != "" {
		action = "check"
		if err := ui.Choose(i18n.T("domain.q", site.Domain), "", &action,
			ui.Option(i18n.T("domain.check"), "check"),
			ui.Option(i18n.T("domain.change"), "set"),
			ui.Option(i18n.T("domain.remove"), "remove"),
		); err != nil {
			return err
		}
	}
	switch action {
	case "check":
		return connectDomain(ctx, site)
	case "remove":
		return removeDomain(ctx, site)
	}

	if err := ui.Explain(i18n.T("domain.about_title"), i18n.T("domain.about")); err != nil {
		return err
	}
	value := site.Domain
	if err := ui.Ask(i18n.T("domain.ask"), i18n.T("domain.ask_help"), "meusite.com.br", &value, func(s string) error {
		_, err := normalizeDomain(s)
		return err
	}); err != nil {
		return err
	}
	d, _ := normalizeDomain(value)
	if old := site.Domain; old != "" && old != d {
		if err := disconnectDomain(ctx, site, old); err != nil {
			ui.Warn(describe(err))
		}
	}
	site.Domain = d
	if err := site.Save(); err != nil {
		return err
	}
	if site.Deploy.Provider == "" {
		ui.OK(i18n.T("domain.saved_unpublished", d))
		return nil
	}
	if err := connectDomain(ctx, site); err != nil {
		return err
	}
	ui.Hint(i18n.T("domain.republish"))
	return nil
}

// connectDomain connects site.Domain where the site is published and shows
// what is still missing (usually a DNS record). It is safe to run again:
// "check" in the menu and every publish call it.
func connectDomain(ctx context.Context, site *config.Site) error {
	if site.Domain == "" {
		return nil
	}
	switch site.Deploy.Provider {
	case "cloudflare":
		return connectCloudflareDomain(ctx, site)
	case "github":
		return connectGitHubDomain(ctx, site)
	}
	ui.OK(i18n.T("domain.saved_unpublished", site.Domain))
	return nil
}

func connectCloudflareDomain(ctx context.Context, site *config.Site) error {
	d := site.Deploy.Cloudflare
	if d.Account == "" || d.Project == "" {
		ui.OK(i18n.T("domain.saved_unpublished", site.Domain))
		return nil
	}
	c, err := cloudflareClient(ctx)
	if err != nil {
		return err
	}
	var st struct {
		active, onCF bool
		msg          string
	}
	if err := ui.Spin(ctx, i18n.T("domain.connecting"), func(ctx context.Context) error {
		dom, err := c.AddDomain(ctx, d.Account, d.Project, site.Domain)
		if err != nil {
			return err
		}
		if !dom.Active() {
			dom, _ = c.RetryDomain(ctx, d.Account, d.Project, site.Domain)
		}
		st.active, st.onCF, st.msg = dom.Active(), dom.OnCloudflare, dom.Message
		return nil
	}); err != nil {
		return err
	}
	if st.active {
		ui.OK(i18n.T("domain.active", site.Domain))
		return nil
	}
	pagesDev := d.Subdomain
	if pagesDev == "" {
		pagesDev = d.Project + ".pages.dev"
	}
	switch {
	case st.onCF:
		ui.Hint(i18n.T("domain.cf_waiting", site.Domain))
	case isApex(site.Domain):
		ui.Explain(i18n.T("domain.dns_title"), i18n.T("domain.cf_apex", site.Domain, "www."+site.Domain, pagesDev))
	default:
		ui.Explain(i18n.T("domain.dns_title"), i18n.T("domain.cf_cname", subLabel(site.Domain), pagesDev))
	}
	if st.msg != "" {
		ui.Hint(st.msg)
	}
	return nil
}

func connectGitHubDomain(ctx context.Context, site *config.Site) error {
	d := site.Deploy.GitHub
	if d.Owner == "" || d.Repo == "" {
		ui.OK(i18n.T("domain.saved_unpublished", site.Domain))
		return nil
	}
	c, _, err := githubClient(ctx)
	if err != nil {
		return err
	}
	var h github.DomainHealth
	var cur github.PagesDomain
	if err := ui.Spin(ctx, i18n.T("domain.connecting"), func(ctx context.Context) error {
		if cur, err = c.GetPagesDomain(ctx, d.Owner, d.Repo); err != nil {
			return err
		}
		if cur.CNAME != site.Domain {
			if err := c.SetPagesDomain(ctx, d.Owner, d.Repo, site.Domain); err != nil {
				return err
			}
		}
		// The first health request only starts GitHub's check.
		for i := 0; i < 5; i++ {
			if h, err = c.PagesHealth(ctx, d.Owner, d.Repo); err != nil || !h.Pending {
				return err
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(2 * time.Second):
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if h.Ready {
		ui.OK(i18n.T("domain.active", site.Domain))
		if !cur.HTTPSEnforced {
			if h.HTTPSEligible && c.EnforceHTTPS(ctx, d.Owner, d.Repo) == nil {
				ui.OK(i18n.T("domain.https_on"))
			} else {
				ui.Hint(i18n.T("domain.https_wait"))
			}
		}
		return nil
	}
	if isApex(site.Domain) {
		ui.Explain(i18n.T("domain.dns_title"), i18n.T("domain.gh_apex", site.Domain, strings.Join(githubPagesIPs, "\n  "), "www."+site.Domain, d.Owner+".github.io"))
	} else {
		ui.Explain(i18n.T("domain.dns_title"), i18n.T("domain.gh_cname", subLabel(site.Domain), d.Owner+".github.io"))
	}
	return nil
}

// disconnectDomain removes domain from where the site is published.
func disconnectDomain(ctx context.Context, site *config.Site, domain string) error {
	switch site.Deploy.Provider {
	case "cloudflare":
		d := site.Deploy.Cloudflare
		if d.Account == "" || d.Project == "" {
			return nil
		}
		c, err := cloudflareClient(ctx)
		if err != nil {
			return err
		}
		return ui.Spin(ctx, i18n.T("domain.removing"), func(ctx context.Context) error {
			return c.RemoveDomain(ctx, d.Account, d.Project, domain)
		})
	case "github":
		d := site.Deploy.GitHub
		if d.Owner == "" || d.Repo == "" {
			return nil
		}
		c, _, err := githubClient(ctx)
		if err != nil {
			return err
		}
		return ui.Spin(ctx, i18n.T("domain.removing"), func(ctx context.Context) error {
			return c.SetPagesDomain(ctx, d.Owner, d.Repo, "")
		})
	}
	return nil
}

func removeDomain(ctx context.Context, site *config.Site) error {
	if ok, err := ui.Confirm(i18n.T("domain.remove_q", site.Domain), i18n.T("domain.remove_help"), false); err != nil || !ok {
		return err
	}
	if err := disconnectDomain(ctx, site, site.Domain); err != nil {
		return err
	}
	site.Domain = ""
	if err := site.Save(); err != nil {
		return err
	}
	ui.OK(i18n.T("domain.removed"))
	ui.Hint(i18n.T("domain.republish"))
	return nil
}
