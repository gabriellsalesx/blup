package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type pagesSite struct {
	HTMLURL   string `json:"html_url"`
	Status    string `json:"status"`
	BuildType string `json:"build_type"`
	CNAME     string `json:"cname"`
	Source    *struct {
		Branch string `json:"branch"`
		Path   string `json:"path"`
	} `json:"source"`
}

// EnsurePages enables GitHub Pages for owner/repo serving branch at "/"
// (classic "deploy from a branch" build), or updates the source when Pages
// is already enabled with a different one. It returns the site URL.
func (c *Client) EnsurePages(ctx context.Context, owner, repo, branch string) (string, error) {
	if branch == "" {
		branch = DefaultBranch
	}
	p := repoPath(owner, repo) + "/pages"
	source := map[string]string{"branch": branch, "path": "/"}

	var site pagesSite
	_, err := c.do(ctx, http.MethodGet, p, nil, &site)
	switch {
	case err == nil:
		if site.Source == nil || site.Source.Branch != branch || site.Source.Path != "/" || site.BuildType == "workflow" {
			_, err := c.do(ctx, http.MethodPut, p, map[string]any{
				"build_type": "legacy",
				"source":     source,
			}, nil)
			if err != nil {
				return "", pagesErr(err)
			}
			if _, err := c.do(ctx, http.MethodGet, p, nil, &site); err != nil {
				return "", pagesErr(err)
			}
		}
	case errors.Is(err, ErrNotFound):
		_, err := c.do(ctx, http.MethodPost, p, map[string]any{
			"build_type": "legacy",
			"source":     source,
		}, &site)
		if err != nil {
			// 409: already enabled (race); update instead.
			if statusOf(err) == http.StatusConflict {
				if _, err := c.do(ctx, http.MethodPut, p, map[string]any{"build_type": "legacy", "source": source}, nil); err != nil {
					return "", pagesErr(err)
				}
				if _, err := c.do(ctx, http.MethodGet, p, nil, &site); err != nil {
					return "", pagesErr(err)
				}
				break
			}
			return "", pagesErr(err)
		}
	default:
		return "", pagesErr(err)
	}
	if site.HTMLURL == "" {
		return ExpectedURL(owner, repo), nil
	}
	u := site.HTMLURL
	if !strings.HasSuffix(u, "/") {
		u += "/"
	}
	return u, nil
}

// pagesErr maps Pages-specific failures to ErrPagesUnavailable.
func pagesErr(err error) error {
	var ae *APIError
	if !errors.As(err, &ae) {
		return err
	}
	text := strings.ToLower(ae.Message + " " + strings.Join(ae.Errors, " "))
	if ae.StatusCode == http.StatusUnprocessableEntity || ae.StatusCode == http.StatusBadRequest ||
		(ae.StatusCode == http.StatusForbidden && strings.Contains(text, "plan")) {
		if strings.Contains(text, "plan") || strings.Contains(text, "private") ||
			strings.Contains(text, "not support") || strings.Contains(text, "upgrade") {
			ae.kind = ErrPagesUnavailable
		}
	}
	return err
}

// Build is the state of a Pages build.
type Build struct {
	Status   string // queued, building, built, errored
	Commit   string
	Error    string
	Duration time.Duration
}

// WaitBuild polls the latest Pages build until it is built or errored.
// If commit is non-empty, builds of other commits are ignored (still
// waiting). It gives up after timeout (default 10 minutes when <= 0).
func (c *Client) WaitBuild(ctx context.Context, owner, repo, commit string, timeout time.Duration) (Build, error) {
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	p := repoPath(owner, repo) + "/pages/builds/latest"
	for {
		var b struct {
			Status   string `json:"status"`
			Commit   string `json:"commit"`
			Duration int64  `json:"duration"`
			Error    struct {
				Message *string `json:"message"`
			} `json:"error"`
		}
		_, err := c.do(ctx, http.MethodGet, p, nil, &b)
		if err != nil && !errors.Is(err, ErrNotFound) {
			if ctx.Err() != nil {
				return Build{}, fmt.Errorf("github: waiting for Pages build: %w", ctx.Err())
			}
			return Build{}, err
		}
		if err == nil && (commit == "" || b.Commit == commit) {
			out := Build{Status: b.Status, Commit: b.Commit, Duration: time.Duration(b.Duration) * time.Millisecond}
			if b.Error.Message != nil {
				out.Error = *b.Error.Message
			}
			switch b.Status {
			case "built":
				return out, nil
			case "errored":
				return out, fmt.Errorf("%w: %s", ErrPagesBuildFailed, out.Error)
			}
		}
		if err := c.doSleep(ctx, 3*time.Second); err != nil {
			return Build{}, fmt.Errorf("github: waiting for Pages build: %w", err)
		}
	}
}

// PagesDomain is the custom domain state of a Pages site.
type PagesDomain struct {
	CNAME         string
	HTTPSEnforced bool
	// CertState is the HTTPS certificate state ("approved" when ready).
	CertState string
}

// GetPagesDomain returns the custom domain of owner/repo's Pages site.
func (c *Client) GetPagesDomain(ctx context.Context, owner, repo string) (PagesDomain, error) {
	var site struct {
		CNAME         string `json:"cname"`
		HTTPSEnforced bool   `json:"https_enforced"`
		Cert          *struct {
			State string `json:"state"`
		} `json:"https_certificate"`
	}
	if _, err := c.do(ctx, http.MethodGet, repoPath(owner, repo)+"/pages", nil, &site); err != nil {
		return PagesDomain{}, pagesErr(err)
	}
	d := PagesDomain{CNAME: site.CNAME, HTTPSEnforced: site.HTTPSEnforced}
	if site.Cert != nil {
		d.CertState = site.Cert.State
	}
	return d, nil
}

// SetPagesDomain sets the custom domain ("" removes it).
func (c *Client) SetPagesDomain(ctx context.Context, owner, repo, domain string) error {
	var cname any
	if domain != "" {
		cname = domain
	}
	_, err := c.do(ctx, http.MethodPut, repoPath(owner, repo)+"/pages", map[string]any{"cname": cname}, nil)
	return pagesErr(err)
}

// EnforceHTTPS turns on "Enforce HTTPS"; GitHub refuses it until the
// certificate of the custom domain is issued.
func (c *Client) EnforceHTTPS(ctx context.Context, owner, repo string) error {
	_, err := c.do(ctx, http.MethodPut, repoPath(owner, repo)+"/pages", map[string]any{"https_enforced": true}, nil)
	return pagesErr(err)
}

// DomainHealth is GitHub's DNS check of the custom domain.
type DomainHealth struct {
	// Pending: GitHub started the check; ask again in a few seconds.
	Pending bool
	// Ready: the DNS points to GitHub Pages and the site is served there.
	Ready bool
	// Apex reports whether the domain is a root domain (needs A records).
	Apex bool
	// HTTPSEligible: a certificate can be issued.
	HTTPSEligible bool
	Reason        string
}

// PagesHealth runs GitHub's DNS check of the custom domain.
func (c *Client) PagesHealth(ctx context.Context, owner, repo string) (DomainHealth, error) {
	var h struct {
		Domain *struct {
			IsApexDomain    bool   `json:"is_apex_domain"`
			IsServedByPages bool   `json:"is_served_by_pages"`
			IsValid         bool   `json:"is_valid"`
			IsHTTPSEligible bool   `json:"is_https_eligible"`
			Reason          string `json:"reason"`
		} `json:"domain"`
	}
	code, err := c.do(ctx, http.MethodGet, repoPath(owner, repo)+"/pages/health", nil, &h)
	if err != nil {
		return DomainHealth{}, pagesErr(err)
	}
	if code == http.StatusAccepted || h.Domain == nil {
		return DomainHealth{Pending: true}, nil
	}
	d := h.Domain
	return DomainHealth{
		Ready:         d.IsValid && d.IsServedByPages,
		Apex:          d.IsApexDomain,
		HTTPSEligible: d.IsHTTPSEligible,
		Reason:        d.Reason,
	}, nil
}
