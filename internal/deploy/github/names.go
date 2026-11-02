package github

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Repository names and the links that let the person create a repository
// and a key limited to it.
//
// Owners and repository names come from site.toml (which also travels
// between computers with the notes) and from the window, and they become
// part of the API path. Only what GitHub itself accepts goes through: no
// "..", no "/", no "?", no "%", no leading "-". Nothing here runs git, so a
// remote URL never reaches a git binary; the names only build REST paths.

// ErrBadName: an owner or repository name GitHub would never accept.
var ErrBadName = errors.New("github: invalid owner or repository name")

// ValidOwner tells whether s can be a GitHub user or organization login:
// 1 to 39 letters, digits and single hyphens, not starting or ending with
// a hyphen.
func ValidOwner(s string) bool {
	if s == "" || len(s) > 39 || s[0] == '-' || s[len(s)-1] == '-' || strings.Contains(s, "--") {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
			return false
		}
	}
	return true
}

// ValidRepoName tells whether s can be a repository name: 1 to 100 ASCII
// letters, digits, ".", "-" and "_", not "." or "..", not starting with "-"
// (it would read as an option to a command line tool) and not ending in
// ".git".
func ValidRepoName(s string) bool {
	if s == "" || len(s) > 100 || s == "." || s == ".." || s[0] == '-' || strings.HasSuffix(strings.ToLower(s), ".git") {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}

// checkRepoPath refuses an API path "/repos/{owner}/{repo}..." whose owner
// or name is not a valid one, before anything is sent.
func checkRepoPath(path string) error {
	rest, ok := strings.CutPrefix(path, "/repos/")
	if !ok {
		return nil
	}
	parts := strings.SplitN(rest, "/", 3)
	if len(parts) < 2 {
		return ErrBadName
	}
	owner, err1 := url.PathUnescape(parts[0])
	name, err2 := url.PathUnescape(strings.SplitN(parts[1], "?", 2)[0])
	if err1 != nil || err2 != nil || !ValidOwner(owner) || !ValidRepoName(name) {
		return fmt.Errorf("%w: %q/%q", ErrBadName, parts[0], parts[1])
	}
	return nil
}

// ErrRepoMissing: the repository does not exist, or the key cannot see
// it, and the key cannot create it (a key limited to chosen repositories
// has no Administration permission). See RepoMissingError.
var ErrRepoMissing = errors.New("github: repository missing or not shared with the key")

// RepoMissingError says which repository the person must create on GitHub
// (CreateURL opens the form already filled in) or add to the key.
type RepoMissingError struct {
	Owner, Name string
	Private     bool
	CreateURL   string
	cause       error
}

func (e *RepoMissingError) Error() string {
	return fmt.Sprintf("github: %s/%s does not exist or the key has no access to it (create it at %s): %v", e.Owner, e.Name, e.CreateURL, e.cause)
}

// Is matches ErrRepoMissing only: the generic "no permission" message
// would send the person to the wrong fix.
func (e *RepoMissingError) Is(target error) bool { return target == ErrRepoMissing }

// NewRepoURL opens GitHub's "new repository" form filled in with the name,
// the owner, the visibility and a description
// (https://docs.github.com/en/repositories/creating-and-managing-repositories/creating-a-new-repository).
func NewRepoURL(owner, name string, private bool, description string) string {
	q := url.Values{}
	q.Set("name", name)
	if owner != "" {
		q.Set("owner", owner)
	}
	if private {
		q.Set("visibility", "private")
	} else {
		q.Set("visibility", "public")
	}
	if description != "" {
		q.Set("description", description)
	}
	return "https://github.com/new?" + q.Encode()
}

// TokenPermissions are the repository permissions of the fine-grained key
// Blup asks for, and nothing else: no Administration (the person creates
// the repositories on GitHub and gives the key only those), no account
// permission. Metadata read is always granted; it is listed so the form
// shows it.
//
//	contents: write   git blobs, trees, commits, refs (site and notes)
//	pages:    write   turn GitHub Pages on, follow builds, own domain
//	issues:   write   comments and reactions of the site (feedback repo)
//	metadata: read    read the repository
//
// https://docs.github.com/en/rest/authentication/permissions-required-for-fine-grained-personal-access-tokens
var TokenPermissions = [][2]string{
	{"contents", "write"},
	{"pages", "write"},
	{"issues", "write"},
	{"metadata", "read"},
}

// TokenURL opens GitHub's form for a fine-grained key, filled in with name,
// description, a year of validity and TokenPermissions. The repositories
// cannot be filled in by link: the person picks "Only select repositories"
// and the ones Blup uses.
// https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens
func TokenURL(name, description string) string {
	q := url.Values{}
	q.Set("name", name)
	q.Set("description", description)
	q.Set("expires_in", "366")
	for _, p := range TokenPermissions {
		q.Set(p[0], p[1])
	}
	return "https://github.com/settings/personal-access-tokens/new?" + q.Encode()
}
