package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDatabase(t *testing.T) {
	site := settingsSite(t)
	schema := DBSchema{
		Columns: []DBColumn{
			{Key: "empresa", Name: "Empresa", Type: "text"},
			{Key: "status", Name: "Status", Type: "select", Options: []DBOption{{Value: "Quente", Color: "red"}, {Value: "Frio", Color: "blue"}}},
		},
		Views: []DBView{{ID: "t", Name: "Tabela", Type: "table"}, {ID: "q", Name: "Quadro", Type: "board", GroupBy: "status"}},
	}
	rel, err := CreateDatabase(site, "", "Clientes", schema)
	if err != nil || rel != "Bancos de dados/Clientes.table" {
		t.Fatalf("create = %q, %v", rel, err)
	}
	// A second one with the same name gets a number (and its own folder).
	if rel2, err := CreateDatabase(site, "", "Clientes", schema); err != nil || rel2 != "Bancos de dados/Clientes 2.table" {
		t.Fatalf("create again = %q, %v", rel2, err)
	}

	rec, err := AddRecord(site, rel, "Maria Souza", map[string]any{"empresa": "Lumen", "status": "Quente", "vazio": ""})
	if err != nil || rec != "Bancos de dados/Clientes/Maria Souza.md" {
		t.Fatalf("add = %q, %v", rec, err)
	}
	if _, err := AddRecord(site, rel, "Maria Souza", nil); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(site.ContentDir(), rec))
	if s := string(b); !strings.Contains(s, "tags:") || !strings.Contains(s, "clientes") || !strings.Contains(s, "empresa: Lumen") || strings.Contains(s, "vazio") {
		t.Fatalf("record = %q", s)
	}

	db, err := ReadDatabase(site, rel)
	if err != nil {
		t.Fatal(err)
	}
	if db.Schema.Tag != "clientes" || len(db.Schema.Columns) != 2 || len(db.Records) != 2 {
		t.Fatalf("db = %+v", db)
	}
	if r := db.Records[0]; r.Name != "Maria Souza" || r.Props["status"] != "Quente" || r.Props["created"] == nil {
		t.Fatalf("record = %+v", r)
	}
	if db.Records[1].Name != "Maria Souza 2" {
		t.Fatalf("second = %+v", db.Records[1])
	}

	// The record knows its database; a note elsewhere doesn't.
	if table, s := DatabaseOf(site, rec); table != rel || s == nil || s.Columns[1].Key != "status" {
		t.Fatalf("database of = %q, %v", table, s)
	}
	if table, _ := DatabaseOf(site, "Posts/Um.md"); table != "" {
		t.Fatalf("a plain note is in %q", table)
	}

	// Saving the columns keeps the records where they are.
	db.Schema.Columns = append(db.Schema.Columns, DBColumn{Key: "telefone", Name: "Telefone", Type: "phone"})
	db.Schema.Records = "elsewhere"
	if err := SaveDatabaseSchema(site, rel, db.Schema); err != nil {
		t.Fatal(err)
	}
	if again, _ := ReadDatabase(site, rel); again.Schema.Records != "Clientes" || len(again.Schema.Columns) != 3 {
		t.Fatalf("saved = %+v", again.Schema)
	}
}
