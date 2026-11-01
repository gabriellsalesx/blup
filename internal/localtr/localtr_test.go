package localtr

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestServeOnlyItsFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, Engine.Name), []byte("wasm"), 0o644); err != nil {
		t.Fatal(err)
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(299) })
	h := Serve(func() string { return dir }, next)
	for path, want := range map[string]int{
		"/local-translate/" + Engine.Name:   200,
		"/local-translate/../../etc/passwd": 404,
		"/local-translate/ready.json":       404,
		"/index.html":                       299,
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://wails.localhost"+path, nil))
		if rec.Code != want {
			t.Errorf("%s: %d, want %d", path, rec.Code, want)
		}
	}
	if Installed(dir) {
		t.Error("an incomplete folder is not an installation")
	}
	if DownloadSize() < 90<<20 || DownloadSize() > 110<<20 {
		t.Errorf("download size %d", DownloadSize())
	}
}

// TestInstall downloads the real files (BLUP_LOCALTR_DIR: where to).
func TestInstall(t *testing.T) {
	dir := os.Getenv("BLUP_LOCALTR_DIR")
	if dir == "" {
		t.Skip("set BLUP_LOCALTR_DIR to download the local translation")
	}
	if err := Install(t.Context(), dir, nil); err != nil {
		t.Fatal(err)
	}
	if !Installed(dir) {
		t.Fatal("not installed after Install")
	}
}
