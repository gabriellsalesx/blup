package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAttachments(t *testing.T) {
	site := settingsSite(t)
	root := site.ContentDir()
	folder := attachmentsFolder(site)
	write(t, root, folder+"/minha foto.png", "png")
	write(t, root, folder+"/sobra.pdf", "pdf")
	write(t, root, "Post.md", "---\nimage: \"[[minha foto.png]]\"\n---\n![a](../"+folder+"/minha%20foto.png)\n")

	list, err := ListAttachments(site)
	if err != nil || len(list) != 2 {
		t.Fatalf("list: %+v %v", list, err)
	}
	for _, a := range list {
		if a.Name == "minha foto.png" && (len(a.Uses) != 1 || a.Kind != "image") || a.Name == "sobra.pdf" && len(a.Uses) != 0 {
			t.Fatalf("uses: %+v", a)
		}
	}
	name, err := RenameAttachment(site, "minha foto.png", "capa nova")
	if err != nil || name != "capa nova.png" {
		t.Fatalf("rename: %q %v", name, err)
	}
	if p := read(t, root, "Post.md"); !strings.Contains(p, "[[capa nova.png]]") || !strings.Contains(p, "/capa%20nova.png)") {
		t.Fatalf("links: %s", p)
	}
	src := filepath.Join(t.TempDir(), "sobra.pdf")
	os.WriteFile(src, []byte("x"), 0o644)
	added, err := AddAttachments(site, []string{src})
	if err != nil || len(added) != 1 || added[0] != "sobra 2.pdf" {
		t.Fatalf("add: %v %v", added, err)
	}
	if _, err := DeleteAttachment(site, "sobra.pdf"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, folder, "sobra.pdf")); err == nil {
		t.Fatal("not deleted")
	}
}

// Programs, installers, scripts and shortcuts of every system are risky to
// open; documents, images and media are not. Windows' tricks (dots and
// spaces at the end, a ":" for a hidden stream) count too.
func TestRiskyAttachment(t *testing.T) {
	risky := []string{"setup.exe", "SETUP.EXE", "a.msi", "run.bat", "run.cmd", "x.com", "x.scr", "x.ps1", "x.vbs", "x.js", "x.jse",
		"x.wsf", "x.hta", "atalho.lnk", "x.reg", "x.jar", "Foo.app", "x.sh", "x.command", "x.AppImage", "x.deb", "x.rpm",
		"x.pkg", "x.dmg", "x.desktop", "x.py", "x.cpl", "x.msix", "x.url", "x.docm", "x.xlsm", "x.iso", "x.chm",
		"setup.exe.", "setup.exe ", "setup.exe. . ", "nota.txt:setup.exe", "foto.png.exe"}
	safe := []string{"foto.png", "doc.pdf", "audio.mp3", "video.mp4", "planilha.xlsx", "texto.docx", "notas.txt", "x.md", "exe", "arquivo", "x.zip", "x.svg", "x.json"}
	for _, n := range risky {
		if !RiskyAttachment(n) {
			t.Errorf("%q passed as safe", n)
		}
	}
	for _, n := range safe {
		if RiskyAttachment(n) {
			t.Errorf("%q passed as risky", n)
		}
	}
	site := settingsSite(t)
	write(t, site.ContentDir(), attachmentsFolder(site)+"/setup.exe", "MZ")
	write(t, site.ContentDir(), attachmentsFolder(site)+"/foto.png", "png")
	list, err := ListAttachments(site)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range list {
		if a.Risky != (a.Name == "setup.exe") {
			t.Errorf("%s: risky = %v", a.Name, a.Risky)
		}
	}
}
