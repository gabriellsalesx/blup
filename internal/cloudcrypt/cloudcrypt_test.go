package cloudcrypt

import (
	"bytes"
	"crypto/rand"
	"errors"
	"strings"
	"testing"
)

// newKeys creates keys or fails the test.
func newKeys(t *testing.T, password string) ([]byte, string, Wrapped) {
	t.Helper()
	master, recovery, w, err := NewKeys(password)
	if err != nil {
		t.Fatal(err)
	}
	return master, recovery, w
}

func TestPasswordAndRecovery(t *testing.T) {
	master, recovery, w := newKeys(t, "cavalo bateria grampo")
	if w.Params.MemoryKiB != 64*1024 || len(w.Params.Salt) != 16 {
		t.Fatalf("params = %+v", w.Params)
	}
	got, err := Unlock(w, "cavalo bateria grampo")
	if err != nil || !bytes.Equal(got, master) {
		t.Fatalf("Unlock = %x, %v", got, err)
	}
	if _, err := Unlock(w, "outra senha"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("wrong password: %v", err)
	}
	// Typed back in lower case, with hyphens instead of spaces.
	typed := strings.ToLower(strings.ReplaceAll(recovery, " ", "-"))
	got, err = Recover(w, typed)
	if err != nil || !bytes.Equal(got, master) {
		t.Fatalf("Recover = %x, %v", got, err)
	}
	if _, err := Recover(w, recovery[:len(recovery)-5]); err == nil {
		t.Fatal("a truncated recovery key was accepted")
	}

	// A new password keeps the recovery key working; the old one stops.
	w2, err := ChangePassword(w, master, "senha nova")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Unlock(w2, "cavalo bateria grampo"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("old password still opens: %v", err)
	}
	if got, err := Unlock(w2, "senha nova"); err != nil || !bytes.Equal(got, master) {
		t.Fatalf("new password: %v", err)
	}
	if got, err := Recover(w2, recovery); err != nil || !bytes.Equal(got, master) {
		t.Fatalf("recovery after change: %v", err)
	}
	if _, _, _, err := NewKeys(""); err == nil {
		t.Fatal("empty password accepted")
	}
}

func TestBlocks(t *testing.T) {
	master := make([]byte, KeySize)
	rand.Read(master)
	k, err := Derive(master)
	if err != nil {
		t.Fatal(err)
	}
	note := []byte(strings.Repeat("# Receita\n\nFarinha, ovos e leite.\n", 200))
	id := k.BlockID(note)
	sealed, err := k.SealBlock(id, note)
	if err != nil {
		t.Fatal(err)
	}
	if len(sealed) >= len(note) {
		t.Fatalf("text was not compressed: %d >= %d", len(sealed), len(note))
	}
	if bytes.Contains(sealed, []byte("Farinha")) {
		t.Fatal("plain text leaked")
	}
	got, err := k.OpenBlock(id, sealed)
	if err != nil || !bytes.Equal(got, note) {
		t.Fatalf("OpenBlock: %v", err)
	}

	// Equal pieces share an id; sealing twice still gives different bytes.
	again, _ := k.SealBlock(id, note)
	if k.BlockID(note) != id || bytes.Equal(again, sealed) {
		t.Fatal("ids must repeat and ciphertexts must not")
	}

	// Another account names the same piece differently.
	other := make([]byte, KeySize)
	rand.Read(other)
	k2, _ := Derive(other)
	if k2.BlockID(note) == id {
		t.Fatal("block ids must depend on the key")
	}
	if _, err := k2.OpenBlock(id, sealed); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("other key opened it: %v", err)
	}

	// A block stored under another id, or tampered with, is refused.
	if _, err := k.OpenBlock(k.BlockID([]byte("x")), sealed); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("swapped id: %v", err)
	}
	for _, i := range []int{0, 1, 5, len(sealed) - 1} {
		bad := bytes.Clone(sealed)
		bad[i] ^= 1
		if _, err := k.OpenBlock(id, bad); !errors.Is(err, ErrCorrupt) {
			t.Fatalf("tampered byte %d: %v", i, err)
		}
	}

	// Random data does not compress and is kept as is.
	noise := make([]byte, 1000)
	rand.Read(noise)
	s, _ := k.SealBlock(k.BlockID(noise), noise)
	if s[1] != flagPlain {
		t.Fatal("incompressible data should not be compressed")
	}
	if got, err := k.OpenBlock(k.BlockID(noise), s); err != nil || !bytes.Equal(got, noise) {
		t.Fatalf("noise: %v", err)
	}
}

func TestManifestsAndNames(t *testing.T) {
	master := make([]byte, KeySize)
	rand.Read(master)
	k, _ := Derive(master)
	m := []byte(`{"Notas/ação.md":{"sha":"abc"}}`)
	sealed, err := k.SealManifest("v1", 7, m)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := k.OpenManifest("v1", 7, sealed); err != nil || !bytes.Equal(got, m) {
		t.Fatalf("OpenManifest: %v", err)
	}
	// The cloud cannot hand version 7 as version 8, or as another vault's.
	if _, err := k.OpenManifest("v1", 8, sealed); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("version swap: %v", err)
	}
	if _, err := k.OpenManifest("v2", 7, sealed); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("vault swap: %v", err)
	}
	name, _ := k.SealName("Meu blog")
	if got, err := k.OpenName(name); err != nil || got != "Meu blog" {
		t.Fatalf("OpenName = %q, %v", got, err)
	}
}

func TestSplit(t *testing.T) {
	if got := Split(nil); len(got) != 1 || len(got[0]) != 0 {
		t.Fatalf("empty file: %d pieces", len(got))
	}
	data := make([]byte, 2*BlockSize+10)
	got := Split(data)
	if len(got) != 3 || len(got[0]) != BlockSize || len(got[2]) != 10 {
		t.Fatalf("pieces: %d", len(got))
	}
}
