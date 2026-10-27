package github

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// DefaultBranch is the branch Publish writes to when none is given.
const DefaultBranch = "gh-pages"

// MaxFileSize is GitHub's hard per-file limit.
const MaxFileSize = 100 << 20

// PublishOptions tunes Publish.
type PublishOptions struct {
	Branch  string // default "gh-pages"
	CNAME   string // custom domain; written to /CNAME when set
	Message string // commit message; default "Publish site"
}

// Phase names reported through Progress.
const (
	PhaseScan    = "scan"    // hashing local files
	PhaseCompare = "compare" // reading the remote tree
	PhaseUpload  = "upload"  // uploading changed files
	PhaseTree    = "tree"    // creating the tree
	PhaseCommit  = "commit"  // creating the commit and moving the branch
	PhaseDone    = "done"
)

// Progress is reported during Publish. Calls are serialized.
type Progress struct {
	Phase      string
	Done       int   // items done in this phase
	Total      int   // items in this phase
	BytesDone  int64 // upload phase only
	BytesTotal int64
	Path       string // file just processed, when meaningful
}

// Result summarizes a Publish.
type Result struct {
	Branch    string
	CommitSHA string // new commit, or current one when Unchanged
	TreeSHA   string
	Files     int   // files in the published tree
	Uploaded  int   // files whose content was sent
	Skipped   int   // files already on GitHub, not re-sent
	Deleted   int   // files removed compared to the previous version
	Bytes     int64 // bytes sent
	Unchanged bool  // nothing changed; no commit was created
	Orphan    bool  // the branch was created by this publish
}

type localFile struct {
	path string // slash-separated, relative
	abs  string // empty for virtual files
	data []byte // content of virtual files
	size int64
	sha  string
}

func (f *localFile) read() ([]byte, error) {
	if f.abs == "" {
		return f.data, nil
	}
	return os.ReadFile(f.abs)
}

// BlobSHA returns the git object id of content as a blob, identical to
// `git hash-object`: sha1("blob <len>\x00" + content).
func BlobSHA(content []byte) string {
	h := sha1.New()
	fmt.Fprintf(h, "blob %d\x00", len(content))
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

func hashFile(abs string, size int64) (string, error) {
	f, err := os.Open(abs)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha1.New()
	fmt.Fprintf(h, "blob %d\x00", size)
	n, err := io.Copy(h, f)
	if err != nil {
		return "", err
	}
	if n != size {
		return "", fmt.Errorf("github: %s changed while reading", abs)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func scanDir(dir string, opts PublishOptions) ([]*localFile, error) {
	st, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("github: %s is not a directory", dir)
	}
	byPath := map[string]*localFile{}
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" && p != dir {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil // symlinks, sockets, ...
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if info.Size() > MaxFileSize {
			return fmt.Errorf("%w: %s (%d bytes)", ErrFileTooLarge, rel, info.Size())
		}
		sha, err := hashFile(p, info.Size())
		if err != nil {
			return err
		}
		byPath[rel] = &localFile{path: rel, abs: p, size: info.Size(), sha: sha}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(byPath) == 0 {
		return nil, fmt.Errorf("%w: %s is empty", ErrNothingToPublish, dir)
	}
	addVirtual := func(path string, data []byte) {
		byPath[path] = &localFile{path: path, data: data, size: int64(len(data)), sha: BlobSHA(data)}
	}
	addVirtual(".nojekyll", nil)
	if cn := strings.TrimSpace(opts.CNAME); cn != "" {
		addVirtual("CNAME", []byte(cn+"\n"))
	}
	files := make([]*localFile, 0, len(byPath))
	for _, f := range byPath {
		files = append(files, f)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	return files, nil
}

func refPath(owner, repo, branch string) string {
	segs := strings.Split(branch, "/")
	for i, s := range segs {
		segs[i] = url.PathEscape(s)
	}
	return repoPath(owner, repo) + "/git/ref/heads/" + strings.Join(segs, "/")
}

type treeEntry struct {
	Path    string  `json:"path"`
	Mode    string  `json:"mode"`
	Type    string  `json:"type"`
	SHA     *string `json:"sha,omitempty"`
	Content *string `json:"content,omitempty"`
}

// Publish mirrors dir onto branch of owner/repo. Unchanged files are not
// uploaded again; files missing from dir are deleted from the branch. A
// .nojekyll file is always added and, when opts.CNAME is set, a CNAME file.
// The branch is created as an orphan when missing and force-updated
// otherwise. progress may be nil.
func (c *Client) Publish(ctx context.Context, owner, repo, dir string, opts PublishOptions, progress func(Progress)) (Result, error) {
	if opts.Branch == "" {
		opts.Branch = DefaultBranch
	}
	if opts.Message == "" {
		opts.Message = "Publish site"
	}
	var pmu sync.Mutex
	report := func(p Progress) {
		if progress == nil {
			return
		}
		pmu.Lock()
		defer pmu.Unlock()
		progress(p)
	}
	res := Result{Branch: opts.Branch}

	report(Progress{Phase: PhaseScan})
	files, err := scanDir(dir, opts)
	if err != nil {
		return res, err
	}
	res.Files = len(files)
	report(Progress{Phase: PhaseScan, Done: len(files), Total: len(files)})

	// Current branch head, if any.
	report(Progress{Phase: PhaseCompare})
	parent, err := c.branchHead(ctx, owner, repo, opts.Branch)
	if err != nil {
		return res, err
	}
	remote := map[string]string{} // path -> blob sha
	known := map[string]bool{}    // blob shas present in the repo
	parentTree := ""
	if parent != "" {
		var cm struct {
			Tree struct {
				SHA string `json:"sha"`
			} `json:"tree"`
		}
		if _, err := c.do(ctx, http.MethodGet, repoPath(owner, repo)+"/git/commits/"+parent, nil, &cm); err != nil {
			return res, err
		}
		parentTree = cm.Tree.SHA
		if err := c.readTree(ctx, owner, repo, parentTree, remote); err != nil {
			return res, err
		}
		for _, s := range remote {
			known[s] = true
		}
	} else {
		res.Orphan = true
	}
	report(Progress{Phase: PhaseCompare, Done: len(remote), Total: len(remote)})

	// Decide what to send.
	inlineLimit := c.InlineTextLimit
	if inlineLimit == 0 {
		inlineLimit = 512 << 10
	}
	inline := map[string][]byte{} // path -> content for inline text files
	var blobs []*localFile        // unique shas to upload as blobs
	pendingSHA := map[string]bool{}
	var inlineFiles []*localFile
	var bytesTotal int64
	for _, f := range files {
		if known[f.sha] {
			res.Skipped++
			continue
		}
		res.Uploaded++
		if inlineLimit > 0 && f.size <= inlineLimit && f.size > 0 {
			data, err := f.read()
			if err != nil {
				return res, err
			}
			if BlobSHA(data) != f.sha {
				return res, fmt.Errorf("github: %s changed during publish", f.path)
			}
			if utf8.Valid(data) && bytes.IndexByte(data, 0) < 0 {
				inline[f.path] = data
				inlineFiles = append(inlineFiles, f)
				bytesTotal += f.size
				continue
			}
		}
		if pendingSHA[f.sha] {
			continue
		}
		pendingSHA[f.sha] = true
		blobs = append(blobs, f)
		bytesTotal += f.size
	}
	for p := range remote {
		if !containsPath(files, p) {
			res.Deleted++
		}
	}

	// Upload blobs with bounded concurrency.
	if err := c.uploadBlobs(ctx, owner, repo, blobs, bytesTotal, report); err != nil {
		return res, err
	}
	res.Bytes = bytesTotal

	// Build the full tree in chunks chained with base_tree.
	maxEntries := c.TreeChunkEntries
	if maxEntries <= 0 {
		maxEntries = 3000
	}
	maxBytes := c.TreeChunkBytes
	if maxBytes <= 0 {
		maxBytes = 16 << 20
	}
	var chunks [][]treeEntry
	var cur []treeEntry
	var curBytes int64
	for _, f := range files {
		e := treeEntry{Path: f.path, Mode: "100644", Type: "blob"}
		var add int64
		if data, ok := inline[f.path]; ok {
			s := string(data)
			e.Content = &s
			add = int64(len(data))
		} else {
			s := f.sha
			e.SHA = &s
		}
		if len(cur) > 0 && (len(cur) >= maxEntries || curBytes+add > maxBytes) {
			chunks = append(chunks, cur)
			cur, curBytes = nil, 0
		}
		cur = append(cur, e)
		curBytes += add
	}
	chunks = append(chunks, cur)
	treeSHA := ""
	for i, ch := range chunks {
		report(Progress{Phase: PhaseTree, Done: i, Total: len(chunks)})
		body := map[string]any{"tree": ch}
		if treeSHA != "" {
			body["base_tree"] = treeSHA
		}
		var out struct {
			SHA string `json:"sha"`
		}
		if _, err := c.do(ctx, http.MethodPost, repoPath(owner, repo)+"/git/trees", body, &out); err != nil {
			return res, err
		}
		treeSHA = out.SHA
	}
	report(Progress{Phase: PhaseTree, Done: len(chunks), Total: len(chunks)})
	res.TreeSHA = treeSHA

	if parent != "" && treeSHA == parentTree {
		res.CommitSHA = parent
		res.Unchanged = true
		res.Uploaded, res.Bytes = 0, 0
		res.Skipped = len(files)
		report(Progress{Phase: PhaseDone})
		return res, nil
	}

	report(Progress{Phase: PhaseCommit})
	parents := []string{}
	if parent != "" {
		parents = append(parents, parent)
	}
	var commit struct {
		SHA string `json:"sha"`
	}
	if _, err := c.do(ctx, http.MethodPost, repoPath(owner, repo)+"/git/commits", map[string]any{
		"message": opts.Message,
		"tree":    treeSHA,
		"parents": parents,
	}, &commit); err != nil {
		return res, err
	}
	res.CommitSHA = commit.SHA

	if err := c.setRef(ctx, owner, repo, opts.Branch, commit.SHA, parent != ""); err != nil {
		return res, err
	}
	report(Progress{Phase: PhaseDone, Done: 1, Total: 1})
	return res, nil
}

func containsPath(files []*localFile, p string) bool {
	i := sort.Search(len(files), func(i int) bool { return files[i].path >= p })
	return i < len(files) && files[i].path == p
}

// branchHead returns the commit sha of branch, or "" if it does not exist.
// An empty repository (no commits at all) is initialized with a README on
// its default branch, since the Git Data API refuses to work on it.
func (c *Client) branchHead(ctx context.Context, owner, repo, branch string) (string, error) {
	const attempts = 4
	for attempt := 0; attempt < attempts; attempt++ {
		var ref struct {
			Object struct {
				SHA  string `json:"sha"`
				Type string `json:"type"`
			} `json:"object"`
		}
		_, err := c.do(ctx, http.MethodGet, refPath(owner, repo, branch), nil, &ref)
		switch {
		case err == nil:
			return ref.Object.SHA, nil
		case errors.Is(err, ErrNotFound):
			return "", nil
		case statusOf(err) == http.StatusConflict:
			// "Git Repository is empty." Either a repository created
			// without auto_init, or one created a moment ago whose initial
			// commit is not visible yet.
			if attempt == 0 {
				if err := c.initEmptyRepo(ctx, owner, repo); err != nil {
					return "", err
				}
			}
			if err := c.doSleep(ctx, backoff(attempt, time.Second)); err != nil {
				return "", err
			}
		default:
			return "", err
		}
	}
	return "", errors.New("github: repository is still empty after initialization")
}

func (c *Client) initEmptyRepo(ctx context.Context, owner, repo string) error {
	content := "# " + repo + "\n\nSite published with blup.\n"
	_, err := c.do(ctx, http.MethodPut, repoPath(owner, repo)+"/contents/README.md", map[string]any{
		"message": "Initialize repository",
		"content": base64.StdEncoding.EncodeToString([]byte(content)),
	}, nil)
	if statusOf(err) == http.StatusUnprocessableEntity {
		return nil // README already there: the repository is being initialized
	}
	return err
}

// readTree fills out (path -> blob sha) with every blob under treeSHA.
func (c *Client) readTree(ctx context.Context, owner, repo, treeSHA string, out map[string]string) error {
	var t struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
			SHA  string `json:"sha"`
		} `json:"tree"`
		Truncated bool `json:"truncated"`
	}
	if _, err := c.do(ctx, http.MethodGet, repoPath(owner, repo)+"/git/trees/"+treeSHA+"?recursive=1", nil, &t); err != nil {
		return err
	}
	if !t.Truncated {
		for _, e := range t.Tree {
			if e.Type == "blob" {
				out[e.Path] = e.SHA
			}
		}
		return nil
	}
	// Over 100,000 entries or 7 MB: walk subtrees one at a time.
	return c.walkTree(ctx, owner, repo, treeSHA, "", out)
}

func (c *Client) walkTree(ctx context.Context, owner, repo, treeSHA, prefix string, out map[string]string) error {
	var t struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
			SHA  string `json:"sha"`
		} `json:"tree"`
	}
	if _, err := c.do(ctx, http.MethodGet, repoPath(owner, repo)+"/git/trees/"+treeSHA, nil, &t); err != nil {
		return err
	}
	for _, e := range t.Tree {
		switch e.Type {
		case "blob":
			out[prefix+e.Path] = e.SHA
		case "tree":
			if err := c.walkTree(ctx, owner, repo, e.SHA, prefix+e.Path+"/", out); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Client) uploadBlobs(ctx context.Context, owner, repo string, blobs []*localFile, bytesTotal int64, report func(Progress)) error {
	if len(blobs) == 0 {
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var (
		mu        sync.Mutex
		done      int
		bytesDone int64
		firstErr  error
		wg        sync.WaitGroup
	)
	report(Progress{Phase: PhaseUpload, Total: len(blobs), BytesTotal: bytesTotal})
	jobs := make(chan *localFile)
	for range c.concurrency() {
		wg.Go(func() {
			for f := range jobs {
				err := c.uploadBlob(ctx, owner, repo, f)
				mu.Lock()
				if err != nil {
					if firstErr == nil {
						firstErr = fmt.Errorf("uploading %s: %w", f.path, err)
						cancel()
					}
					mu.Unlock()
					continue
				}
				done++
				bytesDone += f.size
				p := Progress{Phase: PhaseUpload, Done: done, Total: len(blobs), BytesDone: bytesDone, BytesTotal: bytesTotal, Path: f.path}
				mu.Unlock()
				report(p)
			}
		})
	}
	for _, f := range blobs {
		select {
		case jobs <- f:
		case <-ctx.Done():
		}
		if ctx.Err() != nil {
			break
		}
	}
	close(jobs)
	wg.Wait()
	if firstErr != nil {
		return firstErr
	}
	return ctx.Err()
}

func (c *Client) uploadBlob(ctx context.Context, owner, repo string, f *localFile) error {
	data, err := f.read()
	if err != nil {
		return err
	}
	if BlobSHA(data) != f.sha {
		return fmt.Errorf("github: %s changed during publish", f.path)
	}
	var out struct {
		SHA string `json:"sha"`
	}
	if _, err := c.do(ctx, http.MethodPost, repoPath(owner, repo)+"/git/blobs", map[string]string{
		"content":  base64.StdEncoding.EncodeToString(data),
		"encoding": "base64",
	}, &out); err != nil {
		return err
	}
	if out.SHA != f.sha {
		return fmt.Errorf("github: blob sha mismatch for %s: local %s, remote %s", f.path, f.sha, out.SHA)
	}
	return nil
}

// setRef points branch at sha, creating it when it does not exist.
func (c *Client) setRef(ctx context.Context, owner, repo, branch, sha string, exists bool) error {
	patch := func() error {
		p := strings.Replace(refPath(owner, repo, branch), "/git/ref/", "/git/refs/", 1)
		_, err := c.do(ctx, http.MethodPatch, p, map[string]any{"sha": sha, "force": true}, nil)
		return err
	}
	if exists {
		return patch()
	}
	_, err := c.do(ctx, http.MethodPost, repoPath(owner, repo)+"/git/refs", map[string]string{
		"ref": "refs/heads/" + branch,
		"sha": sha,
	}, nil)
	if statusOf(err) == http.StatusUnprocessableEntity {
		// "Reference already exists": created concurrently; force it.
		return patch()
	}
	return err
}
