package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gabriellsalesx/blup/internal/obsidian"
)

// Two saves of the same file at once each leave a whole version, and no
// temporary file stays behind for the sync to send.
func TestWriteSafeAtOnce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Nota.md")
	var wg sync.WaitGroup
	errs := make([]error, 16)
	for i := range errs {
		wg.Go(func() { errs[i] = writeSafe(path, []byte(strings.Repeat(fmt.Sprint(i%10), 4096))) })
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 4096 || strings.Count(string(data), string(data[:1])) != 4096 {
		t.Fatalf("a mixed or cut file: %d bytes", len(data))
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("files left behind: %v", entries)
	}
}

// The temporary file is hidden: the sync and the lists leave it out.
func TestWriteSafeTempHidden(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Nota.md")
	err := replaceSafe(path, func(tmp *os.File) error {
		if base := filepath.Base(tmp.Name()); !strings.HasPrefix(base, ".") || filepath.Ext(base) == ".md" {
			t.Errorf("temporary file %q would be seen as a note", base)
		}
		return fmt.Errorf("stop")
	})
	if err == nil {
		t.Fatal("the error of fill was lost")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("a failed save left files: %v", entries)
	}
}

// A note with properties broken by hand still keeps the text typed; the
// side panel's change says it could not.
func TestSaveBrokenProperties(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := &NotesService{}
	body := "---\ntags: [a\n---\nTexto novo\n"
	got, err := s.Save("Quebrada.md", body)
	if err != nil {
		t.Fatal(err)
	}
	if got != body {
		t.Fatalf("Save = %q, want the text as typed", got)
	}
	if _, err := s.SetProps("Quebrada.md", "", "", []string{"x"}); err == nil {
		t.Fatal("SetProps on broken properties said it worked")
	}
	path, _ := notePath("Quebrada.md")
	if data, _ := os.ReadFile(path); string(data) != body {
		t.Fatalf("the note changed: %q", data)
	}
}

// Save dates a note, created only once, in the same write as the text.
func TestSaveDates(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := &NotesService{}
	got, err := s.Save("Nova.md", "Oi\n")
	if err != nil {
		t.Fatal(err)
	}
	path, _ := notePath("Nova.md")
	n, err := obsidian.ReadNote(path)
	if err != nil {
		t.Fatal(err)
	}
	if n.String("created") == "" || n.String("updated") == "" || !strings.HasSuffix(got, "Oi\n") {
		t.Fatalf("Save = %q", got)
	}
	again, err := s.Save("Nova.md", "---\ncreated: 2020-01-02\n---\nOi\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(again, "2020-01-02") {
		t.Fatalf("the date it had was replaced: %q", again)
	}
}
