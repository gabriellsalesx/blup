package cloudcrypt

import (
	"encoding/json"
	"os"
	"testing"
)

// O que já foi cifrado continua abrindo: testdata/format1.json foi gerado
// com o código de antes do nome v2 e da cadeia de manifestos (nomes,
// bloco e manifesto no formato 1). Não regenere o arquivo.
type format1Fixture struct {
	Master        []byte `json:"master"`
	Vault         string `json:"vault"`
	NameA         []byte `json:"name_a"`
	NameAPlain    string `json:"name_a_plain"`
	NameB         []byte `json:"name_b"`
	NameBPlain    string `json:"name_b_plain"`
	BlockID       string `json:"block_id"`
	Block         []byte `json:"block"`
	BlockPlain    string `json:"block_plain"`
	ManifestN     int64  `json:"manifest_n"`
	Manifest      []byte `json:"manifest"`
	ManifestPlain string `json:"manifest_plain"`
}

func loadFormat1(t *testing.T) (format1Fixture, *Keys) {
	t.Helper()
	b, err := os.ReadFile("testdata/format1.json")
	if err != nil {
		t.Fatal(err)
	}
	var f format1Fixture
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	k, err := Derive(f.Master)
	if err != nil {
		t.Fatal(err)
	}
	return f, k
}

func TestCompatFormat1StillOpens(t *testing.T) {
	f, k := loadFormat1(t)
	for sealed, want := range map[string]string{string(f.NameA): f.NameAPlain, string(f.NameB): f.NameBPlain} {
		name, legacy, err := k.OpenName(f.Vault, []byte(sealed))
		if err != nil || !legacy || name != want {
			t.Fatalf("format 1 name: %q %v %v (want %q)", name, legacy, err, want)
		}
	}
	piece, err := k.OpenBlock(f.BlockID, f.Block)
	if err != nil || string(piece) != f.BlockPlain {
		t.Fatalf("format 1 block: %q %v", piece, err)
	}
	plain, err := k.OpenManifest(f.Vault, f.ManifestN, f.Manifest)
	if err != nil || string(plain) != f.ManifestPlain {
		t.Fatalf("format 1 manifest: %q %v", plain, err)
	}
	// Selado de novo no formato 2, o mesmo nome abre só no seu cofre.
	again, err := k.SealName(f.Vault, f.NameAPlain)
	if err != nil {
		t.Fatal(err)
	}
	if name, legacy, err := k.OpenName(f.Vault, again); err != nil || legacy || name != f.NameAPlain {
		t.Fatalf("resealed name: %q %v %v", name, legacy, err)
	}
	// As subchaves antigas não mudaram: o mesmo bloco tem o mesmo id.
	if k.BlockID([]byte(f.BlockPlain)) != f.BlockID {
		t.Fatal("block id subkey changed")
	}
}
