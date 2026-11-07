package cfworkers

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

// MaxNameLen is the longest Worker name Blup uses: the name is the first
// label of <name>.<account>.workers.dev, and a DNS label holds 63 characters.
const MaxNameLen = 63

// URL is the address of a Worker on workers.dev, e.g.
// "https://meu-blog.ana.workers.dev/". subdomain is the account's
// workers.dev subdomain ("ana"), as Subdomain returns it.
func URL(name, subdomain string) string {
	return "https://" + Host(name, subdomain) + "/"
}

// Host is "<name>.<subdomain>.workers.dev".
func Host(name, subdomain string) string {
	return name + "." + subdomain + ".workers.dev"
}

func scriptPath(accountID, name string) string {
	return "/accounts/" + url.PathEscape(accountID) + "/workers/scripts/" + url.PathEscape(name)
}

// Subdomain returns the account's workers.dev subdomain ("ana" in
// ana.workers.dev), or ErrNoSubdomain when the account has none yet (a new
// account must pick one before its first Worker goes on workers.dev).
func (c *Client) Subdomain(ctx context.Context, accountID string) (string, error) {
	var r struct {
		Subdomain string `json:"subdomain"`
	}
	_, err := c.do(ctx, http.MethodGet, "/accounts/"+url.PathEscape(accountID)+"/workers/subdomain", "", nil, "", &r)
	if errors.Is(err, ErrNotFound) {
		return "", fmt.Errorf("%w: %w", ErrNoSubdomain, err)
	}
	if err != nil {
		return "", fmt.Errorf("reading the workers.dev subdomain: %w", err)
	}
	if r.Subdomain == "" {
		return "", ErrNoSubdomain
	}
	return r.Subdomain, nil
}

// RegisterSubdomain gives the account its workers.dev subdomain
// (<name>.workers.dev). ErrSubdomainTaken when another account has it.
// Changing it later moves every Worker of the account, so it is only used
// when the account has none.
func (c *Client) RegisterSubdomain(ctx context.Context, accountID, name string) (string, error) {
	if err := ValidateSubdomain(name); err != nil {
		return "", err
	}
	var r struct {
		Subdomain string `json:"subdomain"`
	}
	_, err := c.do(ctx, http.MethodPut, "/accounts/"+url.PathEscape(accountID)+"/workers/subdomain", "", map[string]string{"subdomain": name}, "", &r)
	if err != nil {
		return "", fmt.Errorf("creating the workers.dev subdomain %q: %w", name, err)
	}
	if r.Subdomain == "" {
		r.Subdomain = name
	}
	return r.Subdomain, nil
}

// Exists reports whether the account already has a Worker with this name.
func (c *Client) Exists(ctx context.Context, accountID, name string) (bool, error) {
	_, err := c.do(ctx, http.MethodGet, scriptPath(accountID, name)+"/settings", "", nil, "", nil)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	return err == nil, err
}

// Delete deletes a Worker with its static files (the site goes offline).
// A missing Worker is not an error.
func (c *Client) Delete(ctx context.Context, accountID, name string) error {
	_, err := c.do(ctx, http.MethodDelete, scriptPath(accountID, name), "", nil, "", nil)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

var labelRe = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)

// ValidateName reports whether name is a valid Worker name that also works
// as the first label of its workers.dev address: lowercase a-z, 0-9 and
// '-', not starting/ending with '-', at most 63 characters.
func ValidateName(name string) error {
	if len(name) == 0 || len(name) > MaxNameLen || !labelRe.MatchString(name) {
		return fmt.Errorf("%w: %q (use lowercase letters, digits and '-', up to %d characters)", ErrInvalidName, name, MaxNameLen)
	}
	return nil
}

// ValidateSubdomain checks an account workers.dev subdomain (same rules as
// wrangler: a DNS label of up to 63 characters).
func ValidateSubdomain(name string) error { return ValidateName(name) }

var accentFold = map[rune]string{
	'á': "a", 'à': "a", 'â': "a", 'ã': "a", 'ä': "a", 'å': "a", 'ā': "a",
	'é': "e", 'è': "e", 'ê': "e", 'ë': "e", 'ē': "e",
	'í': "i", 'ì': "i", 'î': "i", 'ï': "i", 'ī': "i",
	'ó': "o", 'ò': "o", 'ô': "o", 'õ': "o", 'ö': "o", 'ø': "o", 'ō': "o",
	'ú': "u", 'ù': "u", 'û': "u", 'ü': "u", 'ū': "u",
	'ç': "c", 'ñ': "n", 'ý': "y", 'ÿ': "y", 'ß': "ss", 'æ': "ae", 'œ': "oe",
	'&': "-e-",
}

// NameFromTitle turns a site title (or an account name) into a valid
// Worker name: accents folded, lowercase, anything else becomes '-', runs
// collapsed, trimmed to 63 chars. Returns "site" when nothing usable is left.
func NameFromTitle(title string) string {
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
	if len(s) > MaxNameLen {
		s = strings.TrimRight(s[:MaxNameLen], "-")
	}
	if s == "" {
		return "site"
	}
	return s
}
