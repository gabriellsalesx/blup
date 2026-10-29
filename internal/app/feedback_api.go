package app

import (
	"context"
	"errors"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/deploy/github"
	"github.com/gabriellsalesx/blup/internal/i18n"
)

// Comments (and votes, reactions) set up from Blup Studio, without
// questions: the public "-feedback" repository on the user's GitHub and the
// login service's app installed on it (enableFeedback asks the same in the
// terminal).

// FeedbackSetup is where the comments setup stands.
type FeedbackSetup struct {
	Repo       string `json:"repo"`       // owner/repo
	App        bool   `json:"app"`        // the login app is installed on it
	InstallURL string `json:"installURL"` // where to install it
	AppName    string `json:"appName"`
}

// SetupFeedback creates the feedback repository (once), turns comments on
// and says whether the login app still needs installing. It needs the
// GitHub key saved (Publish).
func SetupFeedback(ctx context.Context, site *config.Site) (FeedbackSetup, error) {
	creds, err := config.LoadCredentials()
	if err != nil {
		return FeedbackSetup{}, err
	}
	if creds.GitHubToken == "" {
		return FeedbackSetup{}, errors.New(i18n.T("fb.no_token"))
	}
	c, login, err := githubClient(ctx)
	if err != nil {
		return FeedbackSetup{}, err
	}
	f := &site.Feedback
	if f.Repo == "" {
		f.Owner, f.Repo = login, feedbackRepoName(site)
	}
	repo, err := c.EnsureRepoWith(ctx, f.Owner, f.Repo, github.EnsureRepoOptions{
		Description: feedbackRepoDescription(site), Issues: true, NoHomepage: true,
	})
	if err == nil && repo.Private {
		return FeedbackSetup{}, errors.New(i18n.T("fb.private", repo.FullName))
	}
	if err == nil {
		err = c.EnableIssues(ctx, f.Owner, f.Repo)
	}
	if err == nil {
		err = c.EnsureLabel(ctx, f.Owner, f.Repo, feedbackLabel, "6fc0ee", "blup")
	}
	if errors.Is(err, github.ErrPermission) {
		return FeedbackSetup{}, errors.New(i18n.T("fb.need_issues"))
	}
	if err != nil {
		return FeedbackSetup{}, err
	}
	f.Enabled = true
	if err := saved(site); err != nil {
		return FeedbackSetup{}, err
	}
	return FeedbackState(ctx, site)
}

// FeedbackState says whether the login app is installed on the feedback
// repository (without it, people vote and comment on GitHub instead).
func FeedbackState(ctx context.Context, site *config.Site) (FeedbackSetup, error) {
	f := site.Feedback
	out := FeedbackSetup{AppName: f.AppSlug(), InstallURL: "https://github.com/apps/" + f.AppSlug() + "/installations/new"}
	if f.Repo == "" {
		return out, nil
	}
	out.Repo = f.Owner + "/" + f.Repo
	ok, err := appInstalled(ctx, f)
	out.App = ok
	return out, err
}
