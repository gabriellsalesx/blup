package app

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestRemoveNarrations(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	site := settingsSite(t)
	site.Translations = []string{"en"}
	root := site.ContentDir()
	att := attachmentsFolder(site)
	write(t, root, att+"/Um.mp3", "studio")
	write(t, root, att+"/Um (en).mp3", "studio en")
	write(t, root, att+"/minha.mp3", "own")
	write(t, root, att+"/junto.mp3", "shared")
	write(t, root, "Posts/Um.md", "---\ntitle: Um\naudio: '[[Um.mp3]]'\naudio_hash: h1\naudio_seconds: 30\n---\ntexto")
	write(t, root, "Posts/en/Um.md", "---\ntitle: One\ntranslationKey: Posts/Um\naudio: '[[Um (en).mp3]]'\naudio_hash: h2\naudio_seconds: 31\n---\ntext")
	// The person's own recording (no audio_hash): never touched.
	write(t, root, "Posts/Meu.md", "---\ntitle: Meu\naudio: '[[minha.mp3]]'\n---\ntexto")
	// A narration whose file another (own) note links too: the file stays.
	write(t, root, "Posts/Junto.md", "---\ntitle: Junto\naudio: '[[junto.mp3]]'\naudio_hash: h3\n---\ntexto")
	write(t, root, "Posts/Outro.md", "---\ntitle: Outro\naudio: '[[junto.mp3]]'\n---\ntexto")

	n, langs, err := NarrationsMade(site)
	if err != nil || n != 3 || !slices.Equal(langs, []string{"pt", "en"}) {
		t.Fatalf("made %d %v %v", n, langs, err)
	}
	res, err := RemoveNarrations(site)
	if err != nil || res.Notes != 3 || !slices.Equal(res.Langs, []string{"pt", "en"}) {
		t.Fatalf("removed %+v, %v", res, err)
	}
	for _, rel := range []string{"Posts/Um.md", "Posts/en/Um.md", "Posts/Junto.md"} {
		n, _ := ReadSiteNote(site, rel)
		for _, k := range narrationKeys {
			if n.Props[k] != nil {
				t.Fatalf("%s keeps %s", rel, k)
			}
		}
		if n.Body != "texto" && n.Body != "text" {
			t.Fatalf("%s body %q", rel, n.Body)
		}
	}
	if n, _ := ReadSiteNote(site, "Posts/Meu.md"); n.String("audio") != "[[minha.mp3]]" {
		t.Fatal("an own audio keeps its link")
	}
	for _, f := range []string{"minha.mp3", "junto.mp3"} {
		if _, err := os.Stat(filepath.Join(root, att, f)); err != nil {
			t.Fatalf("%s must stay: %v", f, err)
		}
	}
	for _, f := range []string{"Um.mp3", "Um (en).mp3"} {
		if _, err := os.Stat(filepath.Join(root, att, f)); !os.IsNotExist(err) {
			t.Fatalf("%s must leave the attachments", f)
		}
		if _, err := os.Stat(filepath.Join(res.Trash, f)); err != nil {
			t.Fatalf("%s must be in the trash: %v", f, err)
		}
	}
	// One item in the trash, the folder of the narrations.
	items, err := ListTrash(site)
	if err != nil || len(items) != 1 || !items[0].Dir || items[0].Path != res.Trash {
		t.Fatalf("trash %+v %v", items, err)
	}
	// Undo: the files and the properties come back.
	if err := RestoreNarrations(site, res.Trash); err != nil {
		t.Fatal(err)
	}
	if n, _ := ReadSiteNote(site, "Posts/en/Um.md"); n.String("audio") != "[[Um (en).mp3]]" || n.String("audio_hash") != "h2" || n.String("audio_seconds") != "31" {
		t.Fatalf("restored props %v", n.Props)
	}
	if _, err := os.Stat(filepath.Join(root, att, "Um.mp3")); err != nil {
		t.Fatal("the MP3 comes back")
	}
	if items, _ := ListTrash(site); len(items) != 0 {
		t.Fatalf("the trash is empty again: %+v", items)
	}
	// Nothing made by the Studio: nothing to do.
	write(t, root, "Posts/Um.md", "---\ntitle: Um\n---\ntexto")
	write(t, root, "Posts/en/Um.md", "---\ntitle: One\n---\ntext")
	write(t, root, "Posts/Junto.md", "---\ntitle: Junto\n---\ntexto")
	if res, err := RemoveNarrations(site); err != nil || res.Notes != 0 {
		t.Fatalf("nothing: %+v %v", res, err)
	}
}
