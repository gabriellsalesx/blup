package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/deploy/github"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/ui"
)

// Feedback (Bento): votes, reactions and comments live in GitHub issues of
// a public repository named after the site plus "-feedback". On every
// publish the theme lists the pages that get feedback (roadmap items,
// changelog entries, blog posts) in blup-feedback.json; this opens an issue
// for each new page, keeps titles in sync, and writes the issue numbers to
// data/blup_feedback.json for the theme. Visitors read everything live from
// GitHub's public API. With the login service's GitHub App installed on the
// repository (github.com/gabriellsalesx/blup-api), they sign in with GitHub
// and vote, react and comment on the site itself; without it, on GitHub.

const (
	feedbackManifest = "blup-feedback.json"
	feedbackLabel    = "blup"
)

// feedbackDataFile is where the theme finds the issue of each page.
func feedbackDataFile(site *config.Site) string {
	return filepath.Join(site.Dir(), "data", "blup_feedback.json")
}

type feedbackPage struct {
	Key   string `json:"key"`
	Kind  string `json:"kind"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

// feedbackManifestFile is what the theme writes: the site's address and
// the pages that get feedback.
type feedbackManifestFile struct {
	Base  string         `json:"base"`
	Pages []feedbackPage `json:"pages"`
}

type feedbackData struct {
	Repo string `json:"repo"`
	// Site is the address last written to the repository's "website" field,
	// which the login service checks.
	Site string `json:"site,omitempty"`
	// API is the login service, set only while its GitHub App is installed
	// on the repository: the theme then lets visitors vote, react and
	// comment on the site itself.
	API    string         `json:"api,omitempty"`
	Issues map[string]int `json:"issues"`
}

var feedbackMarkerRe = regexp.MustCompile(`<!-- blup:(.+?) -->`)

// feedbackRepoName is the repository of the feedback: the site's name
// plus "-feedback".
func feedbackRepoName(site *config.Site) string {
	return Slug(site.Title) + "-feedback"
}

// readFeedbackManifest reads and deletes the list the theme wrote.
func readFeedbackManifest(dir string) (feedbackManifestFile, error) {
	var m feedbackManifestFile
	path := filepath.Join(dir, feedbackManifest)
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return m, nil
	}
	if err != nil {
		return m, err
	}
	os.Remove(path) // internal: never published
	if err := json.Unmarshal(b, &m); err != nil {
		return m, err
	}
	return m, nil
}

func loadFeedbackData(site *config.Site) feedbackData {
	var d feedbackData
	if b, err := os.ReadFile(feedbackDataFile(site)); err == nil {
		_ = json.Unmarshal(b, &d)
	}
	if d.Issues == nil {
		d.Issues = map[string]int{}
	}
	return d
}

func saveFeedbackData(site *config.Site, d feedbackData) error {
	if err := os.MkdirAll(filepath.Dir(feedbackDataFile(site)), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(feedbackDataFile(site), append(b, '\n'), 0o644)
}

func feedbackBody(site *config.Site, p feedbackPage) string {
	text := map[string]string{
		"pt": "Vote com 👍, reaja ou comente aqui: tudo aparece no site.",
		"en": "Vote with 👍, react or comment here: it all shows up on the site.",
	}[site.Lang]
	if text == "" {
		text = "Vote with 👍, react or comment here: it all shows up on the site."
	}
	link := ""
	if strings.HasPrefix(p.URL, "http") { // a build without the live address has relative links
		link = p.URL + "\n\n"
	}
	return fmt.Sprintf("**%s**\n\n%s%s\n\n<!-- blup:%s -->", p.Title, link, text, p.Key)
}

// feedbackRepoDescription is in the site's language: visitors read it.
func feedbackRepoDescription(site *config.Site) string {
	if site.Lang == "pt" {
		return fmt.Sprintf("Votos, reações e comentários de %s (gerenciado pelo blup)", site.Title)
	}
	return fmt.Sprintf("Votes, reactions and comments of %s (managed by blup)", site.Title)
}

// syncFeedback opens or updates the issue of every page in the manifest of
// dir, keeps the repository's "website" on the site's address and checks
// the login service's App. It reports whether the theme's data changed (the
// site must be built again to show it).
func syncFeedback(ctx context.Context, site *config.Site, dir string) (bool, error) {
	m, err := readFeedbackManifest(dir)
	if err != nil || !site.Feedback.On() || len(m.Pages) == 0 {
		return false, err
	}
	c, err := feedbackClient(ctx)
	if err != nil || c == nil {
		return false, err
	}
	owner, repo := site.Feedback.Owner, site.Feedback.Repo
	data := loadFeedbackData(site)
	before := fmt.Sprint(data.API, data.Issues)
	data.Repo = owner + "/" + repo
	data.Issues = map[string]int{} // pages that are gone lose theirs
	var created, updated int
	err = ui.Spin(ctx, i18n.T("fb.syncing"), func(ctx context.Context) error {
		var err error
		created, updated, err = syncIssues(ctx, c, site, m.Pages, data.Issues)
		if err != nil {
			return err
		}
		// A build without the live address (no --url) has relative links.
		if strings.HasPrefix(m.Base, "http") && data.Site != m.Base {
			if err := c.SetHomepage(ctx, owner, repo, m.Base); err != nil {
				return err
			}
			data.Site = m.Base
		}
		data.API = ""
		if ok, err := appInstalled(ctx, site.Feedback); err == nil && ok {
			data.API = site.Feedback.LoginAPI()
		}
		return nil
	})
	if errors.Is(err, github.ErrPermission) {
		return false, errors.New(i18n.T("fb.need_issues_customize"))
	}
	if err != nil {
		return false, err
	}
	if err := saveFeedbackData(site, data); err != nil {
		return false, err
	}
	if created+updated > 0 {
		ui.OK(i18n.T("fb.synced", created, updated, owner+"/"+repo))
	}
	return fmt.Sprint(data.API, data.Issues) != before, nil
}

// appInstalled asks the login service whether its GitHub App is installed
// on the feedback repository.
func appInstalled(ctx context.Context, f config.Feedback) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	q := url.Values{"repo": {f.Owner + "/" + f.Repo}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.LoginAPI()+"/installed?"+q.Encode(), nil)
	if err != nil {
		return false, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return false, fmt.Errorf("%s: %s", f.LoginAPI(), res.Status)
	}
	var v struct{ Installed bool }
	if err := json.NewDecoder(res.Body).Decode(&v); err != nil {
		return false, err
	}
	return v.Installed, nil
}

// installApp guides the install of the login service's GitHub App on the
// feedback repository. Without it, voting, reacting and commenting happen
// on GitHub instead of on the site.
func installApp(ctx context.Context, site *config.Site) error {
	f := site.Feedback
	if ok, err := appInstalled(ctx, f); err != nil || ok {
		return err
	}
	repo := f.Owner + "/" + f.Repo
	link := "https://github.com/apps/" + f.AppSlug() + "/installations/new"
	if err := ui.Explain(i18n.T("fb.app_title"), i18n.T("fb.app_steps", f.AppSlug(), f.Repo)); err != nil {
		return err
	}
	ui.OpenBrowser(link)
	ui.Hint(i18n.T("browser_fallback", link))
	for {
		done, err := ui.Confirm(i18n.T("fb.app_q"), "", true)
		if err != nil || !done {
			ui.Hint(i18n.T("fb.app_later"))
			return nil
		}
		ok, err := appInstalled(ctx, f)
		if err != nil {
			return err
		}
		if ok {
			ui.OK(i18n.T("fb.app_ok", repo))
			return nil
		}
		ui.Warn(i18n.T("fb.app_missing", f.Repo))
	}
}

// feedbackClient is the GitHub client of the user. Without a terminal and
// without a saved key, feedback is skipped (with a warning) instead of
// asking.
func feedbackClient(ctx context.Context) (*github.Client, error) {
	creds, err := config.LoadCredentials()
	if err != nil {
		return nil, err
	}
	if creds.GitHubToken == "" && !ui.Interactive() {
		ui.Warn(i18n.T("fb.no_token"))
		return nil, nil
	}
	c, _, err := githubClient(ctx)
	return c, err
}

// editFeedback turns feedback on or off (customize, and the new site wizard).
func editFeedback(ctx context.Context, site *config.Site) error {
	if site.Model != "bento" {
		return ui.Explain(i18n.T("fb.title"), i18n.T("fb.bento_only"))
	}
	if err := ui.Explain(i18n.T("fb.title"), i18n.T("fb.about", feedbackRepoName(site))); err != nil {
		return err
	}
	on, err := ui.Confirm(i18n.T("fb.q"), "", site.Feedback.Enabled || site.Feedback.Repo == "")
	if err != nil {
		return err
	}
	if !on {
		site.Feedback.Enabled = false
		ui.OK(i18n.T("fb.off"))
		ui.Hint(i18n.T("fb.republish"))
		return nil
	}
	return enableFeedback(ctx, site)
}

// enableFeedback creates the public "-feedback" repository (once) and
// turns feedback on. Issues are opened on the next publish.
func enableFeedback(ctx context.Context, site *config.Site) error {
	f := &site.Feedback
	var repo github.Repo
	for {
		c, login, err := githubClient(ctx)
		if err != nil {
			return err
		}
		if f.Repo == "" {
			f.Owner, f.Repo = login, feedbackRepoName(site)
		}
		err = ui.Spin(ctx, i18n.T("fb.creating"), func(ctx context.Context) error {
			var err error
			repo, err = c.EnsureRepoWith(ctx, f.Owner, f.Repo, github.EnsureRepoOptions{
				Description: feedbackRepoDescription(site),
				Issues:      true,
				NoHomepage:  true,
			})
			if err != nil || repo.Private {
				return err
			}
			// An existing repository may have issues off.
			if err := c.EnableIssues(ctx, f.Owner, f.Repo); err != nil {
				return err
			}
			// Keys made before feedback existed can't write issues: find
			// out now, not on the next publish.
			return c.EnsureLabel(ctx, f.Owner, f.Repo, feedbackLabel, "6fc0ee", "blup")
		})
		if !errors.Is(err, github.ErrPermission) {
			if err != nil {
				return err
			}
			break
		}
		ui.Error(i18n.T("fb.need_issues"))
		if ok, _ := ui.Confirm(i18n.T("gh.retry"), "", true); !ok {
			return nil // already explained; nothing changes
		}
		if err := forgetGitHubToken(); err != nil {
			return err
		}
	}
	if repo.Private {
		return errors.New(i18n.T("fb.private", repo.FullName))
	}
	f.Enabled = true
	ui.OK(i18n.T("fb.on", repo.FullName))
	if err := installApp(ctx, site); err != nil {
		ui.Warn(i18n.T("fb.failed", describe(err)))
	}
	ui.Hint(i18n.T("fb.republish"))
	return nil
}

// forgetGitHubToken drops the saved key, so the next githubClient asks for
// a new one.
func forgetGitHubToken() error {
	creds, err := config.LoadCredentials()
	if err != nil {
		return err
	}
	creds.GitHubToken = ""
	return creds.Save()
}

// syncIssues opens an issue for each page without one and keeps titles and
// bodies up to date. issues (page key → number) is filled in place.
func syncIssues(ctx context.Context, c *github.Client, site *config.Site, pages []feedbackPage, issues map[string]int) (created, updated int, err error) {
	owner, repo := site.Feedback.Owner, site.Feedback.Repo
	for _, l := range []struct{ name, color string }{
		{feedbackLabel, "6fc0ee"}, {"roadmap", "a99cf5"}, {"changelog", "9ee6c8"}, {"blog", "ffc29e"},
	} {
		if err := c.EnsureLabel(ctx, owner, repo, l.name, l.color, "blup"); err != nil {
			return created, updated, err
		}
	}
	existing, err := c.ListIssues(ctx, owner, repo, feedbackLabel)
	if err != nil {
		return created, updated, err
	}
	byKey := map[string]github.Issue{}
	for _, is := range existing {
		if m := feedbackMarkerRe.FindStringSubmatch(is.Body); m != nil {
			byKey[m[1]] = is
		}
	}
	for _, p := range pages {
		body := feedbackBody(site, p)
		if is, ok := byKey[p.Key]; ok {
			if is.Title != p.Title || strings.TrimSpace(is.Body) != body {
				if err := c.UpdateIssue(ctx, owner, repo, is.Number, p.Title, body); err != nil {
					return created, updated, err
				}
				updated++
			}
			issues[p.Key] = is.Number
			continue
		}
		is, err := c.CreateIssue(ctx, owner, repo, p.Title, body, []string{feedbackLabel, p.Kind})
		if err != nil {
			return created, updated, err
		}
		issues[p.Key] = is.Number
		created++
	}
	return created, updated, nil
}
