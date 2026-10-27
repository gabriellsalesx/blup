// Package lock encrypts password-protected posts in a built site.
//
// When Hugo runs with HUGO_BLUP_MODE=publish, the theme wraps the content
// of each protected post between <blup-begin data-id="ID"> and <blup-end>
// markers and writes Manifest (ID → password) at the site root. Process
// replaces every marked region with its AES-256-GCM ciphertext, deletes the
// manifest and fails if any plaintext region is left. The browser decrypts
// with WebCrypto (theme/assets/js/lock.js) using the same parameters.
package lock

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Manifest is the file the theme writes with the password of each post.
const Manifest = "blup-locks.json"

// Global marks posts that use the site-wide password.
const Global = "__global__"

// Iterations of PBKDF2-SHA256 (OWASP 2023 recommendation).
const Iterations = 600_000

var (
	// ErrNoGlobalPassword means a post asks for the site-wide password but
	// none was set.
	ErrNoGlobalPassword = errors.New("a post uses the general password, but none is set")
	// ErrLeftover means a protected region could not be encrypted.
	ErrLeftover = errors.New("a protected post was left unencrypted")
)

var region = regexp.MustCompile(`(?s)<blup-begin data-id="?([0-9a-f]+)"?\s*></blup-begin>(.*?)<blup-end\s*></blup-end>`)

// Pending reads the manifest and reports whether any post needs the global
// password, without changing anything.
func Pending(dir string) (posts int, needsGlobal bool, err error) {
	m, err := readManifest(dir)
	if err != nil {
		return 0, false, err
	}
	for _, pw := range m {
		if pw == Global {
			needsGlobal = true
		}
	}
	return len(m), needsGlobal, nil
}

// Process encrypts every protected region under dir. global is the site-wide
// password used by posts marked with Global. It returns how many posts were
// encrypted.
func Process(dir, global string) (int, error) {
	m, err := readManifest(dir)
	if err != nil {
		return 0, err
	}
	// The manifest holds passwords: it must never be published.
	defer os.Remove(filepath.Join(dir, Manifest))
	for _, pw := range m {
		if pw == Global && global == "" {
			return 0, ErrNoGlobalPassword // checked before touching any file
		}
	}

	done := map[string]bool{}
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".html") {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if !strings.Contains(string(b), "<blup-begin") {
			return nil
		}
		var failure error
		out := region.ReplaceAllStringFunc(string(b), func(match string) string {
			sm := region.FindStringSubmatch(match)
			id, plain := sm[1], sm[2]
			pw, ok := m[id]
			if !ok {
				failure = fmt.Errorf("%w: %s", ErrLeftover, p)
				return match
			}
			if pw == Global {
				if global == "" {
					failure = ErrNoGlobalPassword
					return match
				}
				pw = global
			}
			sealed, err := Encrypt(pw, []byte(plain))
			if err != nil {
				failure = err
				return match
			}
			done[id] = true
			return sealed.HTML()
		})
		if failure != nil {
			return failure
		}
		if strings.Contains(out, "<blup-begin") {
			return fmt.Errorf("%w: %s", ErrLeftover, p)
		}
		return os.WriteFile(p, []byte(out), 0o644)
	})
	return len(done), err
}

// Sealed is an encrypted post.
type Sealed struct {
	Salt, IV, Ciphertext []byte
	Iterations           int
}

// Encrypt derives an AES-256 key from password with PBKDF2-SHA256 and seals
// plain with AES-GCM (ciphertext followed by the tag, as WebCrypto expects).
func Encrypt(password string, plain []byte) (Sealed, error) {
	s := Sealed{Salt: make([]byte, 16), IV: make([]byte, 12), Iterations: Iterations}
	if _, err := rand.Read(s.Salt); err != nil {
		return s, err
	}
	if _, err := rand.Read(s.IV); err != nil {
		return s, err
	}
	gcm, err := newGCM(password, s.Salt, s.Iterations)
	if err != nil {
		return s, err
	}
	s.Ciphertext = gcm.Seal(nil, s.IV, plain, nil)
	return s, nil
}

// Decrypt reverses Encrypt (used by tests).
func Decrypt(password string, s Sealed) ([]byte, error) {
	gcm, err := newGCM(password, s.Salt, s.Iterations)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, s.IV, s.Ciphertext, nil)
}

// HTML is the element the browser script reads.
func (s Sealed) HTML() string {
	enc := base64.StdEncoding.EncodeToString
	return fmt.Sprintf(`<div hidden data-blup-cipher data-salt="%s" data-iv="%s" data-iter="%d" data-ct="%s"></div>`,
		enc(s.Salt), enc(s.IV), s.Iterations, enc(s.Ciphertext))
}

func newGCM(password string, salt []byte, iter int) (cipher.AEAD, error) {
	key, err := pbkdf2.Key(sha256.New, password, salt, iter, 32)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func readManifest(dir string) (map[string]string, error) {
	b, err := os.ReadFile(filepath.Join(dir, Manifest))
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	m := map[string]string{}
	if len(strings.TrimSpace(string(b))) == 0 {
		return m, nil
	}
	return m, json.Unmarshal(b, &m)
}
