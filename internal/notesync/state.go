package notesync

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// stateVersion is the format version of the state file.
const stateVersion = 1

// state is the last synced snapshot, stored as JSON in Syncer.StateFile:
//
//	{
//	  "version": 1,
//	  "owner": "maria", "repo": "notas", "branch": "main",
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
	Version  int                  `json:"version"`
	Owner    string               `json:"owner"`
	Repo     string               `json:"repo"`
	Branch   string               `json:"branch"`
	Commit   string               `json:"commit"`
	SyncedAt time.Time            `json:"synced_at"`
	Files    map[string]fileState `json:"files"`
}

type fileState struct {
	SHA   string `json:"sha"`
	Size  int64  `json:"size,omitempty"`
	MTime int64  `json:"mtime,omitempty"`
}

// loadState returns the saved state, or nil when there is none or it
// belongs to another repository/branch. A corrupt file is treated as no
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
	if json.Unmarshal(data, &st) != nil || st.Version != stateVersion ||
		!strings.EqualFold(st.Owner, s.Owner) || !strings.EqualFold(st.Repo, s.Repo) ||
		st.Branch != s.branch() {
		return nil, nil
	}
	if st.Files == nil {
		st.Files = map[string]fileState{}
	}
	return &st, nil
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
