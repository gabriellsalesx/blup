// Package cloudcrypt is the end-to-end encryption of notes synced through
// Blup Cloud. Everything is compressed and encrypted on the computer; the
// cloud only stores sealed blocks and the master key wrapped twice (by the
// notes password and by the recovery key), so neither Blup nor anyone who
// breaks into the cloud can read the notes.
//
//   - The master key is 32 random bytes, one per account.
//   - The notes password goes through Argon2id to wrap the master key.
//   - The recovery key is 32 random bytes shown once (base32 groups); it
//     wraps a second copy of the master key.
//   - Subkeys come from the master key through HKDF-SHA256: one encrypts
//     blocks, one names blocks, one encrypts manifests.
//   - A block is a piece of a file (up to BlockSize), compressed when that
//     helps and sealed with XChaCha20-Poly1305. Its id is an HMAC of the
//     plain piece, so equal pieces are stored once without letting the cloud
//     confirm a known content.
//
// Only the standard library and golang.org/x/crypto are used.
package cloudcrypt

import (
	"bytes"
	"compress/flate"
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

// KeySize is the size of the master key and of every subkey.
const KeySize = 32

// BlockSize is the largest piece of a file sealed as one block. It keeps
// every upload well under the Workers request body limit.
const BlockSize = 4 << 20

// format is the first byte of every sealed value.
const format = 1

const (
	flagPlain      = 0
	flagCompressed = 1
)

var (
	// ErrWrongPassword: the notes password (or recovery key) does not open
	// the master key.
	ErrWrongPassword = errors.New("cloudcrypt: wrong notes password or recovery key")
	// ErrCorrupt: a sealed value was changed, truncated or belongs elsewhere.
	ErrCorrupt = errors.New("cloudcrypt: sealed data is corrupt or does not belong here")
)

// Params are the Argon2id settings stored next to the wrapped keys, so they
// can be raised later without breaking old accounts.
type Params struct {
	Time      uint32 `json:"time"`
	MemoryKiB uint32 `json:"memory_kib"`
	Threads   uint8  `json:"threads"`
	Salt      []byte `json:"salt"`
}

// DefaultParams returns the current settings with a fresh salt.
func DefaultParams() (Params, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return Params{}, err
	}
	return Params{Time: 3, MemoryKiB: 64 * 1024, Threads: 4, Salt: salt}, nil
}

// The most a wrapping may ask of this computer. The parameters come from
// the cloud with the wrapped key: without a ceiling, a cloud (or someone
// in the middle) could ask for terabytes of memory and crash the Studio
// every time the notes password is typed. Room is left to raise them.
const (
	maxTime      = 64
	maxMemoryKiB = 4 << 20 // 4 GiB
	maxSalt      = 1024
)

func (p Params) derive(password string) ([]byte, error) {
	if p.Time == 0 || p.MemoryKiB == 0 || p.Threads == 0 || len(p.Salt) < 16 {
		return nil, errors.New("cloudcrypt: invalid key parameters")
	}
	if p.Time > maxTime || p.MemoryKiB > maxMemoryKiB || len(p.Salt) > maxSalt {
		return nil, errors.New("cloudcrypt: key parameters too costly for this computer")
	}
	return argon2.IDKey([]byte(password), p.Salt, p.Time, p.MemoryKiB, p.Threads, KeySize), nil
}

// Wrapped is what the cloud keeps: the master key sealed by the notes
// password and by the recovery key.
type Wrapped struct {
	Params     Params `json:"params"`
	ByPassword []byte `json:"by_password"`
	ByRecovery []byte `json:"by_recovery"`
}

// NewKeys creates the master key and the recovery key (to be shown once)
// and wraps the master key with both.
func NewKeys(password string) (master []byte, recovery string, w Wrapped, err error) {
	if password == "" {
		return nil, "", Wrapped{}, errors.New("cloudcrypt: the notes password is empty")
	}
	master = make([]byte, KeySize)
	rk := make([]byte, KeySize)
	if _, err = rand.Read(master); err != nil {
		return nil, "", Wrapped{}, err
	}
	if _, err = rand.Read(rk); err != nil {
		return nil, "", Wrapped{}, err
	}
	if w, err = wrapPassword(master, password); err != nil {
		return nil, "", Wrapped{}, err
	}
	if w.ByRecovery, err = seal(rk, []byte("blup recovery"), master, false); err != nil {
		return nil, "", Wrapped{}, err
	}
	return master, FormatRecovery(rk), w, nil
}

func wrapPassword(master []byte, password string) (Wrapped, error) {
	p, err := DefaultParams()
	if err != nil {
		return Wrapped{}, err
	}
	kek, err := p.derive(password)
	if err != nil {
		return Wrapped{}, err
	}
	sealed, err := seal(kek, []byte("blup password"), master, false)
	if err != nil {
		return Wrapped{}, err
	}
	return Wrapped{Params: p, ByPassword: sealed}, nil
}

// Unlock opens the master key with the notes password.
func Unlock(w Wrapped, password string) ([]byte, error) {
	kek, err := w.Params.derive(password)
	if err != nil {
		return nil, err
	}
	master, err := open(kek, []byte("blup password"), w.ByPassword)
	if err != nil {
		return nil, ErrWrongPassword
	}
	return master, nil
}

// Recover opens the master key with the recovery key.
func Recover(w Wrapped, recovery string) ([]byte, error) {
	rk, err := ParseRecovery(recovery)
	if err != nil {
		return nil, err
	}
	master, err := open(rk, []byte("blup recovery"), w.ByRecovery)
	if err != nil {
		return nil, ErrWrongPassword
	}
	return master, nil
}

// ChangePassword wraps the master key with a new notes password, keeping the
// recovery key as it is.
func ChangePassword(w Wrapped, master []byte, password string) (Wrapped, error) {
	if password == "" {
		return Wrapped{}, errors.New("cloudcrypt: the notes password is empty")
	}
	nw, err := wrapPassword(master, password)
	if err != nil {
		return Wrapped{}, err
	}
	nw.ByRecovery = w.ByRecovery
	return nw, nil
}

var recoveryEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// FormatRecovery shows a recovery key as groups of four characters.
func FormatRecovery(key []byte) string {
	s := recoveryEncoding.EncodeToString(key)
	var b strings.Builder
	for i := 0; i < len(s); i += 4 {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(s[i:min(i+4, len(s))])
	}
	return b.String()
}

// ParseRecovery reads a recovery key typed back, ignoring case, spaces and
// hyphens.
func ParseRecovery(s string) ([]byte, error) {
	s = strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' || r == '\t' || r == '\n' {
			return -1
		}
		return r
	}, strings.ToUpper(s))
	key, err := recoveryEncoding.DecodeString(s)
	if err != nil || len(key) != KeySize {
		return nil, errors.New("cloudcrypt: the recovery key is incomplete or mistyped")
	}
	return key, nil
}

// Keys are the subkeys of a master key.
type Keys struct {
	block, id, manifest []byte
}

// Derive returns the subkeys of a master key.
func Derive(master []byte) (*Keys, error) {
	if len(master) != KeySize {
		return nil, errors.New("cloudcrypt: invalid master key")
	}
	sub := func(info string) ([]byte, error) {
		return hkdf.Key(sha256.New, master, nil, info, KeySize)
	}
	var k Keys
	var err error
	if k.block, err = sub("blup block v1"); err != nil {
		return nil, err
	}
	if k.id, err = sub("blup block id v1"); err != nil {
		return nil, err
	}
	if k.manifest, err = sub("blup manifest v1"); err != nil {
		return nil, err
	}
	return &k, nil
}

// Split cuts a file into pieces of up to BlockSize. An empty file is one
// empty piece, so it still has a block.
func Split(data []byte) [][]byte {
	if len(data) == 0 {
		return [][]byte{{}}
	}
	var out [][]byte
	for len(data) > 0 {
		n := min(len(data), BlockSize)
		out = append(out, data[:n])
		data = data[n:]
	}
	return out
}

// BlockID names a piece: hex HMAC-SHA256 under the id subkey.
func (k *Keys) BlockID(piece []byte) string {
	m := hmac.New(sha256.New, k.id)
	m.Write(piece)
	return hex.EncodeToString(m.Sum(nil))
}

// SealBlock compresses (when it helps) and encrypts a piece named id.
func (k *Keys) SealBlock(id string, piece []byte) ([]byte, error) {
	return seal(k.block, []byte("block "+id), piece, true)
}

// OpenBlock decrypts a block and checks it is the piece named id.
func (k *Keys) OpenBlock(id string, sealed []byte) ([]byte, error) {
	piece, err := open(k.block, []byte("block "+id), sealed)
	if err != nil {
		return nil, err
	}
	if !hmac.Equal([]byte(k.BlockID(piece)), []byte(id)) {
		return nil, ErrCorrupt
	}
	return piece, nil
}

// SealManifest encrypts the manifest of version n of a vault. Binding both
// to the ciphertext stops the cloud from swapping versions or vaults.
func (k *Keys) SealManifest(vault string, n int64, data []byte) ([]byte, error) {
	return seal(k.manifest, manifestAD(vault, n), data, true)
}

// OpenManifest decrypts the manifest of version n of a vault.
func (k *Keys) OpenManifest(vault string, n int64, sealed []byte) ([]byte, error) {
	return open(k.manifest, manifestAD(vault, n), sealed)
}

// SealName encrypts a vault name (it is not bound to anything else).
func (k *Keys) SealName(name string) ([]byte, error) {
	return seal(k.manifest, []byte("vault name"), []byte(name), false)
}

// OpenName decrypts a vault name.
func (k *Keys) OpenName(sealed []byte) (string, error) {
	b, err := open(k.manifest, []byte("vault name"), sealed)
	return string(b), err
}

func manifestAD(vault string, n int64) []byte {
	return fmt.Appendf(nil, "manifest %s %d", vault, n)
}

// seal is format byte, flag byte, 24-byte nonce, ciphertext. The format and
// flag bytes are authenticated too.
func seal(key, ad, plain []byte, compress bool) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	flag := byte(flagPlain)
	if compress && len(plain) > 0 {
		if c, err := deflate(plain); err == nil && len(c) < len(plain) {
			plain, flag = c, flagCompressed
		}
	}
	out := make([]byte, 2+aead.NonceSize(), 2+aead.NonceSize()+len(plain)+aead.Overhead())
	out[0], out[1] = format, flag
	if _, err := rand.Read(out[2:]); err != nil {
		return nil, err
	}
	return aead.Seal(out, out[2:], plain, append(out[:2:2], ad...)), nil
}

func open(key, ad, sealed []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	head := 2 + aead.NonceSize()
	if len(sealed) < head+aead.Overhead() || sealed[0] != format {
		return nil, ErrCorrupt
	}
	plain, err := aead.Open(nil, sealed[2:head], sealed[head:], append(sealed[:2:2], ad...))
	if err != nil {
		return nil, ErrCorrupt
	}
	switch sealed[1] {
	case flagPlain:
		return plain, nil
	case flagCompressed:
		return inflate(plain)
	}
	return nil, ErrCorrupt
}

func deflate(b []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.DefaultCompression)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(b); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// inflate stops past BlockSize plus slack: a sealed value never holds more,
// so a larger one is not ours.
func inflate(b []byte) ([]byte, error) {
	r := flate.NewReader(bytes.NewReader(b))
	defer r.Close()
	out, err := io.ReadAll(io.LimitReader(r, maxPlain+1))
	if err != nil || len(out) > maxPlain {
		return nil, ErrCorrupt
	}
	return out, nil
}

// maxPlain bounds a decompressed value: a block, or a manifest of a very
// large vault.
const maxPlain = 64 << 20
