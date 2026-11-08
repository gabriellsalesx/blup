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
	n, _ := k.SealName("cofreA", "Meu blog")
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

// O servidor mexe na chave embrulhada que guarda: troca as duas cópias de
// lugar, troca o salt, ou entrega a de outra conta. Nada abre a chave
// mestra (e nada abre uma chave que ele conheça).
func TestAttackWrappedKeyTampering(t *testing.T) {
	master, rec, w := newKeys(t, "senha das notas")
	_, rec2, w2 := newKeys(t, "senha das notas")

	swapped := w
	swapped.ByPassword, swapped.ByRecovery = w.ByRecovery, w.ByPassword
	if _, err := Unlock(swapped, "senha das notas"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("copies swapped, password: %v", err)
	}
	if _, err := Recover(swapped, rec); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("copies swapped, recovery: %v", err)
	}

	salted := w
	salted.Params.Salt = append([]byte(nil), w.Params.Salt...)
	salted.Params.Salt[0] ^= 1
	if _, err := Unlock(salted, "senha das notas"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("salt changed: %v", err)
	}

	// A mesma senha em outra conta abre outra chave mestra, nunca a desta,
	// e a chave de recuperação de uma conta não abre a outra.
	other, err := Unlock(w2, "senha das notas")
	if err != nil || bytes.Equal(other, master) {
		t.Fatalf("other account: %v", err)
	}
	if _, err := Recover(w, rec2); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("recovery key of another account: %v", err)
	}

	// Um bloco selado com a chave de outra conta não abre aqui.
	k, _ := Derive(master)
	k2, _ := Derive(other)
	id := k2.BlockID([]byte("nota"))
	s, _ := k2.SealBlock(id, []byte("nota"))
	if _, err := k.OpenBlock(id, s); err == nil {
		t.Fatal("block of another account opened")
	}
	// O id de um bloco depende da conta: o servidor não liga o mesmo
	// conteúdo entre contas nem confirma um conteúdo conhecido pelo id.
	if k.BlockID([]byte("nota")) == id {
		t.Fatal("block ids equal across accounts")
	}
}

// O servidor troca os nomes de dois cofres da mesma conta (formato 2): o
// nome de um não abre no outro, e mexer no byte de formato para cair na
// leitura antiga também não abre.
func TestAttackVaultNameSwapDetected(t *testing.T) {
	master, _, _ := newKeys(t, "senha")
	k, _ := Derive(master)
	a, _ := k.SealName("cofreA", "Trabalho")
	b, _ := k.SealName("cofreB", "Pessoal")
	if a[0] != formatNameV2 {
		t.Fatalf("format byte %d", a[0])
	}
	if name, legacy, err := k.OpenName("cofreA", a); err != nil || legacy || name != "Trabalho" {
		t.Fatalf("own name: %q %v %v", name, legacy, err)
	}
	if _, _, err := k.OpenName("cofreA", b); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("name of vault B opened as vault A: %v", err)
	}
	if _, _, err := k.OpenName("cofreB", a); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("name of vault A opened as vault B: %v", err)
	}
	// Rebaixar o byte para o formato 1 não faz o nome abrir sem o id.
	down := bytes.Clone(a)
	down[0] = format
	if _, _, err := k.OpenName("cofreB", down); err == nil {
		t.Fatal("downgraded name opened")
	}
	// Um nome não abre como manifesto, nem um manifesto como nome.
	m, _ := k.SealManifest("cofreA", 1, []byte(`{"files":{}}`))
	if _, _, err := k.OpenName("cofreA", m); err == nil {
		t.Fatal("manifest opened as a name")
	}
	if _, err := k.OpenManifest("cofreA", 1, a); err == nil {
		t.Fatal("name opened as a manifest")
	}
	if _, err := k.SealName("", "x"); err == nil {
		t.Fatal("name sealed without a vault id")
	}
}

// Limite que sobra do formato 1: um nome antigo, selado antes do formato 2,
// não está amarrado a cofre nenhum, então ainda abre em qualquer cofre (e
// OpenName avisa que é antigo, para ser selado de novo no próximo PATCH).
func TestKnownLimitLegacyVaultNameNotBound(t *testing.T) {
	master, _, _ := newKeys(t, "senha")
	k, _ := Derive(master)
	a, _ := k.SealLegacyName("Trabalho")
	for _, vault := range []string{"cofreA", "cofreB"} {
		if name, legacy, err := k.OpenName(vault, a); err != nil || !legacy || name != "Trabalho" {
			t.Fatalf("legacy name in %s: %q %v %v", vault, name, legacy, err)
		}
	}
}

// O MAC da cadeia de manifestos depende do cofre, da versão, do conteúdo
// e da conta: o servidor não o calcula nem o reaproveita em outro lugar.
func TestAttackManifestMACBound(t *testing.T) {
	master, _, _ := newKeys(t, "senha")
	other, _, _ := newKeys(t, "senha")
	k, _ := Derive(master)
	k2, _ := Derive(other)
	base := k.ManifestMAC("cofreA", 3, []byte("x"))
	for name, mac := range map[string][]byte{
		"vault":   k.ManifestMAC("cofreB", 3, []byte("x")),
		"version": k.ManifestMAC("cofreA", 4, []byte("x")),
		"content": k.ManifestMAC("cofreA", 3, []byte("y")),
		"account": k2.ManifestMAC("cofreA", 3, []byte("x")),
	} {
		if bytes.Equal(mac, base) {
			t.Fatalf("MAC does not depend on the %s", name)
		}
	}
	if !bytes.Equal(base, k.ManifestMAC("cofreA", 3, []byte("x"))) || len(base) != 32 {
		t.Fatal("MAC is not deterministic")
	}
}
