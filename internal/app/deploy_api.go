package app

import (
	"context"
	"errors"
	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/deploy/cfpages"
	"github.com/gabriellsalesx/blup/internal/deploy/github"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/notesync"
	"github.com/gabriellsalesx/blup/internal/obsidian"
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
// Blup needs already ticked (and no Administration).
func GitHubKeyLink() string { return githubTokenURL() }

// GitHubNewRepoLink opens GitHub's "new repository" form filled in for the
// site ("site", public, named name) or for its notes ("notes", private,
// the name sync uses). A key limited to chosen repositories cannot create
// them, so the person creates them there and gives them to the key. ""
// when the name is not one GitHub accepts.
func GitHubNewRepoLink(site *config.Site, kind, name string) string {
	switch kind {
	case "notes":
		name = site.Sync.Repo
		if name == "" {
			name = Slug(site.Title) + i18n.T("sync.repo_suffix")
		}
		if !github.ValidRepoName(name) {
			return ""
		}
		return github.NewRepoURL(site.Sync.Owner, name, true, "Notes synced with blup")
	case "site":
		name = strings.TrimSpace(name)
		if !github.ValidRepoName(name) {
			return ""
		}
		return github.NewRepoURL("", name, false, "Site published with blup")
	}
	return ""
}

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

// Target is where the site is published: "blup" (Slug), "cloudflare"
// (Account, Project) or "github" (Repo).
type Target struct {
	Provider string `json:"provider"`
	Account  string `json:"account"`
	Project  string `json:"project"`
	Repo     string `json:"repo"`
	Slug     string `json:"slug"`
	// SlugFixed: the site is already on Blup Cloud under Slug.
	SlugFixed bool `json:"slugFixed"`
	// URL is the site's address, once published (read only).
	URL string `json:"url"`
}

// LoadTarget is the site's publishing destination, with suggested names
// for a site never published.
func LoadTarget(site *config.Site) Target {
	d := site.Deploy
	t := Target{Provider: string(d.Provider), Account: d.Cloudflare.Account, Project: d.Cloudflare.Project, Repo: d.GitHub.Repo, Slug: d.Blup.Slug, SlugFixed: d.Blup.ID != ""}
	if t.Slug == "" {
		t.Slug = blupSlug(site.Title)
	}
	if t.Project == "" {
		t.Project = cfpages.ProjectNameFromTitle(site.Title)
	}
	if t.Repo == "" {
		t.Repo = Slug(site.Title)
	}
	switch {
	case d.Provider == config.ProviderBlup && d.Blup.URL != "":
		t.URL = d.Blup.URL
	case d.Provider == config.ProviderCloudflare && d.Cloudflare.Subdomain != "":
		t.URL = siteURL(site, "https://"+d.Cloudflare.Subdomain+"/")
	case d.Provider == config.ProviderGitHub && d.GitHub.Owner != "" && d.GitHub.Repo != "":
		t.URL = siteURL(site, github.ExpectedURL(d.GitHub.Owner, d.GitHub.Repo))
	}
	return t
}

// SetTarget saves where the site is published.
func SetTarget(site *config.Site, t Target) error {
	switch config.Provider(t.Provider) {
	case config.ProviderBlup:
		// The address can change only before the first publish.
		if site.Deploy.Blup.ID == "" {
			slug := strings.TrimSpace(t.Slug)
			if !blupcloud.SiteSlug.MatchString(slug) || strings.Contains(slug, "--") {
				return errors.New(i18n.T("pub.blup_bad_slug"))
			}
			site.Deploy.Blup.Slug = slug
		}
	case config.ProviderCloudflare:
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
	case config.ProviderGitHub:
		if strings.TrimSpace(t.Repo) == "" {
			return errors.New(i18n.T("required"))
		}
		if !github.ValidRepoName(strings.TrimSpace(t.Repo)) {
			return errors.New(i18n.T("gh.bad_repo"))
		}
		site.Deploy.GitHub.Repo = strings.TrimSpace(t.Repo)
	default:
		return errors.New("unknown provider " + t.Provider)
	}
	site.Deploy.Provider = config.Provider(t.Provider)
	return site.Save()
}

// PublishNow builds the site and puts it online where it was set up.
func PublishNow(ctx context.Context, site *config.Site) (string, error) {
	var live string
	var err error
	switch site.Deploy.Provider {
	case config.ProviderBlup:
		live, err = deployBlup(ctx, site)
	case config.ProviderCloudflare:
		var c *cfpages.Client
		if c, err = savedCloudflare(); err == nil {
			live, err = deployCloudflare(ctx, site, c)
		}
	case config.ProviderGitHub:
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
	// TooLarge are files over the plan's (or GitHub's) size limit: they are
	// not synced.
	TooLarge []string `json:"tooLarge"`
}

func tooLarge(rep notesync.Report) []string {
	var out []string
	for _, p := range rep.Skipped {
		if rep.SkipReasons[p] == notesync.SkipTooLarge {
			out = append(out, p)
		}
	}
	return out
}

// syncer prepares where the notes go: Blup Cloud, or the private notes
// repository on GitHub (created the first time).
func syncer(ctx context.Context, site *config.Site) (*notesync.Syncer, string, error) {
	if site.Sync.Remote == RemoteBlup {
		sy, err := blupSyncer(ctx, site)
		return sy, "Blup Cloud", err
	}
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
	// The mark that lets another computer find the site; without it the
	// notes still sync.
	_ = c.AddTopic(ctx, s.Owner, s.Repo, SiteTopic)
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
	// Not on Blup Cloud yet (or let go of): a look makes no vault there,
	// only the Sync button does; everything here would be sent.
	if site.Sync.Remote == RemoteBlup && site.Sync.Vault == "" {
		return SyncInfo{Repo: "Blup Cloud", Send: obsidian.CountNotes(site.ContentDir())}, nil
	}
	sy, repo, err := syncer(ctx, site)
	if err != nil {
		return SyncInfo{}, err
	}
	plan, err := sy.Status(ctx, nil)
	if err != nil {
		if gone := unlinkGoneVault(ctx, site, err); gone != nil {
			return SyncInfo{}, gone
		}
		return SyncInfo{}, errors.New(describe(err))
	}
	return SyncInfo{
		Repo: repo, Send: len(plan.Uploaded) + len(plan.Deleted), Receive: len(plan.Downloaded) + len(plan.DeletedLocal),
		Conflicts: plan.Conflicts, UpToDate: plan.UpToDate && len(plan.Conflicts) == 0, TooLarge: tooLarge(plan),
	}, nil
}

// SyncNow sends and receives the notes, and the site's settings with them.
func SyncNow(ctx context.Context, site *config.Site) (SyncInfo, error) {
	return syncSite(ctx, site, true)
}

// syncSite syncs; share puts this computer's settings next to the notes
// first (a site just brought from GitHub only receives).
// syncMu runs one sync at a time (the button and the live sync share the
// state file).
var syncMu sync.Mutex

func syncSite(ctx context.Context, site *config.Site, share bool) (SyncInfo, error) {
	syncMu.Lock()
	defer syncMu.Unlock()
	sy, repo, err := syncer(ctx, site)
	if err != nil {
		return SyncInfo{}, err
	}
	if share {
		if err := writeShared(site); err != nil {
			return SyncInfo{}, err
		}
	}
	rep, err := sy.Sync(ctx, nil)
	if err != nil {
		if gone := unlinkGoneVault(ctx, site, err); gone != nil {
			return SyncInfo{}, gone
		}
		return SyncInfo{}, errors.New(describe(err))
	}
	// Settings changed on another computer.
	if share && slices.Contains(rep.Downloaded, notesync.SiteFile) {
		if err := readShared(site); err != nil {
			return SyncInfo{}, err
		}
	}
	return SyncInfo{
		Repo: repo, Send: len(rep.Uploaded) + len(rep.Deleted), Receive: len(rep.Downloaded) + len(rep.DeletedLocal),
		Conflicts: rep.ConflictCopies, UpToDate: true, TrashCount: len(rep.DeletedLocal), Trash: rep.Trash, TooLarge: tooLarge(rep),
	}, nil
}

// notesRepoName tells whether name ends like the notes repositories Blup
// names in any of its languages ("site-notas", "site-notes").
func notesRepoName(name string) bool {
	name = strings.ToLower(name)
	for _, suffix := range []string{"-notas", "-notes"} {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}
