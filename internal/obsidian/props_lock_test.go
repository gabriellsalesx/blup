package obsidian

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// The editor's save (SetBody) and a background job's properties
// (SetProperties) on the same note at once: neither loses the other's change.
func TestNoteWritesDontLoseEachOther(t *testing.T) {
	p := filepath.Join(t.TempDir(), "n.md")
	os.WriteFile(p, []byte("---\ntitle: N\n---\nstart"), 0o644)
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(2)
		go func() { defer wg.Done(); _ = SetBody(p, fmt.Sprintf("body %d", i)) }()
		go func() { defer wg.Done(); _ = SetProperties(p, Prop{Key: fmt.Sprintf("k%d", i), Value: i}) }()
	}
	wg.Wait()
	n, err := ReadNote(p)
	if err != nil {
		t.Fatal(err)
	}
	for i := range 50 {
		if n.Props[fmt.Sprintf("k%d", i)] == nil {
			t.Fatalf("property k%d was lost", i)
		}
	}
	if !strings.HasPrefix(n.Body, "body ") || n.String("title") != "N" {
		t.Fatalf("note %q %v", n.Body, n.Props)
	}
	if left, _ := filepath.Glob(filepath.Join(filepath.Dir(p), ".n.md.*")); len(left) != 0 {
		t.Fatalf("temporary files left: %v", left)
	}
}

func TestUpdatePropertiesNeedsTheNote(t *testing.T) {
	p := filepath.Join(t.TempDir(), "gone.md")
	if err := UpdateProperties(p, Prop{Key: "audio", Value: "x"}); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("got %v", err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("a missing note must not be made again")
	}
}
