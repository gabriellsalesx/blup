package app

import (
	"context"
	"net/url"
	"time"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/deploy/github"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/ui"
)

func init() {
	registerErrors(
		knownError{github.ErrBadToken, "gh.err_token"},
		knownError{github.ErrPermission, "gh.err_permission"},
		knownError{github.ErrRepoConflict, "gh.err_repo"},
		knownError{github.ErrRateLimited, "gh.err_rate"},
		knownError{github.ErrPagesUnavailable, "gh.err_pages"},
		knownError{github.ErrPagesBuildFailed, "gh.err_build"},
	)
}

// publish asks where to publish the first time, then repeats the same
// destination on every run.
func publish(ctx context.Context, g *config.Global, site *config.Site) error {
	ui.Step(i18n.T("pub.title", site.Title))
	if site.Deploy.Provider == "" {
		p := "github"
		if err := ui.Choose(i18n.T("pub.where"), i18n.T("pub.where_help"), &p,
			ui.Option(i18n.T("pub.github"), "github"),
			ui.Option(i18n.T("pub.cloudflare"), "cloudflare"),
		); err != nil {
			return err
		}
		site.Deploy.Provider = p
	}
	var live string
	var err error
	switch site.Deploy.Provider {
	case "cloudflare":
		live, err = publishCloudflare(ctx, site)
	default:
		live, err = publishGitHub(ctx, site)
	}
	if err != nil {
		return err
	}
	if err := site.Save(); err != nil {
		return err
	}
	ui.OK(i18n.T("pub.live", ui.Highlight(live)))
	// A custom domain is connected (or checked) on every publish.
	if err := connectDomain(ctx, site); err != nil && !ui.Aborted(err) {
		ui.Warn(describe(err))
	}
	if ok, _ := ui.Confirm(i18n.T("pub.open"), "", true); ok {
		ui.OpenBrowser(live)
	}
	return nil
}

// githubTokenURL opens GitHub's token form already filled in with the
// permissions we need; only "All repositories" must be picked by hand.
func githubTokenURL() string {
	q := url.Values{}
	q.Set("name", "blup")
	q.Set("description", i18n.T("gh.token_description"))
	q.Set("expires_in", "366")
	q.Set("contents", "write")
	q.Set("pages", "write")
	q.Set("administration", "write")
	q.Set("issues", "write")
	return "https://github.com/settings/personal-access-tokens/new?" + q.Encode()
}

func githubClient(ctx context.Context) (*github.Client, string, error) {
	creds, err := config.LoadCredentials()
	if err != nil {
		return nil, "", err
	}
	for {
		if creds.GitHubToken == "" {
			link := githubTokenURL()
			if err := ui.Explain(i18n.T("gh.token_title"), i18n.T("gh.token_steps")); err != nil {
				return nil, "", err
			}
			ui.OpenBrowser(link)
			ui.Hint(i18n.T("browser_fallback", link))
			if err := ui.AskSecret(i18n.T("gh.token_q"), i18n.T("gh.token_help"), &creds.GitHubToken); err != nil {
				return nil, "", err
			}
		}
		c := github.New(creds.GitHubToken)
		var login string
		err := ui.Spin(ctx, i18n.T("gh.checking"), func(ctx context.Context) error {
			var err error
			login, err = c.Viewer(ctx)
			return err
		})
		if err == nil {
			if err := creds.Save(); err != nil {
				return nil, "", err
			}
			ui.OK(i18n.T("gh.logged_in", login))
			return c, login, nil
		}
		ui.Error(describe(err))
		creds.GitHubToken = ""
		if ok, _ := ui.Confirm(i18n.T("gh.retry"), "", true); !ok {
			return nil, "", err
		}
	}
}

func publishGitHub(ctx context.Context, site *config.Site) (string, error) {
	c, login, err := githubClient(ctx)
	if err != nil {
		return "", err
	}
	d := &site.Deploy.GitHub
	if d.Repo == "" {
		d.Repo = Slug(site.Title)
		if err := ui.Ask(i18n.T("gh.repo_q"), i18n.T("gh.repo_help", login), d.Repo, &d.Repo, notEmpty); err != nil {
			return "", err
		}
	}
	d.Owner = login

	if err := ui.Spin(ctx, i18n.T("gh.repo_preparing"), func(ctx context.Context) error {
		_, err := c.EnsureRepo(ctx, login, d.Repo)
		return err
	}); err != nil {
		return "", err
	}

	live := siteURL(site, github.ExpectedURL(login, d.Repo))
	dir, err := build(ctx, site, live)
	if err != nil {
		return "", err
	}

	var res github.Result
	if err := ui.Spin(ctx, i18n.T("pub.uploading"), func(ctx context.Context) error {
		res, err = c.Publish(ctx, login, d.Repo, dir, github.PublishOptions{
			CNAME:   site.Domain,
			Message: "blup " + time.Now().Format("2006-01-02 15:04"),
		}, nil)
		return err
	}); err != nil {
		return "", err
	}
	if res.Unchanged {
		ui.OK(i18n.T("pub.unchanged"))
		return live, nil
	}
	ui.OK(i18n.T("pub.uploaded", res.Uploaded, res.Skipped))

	if err := ui.Spin(ctx, i18n.T("gh.pages_enabling"), func(ctx context.Context) error {
		_, err := c.EnsurePages(ctx, login, d.Repo, github.DefaultBranch)
		return err
	}); err != nil {
		return "", err
	}
	if err := ui.Spin(ctx, i18n.T("gh.pages_building"), func(ctx context.Context) error {
		_, err := c.WaitBuild(ctx, login, d.Repo, res.CommitSHA, 10*time.Minute)
		return err
	}); err != nil {
		ui.Warn(describe(err))
	}
	return live, nil
}
