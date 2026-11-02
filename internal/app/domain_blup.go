package app

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/net/idna"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/i18n"
)

// A site's own domain on Blup Cloud (Max and Ultra), for Blup Studio's
// Publish screen: link it, show the DNS records to create and where it
// stands, check again and remove. The cloud proves the person controls the
// domain (a TXT record it makes) before it serves the site there.

func init() {
	registerErrors(
		knownError{blupcloud.ErrBadHostname, "bdom.bad"},
		knownError{blupcloud.ErrDomainInvalid, "bdom.bad"},
		knownError{blupcloud.ErrDomainPlan, "bdom.plan"},
		knownError{blupcloud.ErrDomainSiteMissing, "bdom.not_published"},
		knownError{blupcloud.ErrDomainTaken, "bdom.taken"},
		knownError{blupcloud.ErrDomainExists, "bdom.exists"},
		knownError{blupcloud.ErrDomainRate, "bdom.rate"},
		knownError{blupcloud.ErrDomainProvider, "bdom.provider"},
		knownError{blupcloud.ErrDomainUnavailable, "bdom.unavailable"},
	)
}

// BlupDomain is the own domain of a site on Blup Cloud.
type BlupDomain struct {
	// Available: the account's plan includes own domains (it comes from
	// the cloud, never from the window).
	Available bool `json:"available"`
	// Published: the site is on Blup Cloud (a domain is linked to a site
	// that exists there).
	Published bool `json:"published"`
	// Domain is nil when the site has none.
	Domain *blupcloud.DomainInfo `json:"domain"`
	// Apex: Domain is a root domain (meusite.com.br): its DNS must accept
	// a CNAME at the root (CNAME flattening or ALIAS).
	Apex bool `json:"apex"`
}

// normalizeBlupHost turns what was typed into the hostname the cloud
// takes: no spaces, lower case, IDNA (café.com -> xn--caf-dma.com). A
// scheme, path or port is refused, not cut, so nothing unexpected goes.
func normalizeBlupHost(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if strings.Contains(s, "://") {
		return "", errors.New(i18n.T("bdom.scheme"))
	}
	s = strings.TrimSuffix(s, ".")
	if a, err := idna.Lookup.ToASCII(s); err == nil {
		s = a
	} else {
		return "", errors.New(i18n.T("bdom.bad"))
	}
	if !blupcloud.ValidHostname(s) {
		return "", errors.New(i18n.T("bdom.bad"))
	}
	return s, nil
}

// blupDomainClient is the signed in client and whether the plan has own
// domains, for a site already on Blup Cloud.
func blupDomainClient(ctx context.Context, site *config.Site) (*blupcloud.Client, BlupDomain, error) {
	var out BlupDomain
	c, _, err := blupClient()
	if err != nil {
		return nil, out, errors.New(describe(err))
	}
	me, err := c.Me(ctx)
	if err != nil {
		return nil, out, errors.New(describe(err))
	}
	out.Available = me.Plan.Limits.CustomDomain
	out.Published = site.Deploy.Blup.ID != ""
	return c, out, nil
}

func (b *BlupDomain) set(d *blupcloud.DomainInfo) {
	b.Domain = d
	b.Apex = d != nil && isApex(d.Hostname)
}

// LoadBlupDomain says where the site's own domain stands (the cloud checks
// the DNS again).
func LoadBlupDomain(ctx context.Context, site *config.Site) (BlupDomain, error) {
	c, out, err := blupDomainClient(ctx, site)
	if err != nil || !out.Published {
		return out, err
	}
	d, err := c.Domain(ctx, site.Deploy.Blup.ID)
	if errors.Is(err, blupcloud.ErrDomainSiteMissing) {
		out.Published = false
		return out, nil
	}
	if err != nil {
		return out, errors.New(describe(err))
	}
	out.set(d)
	return out, nil
}

// SetBlupDomain links hostname to the site, or checks it again when it is
// the one already linked.
func SetBlupDomain(ctx context.Context, site *config.Site, hostname string) (BlupDomain, error) {
	h, err := normalizeBlupHost(hostname)
	if err != nil {
		return BlupDomain{}, err
	}
	c, out, err := blupDomainClient(ctx, site)
	if err != nil {
		return out, err
	}
	if !out.Published {
		return out, errors.New(i18n.T("bdom.not_published"))
	}
	if !out.Available {
		return out, errors.New(i18n.T("bdom.plan"))
	}
	d, err := c.SetDomain(ctx, site.Deploy.Blup.ID, h)
	if err != nil {
		return out, errors.New(describe(err))
	}
	out.set(&d)
	return out, nil
}

// RemoveBlupDomain unlinks the site's own domain (allowed after a
// downgrade too); the site stays at its getblup.site address.
func RemoveBlupDomain(ctx context.Context, site *config.Site) (BlupDomain, error) {
	c, out, err := blupDomainClient(ctx, site)
	if err != nil || !out.Published {
		return out, err
	}
	if err := c.RemoveDomain(ctx, site.Deploy.Blup.ID); err != nil {
		return out, errors.New(describe(err))
	}
	return out, nil
}
