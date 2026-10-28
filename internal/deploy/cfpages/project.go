package cfpages

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

// ProductionBranch is the branch name used for production deployments.
const ProductionBranch = "main"

// MaxProjectNameLen is the Pages limit for project names.
const MaxProjectNameLen = 58

// Project is a Cloudflare Pages project.
type Project struct {
	Name string
	// Subdomain is the *.pages.dev host, e.g. "meu-blog-abc.pages.dev".
	// Cloudflare may add a suffix when the plain name is taken globally, so
	// always use this value (not Name) to build the site's baseURL.
	Subdomain string
	// Domains lists the subdomain plus any custom domains.
	Domains []string
}

// URL is the production URL, e.g. "https://meu-blog-abc.pages.dev/".
func (p Project) URL() string { return "https://" + p.Subdomain + "/" }

type apiProject struct {
	Name             string   `json:"name"`
	Subdomain        string   `json:"subdomain"`
	Domains          []string `json:"domains"`
	ProductionBranch string   `json:"production_branch"`
}

func (p apiProject) toProject() Project {
	return Project{Name: p.Name, Subdomain: p.Subdomain, Domains: p.Domains}
}

func (c *Client) getProject(ctx context.Context, accountID, name string) (apiProject, error) {
	var p apiProject
	_, err := c.do(ctx, http.MethodGet, projectPath(accountID, name), "", nil, "", &p)
	return p, err
}

func projectPath(accountID, name string) string {
	return "/accounts/" + url.PathEscape(accountID) + "/pages/projects/" + url.PathEscape(name)
}

// EnsureProject returns the Pages project, creating it (production branch
// "main", Direct Upload) when it does not exist yet. Use Project.Subdomain /
// URL() as the site's baseURL before building.
func (c *Client) EnsureProject(ctx context.Context, accountID, name string) (Project, error) {
	if err := ValidateProjectName(name); err != nil {
		return Project{}, err
	}
	p, err := c.getProject(ctx, accountID, name)
	if err == nil {
		return p.toProject(), nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Project{}, fmt.Errorf("looking up Pages project %q: %w", name, err)
	}
	body := map[string]any{"name": name, "production_branch": ProductionBranch}
	var created apiProject
	if _, err := c.do(ctx, http.MethodPost, "/accounts/"+url.PathEscape(accountID)+"/pages/projects", "", body, "", &created); err != nil {
		var ae *APIError
		if errors.As(err, &ae) && ae.Status == http.StatusBadRequest && ae.kind == nil {
			// Pages answers 400 for a name it refuses (reserved / in use).
			return Project{}, fmt.Errorf("creating Pages project %q: %w: %w", name, ErrProjectNameTaken, err)
		}
		return Project{}, fmt.Errorf("creating Pages project %q: %w", name, err)
	}
	if created.Subdomain == "" {
		if got, err := c.getProject(ctx, accountID, name); err == nil {
			created = got
		}
	}
	if created.Subdomain == "" {
		created.Subdomain = name + ".pages.dev"
	}
	if created.Name == "" {
		created.Name = name
	}
	return created.toProject(), nil
}

var projectNameRe = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)

// ValidateProjectName reports whether name is a valid Pages project name:
// lowercase a-z, 0-9 and '-', not starting/ending with '-', at most 58 chars.
func ValidateProjectName(name string) error {
	if len(name) == 0 || len(name) > MaxProjectNameLen || !projectNameRe.MatchString(name) {
		return fmt.Errorf("%w: %q (use lowercase letters, digits and '-', up to %d characters)", ErrInvalidProjectName, name, MaxProjectNameLen)
	}
	return nil
}

var accentFold = map[rune]string{
	'á': "a", 'à': "a", 'â': "a", 'ã': "a", 'ä': "a", 'å': "a", 'ā': "a",
	'é': "e", 'è': "e", 'ê': "e", 'ë': "e", 'ē': "e",
	'í': "i", 'ì': "i", 'î': "i", 'ï': "i", 'ī': "i",
	'ó': "o", 'ò': "o", 'ô': "o", 'õ': "o", 'ö': "o", 'ø': "o", 'ō': "o",
	'ú': "u", 'ù': "u", 'û': "u", 'ü': "u", 'ū': "u",
	'ç': "c", 'ñ': "n", 'ý': "y", 'ÿ': "y", 'ß': "ss", 'æ': "ae", 'œ': "oe",
	'&': "-e-",
}

// ProjectNameFromTitle turns a site title into a valid project name:
// accents folded, lowercase, anything else becomes '-', runs collapsed,
// trimmed to 58 chars. Returns "site" when nothing usable is left.
func ProjectNameFromTitle(title string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
		case accentFold[r] != "":
			b.WriteString(accentFold[r])
		case unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r):
			b.WriteByte('-')
		}
	}
	s := b.String()
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	if len(s) > MaxProjectNameLen {
		s = strings.TrimRight(s[:MaxProjectNameLen], "-")
	}
	if s == "" {
		return "site"
	}
	return s
}
