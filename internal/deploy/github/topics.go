package github

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

// OwnRepo is a repository of the authenticated user, with its topics.
type OwnRepo struct {
	Owner   string
	Name    string
	Private bool
	Topics  []string
	Pushed  time.Time
}

// OwnRepos lists the repositories the authenticated user owns, most
// recently pushed first (GET /user/repos, up to 1,000).
func (c *Client) OwnRepos(ctx context.Context) ([]OwnRepo, error) {
	var out []OwnRepo
	for page := 1; page <= 10; page++ {
		var list []struct {
			Name    string    `json:"name"`
			Private bool      `json:"private"`
			Topics  []string  `json:"topics"`
			Pushed  time.Time `json:"pushed_at"`
			Owner   struct {
				Login string `json:"login"`
			} `json:"owner"`
		}
		q := url.Values{"affiliation": {"owner"}, "sort": {"pushed"}, "per_page": {"100"}, "page": {strconv.Itoa(page)}}
		if _, err := c.do(ctx, http.MethodGet, "/user/repos?"+q.Encode(), nil, &list); err != nil {
			return nil, err
		}
		for _, r := range list {
			out = append(out, OwnRepo{Owner: r.Owner.Login, Name: r.Name, Private: r.Private, Topics: r.Topics, Pushed: r.Pushed})
		}
		if len(list) < 100 {
			break
		}
	}
	return out, nil
}

// AddTopic adds a topic to a repository, keeping the ones it has.
func (c *Client) AddTopic(ctx context.Context, owner, name, topic string) error {
	var t struct {
		Names []string `json:"names"`
	}
	if _, err := c.do(ctx, http.MethodGet, repoPath(owner, name)+"/topics", nil, &t); err != nil {
		return err
	}
	if slices.Contains(t.Names, topic) {
		return nil
	}
	t.Names = append(t.Names, topic)
	_, err := c.do(ctx, http.MethodPut, repoPath(owner, name)+"/topics", t, nil)
	return err
}

// FileContent reads one file of a repository's default branch (nil and no
// error when it doesn't exist).
func (c *Client) FileContent(ctx context.Context, owner, name, path string) ([]byte, error) {
	_, data, err := c.send(ctx, http.MethodGet, repoPath(owner, name)+"/contents/"+escapePath(path), nil, "application/vnd.github.raw+json", 1<<20)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	return data, err
}

func escapePath(p string) string {
	segs := strings.Split(p, "/")
	for i, s := range segs {
		segs[i] = url.PathEscape(s)
	}
	return strings.Join(segs, "/")
}
