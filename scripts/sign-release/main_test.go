package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gabriellsalesx/blup/internal/update"
)

func TestGenKeyNeverOverwrites(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "blup-release")
	pub, err := genKey(dir)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(filepath.Join(dir, keyFile))
	if _, err := genKey(dir); err == nil {
		t.Fatal("second -genkey must refuse")
	}
	after, _ := os.ReadFile(filepath.Join(dir, keyFile))
	if string(before) != string(after) {
		t.Fatal("private key was overwritten")
	}
	if runtime.GOOS != "windows" {
		if st, _ := os.Stat(dir); st.Mode().Perm() != 0o700 {
			t.Errorf("folder mode %v", st.Mode().Perm())
		}
		if st, _ := os.Stat(filepath.Join(dir, keyFile)); st.Mode().Perm() != 0o600 {
			t.Errorf("key mode %v", st.Mode().Perm())
		}
	}
	key, err := loadKey("", dir)
	if err != nil {
		t.Fatal(err)
	}
	if !key.Public().(ed25519.PublicKey).Equal(pub) {
		t.Fatal("loaded key does not match the public key")
	}
}

func TestSignFile(t *testing.T) {
	dir := t.TempDir()
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	in := filepath.Join(dir, "checksums.txt")
	os.WriteFile(in, []byte("abc  blup_1.0.0_windows_amd64.zip\n"), 0o644)

	// From the CI variable, as seed and as full private key.
	for _, env := range []string{
		base64.StdEncoding.EncodeToString(priv.Seed()),
		base64.StdEncoding.EncodeToString(priv),
	} {
		key, err := loadKey(env, filepath.Join(dir, "none"))
		if err != nil {
			t.Fatal(err)
		}
		out := filepath.Join(dir, "checksums.txt.sig")
		if err := signFile(key, in, out, []ed25519.PublicKey{pub}); err != nil {
			t.Fatal(err)
		}
		sums, _ := os.ReadFile(in)
		sig, _ := os.ReadFile(out)
		if err := update.VerifyChecksums(sums, sig, []ed25519.PublicKey{pub}); err != nil {
			t.Fatalf("signature does not verify: %v", err)
		}
	}

	// A key the app does not trust writes nothing.
	_, other, _ := ed25519.GenerateKey(rand.Reader)
	out := filepath.Join(dir, "other.sig")
	if err := signFile(other, in, out, []ed25519.PublicKey{pub}); err == nil {
		t.Fatal("signing with an untrusted key must fail")
	}
	if _, err := os.Stat(out); err == nil {
		t.Fatal("signature written for an untrusted key")
	}

	if _, err := loadKey("bm90IGEga2V5", dir); err == nil {
		t.Fatal("wrong-size key accepted")
	}
}
