package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
)

// Attacks through the bindings that take a path from the window: nothing
// outside the open site's notes is read or written, also through links.
func openSite(t *testing.T) (*Studio, string, string) {
	t.Helper()
	site := config.NewSite(t.TempDir())
	site.Vault, site.Folder, site.Title = t.TempDir(), "Site", "T"
	root := site.ContentDir()
	os.MkdirAll(filepath.Join(root, "Posts"), 0o755)
	os.WriteFile(filepath.Join(root, "Posts", "a.md"), []byte("oi"), 0o644)
	os.WriteFile(filepath.Join(root, "img.png"), []byte("PNG"), 0o644)
	out := t.TempDir()
	os.WriteFile(filepath.Join(out, "secret.png"), []byte("SECRET"), 0o644)
	os.WriteFile(filepath.Join(out, "secret.md"), []byte("SECRET"), 0o644)
	if err := os.Symlink(out, filepath.Join(root, "Fora")); err != nil {
		t.Skip("no symlinks:", err)
	}
	os.Symlink(filepath.Join(out, "secret.png"), filepath.Join(root, "Posts", "linked.png"))
	return &Studio{site: site}, root, out
}

func TestBindingPathsStayInside(t *testing.T) {
	s, root, out := openSite(t)
	bad := []string{"../../etc/passwd", "/etc/passwd", "Posts/../../x.md", "Fora/secret.md", "Fora/secret.png", "Posts/linked.png", "Fora/new.md"}
	for _, rel := range bad {
		if _, err := s.ReadNote(rel); err == nil {
			t.Errorf("ReadNote(%q) worked", rel)
		}
		if err := s.SaveNote(rel, "x", nil); err == nil && (strings.HasPrefix(rel, "Fora") || strings.Contains(rel, "linked")) {
			t.Errorf("SaveNote(%q) wrote through a link", rel)
		}
		if _, err := s.BoardFile(rel); err == nil {
			t.Errorf("BoardFile(%q) worked", rel)
		}
	}
	// "/etc/passwd" is the notes' own etc/passwd (joined under the root),
	// never the system's.
	if p, err := s.notePath("/etc/passwd"); err == nil && !strings.HasPrefix(p, root) {
		t.Errorf("notePath(/etc/passwd) = %s", p)
	}
	for _, f := range []string{root + "/../../etc/passwd", filepath.Join(out, "secret.png"), root + "/Fora/secret.png", root + "/Posts/linked.png", root + "-x/a.png"} {
		if _, err := s.ImageData(f); err == nil {
			t.Errorf("ImageData(%q) worked", f)
		}
	}
	for _, src := range []string{"../../../../../../etc/passwd", "../Fora/secret.png", "linked.png", "..%2F..%2FFora%2Fsecret.png"} {
		if d, _ := s.NoteImage("Posts/a.md", src); d != "" {
			t.Errorf("NoteImage(%q) = %.40s", src, d)
		}
	}
	if d, err := s.NoteImage("Posts/a.md", "../img.png"); err != nil || !strings.HasPrefix(d, "data:image/png") {
		t.Errorf("a good image failed: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(out, "secret.md")); string(b) != "SECRET" {
		t.Fatal("a file outside changed")
	}
	if _, err := os.Stat(filepath.Join(out, "new.md")); err == nil {
		t.Fatal("a file was made outside")
	}
	// Attachments by name only.
	for _, name := range []string{"../img.png", "..", ".", "a/b"} {
		if _, err := s.attachmentPath(name); err == nil && name != "a/b" {
			t.Errorf("attachmentPath(%q) worked", name)
		}
	}
	// Creating with a name that tries to climb stays in the folder.
	rel, err := s.CreateNote("Posts", "../../../x")
	if err == nil && !strings.HasPrefix(rel, "Posts/") {
		t.Errorf("CreateNote = %q, %v", rel, err)
	}
	if _, err := s.CreateNote("../..", "x"); err == nil {
		t.Error("CreateNote in ../.. worked")
	}
	if _, err := s.CreateNote("Fora", "x"); err == nil {
		t.Error("CreateNote through a link worked")
	}
}

// Only known sites open in the file manager (nothing runs here: the
// unknown folder is refused before).
func TestOpenSiteFolderRefusesUnknown(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := &Studio{}
	for _, dir := range []string{"/", "/etc", "--help", "file:///etc/passwd"} {
		if err := s.OpenSiteFolder(dir); err == nil {
			t.Errorf("OpenSiteFolder(%q) worked", dir)
		}
	}
}
