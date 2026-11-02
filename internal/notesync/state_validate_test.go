package notesync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// A state file with an entry that cannot be is not trusted at all: the
// next sync compares without it, so nothing is deduced as deleted from it.
func TestInvalidStateIgnored(t *testing.T) {
	dir := t.TempDir()
	s := &Syncer{Owner: "maria", Repo: "notas", Branch: "main", StateFile: filepath.Join(dir, "state.json")}
	good := "0123456789abcdef0123456789abcdef01234567"
	write := func(files map[string]fileState) {
		data, _ := json.Marshal(state{Version: stateVersion, Remote: s.remote().ID(), Commit: "c", Files: files})
		if err := os.WriteFile(s.StateFile, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(map[string]fileState{"Notas/a.md": {SHA: good, Size: 3}})
	if st, err := s.loadState(); err != nil || st == nil || len(st.Files) != 1 {
		t.Fatalf("valid state refused: %v %v", st, err)
	}
	for name, files := range map[string]map[string]fileState{
		"dotdot":    {"../fora.md": {SHA: good}},
		"absolute":  {"/etc/passwd": {SHA: good}},
		"backslash": {"a\\b.md": {SHA: good}},
		"empty seg": {"a//b.md": {SHA: good}},
		"short sha": {"a.md": {SHA: "abc"}},
		"upper sha": {"a.md": {SHA: "0123456789ABCDEF0123456789abcdef01234567"}},
		"neg size":  {"a.md": {SHA: good, Size: -1}},
		"neg mtime": {"a.md": {SHA: good, MTime: -5}},
		"nul":       {"a\x00.md": {SHA: good}},
	} {
		write(files)
		if st, err := s.loadState(); err != nil || st != nil {
			t.Errorf("%s: state trusted (%v, %v)", name, st, err)
		}
	}
}
