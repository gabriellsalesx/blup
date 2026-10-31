package notesync

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/gabriellsalesx/blup/internal/deploy/github"
)

// MaxFileSize is the largest file synced by default (GitHub's per-file
// limit). A Remote with a MaxFileSize method sets its own.
const MaxFileSize = github.MaxFileSize

// Limited is a Remote whose largest file is not MaxFileSize (a plan's limit).
type Limited interface {
	MaxFileSize() int64
}

// ErrNotFastForward is returned by Remote.Commit when someone else saved a
// new version since Head: Sync then compares again.
var ErrNotFastForward = errors.New("notesync: the remote moved on")

// Remote is where the notes are synced to: a GitHub repository or Blup
// Cloud. Sync keeps the three-way rules, conflicts, trash and state; a
// Remote only stores versions of the folder.
type Remote interface {
	// ID names the destination; a state file saved for another ID is
	// ignored.
	ID() string
	// Head reads the current version. write allows preparing the remote
	// (Sync) or not (Status). An empty remote has Version "".
	Head(ctx context.Context, write bool) (Head, error)
	// Upload stores the content of a file at path whose git blob sha is sha,
	// and returns the change that puts it at path.
	Upload(ctx context.Context, path, sha string, data []byte) (Change, error)
	// Copy is the change that puts an existing remote file at path.
	Copy(f RemoteFile, path string) Change
	// Download returns the content of a remote file.
	Download(ctx context.Context, f RemoteFile) ([]byte, error)
	// Commit saves head plus changes as the new version, only if head is
	// still the current one (else ErrNotFastForward). It returns the new
	// version (head.Version when nothing changed).
	Commit(ctx context.Context, head Head, changes []Change, counts Counts) (string, error)
	// Concurrency is how many files are sent or received at once.
	Concurrency() int
}

// Head is a version of the remote folder.
type Head struct {
	Version string
	Files   []RemoteFile
	// Extra is the remote's own data about this version.
	Extra any
}

// RemoteFile is a file of a remote version.
type RemoteFile struct {
	Path string // exact remote path, slash-separated
	SHA  string // git blob sha of the content
	Mode string // "100644", "100755"; anything else is unsupported
	// Ref is the remote's own handle on the content.
	Ref any
}

// Change is one path of a new version.
type Change struct {
	Path   string
	Mode   string
	Delete bool
	SHA    string
	Ref    any
}

// Counts summarize a new version: files updated, added and removed.
type Counts struct{ Updated, Added, Removed int }

// GitHub syncs with Branch of Owner/Repo.
type GitHub struct {
	Client *github.Client
	Owner  string
	Repo   string
	Branch string // default "main"

	mu    sync.Mutex
	known map[string]bool // blob shas already on GitHub
}

func (g *GitHub) branch() string {
	if g.Branch != "" {
		return g.Branch
	}
	return DefaultBranch
}

// ID is "github:owner/repo@branch" (owner and repo in lower case, as GitHub
// ignores their case).
func (g *GitHub) ID() string {
	return "github:" + strings.ToLower(g.Owner+"/"+g.Repo) + "@" + g.branch()
}

func (g *GitHub) Concurrency() int {
	if g.Client.Concurrency > 0 {
		return g.Client.Concurrency
	}
	return 4
}

func (g *GitHub) Head(ctx context.Context, write bool) (Head, error) {
	var init *github.InitFile
	if write {
		i := InitFile
		init = &i
	}
	tip, err := g.Client.BranchTip(ctx, g.Owner, g.Repo, g.branch(), init)
	if err != nil {
		return Head{}, err
	}
	h := Head{Version: tip.Commit, Extra: tip}
	if tip.Tree == "" {
		return h, nil
	}
	tree, err := g.Client.ReadTree(ctx, g.Owner, g.Repo, tip.Tree)
	if err != nil {
		return Head{}, err
	}
	g.mu.Lock()
	if g.known == nil {
		g.known = map[string]bool{}
	}
	for _, f := range tree {
		h.Files = append(h.Files, RemoteFile{Path: f.Path, SHA: f.SHA, Mode: f.Mode})
		g.known[f.SHA] = true
	}
	g.mu.Unlock()
	return h, nil
}

func (g *GitHub) Upload(ctx context.Context, path, sha string, data []byte) (Change, error) {
	ch := Change{Path: path, Mode: "100644", SHA: sha}
	g.mu.Lock()
	known := g.known[sha]
	g.mu.Unlock()
	switch {
	case known:
	case g.Client.CanInline(data):
		ch.Ref = data
	default:
		got, err := g.Client.CreateBlob(ctx, g.Owner, g.Repo, data)
		if err != nil {
			return Change{}, err
		}
		ch.SHA = got
		g.mu.Lock()
		if g.known == nil {
			g.known = map[string]bool{}
		}
		g.known[got] = true // reused if Sync starts over
		g.mu.Unlock()
	}
	return ch, nil
}

func (g *GitHub) Copy(f RemoteFile, path string) Change {
	return Change{Path: path, Mode: f.Mode, SHA: f.SHA}
}

func (g *GitHub) Download(ctx context.Context, f RemoteFile) ([]byte, error) {
	return g.Client.GetBlob(ctx, g.Owner, g.Repo, f.SHA)
}

func (g *GitHub) Commit(ctx context.Context, head Head, changes []Change, n Counts) (string, error) {
	tip, _ := head.Extra.(github.Tip)
	tc := make([]github.TreeChange, len(changes))
	for i, c := range changes {
		tc[i] = github.TreeChange{Path: c.Path, Delete: c.Delete, SHA: c.SHA, Mode: c.Mode}
		if data, ok := c.Ref.([]byte); ok {
			tc[i].Content, tc[i].SHA = data, ""
		}
	}
	tree, err := g.Client.CreateTree(ctx, g.Owner, g.Repo, tip.Tree, tc)
	if err != nil {
		return "", err
	}
	if tree == tip.Tree {
		return tip.Commit, nil
	}
	var parents []string
	if tip.Commit != "" {
		parents = []string{tip.Commit}
	}
	msg := fmt.Sprintf("blup sync: %d updated, %d added, %d removed", n.Updated, n.Added, n.Removed)
	c, err := g.Client.CreateCommit(ctx, g.Owner, g.Repo, msg, tree, parents)
	if err != nil {
		return "", err
	}
	err = g.Client.UpdateBranch(ctx, g.Owner, g.Repo, g.branch(), c, tip.Commit == "")
	if errors.Is(err, github.ErrNotFastForward) {
		return "", ErrNotFastForward
	}
	if err != nil {
		return "", err
	}
	return c, nil
}
