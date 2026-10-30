package app

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/deploy/cfpages"
	"github.com/gabriellsalesx/blup/internal/deploy/github"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/notesync"
)

// Publishing and syncing without questions, for Blup Studio: the accounts
// are connected from a form (the key is pasted there), the destination is
// chosen in the form, and progress goes through ui.Report.

// Connections tells which accounts Blup has a key for. GitHub is the
// account's login ("" without a key); GitHubError says why it could not be
// checked (no internet, revoked key).
type Connections struct {
	GitHub      string `json:"github"`
	GitHubError string `json:"githubError"`
	Cloudflare  bool   `json:"cloudflare"`
}

// LoadConnections checks the saved keys.
func LoadConnections(ctx context.Context) (Connections, error) {
	creds, err := config.LoadCredentials()
	if err != nil {
		return Connections{}, err
	}
	out := Connections{Cloudflare: creds.CloudflareToken != ""}
	if creds.GitHubToken != "" {
		login, err := github.New(creds.GitHubToken).Viewer(ctx)
		if err != nil {
			out.GitHubError = describe(err)
		}
		out.GitHub = login
	}
	return out, nil
}

// GitHubKeyLink opens GitHub's form for a new key, with the permissions
// Blup needs already ticked.
func GitHubKeyLink() string { return githubTokenURL() }

// CloudflareKeyLink opens Cloudflare's form for a new key (Pages: Edit).
func CloudflareKeyLink() string { return cfpages.TokenTemplateURL }

// ConnectGitHub checks a GitHub key and saves it; it returns the login.
func ConnectGitHub(ctx context.Context, token string) (string, error) {
	token = strings.TrimSpace(token)
	login, err := github.New(token).Viewer(ctx)
	if err != nil {
		return "", errors.New(describe(err))
	}
	creds, err := config.LoadCredentials()
	if err != nil {
		return "", err
	}
	creds.GitHubToken = token
	return login, creds.Save()
}

// ConnectCloudflare checks a Cloudflare key and saves it; it returns the
// accounts the key can publish to.
func ConnectCloudflare(ctx context.Context, token string) ([]cfpages.Account, error) {
	c := &cfpages.Client{Token: strings.TrimSpace(token)}
	if err := c.VerifyToken(ctx); err != nil {
		return nil, errors.New(describe(err))
	}
	accounts, err := c.Accounts(ctx)
	if err != nil {
		return nil, errors.New(describe(err))
	}
	creds, err := config.LoadCredentials()
	if err != nil {
		return nil, err
	}
	creds.CloudflareToken = c.Token
	return accounts, creds.Save()
}

// Disconnect forgets the key of "github", "cloudflare" or "openrouter".
func Disconnect(account string) error {
	creds, err := config.LoadCredentials()
	if err != nil {
		return err
	}
	switch account {
	case "github":
		creds.GitHubToken = ""
	case "cloudflare":
		creds.CloudflareToken = ""
	case "openrouter":
		creds.OpenRouterToken = ""
	}
	return creds.Save()
}

// CloudflareAccounts lists the accounts of the saved Cloudflare key.
func CloudflareAccounts(ctx context.Context) ([]cfpages.Account, error) {
	c, err := savedCloudflare()
	if err != nil {
		return nil, err
	}
	a, err := c.Accounts(ctx)
	if err != nil {
		return nil, errors.New(describe(err))
	}
	return a, nil
}

func savedCloudflare() (*cfpages.Client, error) {
	creds, err := config.LoadCredentials()
	if err != nil {
		return nil, err
	}
	if creds.CloudflareToken == "" {
		return nil, errors.New(i18n.T("pub.connect_cloudflare"))
	}
	return &cfpages.Client{Token: creds.CloudflareToken}, nil
}

func savedGitHub(ctx context.Context) (*github.Client, string, error) {
	creds, err := config.LoadCredentials()
	if err != nil {
		return nil, "", err
	}
	if creds.GitHubToken == "" {
		return nil, "", errors.New(i18n.T("pub.connect_github"))
	}
	c := github.New(creds.GitHubToken)
	login, err := c.Viewer(ctx)
	if err != nil {
		return nil, "", errors.New(describe(err))
	}
	return c, login, nil
}

// Target is where the site is published: "cloudflare" (Account, Project)
// or "github" (Repo).
type Target struct {
	Provider string `json:"provider"`
	Account  string `json:"account"`
	Project  string `json:"project"`
	Repo     string `json:"repo"`
	// URL is the site's address, once published (read only).
	URL string `json:"url"`
}

// LoadTarget is the site's publishing destination, with suggested names
// for a site never published.
func LoadTarget(site *config.Site) Target {
	d := site.Deploy
	t := Target{Provider: d.Provider, Account: d.Cloudflare.Account, Project: d.Cloudflare.Project, Repo: d.GitHub.Repo}
	if t.Project == "" {
		t.Project = cfpages.ProjectNameFromTitle(site.Title)
	}
	if t.Repo == "" {
		t.Repo = Slug(site.Title)
	}
	switch {
	case d.Provider == "cloudflare" && d.Cloudflare.Subdomain != "":
		t.URL = siteURL(site, "https://"+d.Cloudflare.Subdomain+"/")
	case d.Provider == "github" && d.GitHub.Owner != "" && d.GitHub.Repo != "":
		t.URL = siteURL(site, github.ExpectedURL(d.GitHub.Owner, d.GitHub.Repo))
	}
	return t
}

// SetTarget saves where the site is published.
func SetTarget(site *config.Site, t Target) error {
	switch t.Provider {
	case "cloudflare":
		if t.Account == "" {
			return errors.New(i18n.T("pub.pick_account"))
		}
		if err := cfpages.ValidateProjectName(t.Project); err != nil {
			return err
		}
		if site.Deploy.Cloudflare.Project != t.Project {
			site.Deploy.Cloudflare.Subdomain = ""
		}
		site.Deploy.Cloudflare.Account, site.Deploy.Cloudflare.Project = t.Account, t.Project
	case "github":
		if strings.TrimSpace(t.Repo) == "" {
			return errors.New(i18n.T("required"))
		}
		site.Deploy.GitHub.Repo = strings.TrimSpace(t.Repo)
	default:
		return errors.New("unknown provider " + t.Provider)
	}
	site.Deploy.Provider = t.Provider
	return site.Save()
}

// PublishNow builds the site and puts it online where it was set up.
func PublishNow(ctx context.Context, site *config.Site) (string, error) {
	var live string
	var err error
	switch site.Deploy.Provider {
	case "cloudflare":
		var c *cfpages.Client
		if c, err = savedCloudflare(); err == nil {
			live, err = deployCloudflare(ctx, site, c)
		}
	case "github":
		var c *github.Client
		var login string
		if c, login, err = savedGitHub(ctx); err == nil {
			live, err = deployGitHub(ctx, site, c, login)
		}
	default:
		return "", errors.New(i18n.T("pub.not_set"))
	}
	if err != nil {
		return "", errors.New(describe(err))
	}
	return live, site.Save()
}

// SyncInfo is what a sync would do (counts), or did.
type SyncInfo struct {
	Repo       string   `json:"repo"`
	Send       int      `json:"send"`
	Receive    int      `json:"receive"`
	Conflicts  []string `json:"conflicts"`
	UpToDate   bool     `json:"upToDate"`
	TrashCount int      `json:"trashCount"`
	Trash      string   `json:"trash"`
}

// syncer prepares the private notes repository (created the first time).
func syncer(ctx context.Context, site *config.Site) (*notesync.Syncer, string, error) {
	c, login, err := savedGitHub(ctx)
	if err != nil {
		return nil, "", err
	}
	s := &site.Sync
	if s.Repo == "" {
		s.Repo = Slug(site.Title) + i18n.T("sync.repo_suffix")
	}
	if s.Owner == "" {
		s.Owner = login
	}
	repo, err := notesync.EnsureRepo(ctx, c, s.Owner, s.Repo)
	if err != nil {
		return nil, "", errors.New(describe(err))
	}
	if !repo.Private {
		return nil, "", errors.New(i18n.T("sync.public_q", repo.FullName))
	}
	if err := site.Save(); err != nil {
		return nil, "", err
	}
	return &notesync.Syncer{
		GH: c, Owner: s.Owner, Repo: s.Repo,
		Dir:       site.ContentDir(),
		StateFile: filepath.Join(site.Dir(), ".blup", "sync-state.json"),
	}, repo.FullName, nil
}

// SyncStatus tells what a sync would send and receive.
func SyncStatus(ctx context.Context, site *config.Site) (SyncInfo, error) {
	sy, repo, err := syncer(ctx, site)
	if err != nil {
		return SyncInfo{}, err
	}
	plan, err := sy.Status(ctx, nil)
	if err != nil {
		return SyncInfo{}, errors.New(describe(err))
	}
	return SyncInfo{
		Repo: repo, Send: len(plan.Uploaded) + len(plan.Deleted), Receive: len(plan.Downloaded) + len(plan.DeletedLocal),
		Conflicts: plan.Conflicts, UpToDate: plan.UpToDate && len(plan.Conflicts) == 0,
	}, nil
}

// SyncNow sends and receives the notes.
func SyncNow(ctx context.Context, site *config.Site) (SyncInfo, error) {
	sy, repo, err := syncer(ctx, site)
	if err != nil {
		return SyncInfo{}, err
	}
	rep, err := sy.Sync(ctx, nil)
	if err != nil {
		return SyncInfo{}, errors.New(describe(err))
	}
	return SyncInfo{
		Repo: repo, Send: len(rep.Uploaded) + len(rep.Deleted), Receive: len(rep.Downloaded) + len(rep.DeletedLocal),
		Conflicts: rep.ConflictCopies, UpToDate: true, TrashCount: len(rep.DeletedLocal), Trash: rep.Trash,
	}, nil
}
