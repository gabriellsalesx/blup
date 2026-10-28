package notesync

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gabriellsalesx/blup/internal/deploy/github"
	"golang.org/x/text/unicode/norm"
)

var fixedNow = time.Date(2026, 10, 28, 14, 30, 0, 0, time.Local)

type computer struct {
	dir string
	s   *Syncer
}

func newComputer(t *testing.T, f *fakeGH, repo string) *computer {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "vault", "site")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return &computer{dir: dir, s: &Syncer{
		GH: f.client(), Owner: "maria", Repo: repo, Dir: dir,
		StateFile: filepath.Join(root, "site", ".blup", "sync-state.json"),
		Now:       func() time.Time { return fixedNow },
	}}
}

func (c *computer) write(t *testing.T, files map[string]string) {
	t.Helper()
	for p, content := range files {
		full := filepath.Join(c.dir, filepath.FromSlash(p))
		if content == "" {
			if err := os.Remove(full); err != nil {
				t.Fatal(err)
			}
			continue
		}
		os.MkdirAll(filepath.Dir(full), 0o755)
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// files returns every file under dir (including dot files).
func (c *computer) files(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	filepath.WalkDir(c.dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			t.Fatal(err)
		}
		if d.Type().IsRegular() {
			b, _ := os.ReadFile(p)
			rel, _ := filepath.Rel(c.dir, p)
			out[filepath.ToSlash(rel)] = string(b)
		}
		return nil
	})
	return out
}

func (c *computer) sync(t *testing.T) Report {
	t.Helper()
	var phases []string
	rep, err := c.s.Sync(context.Background(), func(p Progress) {
		if len(phases) == 0 || phases[len(phases)-1] != p.Phase {
			phases = append(phases, p.Phase)
		}
	})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if phases[len(phases)-1] != PhaseDone {
		t.Fatalf("phases = %v", phases)
	}
	return rep
}

func equalMaps(t *testing.T, what string, got, want map[string]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: got %v, want %v", what, got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("%s: %q = %q, want %q (all: %v)", what, k, got[k], v, got)
		}
	}
}

func eq(t *testing.T, what string, got []string, want ...string) {
	t.Helper()
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if !slices.Equal(got, want) {
		t.Fatalf("%s = %q, want %q", what, got, want)
	}
}

func setup(t *testing.T) (*fakeGH, context.Context) {
	f := newFakeGH(t)
	ctx := context.Background()
	r, err := EnsureRepo(ctx, f.client(), "", "notas")
	if err != nil || !r.Created || !r.Private {
		t.Fatalf("EnsureRepo = %+v, %v", r, err)
	}
	return f, ctx
}

func TestFirstUploadIgnoresDotFilesAndKeepsUnicode(t *testing.T) {
	f, ctx := setup(t)
	a := newComputer(t, f, "notas")
	nfd := norm.NFD.String("Diário/ação.md") // as macOS may store it
	a.write(t, map[string]string{
		"index.md":            "# Home\n",
		"Notas/reunião 1.md":  "olá, mundo\n",
		nfd:                   "acentos\n",
		"anexos/foto.png":     "\x89PNG\x00\x01binary",
		"vazio.md":            "x",
		".obsidian/app.json":  "{}",
		".trash/old.md":       "old",
		".DS_Store":           "junk",
		"Notas/.hidden.md":    "secret",
		"Notas/.git/config":   "[core]",
		"emoji 🎉/日本語 note.md": "unicode\n",
	})

	st, err := a.s.Status(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Uploaded) != 6 || st.UpToDate || f.count("POST") != 1 || f.count("PUT") != 0 {
		t.Fatalf("status = %+v, POST=%d", st, f.count("POST"))
	}

	rep := a.sync(t)
	eq(t, "uploaded", rep.Uploaded, "Diário/ação.md", "Notas/reunião 1.md", "anexos/foto.png", "emoji 🎉/日本語 note.md", "index.md", "vazio.md")
	if rep.UpToDate || rep.Commit == "" || len(rep.Downloaded) != 0 {
		t.Fatalf("report = %+v", rep)
	}
	remote := f.files("notas")
	if _, ok := remote[InitFile.Path]; !ok {
		t.Fatal("init file missing")
	}
	delete(remote, InitFile.Path)
	equalMaps(t, "remote", remote, map[string]string{
		"index.md": "# Home\n", "Notas/reunião 1.md": "olá, mundo\n", "Diário/ação.md": "acentos\n",
		"anexos/foto.png": "\x89PNG\x00\x01binary", "vazio.md": "x", "emoji 🎉/日本語 note.md": "unicode\n",
	})
	// The NFD local name was normalized to NFC remotely.
	if _, ok := f.files("notas")[norm.NFC.String("Diário/ação.md")]; !ok {
		t.Fatal("expected NFC path on GitHub")
	}

	// State file format.
	var s state
	b, err := os.ReadFile(a.s.StateFile)
	if err != nil || json.Unmarshal(b, &s) != nil {
		t.Fatal("state unreadable", err)
	}
	if s.Version != 1 || s.Commit != rep.Commit || len(s.Files) != 6 || s.Files["index.md"].SHA != github.BlobSHA([]byte("# Home\n")) {
		t.Fatalf("state = %+v", s)
	}

	// Second run: nothing to do, no commit, the init file is not downloaded.
	commits := f.commitCount("notas")
	rep = a.sync(t)
	if !rep.UpToDate || f.commitCount("notas") != commits {
		t.Fatalf("second sync = %+v", rep)
	}
	if _, ok := a.files(t)[".blup/README.md"]; ok {
		t.Fatal("dot file downloaded")
	}
	if !strings.HasPrefix(f.repos["notas"].commits[rep.Commit].message, "blup sync: 0 updated, 6 added, 0 removed") {
		t.Fatalf("message = %q", f.repos["notas"].commits[rep.Commit].message)
	}
}

func TestSecondComputerDownloadsAndConflicts(t *testing.T) {
	f, _ := setup(t)
	a := newComputer(t, f, "notas")
	a.write(t, map[string]string{"a.md": "from A\n", "b/c.md": "shared\n", "same.md": "same\n"})
	a.sync(t)

	b := newComputer(t, f, "notas")
	os.RemoveAll(b.dir) // brand new computer: folder does not exist yet
	b.write(t, map[string]string{"a.md": "from B\n", "same.md": "same\n", "only-b.md": "B\n"})
	rep := b.sync(t)
	eq(t, "downloaded", rep.Downloaded, "b/c.md")
	eq(t, "uploaded", rep.Uploaded, "only-b.md")
	eq(t, "conflicts", rep.Conflicts, "a.md")
	copyName := "a (conflict 2026-10-28 1430).md"
	eq(t, "copies", rep.ConflictCopies, copyName)
	equalMaps(t, "B", b.files(t), map[string]string{
		"a.md": "from B\n", copyName: "from A\n", "b/c.md": "shared\n", "same.md": "same\n", "only-b.md": "B\n",
	})
	remote := f.files("notas")
	delete(remote, InitFile.Path)
	equalMaps(t, "remote", remote, b.files(t))
	if msg := f.repos["notas"].commits[rep.Commit].message; msg != "blup sync: 1 updated, 2 added, 0 removed" {
		t.Fatalf("message = %q", msg)
	}

	// A receives everything; no new conflict.
	rep = a.sync(t)
	eq(t, "A downloaded", rep.Downloaded, copyName, "a.md", "only-b.md")
	if len(rep.Conflicts) != 0 || len(rep.Uploaded) != 0 {
		t.Fatalf("A report = %+v", rep)
	}
	equalMaps(t, "A", a.files(t), b.files(t))
	if !b.sync(t).UpToDate || !a.sync(t).UpToDate {
		t.Fatal("expected both up to date")
	}
}

func TestEditsBothWays(t *testing.T) {
	f, _ := setup(t)
	a := newComputer(t, f, "notas")
	a.write(t, map[string]string{"x.md": "x0", "y.md": "y0", "z.md": "z0", "w.md": "w0"})
	a.sync(t)
	b := newComputer(t, f, "notas")
	b.sync(t)

	a.write(t, map[string]string{"x.md": "x1", "z.md": "z-A", "w.md": "same edit"})
	b.write(t, map[string]string{"y.md": "y1", "z.md": "z-B", "w.md": "same edit"})
	ra := a.sync(t)
	eq(t, "A up", ra.Uploaded, "w.md", "x.md", "z.md")
	rb := b.sync(t)
	eq(t, "B up", rb.Uploaded, "y.md")
	eq(t, "B down", rb.Downloaded, "x.md") // w.md: same content both sides
	eq(t, "B conflicts", rb.Conflicts, "z.md")
	ra = a.sync(t)
	eq(t, "A down", ra.Downloaded, "y.md", "z (conflict 2026-10-28 1430).md", "z.md")
	want := map[string]string{"x.md": "x1", "y.md": "y1", "z.md": "z-B", "z (conflict 2026-10-28 1430).md": "z-A", "w.md": "same edit"}
	equalMaps(t, "A", a.files(t), want)
	equalMaps(t, "B", b.files(t), want)

	// A second conflict on the same note in the same minute gets a number.
	a.write(t, map[string]string{"z.md": "z-A2"})
	b.write(t, map[string]string{"z.md": "z-B2"})
	a.sync(t)
	rb = b.sync(t)
	eq(t, "copies", rb.ConflictCopies, "z (conflict 2026-10-28 1430 2).md")
}

func TestDeletionsBothWaysAndTrash(t *testing.T) {
	f, _ := setup(t)
	a := newComputer(t, f, "notas")
	a.write(t, map[string]string{"keep.md": "k", "gone/p.md": "p", "q.md": "q", "r.md": "r"})
	a.sync(t)
	b := newComputer(t, f, "notas")
	b.sync(t)

	// A deletes p.md: removed on GitHub, then moved to B's trash.
	a.write(t, map[string]string{"gone/p.md": ""})
	ra := a.sync(t)
	eq(t, "A deleted", ra.Deleted, "gone/p.md")
	if _, ok := f.files("notas")["gone/p.md"]; ok {
		t.Fatal("not deleted on GitHub")
	}
	rb := b.sync(t)
	eq(t, "B deleted local", rb.DeletedLocal, "gone/p.md")
	if _, err := os.Stat(filepath.Join(b.dir, "gone")); !os.IsNotExist(err) {
		t.Fatal("empty folder left behind")
	}
	if got, err := os.ReadFile(filepath.Join(rb.Trash, "gone", "p.md")); err != nil || string(got) != "p" {
		t.Fatalf("trash: %q %v (trash %s)", got, err, rb.Trash)
	}
	if strings.HasPrefix(rb.Trash, filepath.Dir(b.dir)) {
		t.Fatal("trash must be outside the vault")
	}

	// Deleted on one side, edited on the other: the edit wins.
	a.write(t, map[string]string{"q.md": "q edited"}) // A edits q
	b.write(t, map[string]string{"q.md": ""})         // B deletes q
	b.write(t, map[string]string{"r.md": "r edited"}) // B edits r
	a.write(t, map[string]string{"r.md": ""})         // A deletes r
	a.sync(t)                                         // q edited up, r deleted remotely
	rb = b.sync(t)
	eq(t, "B down", rb.Downloaded, "q.md")
	eq(t, "B up", rb.Uploaded, "r.md")
	a.sync(t)
	want := map[string]string{"keep.md": "k", "q.md": "q edited", "r.md": "r edited"}
	equalMaps(t, "A", a.files(t), want)
	equalMaps(t, "B", b.files(t), want)
}

func TestUntrackedFilesNeverDeleted(t *testing.T) {
	f, _ := setup(t)
	a := newComputer(t, f, "notas")
	a.write(t, map[string]string{"x.md": "x"})
	a.sync(t)
	b := newComputer(t, f, "notas")
	b.sync(t)

	a.write(t, map[string]string{"x.md": ""})
	a.sync(t) // x deleted on GitHub
	// B, before syncing, loses its state (e.g. new install) but has x.md
	// and an untracked file with the same name as nothing remote.
	os.Remove(b.s.StateFile)
	b.write(t, map[string]string{"untracked.md": "u"})
	rb := b.sync(t)
	if len(rb.DeletedLocal) != 0 {
		t.Fatalf("deleted untracked: %+v", rb)
	}
	eq(t, "B up", rb.Uploaded, "untracked.md", "x.md")
	equalMaps(t, "B", b.files(t), map[string]string{"x.md": "x", "untracked.md": "u"})
}

func TestRefMovedRetries(t *testing.T) {
	f, _ := setup(t)
	a := newComputer(t, f, "notas")
	a.write(t, map[string]string{"a.md": "a"})
	a.sync(t)

	a.write(t, map[string]string{"mine.md": "mine"})
	patches := f.count("PATCH")
	moved := 0
	f.hook = func(w http.ResponseWriter, r *http.Request, body map[string]any) bool {
		if r.Method == http.MethodPatch && moved == 0 {
			moved++
			f.pushLocked(f.repos["notas"], map[string]string{"theirs.md": "theirs"})
		}
		return false
	}
	rep := a.sync(t)
	eq(t, "up", rep.Uploaded, "mine.md")
	eq(t, "down", rep.Downloaded, "theirs.md")
	remote := f.files("notas")
	if remote["mine.md"] != "mine" || remote["theirs.md"] != "theirs" || f.count("PATCH")-patches != 2 {
		t.Fatalf("remote = %v, patches = %d", remote, f.count("PATCH"))
	}

	// Always moving: ErrRemoteMoved and nothing changed locally.
	a.write(t, map[string]string{"again.md": "again"})
	n := 0
	f.hook = func(w http.ResponseWriter, r *http.Request, body map[string]any) bool {
		if r.Method == http.MethodPatch {
			n++
			f.pushLocked(f.repos["notas"], map[string]string{"busy.md": strings.Repeat("b", n)})
		}
		return false
	}
	a.s.MaxAttempts = 3
	before := a.files(t)
	_, err := a.s.Sync(context.Background(), nil)
	if !errors.Is(err, ErrRemoteMoved) || n != 3 {
		t.Fatalf("want ErrRemoteMoved after 3 attempts, got %v (%d)", err, n)
	}
	equalMaps(t, "local unchanged", a.files(t), before)
}

func TestStatusAndSafetyGuards(t *testing.T) {
	f, ctx := setup(t)
	a := newComputer(t, f, "notas")
	a.write(t, map[string]string{"a.md": "a", "b.md": "b"})
	a.sync(t)
	f.push("notas", map[string]string{"new.md": "n", "b.md": "b2"})
	a.write(t, map[string]string{"a.md": "a2", "local.md": "l"})

	posts := f.count("POST") + f.count("PATCH")
	st, err := a.s.Status(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	eq(t, "to send", st.Uploaded, "a.md", "local.md")
	eq(t, "to receive", st.Downloaded, "b.md", "new.md")
	if st.UpToDate || f.count("POST")+f.count("PATCH") != posts {
		t.Fatalf("status wrote or wrong: %+v", st)
	}
	if a.files(t)["b.md"] != "b" {
		t.Fatal("status changed files")
	}

	// Empty folder after a sync: refuse to wipe GitHub.
	a.sync(t)
	os.RemoveAll(a.dir)
	os.MkdirAll(a.dir, 0o755)
	if _, err := a.s.Sync(ctx, nil); !errors.Is(err, ErrLocalFolderEmpty) {
		t.Fatalf("want ErrLocalFolderEmpty, got %v", err)
	}
	// Missing folder with state: error, not a mass deletion.
	os.RemoveAll(a.dir)
	if _, err := a.s.Sync(ctx, nil); err == nil {
		t.Fatal("expected error for missing folder")
	}

	// Repo errors are github sentinels.
	c := newComputer(t, f, "nope")
	if _, err := c.s.Status(ctx, nil); err != nil {
		t.Fatalf("missing branch status: %v", err)
	}
	c.write(t, map[string]string{"x.md": "x"})
	if _, err := c.s.Sync(ctx, nil); !errors.Is(err, github.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	c.s.GH.Token = "bad"
	if _, err := c.s.Status(ctx, nil); !errors.Is(err, github.ErrBadToken) {
		t.Fatalf("want ErrBadToken, got %v", err)
	}
}

func TestLargeFilesSkipped(t *testing.T) {
	f, _ := setup(t)
	a := newComputer(t, f, "notas")
	a.write(t, map[string]string{"small.md": "s"})
	big := filepath.Join(a.dir, "video.mp4")
	fh, err := os.Create(big)
	if err != nil {
		t.Fatal(err)
	}
	fh.Truncate(github.MaxFileSize + 1) // sparse: cheap
	fh.Close()
	rep := a.sync(t)
	eq(t, "up", rep.Uploaded, "small.md")
	eq(t, "skipped", rep.Skipped, "video.mp4")
	if rep.SkipReasons["video.mp4"] != SkipTooLarge {
		t.Fatalf("reason = %v", rep.SkipReasons)
	}
	if _, ok := f.files("notas")["video.mp4"]; ok {
		t.Fatal("large file uploaded")
	}
}

func TestDownloadKeepsFileModeAndHashCache(t *testing.T) {
	f, _ := setup(t)
	a := newComputer(t, f, "notas")
	a.write(t, map[string]string{"run.sh": "v1"})
	os.Chmod(filepath.Join(a.dir, "run.sh"), 0o600)
	a.sync(t)
	f.push("notas", map[string]string{"run.sh": "v2"})
	a.sync(t)
	info, err := os.Stat(filepath.Join(a.dir, "run.sh"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, %v", info.Mode(), err)
	}
	if b, _ := os.ReadFile(filepath.Join(a.dir, "run.sh")); string(b) != "v2" {
		t.Fatalf("content = %q", b)
	}
	// No temp files left behind.
	for p := range a.files(t) {
		if strings.Contains(p, ".blup-tmp") {
			t.Fatal("temp file left:", p)
		}
	}
	// Old file: its size/mtime are cached in the state.
	old := filepath.Join(a.dir, "old.md")
	os.WriteFile(old, []byte("o"), 0o644)
	past := time.Now().Add(-time.Hour)
	os.Chtimes(old, past, past)
	a.sync(t)
	var s state
	b, _ := os.ReadFile(a.s.StateFile)
	json.Unmarshal(b, &s)
	if s.Files["old.md"].MTime == 0 || s.Files["run.sh"].MTime != 0 {
		t.Fatalf("cache = %+v", s.Files)
	}
}
