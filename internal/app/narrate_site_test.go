package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/narrate"
	"github.com/gabriellsalesx/blup/internal/obsidian"
)

const narrationBody = "Um texto com palavras suficientes para virar uma narração de verdade."
const narrationBodyEN = "A text with words enough to become a real narration."

// podcastSite is a blog with posts in every situation the batch meets.
func podcastSite(t *testing.T) *config.Site {
	t.Helper()
	site := settingsSite(t)
	site.Translations = []string{"en"}
	root := site.ContentDir()
	tr := func(name, extra string) {
		write(t, root, "Posts/en/"+name+".md", "---\ntitle: "+name+"\ntranslationKey: Posts/"+name+"\n"+extra+"---\n"+narrationBodyEN)
	}
	write(t, root, "Posts/Longo.md", "---\ntitle: Longo\n---\n"+narrationBody)
	tr("Longo", "")
	write(t, root, "Posts/Velho.md", "---\ntitle: Velho\naudio: '[[Velho.mp3]]'\naudio_hash: abc\n---\n"+narrationBody)
	write(t, root, "Posts/Pronto.md", "---\ntitle: Pronto\n---\n"+narrationBody)
	hash, err := NarrationHash(site, "Posts/Pronto.md")
	if err != nil {
		t.Fatal(err)
	}
	write(t, root, attachmentsFolder(site)+"/Pronto.mp3", "mp3")
	if err := obsidian.SetProperties(filepath.Join(root, "Posts", "Pronto.md"), obsidian.Prop{Key: "audio", Value: "[[Pronto.mp3]]"}, obsidian.Prop{Key: "audio_hash", Value: hash}); err != nil {
		t.Fatal(err)
	}
	// The person's own recording: linked, without the Studio's audio_hash.
	write(t, root, attachmentsFolder(site)+"/minha.mp3", "my own")
	write(t, root, "Posts/Proprio.md", "---\ntitle: Proprio\naudio: '[[minha.mp3]]'\n---\n"+narrationBody)
	// Passwords: their own, and the site's general one (yes, true, sim).
	for name, pw := range map[string]string{"Senha": "segredo", "SenhaSim": "sim", "SenhaTrue": "true", "SenhaYes": "yes"} {
		write(t, root, "Posts/"+name+".md", "---\ntitle: "+name+"\npassword: "+pw+"\n---\n"+narrationBody)
	}
	tr("Senha", "")
	// Not posts, and translations of originals that are not posts.
	write(t, root, "Posts/Rascunho.md", "---\ndraft: true\n---\n"+narrationBody)
	tr("Rascunho", "")
	write(t, root, "Posts/Oculto.md", "---\nhidden: true\n---\n"+narrationBody)
	write(t, root, "Posts/Fora.md", "---\npublish: false\n---\n"+narrationBody)
	write(t, root, "Posts/Particular.md", "---\nprivate: sim\n---\n"+narrationBody)
	tr("Particular", "")
	write(t, root, "Privacidade.md", "---\nblup: privacy\n---\n"+narrationBody)
	write(t, root, "Autores/_index.md", "---\nblup: authors\n---\n")
	write(t, root, "Autores/Ana.md", "---\nrole: Editora\n---\n"+narrationBody)
	write(t, root, "Podcast/_index.md", "---\nblup: podcast\n---\n"+narrationBody)
	write(t, root, "Podcast/Extra.md", narrationBody)
	// The vault's note templates (Obsidian's Templates plugin).
	write(t, site.Vault, ".obsidian/templates.json", `{"folder": "`+site.Folder+`/Modelos"}`)
	write(t, root, "Modelos/Post do blog.md", "---\ntitle: \"{{title}}\"\n---\n"+narrationBody)
	return site
}

func planRels(plan NarrationPlan) []string {
	var rels []string
	for _, it := range plan.Items {
		rels = append(rels, it.Rel)
	}
	slices.Sort(rels)
	return rels
}

func TestNarrationTodo(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	site := podcastSite(t)
	plan, err := NarrationTodo(site, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Posts/Longo.md", "Posts/Velho.md", "Posts/en/Longo.md"}; !slices.Equal(planRels(plan), want) {
		t.Fatalf("items %v, want %v", planRels(plan), want)
	}
	if plan.Missing != 1 || plan.Outdated != 1 || plan.Translations["en"] != 1 || plan.Locked != 4 || plan.OwnAudio != 1 || plan.Total != 3 {
		t.Fatalf("plan %+v", plan)
	}
	for _, it := range plan.Items {
		if it.Rel == "Posts/en/Longo.md" && it.Lang != "en" || it.Rel != "Posts/en/Longo.md" && it.Lang != "pt" {
			t.Fatalf("%s read in %s", it.Rel, it.Lang)
		}
	}
	if len(plan.Voices) != 2 || plan.Voices[0].Lang != "pt" || plan.Voices[1].Lang != "en" || plan.Voices[0].Engine != "local" || plan.Voices[0].Notes != 2 {
		t.Fatalf("voices %+v", plan.Voices)
	}
	// No voice in the cloud: nothing to ask Blup Cloud about.
	if c, err := NarrationPlanCost(context.Background(), site, plan, map[string]string{}); c != nil || err != nil {
		t.Fatalf("cost %v, %v", c, err)
	}
}

// One note's narration refuses a translation whose original has a
// password or is not published.
func TestNarrateRefusesTranslationOfLockedOrHiddenOriginal(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	site := podcastSite(t)
	for _, rel := range []string{"Posts/en/Senha.md", "Posts/en/Rascunho.md", "Posts/en/Particular.md"} {
		if _, err := Narrate(context.Background(), site, rel, "", nil); err == nil {
			t.Fatalf("%s must not be narrated", rel)
		}
	}
	if g := originGate(site, "Posts/en/Senha.md"); g != locked {
		t.Fatalf("gate %v", g)
	}
	if g := originGate(site, "Posts/en/Rascunho.md"); g != notPost {
		t.Fatalf("gate %v", g)
	}
}

func stubNarration(t *testing.T, fn func(rel string) (Narration, error)) {
	t.Helper()
	narrateNote = func(_ context.Context, _ *config.Site, rel, _ string, _ func(int, int)) (Narration, error) {
		return fn(rel)
	}
	t.Cleanup(func() { narrateNote = Narrate })
}

func TestNarrateSite(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	site := podcastSite(t)
	plan, err := NarrationTodo(site, nil)
	if err != nil {
		t.Fatal(err)
	}
	var read []string
	stubNarration(t, func(rel string) (Narration, error) {
		read = append(read, rel)
		if rel == "Posts/Velho.md" {
			return Narration{}, errors.New("this one failed")
		}
		return Narration{Link: "[[" + filepath.Base(rel) + ".mp3]]", Hash: "h-" + rel, Seconds: 12.6}, nil
	})
	var steps [][2]int
	res, err := NarrateSite(context.Background(), site, plan, nil, func(done, total int, _ string) { steps = append(steps, [2]int{done, total}) })
	if err != nil || res.Done != 2 || len(res.Failed) != 1 || res.Failed[0].Rel != "Posts/Velho.md" || res.Failed[0].Error != "this one failed" {
		t.Fatalf("res %+v, %v", res, err)
	}
	if len(read) != 3 || steps[len(steps)-1] != [2]int{3, 3} {
		t.Fatalf("read %v, steps %v", read, steps)
	}
	n, _ := ReadSiteNote(site, "Posts/en/Longo.md")
	if n.String("audio") != "[[Longo.md.mp3]]" || n.String("audio_hash") != "h-Posts/en/Longo.md" || n.String("audio_seconds") != "13" {
		t.Fatalf("props %v", n.Props)
	}
	if n, _ := ReadSiteNote(site, "Posts/Velho.md"); n.String("audio_hash") != "abc" {
		t.Fatal("a failed note keeps its old narration")
	}
	if n, _ := ReadSiteNote(site, "Posts/Proprio.md"); n.String("audio") != "[[minha.mp3]]" || n.String("audio_hash") != "" {
		t.Fatal("an audio of the person's own is never touched")
	}
}

// Notes checked again right before each is read: one that became a draft,
// got a password, or whose original got one meanwhile is left out; one
// deleted while it was read is not made again.
func TestNarrateSiteChecksEachNoteAgain(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	site := podcastSite(t)
	root := site.ContentDir()
	write(t, root, "Posts/Outro.md", "---\ntitle: Outro\n---\n"+narrationBody)
	write(t, root, "Posts/Sumiu.md", "---\ntitle: Sumiu\n---\n"+narrationBody)
	plan, err := NarrationTodo(site, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(planRels(plan), "Posts/en/Longo.md") || !slices.Contains(planRels(plan), "Posts/Outro.md") {
		t.Fatalf("plan %v", planRels(plan))
	}
	var read []string
	stubNarration(t, func(rel string) (Narration, error) {
		read = append(read, rel)
		if len(read) == 1 {
			// While the first note is read: one turns into a draft, the
			// original of a translation gets a password.
			write(t, root, "Posts/Outro.md", "---\ntitle: Outro\ndraft: true\n---\n"+narrationBody)
			write(t, root, "Posts/Longo.md", "---\ntitle: Longo\npassword: sim\n---\n"+narrationBody)
		}
		if rel == "Posts/Sumiu.md" {
			os.Remove(filepath.Join(root, "Posts", "Sumiu.md"))
			write(t, root, attachmentsFolder(site)+"/Sumiu.mp3", "audio")
			return Narration{Link: "[[Sumiu.mp3]]", Hash: "h"}, nil
		}
		return Narration{Link: "[[x.mp3]]", Hash: "h"}, nil
	})
	res, err := NarrateSite(context.Background(), site, plan, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range read {
		if rel == "Posts/Outro.md" || rel == "Posts/en/Longo.md" {
			t.Fatalf("%s became ineligible and must not be read (read %v)", rel, read)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "Posts", "Sumiu.md")); !os.IsNotExist(err) {
		t.Fatal("a note deleted while read must not come back")
	}
	if _, err := os.Stat(filepath.Join(root, attachmentsFolder(site), "Sumiu.mp3")); !os.IsNotExist(err) {
		t.Fatal("the audio of a deleted note goes too")
	}
	if res.Skipped != 3 || len(res.Failed) != 0 {
		t.Fatalf("res %+v", res)
	}
}

// The same kind of failure note after note is something every note would
// hit: the batch stops. Failures every note hits at once stop it at once.
func TestNarrateSiteStopsOnRepeatedFailure(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	site := settingsSite(t)
	root := site.ContentDir()
	plan := NarrationPlan{}
	for i := range 5 {
		rel := fmt.Sprintf("Posts/n%d.md", i)
		write(t, root, rel, "---\ntitle: n\n---\n"+narrationBody)
		plan.Items = append(plan.Items, NarrationItem{Rel: rel, Lang: "pt"})
	}
	calls := 0
	// Different texts, same kind (the network): three in a row stop it.
	stubNarration(t, func(rel string) (Narration, error) {
		calls++
		return Narration{}, &fakeNetErr{fmt.Sprintf("dial tcp %d: timeout", calls)}
	})
	res, err := NarrateSite(context.Background(), site, plan, nil, nil)
	if err == nil || calls != sameKindStop || len(res.Failed) != sameKindStop {
		t.Fatalf("calls %d, res %+v, %v", calls, res, err)
	}
	// The month's credits used up: the first note stops it.
	calls = 0
	stubNarration(t, func(string) (Narration, error) {
		calls++
		return Narration{}, friendlyErr{"acabou", narrate.ErrNoCredits}
	})
	if _, err := NarrateSite(context.Background(), site, plan, nil, nil); err == nil || calls != 1 {
		t.Fatalf("credits: calls %d, %v", calls, err)
	}
	// Cancelled: stops at once.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NarrateSite(ctx, site, plan, nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled: %v", err)
	}
}

// fakeNetErr is a network error (net.Error) for the tests.
type fakeNetErr struct{ msg string }

func (e *fakeNetErr) Error() string   { return e.msg }
func (e *fakeNetErr) Timeout() bool   { return true }
func (e *fakeNetErr) Temporary() bool { return true }

func TestNarrateReadyNeedsAVoice(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir()) // no local voice installed
	site := settingsSite(t)
	plan := NarrationPlan{Voices: []NarrationVoice{{Lang: "pt"}}}
	if err := NarrateReady(site, plan, nil); err == nil {
		t.Fatal("no voice chosen and no local voice: the batch can't start")
	}
	var consent *ConsentError
	if err := NarrateReady(site, plan, map[string]string{"pt": "blup:pt-BR-Luana"}); !errors.As(err, &consent) {
		t.Fatalf("a Blup Cloud voice asks for consent first: %v", err)
	}
}
