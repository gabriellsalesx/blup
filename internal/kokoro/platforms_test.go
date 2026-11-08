package kokoro

import (
	"archive/zip"
	"bytes"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var hexSum = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Every system Blup ships the Studio for has an entry, each download is
// pinned or (only the tools, before their release) clearly not, and an
// unpinned entry never counts as supported.
func TestPlatforms(t *testing.T) {
	for _, key := range []string{"linux/amd64", "linux/arm64", "darwin/arm64", "darwin/amd64", "windows/amd64"} {
		p, ok := platforms[key]
		if !ok {
			t.Errorf("%s: missing", key)
			continue
		}
		archive := path.Base(p.ort.url)
		if !strings.HasPrefix(p.ort.url, "https://github.com/microsoft/onnxruntime/releases/download/") ||
			!hexSum.MatchString(p.ort.sha256) || p.ort.size <= 0 {
			t.Errorf("%s: ONNX Runtime not pinned: %+v", key, p.ort)
		}
		top := strings.TrimSuffix(strings.TrimSuffix(archive, ".tgz"), ".zip")
		if !strings.HasPrefix(p.ortLib, top+"/lib/") || p.ortLicense != top+"/LICENSE" {
			t.Errorf("%s: library %q or license %q outside %s", key, p.ortLib, p.ortLicense, top)
		}
		goos, goarch, _ := strings.Cut(key, "/")
		if want := toolsRelease + "blup-voice-tools-" + goos + "-" + goarch + ".tar.gz"; p.tools.url != want {
			t.Errorf("%s: tools at %q, want %q", key, p.tools.url, want)
		}
		switch {
		case hexSum.MatchString(p.tools.sha256) && p.tools.size > 0:
			if !p.ready() {
				t.Errorf("%s: pinned but not ready", key)
			}
		case p.tools.sha256 == "" && p.tools.size == 0:
			if p.ready() {
				t.Errorf("%s: tools without a checksum must not be ready", key)
			}
		default:
			t.Errorf("%s: tools half pinned: %+v", key, p.tools)
		}
	}
	if (platform{}).ready() {
		t.Error("an empty platform is not ready")
	}
}

func zipOf(t *testing.T, files map[string]string, dirs ...string) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, d := range dirs {
		if _, err := zw.Create(d); err != nil {
			t.Fatal(err)
		}
	}
	for name, body := range files {
		h := &zip.FileHeader{Name: name, Method: zip.Deflate}
		h.SetMode(0o755)
		w, err := zw.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// ONNX Runtime for Windows comes as a .zip: only the named files come out.
func TestExtractZip(t *testing.T) {
	top := "onnxruntime-win-x64-1.31.1"
	data := zipOf(t, map[string]string{
		top + "/lib/onnxruntime.dll": "dll",
		top + "/lib/onnxruntime.pdb": "a large file nobody needs",
		top + "/LICENSE":             "MIT",
	}, top+"/", top+"/lib/")
	dir := t.TempDir()
	lib, lic := filepath.Join(dir, "ort", "onnxruntime.dll"), filepath.Join(dir, "ort", "LICENSE")
	if err := extract(data, "x/"+top+".zip", map[string]string{top + "/lib/onnxruntime.dll": lib, top + "/LICENSE": lic}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(lib); string(b) != "dll" {
		t.Errorf("library: %q", b)
	}
	if b, _ := os.ReadFile(lic); string(b) != "MIT" {
		t.Errorf("license: %q", b)
	}
	if entries, _ := os.ReadDir(filepath.Join(dir, "ort")); len(entries) != 2 {
		t.Errorf("%d files out, want 2", len(entries))
	}

	// A file the archive lacks is an error, not a silent gap.
	if err := extract(data, "o.zip", map[string]string{top + "/lib/missing.dll": filepath.Join(dir, "m")}); err == nil {
		t.Error("missing file accepted")
	}
	// Everything into a folder, but nothing out of it.
	all := filepath.Join(dir, "all")
	if err := extract(zipOf(t, map[string]string{"a/b.txt": "b", "c.exe": "c"}), "t.zip", map[string]string{"*": all}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(all, "a", "b.txt")); err != nil {
		t.Error(err)
	}
	evil := zipOf(t, map[string]string{"../../evil.txt": "x"})
	if err := extract(evil, "t.zip", map[string]string{"*": filepath.Join(dir, "safe")}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "evil.txt")); err == nil {
		t.Error("a path with .. left the folder")
	}
	if err := extract([]byte("not a zip"), "t.zip", map[string]string{"*": all}); err == nil {
		t.Error("garbage accepted as a zip")
	}
}
