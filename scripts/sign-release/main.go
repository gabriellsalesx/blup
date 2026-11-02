// Command sign-release signs a release's checksums.txt with the Blup release
// key (Ed25519), so `blup update` can tell the files came from the owner and
// not only from whoever controls the GitHub account.
//
// It is a separate program: nothing here goes into the blup or Blup Studio
// binaries. Usage:
//
//	go run ./scripts/sign-release -genkey
//	    creates the key pair in ~/.config/blup-release (once; never overwrites)
//	    and prints the PUBLIC key to paste into internal/update/keys.go.
//	go run ./scripts/sign-release -in dist/checksums.txt [-out dist/checksums.txt.sig]
//	    signs the exact bytes of the file.
//
// The private key is read from the BLUP_RELEASE_KEY environment variable
// (base64, for CI) or else from ~/.config/blup-release/release.key. It is
// never printed.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gabriellsalesx/blup/internal/update"
)

const (
	keyFile = "release.key"
	pubFile = "release.pub"
	envKey  = "BLUP_RELEASE_KEY"
)

func main() {
	genkey := flag.Bool("genkey", false, "create the key pair in -dir (never overwrites)")
	dir := flag.String("dir", "", "key folder (default ~/.config/blup-release)")
	in := flag.String("in", "", "checksums.txt to sign")
	out := flag.String("out", "", "signature file (default <in>.sig)")
	flag.Parse()

	if *dir == "" {
		d, err := defaultDir()
		if err != nil {
			fail(err)
		}
		*dir = d
	}
	if *genkey {
		pub, err := genKey(*dir)
		if err != nil {
			fail(err)
		}
		fmt.Println("public key (paste into internal/update/keys.go):")
		fmt.Println(base64.StdEncoding.EncodeToString(pub))
		fmt.Println("private key saved in", filepath.Join(*dir, keyFile))
		return
	}
	if *in == "" {
		fail(errors.New("missing -in checksums.txt (or -genkey)"))
	}
	if *out == "" {
		*out = *in + ".sig"
	}
	key, err := loadKey(os.Getenv(envKey), *dir)
	if err != nil {
		fail(err)
	}
	if err := signFile(key, *in, *out, update.PublicKeys()); err != nil {
		fail(err)
	}
	fmt.Println("signed", *in, "->", *out)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "sign-release:", err)
	os.Exit(1)
}

func defaultDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "blup-release"), nil
}

// genKey writes a new pair to dir (0700) as release.key (0600) and
// release.pub. An existing key is never replaced.
func genKey(dir string) (ed25519.PublicKey, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	_ = os.Chmod(dir, 0o700)
	priv := filepath.Join(dir, keyFile)
	if _, err := os.Lstat(priv); err == nil {
		return nil, fmt.Errorf("%s already exists: not overwriting it", priv)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(priv, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, err
	}
	if _, err := f.WriteString(base64.StdEncoding.EncodeToString(key.Seed()) + "\n"); err != nil {
		f.Close()
		os.Remove(priv)
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, pubFile), []byte(base64.StdEncoding.EncodeToString(pub)+"\n"), 0o644); err != nil {
		return nil, err
	}
	return pub, nil
}

// loadKey reads the private key from env (base64) or from dir/release.key.
// Both the 32-byte seed and the 64-byte Go private key are accepted.
func loadKey(env, dir string) (ed25519.PrivateKey, error) {
	text := strings.TrimSpace(env)
	if text == "" {
		b, err := os.ReadFile(filepath.Join(dir, keyFile))
		if err != nil {
			return nil, fmt.Errorf("no %s and no key file: %w", envKey, err)
		}
		text = strings.TrimSpace(string(b))
	}
	raw, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return nil, errors.New("the private key is not valid base64")
	}
	switch len(raw) {
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(raw), nil
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(raw), nil
	}
	return nil, errors.New("the private key has the wrong size")
}

// signFile signs the exact bytes of in and checks the result against the
// keys built into blup, so a release signed with a key the app does not
// know fails here instead of on every user's machine.
func signFile(key ed25519.PrivateKey, in, out string, trusted []ed25519.PublicKey) error {
	data, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	sig := base64.StdEncoding.EncodeToString(ed25519.Sign(key, data)) + "\n"
	if err := update.VerifyChecksums(data, []byte(sig), trusted); err != nil {
		return fmt.Errorf("this key is not one of the keys in internal/update/keys.go: %w", err)
	}
	return os.WriteFile(out, []byte(sig), 0o644)
}
