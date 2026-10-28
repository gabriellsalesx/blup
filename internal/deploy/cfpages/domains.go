package cfpages

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

// Domain is a custom domain of a Pages project.
type Domain struct {
	Name string
	// Status: initializing, pending, active, deactivated, blocked or error.
	Status string
	// Message explains a pending or failed check, when Cloudflare gives one.
	Message string
	// OnCloudflare means the domain's zone is in the account, so Cloudflare
	// manages its DNS record.
	OnCloudflare bool
}

// Active reports whether the domain already serves the site.
func (d Domain) Active() bool { return d.Status == "active" }

type apiDomain struct {
	Name           string `json:"name"`
	Status         string `json:"status"`
	ZoneTag        string `json:"zone_tag"`
	ValidationData struct {
		Status       string `json:"status"`
		ErrorMessage string `json:"error_message"`
	} `json:"validation_data"`
	VerificationData struct {
		Status       string `json:"status"`
		ErrorMessage string `json:"error_message"`
	} `json:"verification_data"`
}

func (a apiDomain) toDomain() Domain {
	d := Domain{Name: a.Name, Status: a.Status, OnCloudflare: a.ZoneTag != ""}
	d.Message = a.VerificationData.ErrorMessage
	if d.Message == "" {
		d.Message = a.ValidationData.ErrorMessage
	}
	return d
}

func domainsPath(accountID, project string) string {
	return projectPath(accountID, project) + "/domains"
}

// AddDomain connects name to the project. A domain already connected is
// returned as it is.
func (c *Client) AddDomain(ctx context.Context, accountID, project, name string) (Domain, error) {
	var a apiDomain
	_, err := c.do(ctx, http.MethodPost, domainsPath(accountID, project), "", map[string]string{"name": name}, "", &a)
	if err != nil {
		if d, gerr := c.GetDomain(ctx, accountID, project, name); gerr == nil {
			return d, nil // already there
		}
		return Domain{}, err
	}
	return a.toDomain(), nil
}

// GetDomain returns the state of a custom domain (ErrNotFound when the
// project does not have it).
func (c *Client) GetDomain(ctx context.Context, accountID, project, name string) (Domain, error) {
	var a apiDomain
	_, err := c.do(ctx, http.MethodGet, domainsPath(accountID, project)+"/"+url.PathEscape(name), "", nil, "", &a)
	return a.toDomain(), err
}

// RetryDomain asks Cloudflare to check a pending domain again.
func (c *Client) RetryDomain(ctx context.Context, accountID, project, name string) (Domain, error) {
	var a apiDomain
	_, err := c.do(ctx, http.MethodPatch, domainsPath(accountID, project)+"/"+url.PathEscape(name), "", map[string]string{}, "", &a)
	if err != nil {
		return c.GetDomain(ctx, accountID, project, name)
	}
	return a.toDomain(), nil
}

// RemoveDomain disconnects a custom domain; one that is not there is fine.
func (c *Client) RemoveDomain(ctx context.Context, accountID, project, name string) error {
	_, err := c.do(ctx, http.MethodDelete, domainsPath(accountID, project)+"/"+url.PathEscape(name), "", nil, "", nil)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}
