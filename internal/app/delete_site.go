package app

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/deploy/github"
)

// Deleting a site from Blup Studio: what it is made of, and taking each
// part away. Folders go to the system's trash (they can come back); what
// is online (the published site, the GitHub repositories) is deleted for
// good, and only when asked.

// SiteParts is what a site is made of, for the "delete" window.
type SiteParts struct {
	Title      string `json:"title"`
	SiteDir    string `json:"siteDir"`
	NotesDir   string `json:"notesDir"`
	NotesOwned bool   `json:"notesOwned"` // a folder Blup made for this site (else a vault the user already had)
	Published  string `json:"published"`  // "Cloudflare Pages: projeto" or "GitHub Pages: dono/repo"
	BlupSite   string `json:"blupSite"`   // its address on Blup Cloud, when it is published there
	SyncRepo   string `json:"syncRepo"`
	Feedback   string `json:"feedback"`
}

// DeleteChoice is what to delete.
type DeleteChoice struct {
	Site      bool `json:"site"`
	Notes     bool `json:"notes"`
	Published bool `json:"published"`
	BlupSite  bool `json:"blupSite"`
	SyncRepo  bool `json:"syncRepo"`
	Feedback  bool `json:"feedback"`
}

// PartsOf lists the parts of the site in dir.
func PartsOf(dir string) (SiteParts, error) {
	site, err := config.LoadSite(dir)
	if err != nil {
		return SiteParts{}, err
	}
	p := SiteParts{Title: site.Title, SiteDir: dir, NotesDir: site.ContentDir()}
	// Blup's own notes folder: the whole vault, made in Documents/Blup.
	p.NotesOwned = site.Folder == "" && filepath.Dir(filepath.Clean(site.Vault)) == filepath.Dir(DefaultNotesDir("x"))
	switch site.Deploy.Provider {
	case "cloudflare":
		if site.Deploy.Cloudflare.Project != "" {
			p.Published = "Cloudflare Pages: " + site.Deploy.Cloudflare.Project
		}
	case "github":
		if site.Deploy.GitHub.Repo != "" {
			p.Published = "GitHub Pages: " + site.Deploy.GitHub.Owner + "/" + site.Deploy.GitHub.Repo
		}
	}
	// On Blup Cloud whatever it publishes to now: a site moved to Cloudflare
	// or GitHub keeps its getblup.site address until it is taken down.
	if b := site.Deploy.Blup; b.ID != "" {
		p.BlupSite = cmp.Or(strings.TrimPrefix(b.URL, "https://"), b.Slug+".getblup.site")
	}
	if site.Sync.Repo != "" {
		p.SyncRepo = site.Sync.Owner + "/" + site.Sync.Repo
	}
	if site.Feedback.Repo != "" {
		p.Feedback = site.Feedback.Owner + "/" + site.Feedback.Repo
	}
	return p, nil
}

// DeleteProblem is a part that couldn't be deleted, and where to do it by
// hand.
type DeleteProblem struct {
	Part string `json:"part"`
	Err  string `json:"err"`
	Link string `json:"link"`
}

// DeleteSite takes the site out of Blup's list and deletes the parts
// chosen. Online parts that fail (a key without the permission) come back
// as problems with a link to delete them by hand; the rest goes on.
func DeleteSite(ctx context.Context, dir string, c DeleteChoice) ([]DeleteProblem, error) {
	site, err := config.LoadSite(dir)
	if err != nil {
		return nil, err
	}
	problems := []DeleteProblem{}
	fail := func(part string, err error, link string) {
		problems = append(problems, DeleteProblem{Part: part, Err: describe(err), Link: link})
	}
	if c.Published {
		switch site.Deploy.Provider {
		case "cloudflare":
			d := site.Deploy.Cloudflare
			if d.Project != "" {
				if cf, err := cloudflareClient(ctx); err != nil {
					fail("published", err, "https://dash.cloudflare.com/")
				} else if err := cf.DeleteProject(ctx, d.Account, d.Project); err != nil {
					fail("published", err, "https://dash.cloudflare.com/"+d.Account+"/pages/view/"+d.Project+"/settings/production")
				}
			}
		case "github":
			d := site.Deploy.GitHub
			if d.Repo != "" {
				deleteRepo(ctx, d.Owner, d.Repo, "published", fail)
			}
		}
	}
	if c.BlupSite && site.Deploy.Blup.ID != "" && deleteBlupSite(ctx, site.Deploy.Blup.ID, fail) && !c.Site {
		// The folder stays: it no longer points at the address taken down.
		site.Deploy.Blup = config.BlupSite{}
		if site.Deploy.Provider == "blup" {
			site.Deploy.Provider = ""
		}
		if err := site.Save(); err != nil {
			return problems, err
		}
	}
	if c.SyncRepo && site.Sync.Repo != "" {
		deleteRepo(ctx, site.Sync.Owner, site.Sync.Repo, "syncRepo", fail)
	}
	if c.Feedback && site.Feedback.Repo != "" {
		deleteRepo(ctx, site.Feedback.Owner, site.Feedback.Repo, "feedback", fail)
	}
	if c.Notes {
		if err := ToSystemTrash(site.ContentDir()); err != nil {
			return problems, err
		}
	}
	if c.Site {
		if err := ToSystemTrash(dir); err != nil {
			return problems, err
		}
	}
	return problems, ForgetSite(dir)
}

func deleteRepo(ctx context.Context, owner, repo, part string, fail func(string, error, string)) {
	link := "https://github.com/" + owner + "/" + repo + "/settings#danger-zone"
	c, _, err := githubClient(ctx)
	if err != nil {
		fail(part, err, link)
		return
	}
	if err := c.DeleteRepo(ctx, owner, repo); err != nil {
		if errors.Is(err, github.ErrPermission) {
			err = errors.New("the GitHub key can't delete repositories")
		}
		fail(part, err, link)
	}
}

// deleteBlupSite takes the site down from Blup Cloud. One already gone
// (deleted in the panel) is fine.
func deleteBlupSite(ctx context.Context, id string, fail func(string, error, string)) bool {
	link := blupcloud.BaseURL() + "/conta"
	c, _, err := blupClient()
	if err != nil {
		fail("blupSite", errors.New(describe(err)), link)
		return false
	}
	err = c.DeleteSite(ctx, id)
	if err == nil {
		return true
	}
	// Not found: already gone when the account no longer lists it.
	if sites, _, lerr := c.Sites(ctx); lerr == nil && !slices.ContainsFunc(sites, func(s blupcloud.Site) bool { return s.ID == id }) {
		return true
	}
	fail("blupSite", errors.New(describe(err)), link)
	return false
}

// ForgetSite takes a site out of Blup's list (nothing is deleted).
func ForgetSite(dir string) error {
	g, err := config.LoadGlobal()
	if err != nil {
		return err
	}
	g.Sites = slices.DeleteFunc(g.Sites, func(s string) bool { return filepath.Clean(s) == filepath.Clean(dir) })
	return g.Save()
}

// ToSystemTrash moves a folder to the system's trash (Linux, macOS,
// Windows), from where it can be restored.
func ToSystemTrash(path string) error {
	path = filepath.Clean(path)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	home, _ := os.UserHomeDir()
	if path == "/" || path == home || path == filepath.Dir(home) {
		return errors.New("refusing to delete " + path)
	}
	switch runtime.GOOS {
	case "windows":
		script := "Add-Type -AssemblyName Microsoft.VisualBasic; [Microsoft.VisualBasic.FileIO.FileSystem]::DeleteDirectory('" +
			strings.ReplaceAll(path, "'", "''") + "', 'OnlyErrorDialogs', 'SendToRecycleBin')"
		return exec.Command("powershell", "-NoProfile", "-Command", script).Run()
	case "darwin":
		return moveDir(path, freeName(filepath.Join(home, ".Trash"), filepath.Base(path)))
	}
	// The freedesktop.org trash: the file, and a note of where it was.
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" {
		data = filepath.Join(home, ".local", "share")
	}
	trash := filepath.Join(data, "Trash")
	dst := freeName(filepath.Join(trash, "files"), filepath.Base(path))
	if err := os.MkdirAll(filepath.Join(trash, "info"), 0o700); err != nil {
		return err
	}
	u := (&url.URL{Path: path}).EscapedPath()
	info := fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n", u, time.Now().Format("2006-01-02T15:04:05"))
	if err := os.WriteFile(filepath.Join(trash, "info", filepath.Base(dst)+".trashinfo"), []byte(info), 0o600); err != nil {
		return err
	}
	return moveDir(path, dst)
}

// freeName is dir/name, or dir/name.2, .3... when taken.
func freeName(dir, name string) string {
	p := filepath.Join(dir, name)
	for i := 2; ; i++ {
		if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
			return p
		}
		p = filepath.Join(dir, fmt.Sprintf("%s.%d", name, i))
	}
}
