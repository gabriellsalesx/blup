package app

import (
	"os"
	"path/filepath"
	"slices"
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

// A database renamed or moved takes its records' folder along; a records'
// folder renamed by itself is written in its .table.
func TestDatabaseFollowsMoves(t *testing.T) {
	site := settingsSite(t)
	rel, err := CreateDatabase(site, "", "Clientes", DBSchema{})
	if err != nil {
		t.Fatal(err)
	}
	rec, _ := AddRecord(site, rel, "Ana", nil)
	if err := os.WriteFile(filepath.Join(site.ContentDir(), "Nota.md"), []byte("Ver [Ana](Bancos%20de%20dados/Clientes/Ana.md)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	to, err := RenameItem(site, rel, "Contatos")
	if err != nil || to != "Bancos de dados/Contatos.table" {
		t.Fatalf("rename = %q, %v", to, err)
	}
	db, err := ReadDatabase(site, to)
	if err != nil || db.Schema.Records != "Contatos" || len(db.Records) != 1 {
		t.Fatalf("after rename = %+v, %v (record was %s)", db, err, rec)
	}
	if b, _ := os.ReadFile(filepath.Join(site.ContentDir(), "Nota.md")); !strings.Contains(string(b), "Contatos/Ana.md") {
		t.Fatalf("link not fixed: %s", b)
	}
	moved, err := MoveItem(site, to, "")
	if err != nil || moved != "Contatos.table" {
		t.Fatalf("move = %q, %v", moved, err)
	}
	if db, _ := ReadDatabase(site, moved); len(db.Records) != 1 {
		t.Fatalf("after move = %+v", db)
	}
	if _, err := RenameItem(site, "Contatos", "Pessoas"); err != nil {
		t.Fatal(err)
	}
	if db, _ := ReadDatabase(site, moved); db.Schema.Records != "Pessoas" || len(db.Records) != 1 {
		t.Fatalf("after folder rename = %+v", db)
	}
	if table, _ := DatabaseOf(site, "Pessoas/Ana.md"); table != "Contatos.table" {
		t.Fatalf("record's database = %q", table)
	}
}

// A .table from elsewhere (a sync, a hand edit, another tool) is fixed on
// reading into one the Studio can show; fields the Studio doesn't know are
// kept when it saves.
func TestReadSchemaFixesBadFiles(t *testing.T) {
	site := settingsSite(t)
	root := site.ContentDir()
	cases := map[string]string{
		"null":     `null`,
		"garbage":  `{"version": -3, "records": "../../x", "columns": null, "views": null}`,
		"columns":  `{"version": 1, "records": "Clientes", "columns": [{"key": "a", "name": "", "type": "virus"}, {"key": "a", "type": "text"}, {"key": " ", "type": "text"}, {"key": "b", "type": "select", "options": [{"value": "x"}, {"value": "x"}, {"value": ""}]}], "views": []}`,
		"views":    `{"version": 1, "records": "Clientes", "columns": [{"key": "a", "type": "select"}], "views": [{"id": "v", "type": "board", "groupBy": "nada", "sort": "-nada", "filter": "nada", "match": "m"}, {"id": "v", "type": "pizza", "groupBy": "a", "sort": "-a", "filter": "a", "match": "m"}, {"type": "table"}]}`,
		"huge num": `{"version": 1e400}`,
	}
	for name, body := range cases {
		write(t, root, "Bancos/T.table", body)
		s, err := readSchema(filepath.Join(root, "Bancos", "T.table"))
		if name == "huge num" {
			if err == nil {
				t.Errorf("%s: read without error", name)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if s.Version < 1 || !recordsName(s.Records) || s.Columns == nil || s.Views == nil {
			t.Errorf("%s: %+v", name, s)
		}
		keys := map[string]bool{}
		for _, c := range s.Columns {
			if c.Key == "" || keys[c.Key] || !slices.Contains(columnTypes, c.Type) || c.Name == "" {
				t.Errorf("%s: bad column %+v", name, c)
			}
			keys[c.Key] = true
			seen := map[string]bool{}
			for _, o := range c.Options {
				if o.Value == "" || seen[o.Value] {
					t.Errorf("%s: bad option %+v", name, o)
				}
				seen[o.Value] = true
			}
		}
		ids := map[string]bool{}
		for _, v := range s.Views {
			if v.ID == "" || ids[v.ID] || (v.Type != "table" && v.Type != "board") {
				t.Errorf("%s: bad view %+v", name, v)
			}
			ids[v.ID] = true
			if v.GroupBy != "" && !keys[v.GroupBy] || v.Filter != "" && !keys[v.Filter] || strings.TrimPrefix(v.Sort, "-") != "" && !keys[strings.TrimPrefix(v.Sort, "-")] {
				t.Errorf("%s: view on a missing column %+v", name, v)
			}
		}
		if name == "views" && (len(s.Views) != 3 || s.Views[1].GroupBy != "a" || s.Views[1].Sort != "-a" || s.Views[1].Match != "m") {
			t.Errorf("good references were dropped: %+v", s.Views)
		}
		if name == "columns" && (len(s.Columns) != 2 || s.Columns[0].Type != "text" || len(s.Columns[1].Options) != 1) {
			t.Errorf("columns: %+v", s.Columns)
		}
	}

	write(t, root, "Bancos/T.table", `{"version": 1, "records": "T", "futuro": {"x": 1}, "columns": [], "views": []}`)
	if err := SaveDatabaseSchema(site, "Bancos/T.table", DBSchema{Columns: []DBColumn{{Key: "a", Type: "nope"}}}); err != nil {
		t.Fatal(err)
	}
	b := read(t, root, "Bancos/T.table")
	if !strings.Contains(b, `"futuro"`) || !strings.Contains(b, `"type": "text"`) {
		t.Errorf("saved: %s", b)
	}
}
