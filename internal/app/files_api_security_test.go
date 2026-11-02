package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Attacks on the paths the Studio's window sends (rename, move, delete,
// attachments, databases) and on links planted in a synced or downloaded
// vault: nothing outside the notes folder is read, written or moved.

// outside is a folder next to the notes, standing for ~/.ssh or ~/.config.
func outside(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	write(t, d, "secret.txt", "SECRET\n")
	return d
}

func symlink(t *testing.T, target, link string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(link), 0o755)
	if err := os.Symlink(target, link); err != nil {
		t.Skip("no symlinks here:", err)
	}
}

func TestNotesPathRefusesTraversal(t *testing.T) {
	site := settingsSite(t)
	out := outside(t)
	symlink(t, out, filepath.Join(site.ContentDir(), "Fora"))
	symlink(t, filepath.Join(out, "secret.txt"), filepath.Join(site.ContentDir(), "Posts", "link.md"))
	for _, rel := range []string{"", ".", "../../etc/passwd", "../" + filepath.Base(site.Dir()), "/etc/passwd", "Posts/../../x.md", "Fora/secret.txt", "Fora/new.md", "Posts/link.md"} {
		if p, err := notesPath(site, rel); err == nil && !strings.HasPrefix(p, site.ContentDir()+string(filepath.Separator)) {
			t.Errorf("%q let out: %s", rel, p)
		} else if err == nil && (strings.HasPrefix(rel, "Fora") || rel == "Posts/link.md") {
			t.Errorf("%q followed a link out: %s", rel, p)
		}
	}
	if _, err := DeleteItem(site, "../../etc"); err == nil {
		t.Error("DeleteItem ../../etc")
	}
	if _, err := DeleteItem(site, "Fora/secret.txt"); err == nil {
		t.Error("DeleteItem through a link")
	}
	if _, err := MoveItem(site, "Posts/Um.md", "../.."); err == nil {
		t.Error("MoveItem out of the notes")
	}
	if _, err := MoveItem(site, "Posts/Um.md", "Fora"); err == nil {
		t.Error("MoveItem into a link out")
	}
	if _, err := RenameItem(site, "Posts/Um.md", "../../x"); err == nil {
		if _, err := os.Stat(filepath.Join(site.ContentDir(), "Posts", "....x.md")); err != nil {
			t.Error("RenameItem with ../ went somewhere else")
		}
	}
	if b, _ := os.ReadFile(filepath.Join(out, "secret.txt")); string(b) != "SECRET\n" {
		t.Fatal("the file outside changed")
	}
}

// A note that is a link to a file outside (~/.bashrc named x.md) is not
// rewritten when the links follow a rename.
func TestFixLinksSkipsLinkedNotes(t *testing.T) {
	site := settingsSite(t)
	out := t.TempDir()
	victim := filepath.Join(out, "bashrc")
	text := "alias x='see [um](Posts/Um.md)'\n"
	write(t, out, "bashrc", text)
	symlink(t, victim, filepath.Join(site.ContentDir(), "evil.md"))
	if _, err := RenameItem(site, "Posts/Um.md", "Dois"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(victim); string(b) != text {
		t.Fatalf("the file outside was rewritten: %q", b)
	}
}

func TestAttachmentNamesRefuseTraversal(t *testing.T) {
	site := settingsSite(t)
	dir := filepath.Join(site.ContentDir(), attachmentsFolder(site))
	write(t, dir, "a.png", "png")
	for _, bad := range []string{"", ".", "..", "../_index.md", "../../x", "a/b.png", ".hidden"} {
		if _, err := DeleteAttachment(site, bad); err == nil {
			t.Errorf("DeleteAttachment(%q) worked", bad)
		}
		if _, err := RenameAttachment(site, bad, "novo"); err == nil {
			t.Errorf("RenameAttachment(%q) worked", bad)
		}
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatal("the attachments folder went away:", err)
	}
	if _, err := os.Stat(filepath.Join(site.ContentDir(), "_index.md")); err != nil {
		t.Fatal("the home note went away:", err)
	}
	if n, err := RenameAttachment(site, "a.png", "b"); err != nil || n != "b.png" {
		t.Fatalf("a good rename failed: %q %v", n, err)
	}
}

// A link to nothing planted in the attachments folder, named like the
// image being pasted, doesn't make the file where it points.
func TestPasteImageDoesNotFollowDanglingLink(t *testing.T) {
	site := settingsSite(t)
	out := t.TempDir()
	target := filepath.Join(out, "autostart.desktop")
	dir := filepath.Join(site.ContentDir(), attachmentsFolder(site))
	symlink(t, target, filepath.Join(dir, "foto.png"))
	md, err := PasteImage(site, "Posts/Um.md", "foto.png", []byte("PNG"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); err == nil {
		t.Fatal("the image was written through the link")
	}
	if !strings.Contains(md, "foto%202.png") {
		t.Fatalf("md = %q", md)
	}
}

// An attachments folder that is a link out of the notes isn't written.
func TestPasteImageRefusesLinkedFolder(t *testing.T) {
	site := settingsSite(t)
	out := t.TempDir()
	symlink(t, out, filepath.Join(site.ContentDir(), attachmentsFolder(site)))
	if _, err := PasteImage(site, "Posts/Um.md", "x.png", []byte("PNG")); err == nil {
		t.Fatal("pasted into a folder outside the notes")
	}
	src := filepath.Join(t.TempDir(), "y.png")
	os.WriteFile(src, []byte("PNG"), 0o644)
	if _, err := AddAttachments(site, []string{src}); err == nil {
		t.Fatal("added into a folder outside the notes")
	}
	if entries, _ := os.ReadDir(out); len(entries) != 0 {
		t.Fatalf("files outside: %v", entries)
	}
}

// A .table from elsewhere naming "../../somewhere" as its records' folder
// neither lists nor creates notes there.
func TestDatabaseRecordsStayInside(t *testing.T) {
	site := settingsSite(t)
	rel, err := CreateDatabase(site, "", "Clientes", DBSchema{})
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	write(t, out, "leak.md", "---\nsenha: 123\n---\nSEGREDO\n")
	evil, _ := filepath.Rel(filepath.Dir(filepath.Join(site.ContentDir(), filepath.FromSlash(rel))), out)
	p := filepath.Join(site.ContentDir(), filepath.FromSlash(rel))
	os.WriteFile(p, []byte(`{"version":1,"records":`+strconvQuote(evil)+`}`), 0o644)

	db, err := ReadDatabase(site, rel)
	if err == nil {
		for _, r := range db.Records {
			if strings.Contains(r.Text, "SEGREDO") {
				t.Fatal("read a note outside the vault")
			}
		}
	}
	if _, err := AddRecord(site, rel, "Novo", nil); err == nil {
		if _, err := os.Stat(filepath.Join(out, "Novo.md")); err == nil {
			t.Fatal("a record was written outside the vault")
		}
	}
	if s, _ := readSchema(p); s.Records != "Clientes" {
		t.Fatalf("records = %q, want the default", s.Records)
	}

	// A records' folder that is a link out.
	os.RemoveAll(filepath.Join(filepath.Dir(p), "Clientes"))
	symlink(t, out, filepath.Join(filepath.Dir(p), "Clientes"))
	os.WriteFile(p, []byte(`{"version":1,"records":"Clientes"}`), 0o644)
	if _, err := ReadDatabase(site, rel); err == nil {
		t.Error("read the records through a link out")
	}
	if _, err := AddRecord(site, rel, "Outro", nil); err == nil {
		t.Error("added a record through a link out")
	}
	if _, err := os.Stat(filepath.Join(out, "Outro.md")); err == nil {
		t.Fatal("a record was written outside the vault")
	}
}

func strconvQuote(s string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`) + `"`
}

// The guide for agents isn't written through a link to nothing.
func TestAgentGuidesDoNotFollowLinks(t *testing.T) {
	site := settingsSite(t)
	out := t.TempDir()
	target := filepath.Join(out, "planted")
	symlink(t, target, filepath.Join(site.ContentDir(), "CLAUDE.md"))
	if err := WriteAgentGuides(site); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); err == nil {
		t.Fatal("the guide was written through the link")
	}
	if _, err := os.Stat(filepath.Join(site.ContentDir(), "AGENTS.md")); err != nil {
		t.Fatal("AGENTS.md missing:", err)
	}
}

// Within, by itself: the check the Studio's notePath relies on.
func TestWithinAttacks(t *testing.T) {
	root := t.TempDir()
	out := outside(t)
	symlink(t, out, filepath.Join(root, "dir-link"))
	symlink(t, filepath.Join(out, "nothing"), filepath.Join(root, "dangling"))
	for p, want := range map[string]bool{
		root:                                 true,
		filepath.Join(root, "a.md"):          true,
		filepath.Join(root, "new/deep/a.md"): true,
		filepath.Join(root, "../x"):          false,
		filepath.Join(root, "dir-link"):      false,
		filepath.Join(root, "dir-link/s.md"): false,
		filepath.Join(root, "dangling"):      false,
		root + "-sibling/a.md":               false,
	} {
		if got := Within(root, p); got != want {
			t.Errorf("Within(%q) = %v, want %v", p, got, want)
		}
	}
}
