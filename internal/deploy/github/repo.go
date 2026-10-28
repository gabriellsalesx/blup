package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Repo describes a repository.
type Repo struct {
	Owner         string
	Name          string
	FullName      string
	DefaultBranch string
	Private       bool
	HTMLURL       string
	// Created is true when EnsureRepo created it now.
	Created bool
}

type apiRepo struct {
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Private       bool   `json:"private"`
	HTMLURL       string `json:"html_url"`
	DefaultBranch string `json:"default_branch"`
	Owner         struct {
		Login string `json:"login"`
	} `json:"owner"`
	Permissions *struct {
		Admin bool `json:"admin"`
		Push  bool `json:"push"`
	} `json:"permissions"`
}

func (r apiRepo) toRepo(created bool) Repo {
	return Repo{
		Owner:         r.Owner.Login,
		Name:          r.Name,
		FullName:      r.FullName,
		DefaultBranch: r.DefaultBranch,
		Private:       r.Private,
		HTMLURL:       r.HTMLURL,
		Created:       created,
	}
}

// Viewer returns the login of the token's user (GET /user).
func (c *Client) Viewer(ctx context.Context) (string, error) {
	var u struct {
		Login string `json:"login"`
	}
	if _, err := c.do(ctx, http.MethodGet, "/user", nil, &u); err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrPermission) {
			// GET /user only fails like this for unusable tokens.
			return "", fmt.Errorf("%w: %w", ErrBadToken, err)
		}
		return "", err
	}
	if u.Login == "" {
		return "", ErrBadToken
	}
	return u.Login, nil
}

func repoPath(owner, name string) string {
	return "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(name)
}

// EnsureRepo returns owner/name, creating it as a PUBLIC repository with an
// initial commit (auto_init) when it does not exist. owner may be empty to
// mean the authenticated user. Only repositories of the authenticated user
// can be created; an existing repository the token cannot push to, or a name
// that belongs to another account, yields ErrRepoConflict.
func (c *Client) EnsureRepo(ctx context.Context, owner, name string) (Repo, error) {
	return c.EnsureRepoWith(ctx, owner, name, EnsureRepoOptions{})
}

// EnsureRepoOptions tunes EnsureRepoWith. The zero value matches EnsureRepo.
type EnsureRepoOptions struct {
	// Private creates the repository as private (free on every GitHub plan).
	// It has no effect on a repository that already exists: check
	// Repo.Private in the result.
	Private bool
	// Description of a created repository; default "Site published with blup".
	Description string
	// Empty creates the repository without an initial commit (no README).
	// The first BranchTip call with an InitFile initializes it.
	Empty bool
}

// EnsureRepoWith is EnsureRepo with options (e.g. a private repository).
// The homepage is set to the Pages URL only for public repositories.
func (c *Client) EnsureRepoWith(ctx context.Context, owner, name string, opts EnsureRepoOptions) (Repo, error) {
	if name == "" {
		return Repo{}, errors.New("github: repository name is empty")
	}
	login, err := c.Viewer(ctx)
	if err != nil {
		return Repo{}, err
	}
	if owner == "" {
		owner = login
	}
	var r apiRepo
	_, err = c.do(ctx, http.MethodGet, repoPath(owner, name), nil, &r)
	if err == nil {
		if r.Permissions != nil && !r.Permissions.Push && !r.Permissions.Admin {
			return Repo{}, fmt.Errorf("%w: %s exists but this account cannot publish to it", ErrRepoConflict, r.FullName)
		}
		return r.toRepo(false), nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Repo{}, err
	}
	if !strings.EqualFold(owner, login) {
		return Repo{}, fmt.Errorf("%w: cannot create a repository under %q while logged in as %q", ErrRepoConflict, owner, login)
	}
	desc := opts.Description
	if desc == "" {
		desc = "Site published with blup"
	}
	body := map[string]any{
		"name":         name,
		"private":      opts.Private,
		"auto_init":    !opts.Empty,
		"has_issues":   false,
		"has_projects": false,
		"has_wiki":     false,
		"description":  desc,
	}
	if !opts.Private {
		body["homepage"] = ExpectedURL(login, name)
	}
	_, err = c.do(ctx, http.MethodPost, "/user/repos", body, &r)
	if err != nil {
		var ae *APIError
		if errors.As(err, &ae) && ae.StatusCode == http.StatusUnprocessableEntity {
			ae.kind = ErrRepoConflict
		}
		return Repo{}, err
	}
	return r.toRepo(true), nil
}

// ExpectedURL is the GitHub Pages URL for owner/repo without custom domain:
// https://<owner>.github.io/ for the user site repository <owner>.github.io
// (case-insensitive), https://<owner>.github.io/<repo>/ otherwise.
func ExpectedURL(owner, repo string) string {
	host := strings.ToLower(owner) + ".github.io"
	if strings.EqualFold(repo, host) {
		return "https://" + host + "/"
	}
	return "https://" + host + "/" + repo + "/"
}
