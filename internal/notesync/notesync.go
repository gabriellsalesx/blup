// Package notesync keeps a folder of notes (the site folder inside an
// Obsidian vault) in two-way sync with a Remote: a private GitHub repository
// (through the GitHub REST API only: no git binary and no git knowledge
// needed) or Blup Cloud.
//
// Each sync compares three versions of every file: the local one, the one on
// GitHub, and the one recorded at the last sync (the state file, kept
// outside the vault). A side that changed wins over a side that did not;
// when both changed differently the local file is kept and the GitHub
// version is saved next to it as "Name (conflict YYYY-MM-DD HHMM).ext", and
// both are uploaded, so nothing is ever lost. Files deleted here because
// they were deleted elsewhere are moved to a trash folder outside the vault.
// Only files the state file says were synced before are ever deleted.
//
// Dot files and dot folders (.obsidian, .trash, .git, .DS_Store, ...) are
// ignored on both sides (except SiteFile), as are files over GitHub's 100 MiB limit.
//
// Token permissions (fine-grained personal access token): Contents read and
// write (blobs, trees, commits, refs, initializing an empty repository),
// Metadata read (always granted), and Administration read and write only to
// create the repository ([EnsureRepo]). With the OAuth Device Flow the
// "repo" scope covers private repositories. Private repositories are free.
package notesync

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gabriellsalesx/blup/internal/deploy/github"
)

// DefaultBranch is the branch notes are synced to when Syncer.Branch is empty.
const DefaultBranch = "main"

// DefaultMaxAttempts bounds how often Sync starts over when someone else
// synced at the same time.
const DefaultMaxAttempts = 5

var (
	// ErrRemoteMoved: GitHub kept receiving other syncs while this one ran,
	// and all attempts were used up. Nothing was changed locally; try again.
	ErrRemoteMoved = errors.New("notesync: notes on GitHub kept changing during the sync")
	// ErrLocalFolderEmpty: the notes folder has no files although several
	// notes were synced from it before. Sync refuses to delete them all on
	// GitHub (most likely the folder was moved or the wrong folder was
	// given). Deleting the state file overrides it (then nothing is deleted
	// and everything is downloaded again).
	ErrLocalFolderEmpty = errors.New("notesync: the notes folder is empty but was synced before")
)

// InitFile is written to initialize an empty repository. It lives in a dot
// folder, so it is never downloaded into the vault.
var InitFile = github.InitFile{
	Path:    ".blup/README.md",
	Content: []byte("# Notes\n\nThese notes are kept in sync by blup.\nEdit them in Obsidian and run `blup sync`.\n"),
	Message: "Start notes backup",
}

// EnsureRepo returns owner/name, creating it as a PRIVATE, empty repository
// when it does not exist (owner "" = the token's user). An existing
// repository is used as is: check Repo.Private to warn about public ones.
func EnsureRepo(ctx context.Context, gh *github.Client, owner, name string) (github.Repo, error) {
	return gh.EnsureRepoWith(ctx, owner, name, github.EnsureRepoOptions{
		Private:     true,
		Empty:       true,
		Description: "Notes synced with blup",
	})
}

// Syncer syncs Dir with Remote, or with Branch of Owner/Repo on GitHub
// when Remote is nil.
type Syncer struct {
	Remote Remote

	GH     *github.Client
	Owner  string
	Repo   string
	Branch string // default "main"
	// Dir is the local folder (the site folder inside the vault).
	Dir string
	// StateFile keeps the last synced snapshot, e.g.
	// <site>/.blup/sync-state.json. Keep it outside the vault.
	StateFile string
	// TrashDir receives files deleted locally because they were deleted on
	// GitHub, in a "YYYY-MM-DD HHMMSS" subfolder. Default: "sync-trash"
	// next to StateFile.
	TrashDir string
	// MaxAttempts when the branch moves during a sync (default 5).
	MaxAttempts int
	// Now is the clock (default time.Now), used for conflict names.
	Now func() time.Time

	gh *GitHub // Remote built from GH, Owner, Repo and Branch
}

func (s *Syncer) remote() Remote {
	if s.Remote != nil {
		return s.Remote
	}
	if s.gh == nil {
		s.gh = &GitHub{Client: s.GH, Owner: s.Owner, Repo: s.Repo, Branch: s.Branch}
	}
	return s.gh
}

// Phase names reported through Progress.
const (
	PhaseScan     = "scan"     // hashing local files (Done grows, Total 0)
	PhaseCompare  = "compare"  // reading the list of files on GitHub
	PhaseUpload   = "upload"   // sending files
	PhaseCommit   = "commit"   // saving the new version on GitHub
	PhaseDownload = "download" // receiving files and applying deletions
	PhaseRetry    = "retry"    // someone else synced meanwhile; starting over
	PhaseDone     = "done"
)

// Progress is reported during Sync and Status. Calls are serialized.
type Progress struct {
	Phase string
	Done  int
	Total int
	Path  string // file just processed, when meaningful
}

// SkipReason says why a file was left out.
type SkipReason string

const (
	SkipTooLarge      SkipReason = "too_large"           // over the remote's (or plan's) limit
	SkipInvalidName   SkipReason = "invalid_name"        // name not valid on this computer
	SkipDuplicateName SkipReason = "duplicate_name"      // two names differing only in Unicode form
	SkipUnsupported   SkipReason = "unsupported"         // symlink or submodule on GitHub
	SkipChanged       SkipReason = "changed_during_sync" // edited while syncing; next sync handles it
	SkipWriteFailed   SkipReason = "write_failed"        // could not write, delete or download it
)

// Report says what a sync did (or, from Status, would do). Paths are
// relative to Dir, with forward slashes, sorted.
type Report struct {
	Uploaded   []string // sent to GitHub: new or changed here
	Downloaded []string // received: new or changed elsewhere
	Deleted    []string // removed from GitHub because they were deleted here
	// DeletedLocal were removed here because they were deleted elsewhere;
	// they were moved to Trash.
	DeletedLocal []string
	// Conflicts changed on both sides; the local version was kept (and
	// uploaded). ConflictCopies[i] holds the GitHub version of Conflicts[i]
	// (empty in Status, or when the copy could not be written).
	Conflicts      []string
	ConflictCopies []string
	Skipped        []string
	SkipReasons    map[string]SkipReason
	Commit         string // remote commit the folder now matches
	Trash          string // folder holding DeletedLocal files, if any
	UpToDate       bool   // nothing to send or receive
}

func (r *Report) skip(path string, why SkipReason) {
	if r.SkipReasons == nil {
		r.SkipReasons = map[string]SkipReason{}
	}
	if _, ok := r.SkipReasons[path]; ok {
		return
	}
	r.SkipReasons[path] = why
	r.Skipped = append(r.Skipped, path)
}

func (r *Report) finish() {
	for _, l := range [][]string{r.Uploaded, r.Downloaded, r.Deleted, r.DeletedLocal, r.Skipped} {
		sort.Strings(l)
	}
	r.UpToDate = len(r.Uploaded)+len(r.Downloaded)+len(r.Deleted)+len(r.DeletedLocal)+len(r.Conflicts) == 0
}

func (s *Syncer) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Syncer) trashDir() string {
	if s.TrashDir != "" {
		return s.TrashDir
	}
	return filepath.Join(filepath.Dir(s.StateFile), "sync-trash")
}

func (s *Syncer) concurrency() int { return s.remote().Concurrency() }

func (s *Syncer) maxFileSize() int64 {
	if l, ok := s.remote().(Limited); ok {
		return l.MaxFileSize()
	}
	return MaxFileSize
}

func (s *Syncer) check() error {
	switch {
	case s.Remote == nil && s.GH == nil:
		return errors.New("notesync: no remote")
	case s.Remote == nil && (s.Owner == "" || s.Repo == ""):
		return errors.New("notesync: repository owner and name are required")
	case s.Dir == "":
		return errors.New("notesync: notes folder is required")
	case s.StateFile == "":
		return errors.New("notesync: state file is required")
	}
	return nil
}

type op int

const (
	opNone op = iota
	opUpload
	opDownload
	opDeleteRemote
	opDeleteLocal
	opConflict
)

type entry struct {
	key     string
	l       *localFile
	r       *remoteFile
	base    string // sha at the last sync, "" when untracked
	op      op
	copyKey string // conflict copy path

	// filled while syncing
	sent    string // sha actually uploaded
	applied bool   // local side effect succeeded
}

// decide is the three-way comparison of one path.
func decide(l, r, b string) op {
	switch {
	case l == r:
		return opNone
	case l == b: // only GitHub changed
		if r == "" {
			return opDeleteLocal
		}
		return opDownload
	case r == b: // only this computer changed
		if l == "" {
			return opDeleteRemote
		}
		return opUpload
	case l == "": // deleted here, edited elsewhere: keep the edit
		return opDownload
	case r == "": // deleted elsewhere, edited here: keep the edit
		return opUpload
	}
	return opConflict
}

// plan compares local, remote and the last snapshot.
func plan(local map[string]*localFile, remote map[string]*remoteFile, prev *state, skip map[string]bool) []*entry {
	keys := map[string]bool{}
	for k := range local {
		keys[k] = true
	}
	for k := range remote {
		keys[k] = true
	}
	if prev != nil {
		for k := range prev.Files {
			keys[k] = true
		}
	}
	var out []*entry
	for k := range keys {
		if skip[k] || ignoredPath(k) {
			continue
		}
		e := &entry{key: k, l: local[k], r: remote[k]}
		if prev != nil {
			e.base = prev.Files[k].SHA
		}
		var ls, rs string
		if e.l != nil {
			ls = e.l.sha
		}
		if e.r != nil {
			rs = e.r.f.SHA
		}
		e.op = decide(ls, rs, e.base)
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].key < out[j].key })
	return out
}

// snapshot reads both sides and plans. head.Version is "" for an empty
// remote.
func (s *Syncer) snapshot(ctx context.Context, prev *state, write bool, report func(Progress)) (Head, []*entry, map[string]*localFile, Report, error) {
	var rep Report
	skip := map[string]bool{}
	report(Progress{Phase: PhaseCompare})
	head, err := s.remote().Head(ctx, write)
	if err != nil {
		return head, nil, nil, rep, err
	}
	remote := remoteFiles(head.Files, &rep, skip)
	report(Progress{Phase: PhaseCompare, Done: len(remote), Total: len(remote)})
	local, err := s.scan(prev, &rep, skip, report)
	if err != nil {
		return head, nil, nil, rep, err
	}
	entries := plan(local, remote, prev, skip)
	if len(local) == 0 && prev != nil {
		dels := 0
		for _, e := range entries {
			if e.op == opDeleteRemote {
				dels++
			}
		}
		if dels > 1 {
			return head, nil, nil, rep, fmt.Errorf("%w: %s", ErrLocalFolderEmpty, s.Dir)
		}
	}
	return head, entries, local, rep, nil
}

func fill(rep *Report, entries []*entry, done bool) {
	for _, e := range entries {
		if done && !e.applied && (e.op == opDownload || e.op == opDeleteLocal) {
			continue
		}
		switch e.op {
		case opUpload:
			rep.Uploaded = append(rep.Uploaded, e.key)
		case opDownload:
			rep.Downloaded = append(rep.Downloaded, e.key)
		case opDeleteRemote:
			rep.Deleted = append(rep.Deleted, e.key)
		case opDeleteLocal:
			rep.DeletedLocal = append(rep.DeletedLocal, e.key)
		case opConflict:
			rep.Conflicts = append(rep.Conflicts, e.key)
			c := ""
			if done && e.applied {
				c = e.copyKey
			}
			rep.ConflictCopies = append(rep.ConflictCopies, c)
		}
	}
	rep.finish()
}

func serialize(progress func(Progress)) func(Progress) {
	var mu sync.Mutex
	return func(p Progress) {
		if progress == nil {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		progress(p)
	}
}

func (s *Syncer) prepareDir(prev *state) error {
	st, err := os.Stat(s.Dir)
	switch {
	case err == nil && !st.IsDir():
		return fmt.Errorf("notesync: %s is not a folder", s.Dir)
	case err == nil:
		return nil
	case errors.Is(err, os.ErrNotExist) && prev == nil:
		// First sync on a new computer: the folder will be filled.
		return os.MkdirAll(s.Dir, 0o755)
	}
	return err
}

// Status reports what Sync would do now, without changing anything locally
// or on GitHub. Report.Commit is the current remote commit.
func (s *Syncer) Status(ctx context.Context, progress func(Progress)) (Report, error) {
	if err := s.check(); err != nil {
		return Report{}, err
	}
	report := serialize(progress)
	prev, err := s.loadState()
	if err != nil {
		return Report{}, err
	}
	head, entries, _, rep, err := s.snapshot(ctx, prev, false, report)
	if err != nil {
		return Report{}, err
	}
	fill(&rep, entries, false)
	rep.Commit = head.Version
	report(Progress{Phase: PhaseDone})
	return rep, nil
}

// Sync uploads local changes, downloads remote ones and resolves conflicts,
// creating at most one commit on GitHub. progress may be nil.
func (s *Syncer) Sync(ctx context.Context, progress func(Progress)) (Report, error) {
	if err := s.check(); err != nil {
		return Report{}, err
	}
	report := serialize(progress)
	prev, err := s.loadState()
	if err != nil {
		return Report{}, err
	}
	if err := s.prepareDir(prev); err != nil {
		return Report{}, err
	}
	attempts := s.MaxAttempts
	if attempts <= 0 {
		attempts = DefaultMaxAttempts
	}
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			report(Progress{Phase: PhaseRetry, Done: attempt, Total: attempts})
		}
		head, entries, local, rep, err := s.snapshot(ctx, prev, true, report)
		if err != nil {
			return Report{}, err
		}
		s.nameConflicts(entries, local)

		changes, counts, err := s.prepareUploads(ctx, entries, report)
		if err != nil {
			return Report{}, err
		}
		commit := head.Version
		if len(changes) > 0 {
			report(Progress{Phase: PhaseCommit})
			commit, err = s.remote().Commit(ctx, head, changes, counts)
			if errors.Is(err, ErrNotFastForward) {
				continue // someone else synced: compare again
			}
			if err != nil {
				return Report{}, err
			}
			report(Progress{Phase: PhaseCommit, Done: 1, Total: 1})
		}

		trash := filepath.Join(s.trashDir(), s.now().Format("2006-01-02 150405"))
		s.applyLocal(ctx, entries, trash, &rep, report)
		fill(&rep, entries, true)
		rep.Commit = commit
		if len(rep.DeletedLocal) > 0 {
			rep.Trash = trash
		}
		if err := s.saveState(s.newState(prev, entries, commit)); err != nil {
			return rep, fmt.Errorf("notesync: saving sync state: %w", err)
		}
		report(Progress{Phase: PhaseDone})
		return rep, nil
	}
	return Report{}, fmt.Errorf("%w (%d attempts)", ErrRemoteMoved, attempts)
}

// nameConflicts picks a free "Name (conflict YYYY-MM-DD HHMM).ext" for each
// conflict.
func (s *Syncer) nameConflicts(entries []*entry, local map[string]*localFile) {
	taken := map[string]bool{}
	for _, e := range entries {
		taken[e.key] = true
	}
	stamp := s.now().Format("2006-01-02 1504")
	for _, e := range entries {
		if e.op != opConflict {
			continue
		}
		dir, name := "", e.key
		if i := strings.LastIndex(e.key, "/"); i >= 0 {
			dir, name = e.key[:i+1], e.key[i+1:]
		}
		ext := filepath.Ext(name)
		stem := strings.TrimSuffix(name, ext)
		if stem == "" {
			stem, ext = name, ""
		}
		for n := 1; ; n++ {
			suffix := ""
			if n > 1 {
				suffix = fmt.Sprintf(" %d", n)
			}
			k := dir + stem + " (conflict " + stamp + suffix + ")" + ext
			if taken[k] {
				continue
			}
			if _, err := os.Lstat(filepath.Join(s.Dir, filepath.FromSlash(k))); err == nil {
				continue
			}
			taken[k] = true
			e.copyKey = k
			break
		}
	}
}

func remoteMode(r *remoteFile) string {
	if r != nil && r.f.Mode == "100755" {
		return r.f.Mode
	}
	return "100644"
}

// prepareUploads reads and sends local files and returns the changes of the
// new version.
func (s *Syncer) prepareUploads(ctx context.Context, entries []*entry, report func(Progress)) ([]Change, Counts, error) {
	var counts Counts
	var jobs []*entry
	for _, e := range entries {
		switch e.op {
		case opUpload, opConflict:
			jobs = append(jobs, e)
		}
	}
	changes := make([]Change, len(jobs))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var (
		mu       sync.Mutex
		firstErr error
		done     int
		wg       sync.WaitGroup
	)
	work := make(chan int)
	report(Progress{Phase: PhaseUpload, Total: len(jobs)})
	for range s.concurrency() {
		wg.Go(func() {
			for i := range work {
				e := jobs[i]
				ch, err := s.uploadOne(ctx, e)
				mu.Lock()
				if err != nil {
					if firstErr == nil {
						firstErr = fmt.Errorf("notesync: sending %s: %w", e.key, err)
						cancel()
					}
					mu.Unlock()
					continue
				}
				changes[i] = ch
				done++
				p := Progress{Phase: PhaseUpload, Done: done, Total: len(jobs), Path: e.key}
				mu.Unlock()
				report(p)
			}
		})
	}
	for i := range jobs {
		select {
		case work <- i:
		case <-ctx.Done():
		}
		if ctx.Err() != nil {
			break
		}
	}
	close(work)
	wg.Wait()
	if firstErr != nil {
		return nil, counts, firstErr
	}
	if err := ctx.Err(); err != nil {
		return nil, counts, err
	}
	for _, e := range entries {
		switch e.op {
		case opUpload:
			if e.r != nil {
				counts.Updated++
			} else {
				counts.Added++
			}
		case opConflict:
			counts.Updated++
			counts.Added++
			changes = append(changes, s.remote().Copy(e.r.f, e.copyKey))
		case opDeleteRemote:
			counts.Removed++
			changes = append(changes, Change{Path: e.r.f.Path, Delete: true})
		}
	}
	return changes, counts, nil
}

func (s *Syncer) uploadOne(ctx context.Context, e *entry) (Change, error) {
	data, err := os.ReadFile(e.l.abs)
	if err != nil {
		return Change{}, err
	}
	if int64(len(data)) > s.maxFileSize() {
		return Change{}, github.ErrFileTooLarge
	}
	sha := github.BlobSHA(data)
	if sha != e.l.sha {
		e.l.cacheable = false // edited meanwhile: send what was read
	}
	e.sent = sha
	path := e.key
	if e.r != nil {
		path = e.r.f.Path
	}
	ch, err := s.remote().Upload(ctx, path, sha, data)
	if err != nil {
		return Change{}, err
	}
	ch.Path, ch.Mode = path, remoteMode(e.r)
	return ch, nil
}

// applyLocal downloads files, writes conflict copies and moves deleted
// files to the trash. Failures are reported as skipped, never fatal: the
// state keeps the old snapshot for them, so the next sync retries.
func (s *Syncer) applyLocal(ctx context.Context, entries []*entry, trash string, rep *Report, report func(Progress)) {
	var jobs []*entry
	for _, e := range entries {
		switch e.op {
		case opDownload, opDeleteLocal, opConflict:
			jobs = append(jobs, e)
		}
	}
	if len(jobs) == 0 {
		return
	}
	var (
		mu   sync.Mutex
		done int
		wg   sync.WaitGroup
	)
	work := make(chan *entry)
	report(Progress{Phase: PhaseDownload, Total: len(jobs)})
	for range s.concurrency() {
		wg.Go(func() {
			for e := range work {
				why := s.applyOne(ctx, e, trash)
				mu.Lock()
				if why != "" {
					key := e.key
					if e.op == opConflict {
						key = e.copyKey
					}
					rep.skip(key, why)
				} else {
					e.applied = true
				}
				done++
				p := Progress{Phase: PhaseDownload, Done: done, Total: len(jobs), Path: e.key}
				mu.Unlock()
				report(p)
			}
		})
	}
	for _, e := range jobs {
		work <- e
	}
	close(work)
	wg.Wait()
	sort.Strings(rep.Skipped)
}

// unchanged reports whether the local file still has the content planned on.
func unchanged(abs string, l *localFile) bool {
	if l == nil {
		_, err := os.Lstat(abs)
		return errors.Is(err, os.ErrNotExist)
	}
	sha, _, err := hashFile(abs)
	return err == nil && sha == l.sha
}

func (s *Syncer) localPath(key string) string {
	return filepath.Join(s.Dir, filepath.FromSlash(key))
}

func (s *Syncer) applyOne(ctx context.Context, e *entry, trash string) SkipReason {
	switch e.op {
	case opDownload, opConflict:
		target := s.localPath(e.key)
		if e.op == opDownload && e.l != nil {
			target = e.l.abs
		}
		if e.op == opConflict {
			target = s.localPath(e.copyKey)
		}
		data, err := s.remote().Download(ctx, e.r.f)
		if err != nil {
			return SkipWriteFailed
		}
		var expect *localFile
		if e.op == opDownload {
			expect = e.l
		}
		if !unchanged(target, expect) {
			return SkipChanged
		}
		perm := os.FileMode(0o644)
		if e.r.f.Mode == "100755" {
			perm = 0o755
		}
		if e.op == opDownload && e.l != nil {
			perm = e.l.perm
		}
		if writeAtomic(target, data, perm) != nil {
			return SkipWriteFailed
		}
	case opDeleteLocal:
		if !unchanged(e.l.abs, e.l) {
			return SkipChanged
		}
		dst := filepath.Join(trash, filepath.FromSlash(e.key))
		if moveFile(e.l.abs, dst) != nil {
			return SkipWriteFailed
		}
		s.removeEmptyParents(filepath.Dir(e.l.abs))
	}
	return ""
}

// moveFile renames src to dst, copying across file systems.
func moveFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if os.Rename(src, dst) == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		in.Close()
		return err
	}
	_, err = io.Copy(out, in)
	in.Close()
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(dst)
		return err
	}
	return os.Remove(src)
}

func (s *Syncer) removeEmptyParents(dir string) {
	root := filepath.Clean(s.Dir)
	for dir = filepath.Clean(dir); dir != root && strings.HasPrefix(dir, root+string(filepath.Separator)); dir = filepath.Dir(dir) {
		if os.Remove(dir) != nil { // fails when not empty
			return
		}
	}
}

// newState records the snapshot both sides now agree on.
func (s *Syncer) newState(prev *state, entries []*entry, commit string) *state {
	st := &state{
		Version: stateVersion, Remote: s.remote().ID(),
		Commit: commit, SyncedAt: s.now().UTC(), Files: map[string]fileState{},
	}
	// Paths left out of the plan (skipped) keep their old snapshot.
	planned := map[string]bool{}
	for _, e := range entries {
		planned[e.key] = true
	}
	if prev != nil {
		for k, v := range prev.Files {
			if !planned[k] {
				st.Files[k] = fileState{SHA: v.SHA}
			}
		}
	}
	fromLocal := func(l *localFile, sha string) fileState {
		fs := fileState{SHA: sha}
		if l.cacheable && l.sha == sha {
			fs.Size, fs.MTime = l.size, l.mtime
		}
		return fs
	}
	keep := func(e *entry) {
		if e.base != "" {
			st.Files[e.key] = fileState{SHA: e.base}
		}
	}
	for _, e := range entries {
		switch e.op {
		case opNone:
			if e.l != nil {
				st.Files[e.key] = fromLocal(e.l, e.l.sha)
			}
		case opUpload:
			st.Files[e.key] = fromLocal(e.l, e.sent)
		case opConflict:
			st.Files[e.key] = fromLocal(e.l, e.sent)
			if e.applied {
				st.Files[e.copyKey] = fileState{SHA: e.r.f.SHA}
			}
		case opDeleteRemote:
			// gone on both sides
		case opDownload:
			if e.applied {
				st.Files[e.key] = fileState{SHA: e.r.f.SHA}
			} else {
				keep(e)
			}
		case opDeleteLocal:
			if !e.applied {
				keep(e)
			}
		}
	}
	return st
}
