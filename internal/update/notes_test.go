package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

const trilingual = `## Português

### Blup Notes (celular)
- Pronto para a Google Play.

## English

### Blup Notes (phone)
- Ready for Google Play.

## Español

### Blup Notes (móvil)
- Listo para Google Play.

Os arquivos estão listados abaixo com ` + "`studio-checksums.txt`" + ` e a assinatura. / The files are listed below with ` + "`studio-checksums.txt`" + `.`

func TestNotesFor(t *testing.T) {
	cases := []struct{ notes, lang, want string }{
		{trilingual, "pt", "### Blup Notes (celular)\n- Pronto para a Google Play."},
		{trilingual, "en", "### Blup Notes (phone)\n- Ready for Google Play."},
		{trilingual, "es", "### Blup Notes (móvil)\n- Listo para Google Play."},
		{trilingual, "fr", "### Blup Notes (phone)\n- Ready for Google Play."},
		// Older notes, without Spanish: English stands in.
		{"## Português\n- Algo.\n\n## English\n- Something.", "es", "- Something."},
		// No sections: as they are, without the checksums line.
		{"- One fix.\nThe files with `checksums.txt`.", "pt", "- One fix."},
	}
	for _, c := range cases {
		if got := NotesFor(c.notes, c.lang); got != c.want {
			t.Errorf("NotesFor(%q) = %q, want %q", c.lang, got, c.want)
		}
	}
}

func TestSince(t *testing.T) {
	h := []VersionNotes{{"1.1.0", "## English\n- c"}, {"1.0.9", "## English\n- b"}, {"1.0.8", "## English\n- a"}, {"1.0.7", "## English\n- old"}}
	got := Since(h, "1.0.7", "1.1.0", "en")
	want := []VersionNotes{{"1.1.0", "- c"}, {"1.0.9", "- b"}, {"1.0.8", "- a"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
	// Nothing past the latest the Studio was told about.
	if got := Since(h, "1.0.8", "1.0.9", "en"); len(got) != 1 || got[0].Version != "1.0.9" {
		t.Fatalf("got %v", got)
	}
	if got := Since(h, "1.1.0", "1.1.0", "en"); len(got) != 0 {
		t.Fatalf("up to date: got %v", got)
	}
}

func TestReleases(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/"+Repo+"/releases" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(`[
			{"tag_name":"v1.0.8","body":"a","html_url":"u8"},
			{"tag_name":"voice-tools-2","body":"tools","prerelease":true},
			{"tag_name":"v1.0.10","body":"c","html_url":"u10"},
			{"tag_name":"v1.1.0-rc1","body":"rc","prerelease":true},
			{"tag_name":"v1.0.9","body":"b","draft":true},
			{"tag_name":"v1.0.9","body":"b","html_url":"u9"}
		]`))
	}))
	defer srv.Close()
	rels, err := (&Client{BaseURL: srv.URL}).Releases(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, r := range rels {
		got = append(got, r.Version)
	}
	if !reflect.DeepEqual(got, []string{"1.0.10", "1.0.9", "1.0.8"}) {
		t.Fatalf("got %v", got)
	}
}

// Should GitHub's "latest" be a Blup Notes release, the computer's newest
// version is the latest, with its files.
func TestLatestSkipsNotesRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/" + Repo + "/releases/latest":
			w.Write([]byte(`{"tag_name":"notes-v1.1.1","body":"phone","assets":[{"name":"blup-notes_1.1.1_android.apk","browser_download_url":"https://x/apk"}]}`))
		case "/repos/" + Repo + "/releases":
			w.Write([]byte(`[{"tag_name":"notes-v1.1.1","body":"phone"},{"tag_name":"v1.1.0","body":"computer","assets":[{"name":"checksums.txt","browser_download_url":"https://x/sums"}]}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	rel, err := (&Client{BaseURL: srv.URL}).Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rel.Version != "1.1.0" || rel.Notes != "computer" || rel.assets["checksums.txt"] != "https://x/sums" {
		t.Fatalf("got %+v", rel)
	}
}
