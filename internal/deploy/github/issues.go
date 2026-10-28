package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// Issue is the part of a GitHub issue the feedback of a site uses.
type Issue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
	URL    string `json:"html_url"`
}

// EnableIssues turns issues on in a repository (a no-op when they are on).
func (c *Client) EnableIssues(ctx context.Context, owner, repo string) error {
	_, err := c.do(ctx, http.MethodPatch, repoPath(owner, repo), map[string]any{"has_issues": true}, nil)
	return err
}

// EnsureLabel creates a label (name, color without "#") unless it exists.
func (c *Client) EnsureLabel(ctx context.Context, owner, repo, name, color, description string) error {
	_, err := c.do(ctx, http.MethodPost, repoPath(owner, repo)+"/labels", map[string]string{
		"name": name, "color": color, "description": description,
	}, nil)
	if statusOf(err) == http.StatusUnprocessableEntity {
		return nil // already there
	}
	return err
}

// ListIssues returns every issue (open and closed) with label, oldest first.
func (c *Client) ListIssues(ctx context.Context, owner, repo, label string) ([]Issue, error) {
	var all []Issue
	for page := 1; page <= 20; page++ {
		q := url.Values{"state": {"all"}, "labels": {label}, "per_page": {"100"}, "page": {fmt.Sprint(page)}, "sort": {"created"}, "direction": {"asc"}}
		var batch []Issue
		if _, err := c.do(ctx, http.MethodGet, repoPath(owner, repo)+"/issues?"+q.Encode(), nil, &batch); err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < 100 {
			break
		}
	}
	return all, nil
}

// CreateIssue opens an issue with the given labels.
func (c *Client) CreateIssue(ctx context.Context, owner, repo, title, body string, labels []string) (Issue, error) {
	var out Issue
	_, err := c.do(ctx, http.MethodPost, repoPath(owner, repo)+"/issues", map[string]any{
		"title": title, "body": body, "labels": labels,
	}, &out)
	return out, err
}

// UpdateIssue changes the title and body of an issue.
func (c *Client) UpdateIssue(ctx context.Context, owner, repo string, number int, title, body string) error {
	_, err := c.do(ctx, http.MethodPatch, fmt.Sprintf("%s/issues/%d", repoPath(owner, repo), number), map[string]any{
		"title": title, "body": body,
	}, nil)
	return err
}
