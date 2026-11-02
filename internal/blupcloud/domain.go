package blupcloud

import (
	"context"
	"errors"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

// A site's own domain on Blup Cloud (Max and Ultra), besides
// <slug>.getblup.site, which keeps working. One hostname per site. The
// contract is the cloud's (cloud/worker, agreed with its team):
//
//	GET    /api/sites/:id/domain  -> DomainInfo | null
//	POST   /api/sites/:id/domain  {hostname} -> DomainInfo (the same hostname
//	                              again checks it again)
//	DELETE /api/sites/:id/domain  -> {ok: true}
//
// Ownership: the cloud makes a TXT record _blup.<hostname> with a random
// value for each link, so another account cannot take a domain it does not
// control; the first answer is "pending_ownership" with that TXT and the
// CNAME. The CNAME's target comes in the answer and is never fixed here.

// Domain errors, by the cloud's code.
var (
	// ErrDomainInvalid (400 INVALID): the cloud did not accept the hostname.
	ErrDomainInvalid = errors.New("blupcloud: invalid domain")
	// ErrDomainPlan (403 PLAN): own domains come with Max and Ultra.
	ErrDomainPlan = errors.New("blupcloud: the plan does not include own domains")
	// ErrDomainSiteMissing (404 NOT_FOUND): the site is not on Blup Cloud
	// (or is not this account's).
	ErrDomainSiteMissing = errors.New("blupcloud: site not found")
	// ErrDomainTaken (409 DOMAIN_TAKEN): another site has that domain.
	ErrDomainTaken = errors.New("blupcloud: the domain is linked to another site")
	// ErrDomainExists (409 DOMAIN_EXISTS): the site has another domain;
	// remove it first.
	ErrDomainExists = errors.New("blupcloud: the site already has another domain")
	// ErrDomainRate (429 TOO_MANY_REQUESTS).
	ErrDomainRate = errors.New("blupcloud: too many requests, try again in a moment")
	// ErrDomainProvider (502 DOMAIN_PROVIDER): the cloud's provider failed.
	ErrDomainProvider = errors.New("blupcloud: the domain provider failed")
	// ErrDomainUnavailable (503 DOMAIN_UNAVAILABLE): own domains are off
	// for now.
	ErrDomainUnavailable = errors.New("blupcloud: own domains are unavailable right now")
	// ErrBadHostname: refused here, before anything is sent.
	ErrBadHostname = errors.New("blupcloud: not a domain name")
)

// DomainRecord is a DNS record to create where the domain was bought.
type DomainRecord struct {
	Type  string `json:"type"` // "TXT" or "CNAME"
	Name  string `json:"name"`
	Value string `json:"value"`
}

// DomainInfo is a site's own domain and where it stands. Status:
// "pending_ownership" (the TXT record is missing), "pending" (being set up),
// "active", "error" or "disabled" (turned off when the grace after a plan
// without own domains ran out; a new link on Max or Ultra opens it again).
// SSL: "pending", "active" or "error".
type DomainInfo struct {
	Hostname string         `json:"hostname"`
	Status   string         `json:"status"`
	SSL      string         `json:"ssl"`
	Records  []DomainRecord `json:"records"`
	// DisableAt (ISO 8601, UTC) is set while the account's plan no longer
	// includes own domains: the domain keeps working until then (14 days).
	// Nil otherwise.
	DisableAt *string `json:"disableAt"`
}

// Official tells whether the domain is the site's address for links
// (canonical, feeds, sitemap): active with its HTTPS, and the plan
// includes it or the grace after a downgrade has not run out.
func (d *DomainInfo) Official(planAllows bool, now time.Time) bool {
	if d == nil || d.Status != "active" || d.SSL != "active" {
		return false
	}
	if planAllows {
		return true
	}
	if d.DisableAt == nil {
		return false
	}
	at, err := time.Parse(time.RFC3339, *d.DisableAt)
	return err == nil && at.After(now)
}

var domainStatuses = map[string]bool{"pending_ownership": true, "pending": true, "active": true, "error": true, "disabled": true}
var sslStatuses = map[string]bool{"pending": true, "active": true, "error": true}

// blocked are Blup's own domains: a site's own domain is never one of
// them or below them.
var blocked = []string{"getblup.io", "getblup.site"}

// ValidHostname tells whether h is a bare domain name the cloud can link:
// lower case ASCII (IDNA already applied), no scheme, path, port or user,
// not an IP, at least two labels, 1 to 63 characters per label, 253 in
// all, not a local name and not under Blup's own domains.
func ValidHostname(h string) bool {
	if h == "" || len(h) > 253 || strings.ContainsAny(h, ":/?#@[]\\% \t\r\n") {
		return false
	}
	if _, err := netip.ParseAddr(h); err == nil {
		return false
	}
	labels := strings.Split(h, ".")
	if len(labels) < 2 {
		return false
	}
	for _, l := range labels {
		if l == "" || len(l) > 63 || l[0] == '-' || l[len(l)-1] == '-' {
			return false
		}
		for i := 0; i < len(l); i++ {
			c := l[i]
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	tld := labels[len(labels)-1]
	if tld[0] >= '0' && tld[0] <= '9' { // 1.2.3.4-like, or a numeric TLD
		return false
	}
	switch tld {
	case "localhost", "local", "internal", "test", "invalid", "example", "home", "lan", "arpa", "onion":
		return false
	}
	for _, b := range blocked {
		if h == b || strings.HasSuffix(h, "."+b) {
			return false
		}
	}
	return true
}

func domainPath(siteID string) (string, error) {
	if !ValidVaultID(siteID) { // site ids are UUIDs, like vault ids
		return "", ErrDomainSiteMissing
	}
	return "/sites/" + url.PathEscape(siteID) + "/domain", nil
}

func domainErr(err error) error {
	var e *apiError
	if errors.As(err, &e) {
		switch e.Code {
		case "INVALID":
			return ErrDomainInvalid
		case "PLAN":
			return ErrDomainPlan
		case "NOT_FOUND":
			return ErrDomainSiteMissing
		case "DOMAIN_TAKEN":
			return ErrDomainTaken
		case "DOMAIN_EXISTS":
			return ErrDomainExists
		case "TOO_MANY_REQUESTS":
			return ErrDomainRate
		case "DOMAIN_PROVIDER":
			return ErrDomainProvider
		case "DOMAIN_UNAVAILABLE":
			return ErrDomainUnavailable
		}
		switch e.Status {
		case http.StatusNotFound:
			return ErrDomainSiteMissing
		case http.StatusTooManyRequests:
			return ErrDomainRate
		}
	}
	return err
}

// clean keeps what the window can show safely: known statuses only, and
// records of the two known types with bounded text.
func (d *DomainInfo) clean() error {
	if !ValidHostname(d.Hostname) || !domainStatuses[d.Status] {
		return errors.New("blupcloud: unexpected domain answer")
	}
	if !sslStatuses[d.SSL] {
		d.SSL = "pending"
	}
	out := make([]DomainRecord, 0, len(d.Records))
	for _, r := range d.Records {
		if (r.Type != "TXT" && r.Type != "CNAME") || r.Name == "" || len(r.Name) > 253 || len(r.Value) > 1024 || len(out) >= 10 {
			continue
		}
		out = append(out, r)
	}
	d.Records = out
	if d.DisableAt != nil {
		if _, err := time.Parse(time.RFC3339, *d.DisableAt); err != nil {
			d.DisableAt = nil
		}
	}
	return nil
}

// Domain is the site's own domain, or nil when it has none. It also
// checks the DNS again on the cloud.
func (c *Client) Domain(ctx context.Context, siteID string) (*DomainInfo, error) {
	p, err := domainPath(siteID)
	if err != nil {
		return nil, err
	}
	var d *DomainInfo
	if err := c.do(ctx, http.MethodGet, p, nil, "", &d); err != nil {
		return nil, domainErr(err)
	}
	if d == nil {
		return nil, nil
	}
	if err := d.clean(); err != nil {
		return nil, err
	}
	return d, nil
}

// SetDomain links hostname to the site (again: checks it again).
func (c *Client) SetDomain(ctx context.Context, siteID, hostname string) (DomainInfo, error) {
	if !ValidHostname(hostname) {
		return DomainInfo{}, ErrBadHostname
	}
	p, err := domainPath(siteID)
	if err != nil {
		return DomainInfo{}, err
	}
	var d DomainInfo
	if err := c.do(ctx, http.MethodPost, p, map[string]string{"hostname": hostname}, "", &d); err != nil {
		return DomainInfo{}, domainErr(err)
	}
	return d, d.clean()
}

// RemoveDomain unlinks the site's own domain.
func (c *Client) RemoveDomain(ctx context.Context, siteID string) error {
	p, err := domainPath(siteID)
	if err != nil {
		return err
	}
	return domainErr(c.do(ctx, http.MethodDelete, p, nil, "", nil))
}
