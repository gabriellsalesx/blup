package blupcloud

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

// The history of a vault, read on this computer: the cloud keeps old
// versions sealed, and only here are they opened and compared. Sealed
// manifests never change, so they are kept in Remote.Cache once fetched.

// ErrNotInHistory: the file is not in that version.
var ErrNotInHistory = errors.New("blupcloud: the file is not in that version")

// NoteVersion is one content a file had: the version where it was saved.
type NoteVersion struct {
	N    int64     `json:"n"`
	At   time.Time `json:"at"`
	Size int64     `json:"size"`
	// Current: the content of the newest version.
	Current bool `json:"current"`
}

// DeletedFile is a file of an old version that the newest one no longer has.
type DeletedFile struct {
	Path string    `json:"path"`
	N    int64     `json:"n"`  // the last version that had it
	At   time.Time `json:"at"` // when that version was saved
}

// manifestAt reads (and caches) the manifest of version n.
func (r *Remote) manifestAt(ctx context.Context, n int64) (manifest, error) {
	if !ValidVaultID(r.Vault) || n <= 0 {
		return manifest{}, ErrBadVault
	}
	var sealed []byte
	cached := ""
	if r.Cache != "" {
		cached = filepath.Join(r.Cache, strconv.FormatInt(n, 10))
		sealed, _ = os.ReadFile(cached)
	}
	fresh := sealed == nil
	if fresh {
		var err error
		if sealed, err = r.Client.Manifest(ctx, r.Vault, n); err != nil {
			return manifest{}, err
		}
	}
	plain, err := r.Keys.OpenManifest(r.Vault, n, sealed)
	if err != nil {
		if !fresh {
			_ = os.Remove(cached) // a damaged copy: fetched again next time
		}
		return manifest{}, err
	}
	m, err := parseManifest(plain)
	if err != nil {
		if !fresh {
			_ = os.Remove(cached)
		}
		return manifest{}, err
	}
	// Kept only once it opened and made sense: the cache never holds what
	// the cloud made up.
	if fresh && cached != "" && os.MkdirAll(r.Cache, 0o700) == nil {
		_ = os.WriteFile(cached, sealed, 0o600)
	}
	return m, nil
}

// eachVersion calls fn for every version kept, newest first, until it
// returns false.
func (r *Remote) eachVersion(ctx context.Context, fn func(VersionInfo, manifest) bool) error {
	var before int64
	for {
		page, err := r.Client.Versions(ctx, r.Vault, before)
		if err != nil {
			return err
		}
		for _, v := range page {
			m, err := r.manifestAt(ctx, v.N)
			if err != nil {
				return err
			}
			if !fn(v, m) {
				return nil
			}
			before = v.N
		}
		// The cloud checks every page (Versions): numbers only go down,
		// so the loop ends; a full page that did not move is an error.
		if len(page) < versionsPage || before <= 1 {
			return nil
		}
	}
}

// FileHistory lists the contents path had, newest first (at most limit):
// each with the version where it was saved.
func (r *Remote) FileHistory(ctx context.Context, path string, limit int) ([]NoteVersion, error) {
	var out []NoteVersion
	sha := ""
	first := true
	err := r.eachVersion(ctx, func(v VersionInfo, m manifest) bool {
		f, ok := m.Files[path]
		if !ok {
			sha = ""
			first = false
			return true
		}
		if f.SHA == sha && len(out) > 0 {
			// The same content, saved earlier: that is when it was saved.
			out[len(out)-1].N, out[len(out)-1].At = v.N, v.CreatedAt
		} else {
			if len(out) == limit {
				return false
			}
			out = append(out, NoteVersion{N: v.N, At: v.CreatedAt, Size: f.Size, Current: first})
			sha = f.SHA
		}
		first = false
		return true
	})
	return out, err
}

// ReadAt returns the content path had in version n.
func (r *Remote) ReadAt(ctx context.Context, path string, n int64) ([]byte, error) {
	m, err := r.manifestAt(ctx, n)
	if err != nil {
		return nil, err
	}
	f, ok := m.Files[path]
	if !ok {
		return nil, ErrNotInHistory
	}
	return r.download(ctx, f)
}

// Deleted lists the files of the versions kept that the newest version no
// longer has, the most recently deleted first.
func (r *Remote) Deleted(ctx context.Context) ([]DeletedFile, error) {
	var head map[string]manifestFile
	found := map[string]DeletedFile{}
	err := r.eachVersion(ctx, func(v VersionInfo, m manifest) bool {
		if head == nil {
			head = m.Files
			return true
		}
		for p := range m.Files {
			if _, ok := head[p]; ok {
				continue
			}
			if _, ok := found[p]; !ok {
				found[p] = DeletedFile{Path: p, N: v.N, At: v.CreatedAt}
			}
		}
		return true
	})
	out := make([]DeletedFile, 0, len(found))
	for _, d := range found {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].N != out[j].N {
			return out[i].N > out[j].N
		}
		return out[i].Path < out[j].Path
	})
	return out, err
}
