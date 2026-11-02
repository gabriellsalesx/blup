package blupcloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/gabriellsalesx/blup/internal/cloudcrypt"
	"github.com/gabriellsalesx/blup/internal/deploy/github"
	"github.com/gabriellsalesx/blup/internal/notesync"
)

// manifest is a version of a vault: every file with the blocks holding it.
// It is sealed before leaving the computer.
type manifest struct {
	Files map[string]manifestFile `json:"files"`
}

type manifestFile struct {
	SHA    string   `json:"sha"` // git blob sha of the content
	Size   int64    `json:"size"`
	Mode   string   `json:"mode"`
	Blocks []string `json:"blocks"`
}

// Remote syncs a folder of notes with a vault (notesync.Remote).
type Remote struct {
	Client *Client
	Keys   *cloudcrypt.Keys
	Vault  string
	// MaxFile is the plan's largest file in bytes (0 = notesync's default).
	MaxFile int64
	// Cache is a folder for the sealed manifests of old versions (they never
	// change); "" fetches them every time.
	Cache string

	mu    sync.Mutex
	known map[string][]string // sha -> blocks already in the vault
}

type head struct {
	n int64
	m manifest
}

func (r *Remote) ID() string { return "blup:" + r.Vault }

func (r *Remote) Concurrency() int { return 4 }

// Older: a vault's versions are numbered and only grow (notesync.Ordered).
// A version that is not a number counts as older, so it is refused.
func (r *Remote) Older(v, than string) bool {
	a, err1 := strconv.ParseInt(v, 10, 64)
	b, err2 := strconv.ParseInt(than, 10, 64)
	if err2 != nil {
		return false // nothing known to compare with
	}
	return err1 != nil || a < b
}

func (r *Remote) MaxFileSize() int64 {
	if r.MaxFile > 0 {
		return r.MaxFile
	}
	return notesync.MaxFileSize
}

func (r *Remote) Head(ctx context.Context, _ bool) (notesync.Head, error) {
	if !ValidVaultID(r.Vault) {
		return notesync.Head{}, ErrBadVault
	}
	n, err := r.Client.VaultHead(ctx, r.Vault)
	if err != nil {
		return notesync.Head{}, err
	}
	h := head{n: n, m: manifest{Files: map[string]manifestFile{}}}
	if n > 0 {
		sealed, err := r.Client.Manifest(ctx, r.Vault, n)
		if err != nil {
			return notesync.Head{}, err
		}
		plain, err := r.Keys.OpenManifest(r.Vault, n, sealed)
		if err != nil {
			return notesync.Head{}, err
		}
		// Checked before anything here changes: a manifest that makes
		// no sense is an error, never an empty vault (which would delete
		// the notes on this computer).
		if h.m, err = parseManifest(plain); err != nil {
			return notesync.Head{}, err
		}
	}
	out := notesync.Head{Extra: h}
	if n > 0 {
		out.Version = strconv.FormatInt(n, 10)
	}
	r.mu.Lock()
	if r.known == nil {
		r.known = map[string][]string{}
	}
	for path, f := range h.m.Files {
		out.Files = append(out.Files, notesync.RemoteFile{Path: path, SHA: f.SHA, Mode: f.Mode, Ref: f})
		r.known[f.SHA] = f.Blocks
	}
	r.mu.Unlock()
	return out, nil
}

func (r *Remote) Upload(ctx context.Context, path, sha string, data []byte) (notesync.Change, error) {
	ch := notesync.Change{Path: path, SHA: sha}
	r.mu.Lock()
	blocks, ok := r.known[sha]
	r.mu.Unlock()
	if ok {
		ch.Ref = manifestFile{SHA: sha, Size: int64(len(data)), Blocks: blocks}
		return ch, nil
	}
	pieces := cloudcrypt.Split(data)
	ids := make([]string, len(pieces))
	for i, p := range pieces {
		ids[i] = r.Keys.BlockID(p)
	}
	missing, err := r.Client.Missing(ctx, ids)
	if err != nil {
		return notesync.Change{}, err
	}
	need := map[string]bool{}
	for _, id := range missing {
		need[id] = true
	}
	for i, p := range pieces {
		if !need[ids[i]] {
			continue
		}
		sealed, err := r.Keys.SealBlock(ids[i], p)
		if err != nil {
			return notesync.Change{}, err
		}
		if err := r.Client.PutBlock(ctx, ids[i], sealed); err != nil {
			return notesync.Change{}, err
		}
		delete(need, ids[i]) // a file repeating a piece sends it once
	}
	r.mu.Lock()
	if r.known == nil {
		r.known = map[string][]string{}
	}
	r.known[sha] = ids // reused if Sync starts over
	r.mu.Unlock()
	ch.Ref = manifestFile{SHA: sha, Size: int64(len(data)), Blocks: ids}
	return ch, nil
}

func (r *Remote) Copy(f notesync.RemoteFile, path string) notesync.Change {
	return notesync.Change{Path: path, Mode: f.Mode, SHA: f.SHA, Ref: f.Ref}
}

func (r *Remote) Download(ctx context.Context, f notesync.RemoteFile) ([]byte, error) {
	mf, ok := f.Ref.(manifestFile)
	if !ok {
		return nil, errors.New("blupcloud: unknown file")
	}
	return r.download(ctx, mf)
}

func (r *Remote) download(ctx context.Context, mf manifestFile) ([]byte, error) {
	if err := mf.check(); err != nil {
		return nil, cloudcrypt.ErrCorrupt
	}
	var buf bytes.Buffer
	buf.Grow(int(min(mf.Size, 64<<20)))
	for _, id := range mf.Blocks {
		sealed, err := r.Client.Block(ctx, id)
		if err != nil {
			return nil, err
		}
		piece, err := r.Keys.OpenBlock(id, sealed)
		if err != nil {
			return nil, err
		}
		// Never more than the manifest said: blocks larger than
		// announced stop here instead of filling the memory.
		if len(piece) > cloudcrypt.BlockSize || int64(buf.Len())+int64(len(piece)) > min(mf.Size, maxManifestFile) {
			return nil, cloudcrypt.ErrCorrupt
		}
		buf.Write(piece)
	}
	if int64(buf.Len()) != mf.Size || github.BlobSHA(buf.Bytes()) != mf.SHA {
		return nil, cloudcrypt.ErrCorrupt
	}
	return buf.Bytes(), nil
}

func (r *Remote) Commit(ctx context.Context, h notesync.Head, changes []notesync.Change, _ notesync.Counts) (string, error) {
	cur, _ := h.Extra.(head)
	next := manifest{Files: make(map[string]manifestFile, len(cur.m.Files)+len(changes))}
	for p, f := range cur.m.Files {
		next.Files[p] = f
	}
	for _, c := range changes {
		if c.Delete {
			delete(next.Files, c.Path)
			continue
		}
		mf, ok := c.Ref.(manifestFile)
		if !ok {
			return "", fmt.Errorf("blupcloud: no content for %s", c.Path)
		}
		mf.SHA, mf.Mode = c.SHA, c.Mode
		next.Files[c.Path] = mf
	}
	var blocks []string
	seen := map[string]bool{}
	for _, f := range next.Files {
		for _, id := range f.Blocks {
			if !seen[id] {
				seen[id] = true
				blocks = append(blocks, id)
			}
		}
	}
	plain, err := json.Marshal(next)
	if err != nil {
		return "", err
	}
	sealed, err := r.Keys.SealManifest(r.Vault, cur.n+1, plain)
	if err != nil {
		return "", err
	}
	n, err := r.Client.Commit(ctx, r.Vault, cur.n, blocks, sealed)
	if errors.Is(err, ErrMoved) {
		return "", notesync.ErrNotFastForward
	}
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(n, 10), nil
}
