package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A folder at the top of the notes deleted for good (Hugo's watcher misses
// it) starts the preview again; one moved away, or a deeper one, doesn't.
func TestFolderGoneRestartsPreview(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"Docs/Guides", "Roadmap", "Moved"} {
		os.MkdirAll(filepath.Join(root, d), 0o755)
		os.WriteFile(filepath.Join(root, d, "nota.md"), []byte("x"), 0o644)
	}
	gone := make(chan struct{}, 8)
	before := notesChanged
	notesChanged = func(string) {}
	watchMu.Lock()
	onFolderGone = func() { gone <- struct{}{} }
	watchMu.Unlock()
	watchNotes(root)
	t.Cleanup(func() {
		watchNotes("")
		watchMu.Lock()
		onFolderGone = nil
		watchMu.Unlock()
		notesChanged = before
	})
	quiet := func(what string) {
		t.Helper()
		select {
		case <-gone:
			t.Fatalf("%s restarted the preview", what)
		case <-time.After(300 * time.Millisecond):
		}
	}

	os.RemoveAll(filepath.Join(root, "Docs", "Guides"))
	quiet("a deeper folder")
	os.Rename(filepath.Join(root, "Moved"), filepath.Join(t.TempDir(), "Moved"))
	quiet("a moved folder")
	os.Remove(filepath.Join(root, "Roadmap", "nota.md"))
	quiet("a deleted note")
	os.RemoveAll(filepath.Join(root, "Roadmap"))
	select {
	case <-gone:
	case <-time.After(3 * time.Second):
		t.Fatal("a deleted section folder didn't restart the preview")
	}
	quiet("the same folder twice")
}
