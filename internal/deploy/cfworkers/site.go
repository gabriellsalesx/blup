package cfworkers

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

// MaxScriptNameLen keeps <script> a valid DNS label in
// <script>.<subdomain>.workers.dev.
const MaxScriptNameLen = 63

// Site describes where an assets-only Worker is (or will be) served.
type Site struct {
	// Script is the Worker name.
	Script string
	// AccountSubdomain is the account's workers.dev label, e.g. "gabriell".
	AccountSubdomain string
	// Host is "<script>.<account-subdomain>.workers.dev".
	Host string
	// URL is "https://<Host>/" — use it as the Hugo baseURL before building.
	URL string
	// Exists reports whether a Worker with this name already exists in the
	// account (a deploy would replace it). Best effort.
	Exists bool
}

func scriptPath(accountID, script string) string {
	return "/accounts/" + url.PathEscape(accountID) + "/workers/scripts/" + url.PathEscape(script)
}

// accountSubdomain returns the account's workers.dev label.
func (c *Client) accountSubdomain(ctx context.Context, accountID string) (string, error) {
	// Note (from wrangler): the API docs call the field "name" but it is "subdomain".
	var r struct {
		Subdomain string `json:"subdomain"`
		Name      string `json:"name"`
	}
	_, err := c.get(ctx, "/accounts/"+url.PathEscape(accountID)+"/workers/subdomain", &r)
	if err != nil {
		var ae *APIError
		if errors.As(err, &ae) && ae.hasCode(codeSubdomainNotFound) {
			return "", fmt.Errorf("open %s once to choose one: %w", WorkersOnboardingURL(accountID), ErrNoWorkersSubdomain)
		}
		return "", fmt.Errorf("reading the account's workers.dev subdomain: %w", err)
	}
	s := r.Subdomain
	if s == "" {
		s = r.Name
	}
	if s == "" {
		return "", fmt.Errorf("open %s once to choose one: %w", WorkersOnboardingURL(accountID), ErrNoWorkersSubdomain)
	}
	return s, nil
}

// EnsureSite validates the Worker name and returns the final public URL
// (https://<script>.<account-subdomain>.workers.dev/) BEFORE anything is
// built or uploaded. It creates nothing: the Worker is created by Deploy.
// If the account has no workers.dev subdomain it returns
// ErrNoWorkersSubdomain (the user must open the Workers dashboard once).
func (c *Client) EnsureSite(ctx context.Context, accountID, script string) (Site, error) {
	if err := ValidateScriptName(script); err != nil {
		return Site{}, err
	}
	sub, err := c.accountSubdomain(ctx, accountID)
	if err != nil {
		return Site{}, err
	}
	host := script + "." + sub + ".workers.dev"
	site := Site{Script: script, AccountSubdomain: sub, Host: host, URL: "https://" + host + "/"}
	// Same endpoint wrangler uses to read a Worker's subdomain settings.
	if _, err := c.get(ctx, "/accounts/"+url.PathEscape(accountID)+"/workers/workers/"+url.PathEscape(script), nil); err == nil {
		site.Exists = true
	} else if errors.Is(err, ErrInvalidToken) || errors.Is(err, ErrMissingPermission) {
		return Site{}, fmt.Errorf("checking worker %q: %w", script, err)
	}
	return site, nil
}

var scriptNameRe = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)

// ValidateScriptName reports whether name works as a Worker name and as the
// first label of its workers.dev hostname: lowercase a-z, 0-9 and '-', not
// starting/ending with '-', at most 63 characters.
func ValidateScriptName(name string) error {
	if len(name) == 0 || len(name) > MaxScriptNameLen || !scriptNameRe.MatchString(name) {
		return fmt.Errorf("%w: %q (use lowercase letters, digits and '-', up to %d characters)", ErrInvalidScriptName, name, MaxScriptNameLen)
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

// ScriptNameFromTitle turns a site title into a valid Worker name: accents
// folded, lowercase, anything else becomes '-', runs collapsed, trimmed to
// 63 chars. Returns "site" when nothing usable is left.
func ScriptNameFromTitle(title string) string {
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
	if len(s) > MaxScriptNameLen {
		s = strings.TrimRight(s[:MaxScriptNameLen], "-")
	}
	if s == "" {
		return "site"
	}
	return s
}
