// Package config reads and writes the CLI state: the global file (language,
// known sites), the private credentials file and each site's blup.toml.
package config

import (
	"errors"
	"os"
	"path/filepath"
	"slices"

	"github.com/pelletier/go-toml/v2"
)

// SiteFile is the name of the per-site settings file.
const SiteFile = "blup.toml"

// Global is ~/.config/blup/config.toml.
type Global struct {
	Lang  string   `toml:"lang"`
	Sites []string `toml:"sites"`
}

// Credentials is ~/.config/blup/credentials.toml (mode 0600).
type Credentials struct {
	CloudflareToken string `toml:"cloudflare_token,omitempty"`
	GitHubToken     string `toml:"github_token,omitempty"`
}

// Site is <site>/blup.toml.
type Site struct {
	Title       string `toml:"title"`
	Description string `toml:"description"`
	Author      string `toml:"author"`
	Lang        string `toml:"lang"`   // "pt" or "en": language of the site itself
	Model       string `toml:"model"`  // theme: "mochi" (personal blog) or "bento" (SaaS)
	Vault       string `toml:"vault"`  // absolute path of the Obsidian vault
	Folder      string `toml:"folder"` // folder inside the vault published as the site; "" = whole vault
	Domain      string `toml:"domain,omitempty"`
	AuthorBio   string `toml:"author_bio,omitempty"`
	// Password is the general password of protected notes ("password: sim").
	// It stays in this file, on the user's computer, and is never published.
	Password string `toml:"password,omitempty"`
	Logo     string `toml:"logo,omitempty"`   // image file chosen by the user
	Avatar   string `toml:"avatar,omitempty"` // author photo
	// Menu is the top menu. Empty means automatic: Home plus the top-level
	// folders of the site.
	Menu   []MenuItem `toml:"menu,omitempty"`
	Social []Social   `toml:"social,omitempty"`
	Deploy Deploy     `toml:"deploy"`

	dir string
}

// MenuItem is one entry of the top menu: a page of the site (Page, a path
// such as "/receitas") or an external link (URL).
type MenuItem struct {
	Name string `toml:"name"`
	Page string `toml:"page,omitempty"`
	URL  string `toml:"url,omitempty"`
}

// Social is a profile link shown in the footer.
type Social struct {
	Network string `toml:"network"` // github, x, instagram, linkedin, youtube, facebook, whatsapp, mail
	URL     string `toml:"url"`
}

// Deploy remembers where the site was published.
type Deploy struct {
	Provider   string     `toml:"provider,omitempty"` // "cloudflare" or "github"
	Cloudflare Cloudflare `toml:"cloudflare,omitempty"`
	GitHub     GitHub     `toml:"github,omitempty"`
}

type Cloudflare struct {
	Account   string `toml:"account,omitempty"`
	Project   string `toml:"project,omitempty"`
	Subdomain string `toml:"subdomain,omitempty"`
}

type GitHub struct {
	Owner string `toml:"owner,omitempty"`
	Repo  string `toml:"repo,omitempty"`
}

// Dir is the folder where the CLI keeps its files.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "blup"), nil
}

// LoadGlobal returns the global config; a missing file is not an error.
func LoadGlobal() (Global, error) {
	var g Global
	err := load("config.toml", &g)
	return g, err
}

// Save writes the global config.
func (g Global) Save() error { return save("config.toml", g, 0o644) }

// AddSite remembers a site folder (most recent first).
func (g *Global) AddSite(dir string) {
	g.Sites = slices.DeleteFunc(g.Sites, func(s string) bool { return s == dir })
	g.Sites = append([]string{dir}, g.Sites...)
}

// ExistingSites returns the remembered sites whose folder still exists.
func (g Global) ExistingSites() []string {
	var out []string
	for _, s := range g.Sites {
		if _, err := os.Stat(filepath.Join(s, SiteFile)); err == nil {
			out = append(out, s)
		}
	}
	return out
}

// LoadCredentials returns the stored tokens.
func LoadCredentials() (Credentials, error) {
	var c Credentials
	err := load("credentials.toml", &c)
	return c, err
}

// Save writes the credentials readable only by the user.
func (c Credentials) Save() error { return save("credentials.toml", c, 0o600) }

// LoadSite reads dir/blup.toml.
func LoadSite(dir string) (*Site, error) {
	b, err := os.ReadFile(filepath.Join(dir, SiteFile))
	if err != nil {
		return nil, err
	}
	s := &Site{dir: dir}
	if err := toml.Unmarshal(b, s); err != nil {
		return nil, err
	}
	// Sites created before the themes had names.
	switch s.Model {
	case "blog":
		s.Model = "mochi"
	case "docs":
		s.Model = "bento"
	}
	return s, nil
}

// NewSite returns an empty site bound to dir.
func NewSite(dir string) *Site { return &Site{dir: dir} }

// Dir is the site folder.
func (s *Site) Dir() string { return s.dir }

// ContentDir is the vault folder published as the site.
func (s *Site) ContentDir() string { return filepath.Join(s.Vault, s.Folder) }

// Save writes dir/blup.toml.
func (s *Site) Save() error {
	b, err := toml.Marshal(s)
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(s.dir, SiteFile), b, 0o644)
}

func load(name string, v any) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	b, err := os.ReadFile(filepath.Join(dir, name))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return toml.Unmarshal(b, v)
}

func save(name string, v any, mode os.FileMode) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, err := toml.Marshal(v)
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(dir, name), b, mode)
}

func writeAtomic(path string, b []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
