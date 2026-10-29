package app

import (
	"context"
	"time"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/deploy/github"
)

// The custom domain for Blup Studio's Publish screen, without questions:
// connect it where the site is published and say what is still missing,
// as data (the CLI's editDomain asks and prints the same steps).

// DNSRecord is a record to create where the domain was bought.
type DNSRecord struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// DomainStatus is where a custom domain stands.
type DomainStatus struct {
	Domain string `json:"domain"`
	// State: "none" (no domain), "unpublished" (connected on the first
	// publish), "active", "dns" (records to create), "waiting" (the host
	// is setting it up) or "apex-cloudflare" (a root domain on Cloudflare
	// Pages: it must be added to the Cloudflare account).
	State   string      `json:"state"`
	Records []DNSRecord `json:"records"`
	WWW     string      `json:"www"` // for apex-cloudflare: the www. address that works with a record
	Message string      `json:"message"`
	HTTPS   bool        `json:"https"` // GitHub: HTTPS is on
}

// SetDomain saves a new custom domain (leaving the old one) and connects it.
func SetDomain(ctx context.Context, site *config.Site, value string) (DomainStatus, error) {
	d, err := normalizeDomain(value)
	if err != nil {
		return DomainStatus{}, err
	}
	if old := site.Domain; old != "" && old != d {
		_ = disconnectDomainQuiet(ctx, site, old)
	}
	site.Domain = d
	if err := site.Save(); err != nil {
		return DomainStatus{}, err
	}
	return CheckDomain(ctx, site)
}

// RemoveDomain goes back to the free address.
func RemoveDomain(ctx context.Context, site *config.Site) error {
	if site.Domain == "" {
		return nil
	}
	if err := disconnectDomainQuiet(ctx, site, site.Domain); err != nil {
		return err
	}
	site.Domain = ""
	return site.Save()
}

// CheckDomain connects the domain (again) and reports where it stands.
func CheckDomain(ctx context.Context, site *config.Site) (DomainStatus, error) {
	st := DomainStatus{Domain: site.Domain, State: "none", Records: []DNSRecord{}}
	if site.Domain == "" {
		return st, nil
	}
	switch site.Deploy.Provider {
	case "cloudflare":
		d := site.Deploy.Cloudflare
		if d.Account == "" || d.Project == "" {
			st.State = "unpublished"
			return st, nil
		}
		c, err := cloudflareClient(ctx)
		if err != nil {
			return st, err
		}
		dom, err := c.AddDomain(ctx, d.Account, d.Project, site.Domain)
		if err != nil {
			return st, err
		}
		if !dom.Active() {
			dom, _ = c.RetryDomain(ctx, d.Account, d.Project, site.Domain)
		}
		st.Message = dom.Message
		pagesDev := firstOf(d.Subdomain, d.Project+".pages.dev")
		switch {
		case dom.Active():
			st.State = "active"
		case dom.OnCloudflare:
			st.State = "waiting"
		case isApex(site.Domain):
			st.State, st.WWW = "apex-cloudflare", "www."+site.Domain
			st.Records = append(st.Records, DNSRecord{"CNAME", "www", pagesDev})
		default:
			st.State = "dns"
			st.Records = append(st.Records, DNSRecord{"CNAME", subLabel(site.Domain), pagesDev})
		}
		return st, nil
	case "github":
		d := site.Deploy.GitHub
		if d.Owner == "" || d.Repo == "" {
			st.State = "unpublished"
			return st, nil
		}
		c, _, err := githubClient(ctx)
		if err != nil {
			return st, err
		}
		cur, err := c.GetPagesDomain(ctx, d.Owner, d.Repo)
		if err != nil {
			return st, err
		}
		if cur.CNAME != site.Domain {
			if err := c.SetPagesDomain(ctx, d.Owner, d.Repo, site.Domain); err != nil {
				return st, err
			}
		}
		var h github.DomainHealth
		for i := 0; i < 5; i++ { // the first request only starts GitHub's check
			if h, err = c.PagesHealth(ctx, d.Owner, d.Repo); err != nil || !h.Pending {
				break
			}
			select {
			case <-ctx.Done():
				return st, ctx.Err()
			case <-time.After(2 * time.Second):
			}
		}
		if err != nil {
			return st, err
		}
		if h.Ready {
			st.State, st.HTTPS = "active", cur.HTTPSEnforced
			if !cur.HTTPSEnforced && h.HTTPSEligible && c.EnforceHTTPS(ctx, d.Owner, d.Repo) == nil {
				st.HTTPS = true
			}
			return st, nil
		}
		st.State = "dns"
		if isApex(site.Domain) {
			for _, ip := range githubPagesIPs {
				st.Records = append(st.Records, DNSRecord{"A", "@", ip})
			}
			st.Records = append(st.Records, DNSRecord{"CNAME", "www", d.Owner + ".github.io"})
		} else {
			st.Records = append(st.Records, DNSRecord{"CNAME", subLabel(site.Domain), d.Owner + ".github.io"})
		}
		return st, nil
	}
	st.State = "unpublished"
	return st, nil
}

// disconnectDomainQuiet removes domain from the host, without spinners.
func disconnectDomainQuiet(ctx context.Context, site *config.Site, domain string) error {
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
		return c.RemoveDomain(ctx, d.Account, d.Project, domain)
	case "github":
		d := site.Deploy.GitHub
		if d.Owner == "" || d.Repo == "" {
			return nil
		}
		c, _, err := githubClient(ctx)
		if err != nil {
			return err
		}
		return c.SetPagesDomain(ctx, d.Owner, d.Repo, "")
	}
	return nil
}
