package blupcloud

import (
	"context"
	"crypto/hmac"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gabriellsalesx/blup/internal/notesync"
)

// The manifest chain. Every manifest saved now carries, sealed inside it,
// the chain MACs (cloudcrypt.ManifestMAC) of the versions right before it:
//
//	"prev": {"n": 41, "macs": ["<MAC of 41>", "<MAC of 40>", ...]}
//
// at most chainBack of them, newest first. A computer keeps the MAC of the
// last version it synced (notesync state, "chain") and only accepts a head
// that descends from it: the same version with the same MAC, or a newer
// one that lists that MAC at the right place. So the cloud can't show it an
// old version (also refused by number, Older) nor a version of another
// branch (a fork: two different versions with the same number, each
// shown to a different computer).
//
// When the head is more than chainBack versions ahead, the chain is
// followed through the versions the cloud still keeps (history of the
// paid plans), each one checked against the MAC listed by the next. When
// the version needed is no longer kept, the chain can't be followed and
// the head is accepted (the limit in docs/seguranca.md).
//
// Manifests sealed before the chain have no "prev": they are accepted and
// the chain starts at the first manifest that has it. A manifest without
// "prev" newer than a chained one can only have been saved by an older
// Blup (the cloud can't seal one), so it is accepted too and the chain
// starts again after it.

// chainBack is how many earlier MACs a manifest carries (about 3 KB).
const chainBack = 64

// chainWalk bounds the versions fetched to follow the chain once. Past it
// the chain is not followed any further and the head is accepted, as when
// the versions needed are no longer kept: refusing would lock a computer
// that was away for long out of its notes for good (a var for the tests).
var chainWalk = 4096

// chainLink is the "prev" of a manifest of version n: N is n-1, MACs the
// MACs of N, N-1, ... (none when N is 0, the first version of a vault).
type chainLink struct {
	N    int64    `json:"n"`
	MACs [][]byte `json:"macs"`
}

// check validates a link read from the manifest of version n.
func (l *chainLink) check(n int64) error {
	switch {
	case l.N != n-1:
		return fmt.Errorf("prev names version %d", l.N)
	case l.MACs == nil || len(l.MACs) > chainBack || int64(len(l.MACs)) > l.N:
		return errors.New("prev MACs")
	case l.N > 0 && len(l.MACs) == 0:
		return errors.New("prev without MACs")
	}
	for _, m := range l.MACs {
		if len(m) != 32 {
			return errors.New("prev MAC size")
		}
	}
	return nil
}

// link is the "prev" of the version after h.
func (h head) link() *chainLink {
	l := &chainLink{N: h.n, MACs: [][]byte{}}
	if h.n == 0 {
		return l
	}
	l.MACs = append(l.MACs, h.mac)
	if h.m.Prev != nil {
		l.MACs = append(l.MACs, h.m.Prev.MACs[:min(len(h.m.Prev.MACs), chainBack-1)]...)
	}
	return l
}

// openManifest opens and checks the sealed manifest of version n, and
// returns its chain MAC.
func (r *Remote) openManifest(n int64, sealed []byte) (manifest, []byte, error) {
	plain, err := r.Keys.OpenManifest(r.Vault, n, sealed)
	if err != nil {
		return manifest{}, nil, err
	}
	m, err := parseManifest(plain)
	if err != nil {
		return manifest{}, nil, err
	}
	if m.Prev != nil {
		if err := m.Prev.check(n); err != nil {
			return manifest{}, nil, fmt.Errorf("%w: manifest %d: %w", ErrBadAnswer, n, err)
		}
	}
	return m, r.Keys.ManifestMAC(r.Vault, n, plain), nil
}

func (r *Remote) remember(n int64, mac []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.macs == nil {
		r.macs = map[int64][]byte{}
	}
	r.macs[n] = mac
}

// Token is the chain MAC of a version this Remote read or saved, for the
// sync state ("" when unknown).
func (r *Remote) Token(version string) string {
	n, err := strconv.ParseInt(version, 10, 64)
	if err != nil {
		return ""
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if mac := r.macs[n]; mac != nil {
		return base64.StdEncoding.EncodeToString(mac)
	}
	return ""
}

// Forget drops the chain MACs this Remote remembers and the cached
// manifests of old versions (notesync.Forgetter): after the person chose to
// compare from scratch, versions of the branch left behind must not be
// read again as if they were part of the one accepted now (following the
// chain through them would stop the sync once more).
func (r *Remote) Forget() error {
	r.mu.Lock()
	r.macs = nil
	r.mu.Unlock()
	if r.Cache == "" {
		return nil
	}
	return os.RemoveAll(r.Cache)
}

// Continues reports whether the head h descends from version, whose chain
// MAC is token (notesync.Chained). It returns notesync.ErrForked when it
// doesn't.
func (r *Remote) Continues(ctx context.Context, h notesync.Head, version, token string) error {
	hd, ok := h.Extra.(head)
	n0, err1 := strconv.ParseInt(version, 10, 64)
	mac0, err2 := base64.StdEncoding.DecodeString(token)
	if !ok || err1 != nil || err2 != nil || len(mac0) != 32 || n0 <= 0 {
		return nil // nothing known to compare with
	}
	forked := func(at int64) error {
		return fmt.Errorf("%w (version %d does not continue version %d synced here)", notesync.ErrForked, at, n0)
	}
	switch {
	case hd.n < n0:
		return fmt.Errorf("%w (%d, synced %d)", notesync.ErrRolledBack, hd.n, n0)
	case hd.n == n0:
		if !hmac.Equal(hd.mac, mac0) {
			return forked(hd.n)
		}
		return nil
	}
	cur, at := hd.m.Prev, hd.n
	for range chainWalk {
		if cur == nil {
			return nil // saved by an older Blup: the chain starts again
		}
		// cur.MACs[i] is the MAC of version at-1-i.
		if i := at - 1 - n0; i < int64(len(cur.MACs)) {
			if !hmac.Equal(cur.MACs[i], mac0) {
				return forked(hd.n)
			}
			return nil
		}
		// Further back than this manifest lists: follow the chain through
		// the oldest version it names, if the cloud still keeps it.
		older := at - int64(len(cur.MACs))
		if len(cur.MACs) == 0 || older <= n0 {
			return forked(hd.n)
		}
		m, err := r.manifestAt(ctx, older)
		var e *apiError
		if errors.As(err, &e) && e.Status == http.StatusNotFound {
			return nil // no longer kept: the chain can't be followed
		}
		if err != nil {
			return err
		}
		r.mu.Lock()
		mac := r.macs[older]
		r.mu.Unlock()
		if !hmac.Equal(mac, cur.MACs[len(cur.MACs)-1]) {
			return forked(hd.n)
		}
		cur, at = m.Prev, older
	}
	return nil // too far back to follow: as when no longer kept
}
