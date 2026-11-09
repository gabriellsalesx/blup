package app

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/obsidian"
)

const narrationBody = "Um texto com palavras suficientes para virar uma narração de verdade."

// podcastSite is a blog with posts in every situation the batch meets.
func podcastSite(t *testing.T) *config.Site {
	t.Helper()
	site := settingsSite(t)
	site.Translations = []string{"en"}
	root := site.ContentDir()
	write(t, root, "Posts/Longo.md", "---\ntitle: Longo\n---\n"+narrationBody)
	write(t, root, "Posts/en/Longo.md", "---\ntitle: Long\ntranslationKey: Posts/Longo\n---\nA text with words enough to become a real narration.")
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
	write(t, root, "Posts/Senha.md", "---\ntitle: Senha\npassword: segredo\n---\n"+narrationBody)
	write(t, root, "Posts/en/Senha.md", "---\ntitle: Password\ntranslationKey: Posts/Senha\n---\nA text with words enough to become a real narration.")
	write(t, root, "Posts/Rascunho.md", "---\ndraft: true\n---\n"+narrationBody)
	write(t, root, "Posts/Oculto.md", "---\nhidden: true\n---\n"+narrationBody)
	write(t, root, "Posts/Fora.md", "---\npublish: false\n---\n"+narrationBody)
	write(t, root, "Posts/Particular.md", "---\nprivate: sim\n---\n"+narrationBody)
	write(t, root, "Privacidade.md", "---\nblup: privacy\n---\n"+narrationBody)
	write(t, root, "Autores/_index.md", "---\nblup: authors\n---\n")
	write(t, root, "Autores/Ana.md", "---\nrole: Editora\n---\n"+narrationBody)
	write(t, root, "Podcast/_index.md", "---\nblup: podcast\n---\n"+narrationBody)
	write(t, root, "Podcast/Extra.md", narrationBody)
	return site
}

func TestNarrationTodo(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	site := podcastSite(t)
	plan, err := NarrationTodo(site, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	var rels []string
	for _, it := range plan.Items {
		rels = append(rels, it.Rel)
	}
	slices.Sort(rels)
	if want := []string{"Posts/Longo.md", "Posts/Velho.md", "Posts/en/Longo.md"}; !slices.Equal(rels, want) {
		t.Fatalf("items %v, want %v", rels, want)
	}
	if plan.Missing != 1 || plan.Outdated != 1 || plan.Translations["en"] != 1 || plan.Locked != 1 || plan.Total != 3 {
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

func TestNarrateSite(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	site := podcastSite(t)
	plan, err := NarrationTodo(site, nil)
	if err != nil {
		t.Fatal(err)
	}
	var read []string
	narrateNote = func(_ context.Context, _ *config.Site, rel, voice string, _ func(int, int)) (Narration, error) {
		read = append(read, rel)
		if rel == "Posts/Velho.md" {
			return Narration{}, errors.New("this one failed")
		}
		return Narration{Link: "[[" + filepath.Base(rel) + ".mp3]]", Hash: "h-" + rel, Seconds: 12.6}, nil
	}
	defer func() { narrateNote = Narrate }()
	var steps [][2]int
	res, err := NarrateSite(context.Background(), site, plan, nil, func(done, total int, _ string) { steps = append(steps, [2]int{done, total}) })
	if err != nil || res.Done != 2 || !slices.Equal(res.Failed, []string{"Posts/Velho.md"}) {
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
}

// The same failure note after note is something every note would hit:
// the batch stops instead of going through all of them.
func TestNarrateSiteStopsOnRepeatedFailure(t *testing.T) {
	site := settingsSite(t)
	plan := NarrationPlan{}
	for _, r := range []string{"a.md", "b.md", "c.md", "d.md", "e.md"} {
		plan.Items = append(plan.Items, NarrationItem{Rel: r, Lang: "pt"})
	}
	calls := 0
	narrateNote = func(context.Context, *config.Site, string, string, func(int, int)) (Narration, error) {
		calls++
		return Narration{}, errors.New("the allowance ran out")
	}
	defer func() { narrateNote = Narrate }()
	res, err := NarrateSite(context.Background(), site, plan, nil, nil)
	if err == nil || calls != sameErrorStop || len(res.Failed) != sameErrorStop {
		t.Fatalf("calls %d, res %+v, %v", calls, res, err)
	}
	// Cancelled: stops at once.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NarrateSite(ctx, site, plan, nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled: %v", err)
	}
}

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
