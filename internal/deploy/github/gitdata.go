package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"unicode/utf8"
)

// Low-level Git Data API helpers used by two-way note syncing
// (internal/notesync). Publish keeps its own force-push flow; these never
// force a branch update.

// ErrNotFastForward: the branch moved since it was read (someone else
// pushed), so a non-forced update was refused.
var ErrNotFastForward = errors.New("github: branch moved; update is not a fast-forward")

// InitFile is the file written to initialize an empty repository (the Git
// Data API refuses to work on a repository without commits).
type InitFile struct {
	Path    string // slash-separated, e.g. ".blup/README.md"
	Content []byte
	Message string // commit message; default "Initialize repository"
}

// Tip is the head of a branch. Both fields are empty when the branch does
// not exist.
type Tip struct {
	Commit string
	Tree   string
}

// BranchTip returns the head commit and its tree. An empty repository is
// initialized with init (on its default branch) when init is non-nil;
// with init == nil it is reported as a missing branch and nothing is written.
func (c *Client) BranchTip(ctx context.Context, owner, repo, branch string, init *InitFile) (Tip, error) {
	commit, err := c.branchHeadInit(ctx, owner, repo, branch, init)
	if err != nil || commit == "" {
		return Tip{}, err
	}
	var cm struct {
		Tree struct {
			SHA string `json:"sha"`
		} `json:"tree"`
	}
	if _, err := c.do(ctx, http.MethodGet, repoPath(owner, repo)+"/git/commits/"+commit, nil, &cm); err != nil {
		return Tip{}, err
	}
	return Tip{Commit: commit, Tree: cm.Tree.SHA}, nil
}

// TreeFile is a blob entry of a tree.
type TreeFile struct {
	Path string // slash-separated, relative to the repository root
	SHA  string // blob sha
	Mode string // "100644", "100755" or "120000" (symlink)
}

// ReadTree lists every blob under treeSHA recursively (handling truncated
// responses), sorted by path.
func (c *Client) ReadTree(ctx context.Context, owner, repo, treeSHA string) ([]TreeFile, error) {
	m := map[string]TreeFile{}
	if err := c.readTreeFiles(ctx, owner, repo, treeSHA, m); err != nil {
		return nil, err
	}
	out := make([]TreeFile, 0, len(m))
	for _, f := range m {
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// CreateBlob uploads data as a blob and returns its sha (checked against
// BlobSHA). Files over MaxFileSize yield ErrFileTooLarge.
func (c *Client) CreateBlob(ctx context.Context, owner, repo string, data []byte) (string, error) {
	if len(data) > MaxFileSize {
		return "", ErrFileTooLarge
	}
	want := BlobSHA(data)
	var out struct {
		SHA string `json:"sha"`
	}
	if _, err := c.do(ctx, http.MethodPost, repoPath(owner, repo)+"/git/blobs", map[string]string{
		"content":  base64.StdEncoding.EncodeToString(data),
		"encoding": "base64",
	}, &out); err != nil {
		return "", err
	}
	if out.SHA != want {
		return "", fmt.Errorf("github: blob sha mismatch: local %s, remote %s", want, out.SHA)
	}
	return out.SHA, nil
}

// GetBlob downloads a blob's raw content (up to MaxFileSize) and verifies
// its sha.
func (c *Client) GetBlob(ctx context.Context, owner, repo, sha string) ([]byte, error) {
	_, data, err := c.send(ctx, http.MethodGet, repoPath(owner, repo)+"/git/blobs/"+sha, nil, "application/vnd.github.raw+json", MaxFileSize)
	if err != nil {
		return nil, err
	}
	if got := BlobSHA(data); got != sha {
		return nil, fmt.Errorf("github: downloaded blob %s has sha %s", sha, got)
	}
	return data, nil
}

// CanInline reports whether data can be sent inline in a tree request
// (non-empty UTF-8 text without NUL bytes, up to InlineTextLimit), saving
// one blob request.
func (c *Client) CanInline(data []byte) bool {
	limit := c.InlineTextLimit
	if limit == 0 {
		limit = 512 << 10
	}
	return limit > 0 && len(data) > 0 && int64(len(data)) <= limit &&
		utf8.Valid(data) && bytes.IndexByte(data, 0) < 0
}

// TreeChange is one change applied on top of a base tree.
type TreeChange struct {
	Path string // slash-separated
	// Delete removes Path from the tree.
	Delete bool
	// Content, when non-nil, is sent inline (see CanInline); otherwise SHA
	// must name an existing blob.
	Content []byte
	SHA     string
	Mode    string // default "100644"
}

// CreateTree applies changes on top of baseTree ("" for none) and returns
// the new tree sha. Large change sets are split into chunks chained with
// base_tree. With no changes it returns baseTree.
func (c *Client) CreateTree(ctx context.Context, owner, repo, baseTree string, changes []TreeChange) (string, error) {
	maxEntries := c.TreeChunkEntries
	if maxEntries <= 0 {
		maxEntries = 3000
	}
	maxBytes := c.TreeChunkBytes
	if maxBytes <= 0 {
		maxBytes = 16 << 20
	}
	var chunks [][]map[string]any
	var cur []map[string]any
	var curBytes int64
	for _, ch := range changes {
		if ch.Path == "" || strings.HasPrefix(ch.Path, "/") {
			return "", fmt.Errorf("github: invalid tree path %q", ch.Path)
		}
		mode := ch.Mode
		if mode == "" {
			mode = "100644"
		}
		e := map[string]any{"path": ch.Path, "mode": mode, "type": "blob"}
		var add int64
		switch {
		case ch.Delete:
			e["sha"] = nil
		case ch.Content != nil:
			e["content"] = string(ch.Content)
			add = int64(len(ch.Content))
		case ch.SHA != "":
			e["sha"] = ch.SHA
		default:
			return "", fmt.Errorf("github: tree change for %q has no content", ch.Path)
		}
		if len(cur) > 0 && (len(cur) >= maxEntries || curBytes+add > maxBytes) {
			chunks = append(chunks, cur)
			cur, curBytes = nil, 0
		}
		cur = append(cur, e)
		curBytes += add
	}
	if len(cur) > 0 {
		chunks = append(chunks, cur)
	}
	tree := baseTree
	for _, ch := range chunks {
		body := map[string]any{"tree": ch}
		if tree != "" {
			body["base_tree"] = tree
		}
		var out struct {
			SHA string `json:"sha"`
		}
		if _, err := c.do(ctx, http.MethodPost, repoPath(owner, repo)+"/git/trees", body, &out); err != nil {
			return "", err
		}
		tree = out.SHA
	}
	return tree, nil
}

// CreateCommit creates a commit object and returns its sha.
func (c *Client) CreateCommit(ctx context.Context, owner, repo, message, tree string, parents []string) (string, error) {
	if parents == nil {
		parents = []string{}
	}
	var out struct {
		SHA string `json:"sha"`
	}
	if _, err := c.do(ctx, http.MethodPost, repoPath(owner, repo)+"/git/commits", map[string]any{
		"message": message,
		"tree":    tree,
		"parents": parents,
	}, &out); err != nil {
		return "", err
	}
	return out.SHA, nil
}

// UpdateBranch moves branch to sha WITHOUT force (create == true creates
// the branch instead). If the branch moved meanwhile (or, when creating, it
// already exists) the error wraps ErrNotFastForward.
func (c *Client) UpdateBranch(ctx context.Context, owner, repo, branch, sha string, create bool) error {
	if create {
		_, err := c.do(ctx, http.MethodPost, repoPath(owner, repo)+"/git/refs", map[string]string{
			"ref": "refs/heads/" + branch,
			"sha": sha,
		}, nil)
		if statusOf(err) == http.StatusUnprocessableEntity && strings.Contains(strings.ToLower(errMessage(err)), "already exists") {
			return fmt.Errorf("%w: %w", ErrNotFastForward, err)
		}
		return err
	}
	p := strings.Replace(refPath(owner, repo, branch), "/git/ref/", "/git/refs/", 1)
	_, err := c.do(ctx, http.MethodPatch, p, map[string]any{"sha": sha, "force": false}, nil)
	if statusOf(err) == http.StatusUnprocessableEntity {
		m := strings.ToLower(errMessage(err))
		if strings.Contains(m, "fast forward") || strings.Contains(m, "fast-forward") {
			return fmt.Errorf("%w: %w", ErrNotFastForward, err)
		}
	}
	return err
}

func errMessage(err error) string {
	var e *APIError
	if errors.As(err, &e) {
		return e.Message + " " + strings.Join(e.Errors, " ")
	}
	return ""
}
