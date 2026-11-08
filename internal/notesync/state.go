package notesync

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// stateVersion is the format version of the state file.
const stateVersion = 1

// state is the last synced snapshot, stored as JSON in Syncer.StateFile:
//
//	{
//	  "version": 1,
//	  "remote": "github:maria/notas@main",
//	  "commit": "<sha of the remote commit the snapshot matches>",
//	  "synced_at": "2026-10-28T14:30:00Z",
//	  "files": {
//	    "Notas/ação.md": {"sha": "<git blob sha>", "size": 120, "mtime": 1761661800000000000}
//	  }
//	}
//
// files holds every path (NFC, forward slashes, relative to Dir) that was
// identical locally and on GitHub after the last sync. size/mtime (Unix
// nanoseconds) are a hashing cache for the local copy; mtime 0 means "always
// re-hash".
type state struct {
	Version int    `json:"version"`
	Remote  string `json:"remote"`
	// Owner, Repo and Branch name the GitHub remote in files saved before
	// Remote existed.
	Owner  string `json:"owner,omitempty"`
	Repo   string `json:"repo,omitempty"`
	Branch string `json:"branch,omitempty"`
	Commit string `json:"commit"`
	// Chain is the remote's token for Commit (Chained); "" when the remote
	// has none or the state was saved before it existed.
	Chain    string               `json:"chain,omitempty"`
	SyncedAt time.Time            `json:"synced_at"`
	Files    map[string]fileState `json:"files"`
}

type fileState struct {
	SHA   string `json:"sha"`
	Size  int64  `json:"size,omitempty"`
	MTime int64  `json:"mtime,omitempty"`
}

// loadState returns the saved state, or nil when there is none or it
// belongs to another remote. A corrupt file is treated as no
// state: the next sync then never deletes anything (it only adds, and turns
// differences into conflict copies), which is the safe direction.
func (s *Syncer) loadState() (*state, error) {
	data, err := os.ReadFile(s.StateFile)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var st state
	if json.Unmarshal(data, &st) != nil || st.Version != stateVersion {
		return nil, nil
	}
	if st.Remote == "" && st.Owner != "" {
		st.Remote = (&GitHub{Owner: st.Owner, Repo: st.Repo, Branch: st.Branch}).ID()
	}
	if st.Remote != s.remote().ID() {
		return nil, nil
	}
	st.Owner, st.Repo, st.Branch = "", "", ""
	if st.Files == nil {
		st.Files = map[string]fileState{}
	}
	// One entry that cannot be (a path out of the folder, a hash that is
	// not one) makes the whole file untrusted: exclusions are never
	// deduced from it, the sync compares without a previous state.
	for p, f := range st.Files {
		if !validStatePath(p) || !validStateSHA(f.SHA) || f.Size < 0 || f.MTime < 0 {
			return nil, nil
		}
	}
	return &st, nil
}

// validStatePath is a path relative to Dir as the state keeps it:
// forward slashes, no "." or ".." part, no NUL or backslash, valid UTF-8.
func validStatePath(p string) bool {
	if p == "" || len(p) > 4096 || !utf8.ValidString(p) || strings.ContainsAny(p, "\\\x00") || p[0] == '/' {
		return false
	}
	for _, part := range strings.Split(p, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

// validStateSHA is a git blob sha: 40 lower case hex digits.
func validStateSHA(s string) bool {
	if len(s) != 40 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if c := s[i]; !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// Forget drops what this computer knows of the last sync: the base of
// every file and the chain of versions (the state file), and what the
// Remote keeps of the versions it read (Forgetter). No note is touched,
// here or on the remote. The next Sync then compares as on a new computer:
// without a base, decide only uploads (only here), downloads (only there)
// or keeps both versions as a conflict copy (different on each side); it
// never deletes a file on either side, and accepts the head the remote
// shows now. So it is never called on its own: only when the person,
// told that the sync stopped (Stuck), asks to compare from scratch.
func (s *Syncer) Forget() error {
	if err := s.check(); err != nil {
		return err
	}
	if f, ok := s.remote().(Forgetter); ok {
		if err := f.Forget(); err != nil {
			return err
		}
	}
	if err := os.Remove(s.StateFile); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (s *Syncer) saveState(st *state) error {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(s.StateFile, append(data, '\n'), 0o644)
}

// writeAtomic writes data to a temporary dot-file next to path and renames
// it into place, so readers never see a half-written file.
func writeAtomic(path string, data []byte, perm fs.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".blup-tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	ok := false
	defer func() {
		if !ok {
			f.Close()
			os.Remove(tmp)
		}
	}()
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp, perm); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	ok = true
	return nil
}
