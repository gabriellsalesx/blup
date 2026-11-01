package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/config"
)

// A vault's database shows as one item: its records' folder, with the
// database's kind and its .table, which isn't listed by itself.
func TestTreeDatabase(t *testing.T) {
	site := config.NewSite(t.TempDir())
	site.Vault, site.Kind = t.TempDir(), config.KindNotes
	rel, err := app.CreateDatabase(site, "Bancos de dados", "Clientes", app.DBSchema{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.AddRecord(site, rel, "Ana", nil); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(site.Vault, "Bancos de dados", "Outra pasta"), 0o755); err != nil {
		t.Fatal(err)
	}
	root, err := (&Studio{site: site}).Tree()
	if err != nil {
		t.Fatal(err)
	}
	dbs := root.Children[0].Children
	if len(dbs) != 2 {
		t.Fatalf("items = %+v", dbs)
	}
	db := dbs[0]
	if db.Name != "Clientes" || !db.Dir || db.Kind != KindDatabase || db.Table != rel || len(db.Children) != 1 {
		t.Fatalf("database = %+v", db)
	}
	if other := dbs[1]; other.Kind != "" || other.Table != "" {
		t.Fatalf("a plain folder = %+v", other)
	}
}
