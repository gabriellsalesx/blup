package cloudcrypt

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

// Ataques de um servidor malicioso contra o que ele guarda.

// O servidor manda parâmetros do Argon2 absurdos junto da chave embrulhada:
// o Studio recusa na hora, em vez de tentar alocar terabytes.
func TestAttackCostlyParamsRefused(t *testing.T) {
	_, _, w := newKeys(t, "senha")
	for _, p := range []Params{
		{Time: 3, MemoryKiB: 1 << 31, Threads: 4, Salt: w.Params.Salt},
		{Time: 1 << 30, MemoryKiB: 64 * 1024, Threads: 4, Salt: w.Params.Salt},
		{Time: 3, MemoryKiB: 64 * 1024, Threads: 4, Salt: make([]byte, 1<<20)},
	} {
		bad := w
		bad.Params = p
		start := time.Now()
		if _, err := Unlock(bad, "senha"); err == nil || errors.Is(err, ErrWrongPassword) {
			t.Fatalf("params %+v: %v", p.Time, err)
		}
		if time.Since(start) > time.Second {
			t.Fatal("costly params were tried")
		}
	}
	// Parâmetros fracos não abrem a chave (nada vaza: a senha não sai).
	weak := w
	weak.Params.Time, weak.Params.MemoryKiB = 1, 8
	if _, err := Unlock(weak, "senha"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("weak params: %v", err)
	}
}

// O servidor troca o manifesto de um cofre pelo de outro, ou de uma versão
// por outra, ou um bloco por outro: tudo é recusado.
func TestAttackSwapsRefused(t *testing.T) {
	master, _, _ := newKeys(t, "senha")
	k, _ := Derive(master)
	m5, _ := k.SealManifest("cofreA", 5, []byte(`{"files":{}}`))
	if _, err := k.OpenManifest("cofreA", 6, m5); err == nil {
		t.Fatal("manifest of version 5 opened as version 6")
	}
	if _, err := k.OpenManifest("cofreB", 5, m5); err == nil {
		t.Fatal("manifest of vault A opened as vault B")
	}
	a, b := []byte("nota A"), []byte("nota B")
	ida, idb := k.BlockID(a), k.BlockID(b)
	sb, _ := k.SealBlock(idb, b)
	if _, err := k.OpenBlock(ida, sb); err == nil {
		t.Fatal("block B opened as block A")
	}
	// Trocar o flag (comprimido/puro) é detectado: ele é autenticado.
	sa, _ := k.SealBlock(ida, bytes.Repeat(a, 100))
	sa[1] ^= 1
	if _, err := k.OpenBlock(ida, sa); err == nil {
		t.Fatal("flag change not detected")
	}
	// O nome de um cofre e um manifesto não se confundem.
	n, _ := k.SealName("Meu blog")
	if _, err := k.OpenManifest("cofreA", 1, n); err == nil {
		t.Fatal("name opened as manifest")
	}
}

// Cada selagem usa um nonce novo (24 bytes aleatórios): o mesmo texto dá
// cifras diferentes, e a chave de recuperação tem 256 bits.
func TestAttackNoncesAndRecoveryEntropy(t *testing.T) {
	master, rec, _ := newKeys(t, "senha")
	k, _ := Derive(master)
	seen := map[string]bool{}
	for range 200 {
		s, _ := k.SealBlock("x", []byte("igual"))
		nonce := string(s[2:26])
		if seen[nonce] {
			t.Fatal("nonce repeated")
		}
		seen[nonce] = true
	}
	if key, err := ParseRecovery(rec); err != nil || len(key) != 32 {
		t.Fatalf("recovery key: %d bytes, %v", len(key), err)
	}
}
