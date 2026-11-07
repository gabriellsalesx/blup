package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gabriellsalesx/blup/internal/config"
)

// The voice models and the translation order come from the staff, and the
// Studio's own defaults stand while Blup Cloud can't be read.
func TestStudioAIFromStaff(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	before, beforeAt := studioAIState.ai, studioAIState.at
	studioAIState.ai, studioAIState.at = nil, time.Time{}
	t.Cleanup(func() { studioAIState.ai, studioAIState.at = before, beforeAt })
	staff := `{"voice":{"plan":{"models":["vendor/plano","microsoft/mai-voice-2.1"],"recommended":"microsoft/mai-voice-2.1"},"own":{"models":["vendor/propria"],"recommended":null}},"translate":{"plan":["anthropic/claude-haiku-5.5"],"own":["x/um","y/dois"]}}`
	cloud := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/studio-ai" {
			w.Write([]byte(staff))
			return
		}
		w.WriteHeader(404)
	}))
	defer cloud.Close()
	router := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[
			{"id":"microsoft/mai-voice-2.1","name":"Microsoft AI: MAI-Voice-2.1","pricing":{"prompt":"0.000022"},"supported_voices":["pt-BR-Luana:MAI-Voice-2.1"]},
			{"id":"vendor/plano","name":"Vendor: Plano","pricing":{"prompt":"0.00001"},"supported_voices":["pt-BR-Ana"]},
			{"id":"vendor/propria","name":"Vendor: Própria","pricing":{"prompt":"0.00003"},"supported_voices":["pt-BR-Bia"]},
			{"id":"vendor/fora","name":"Vendor: Fora","pricing":{"prompt":"0.00001"},"supported_voices":["pt-BR-Caio"]}]}`))
	}))
	defer router.Close()
	openRouterURL = router.URL
	defer func() { openRouterURL = "" }()
	ctx := context.Background()

	// Never reached the Cloud: nothing is offered (no defaults).
	t.Setenv("BLUP_CLOUD_URL", "http://127.0.0.1:9")
	if ai := studioAI(ctx); len(ai.Voice.Own.Models) != 0 || len(ai.Translate.Own) != 0 {
		t.Fatalf("offline = %+v", ai)
	}
	if ms, _ := NarrationModels(ctx, "pt"); len(ms.Own) != 0 || len(ms.Plan) != 0 {
		t.Fatalf("offline models = %+v", ms)
	}
	t.Setenv("BLUP_CLOUD_URL", cloud.URL)
	ms, err := NarrationModels(ctx, "pt")
	if err != nil {
		t.Fatal(err)
	}
	ids := func(l []ModelOption) (out []string) {
		for _, m := range l {
			out = append(out, m.ID)
		}
		return out
	}
	// Recommended only where the owner marked one.
	if got := ids(ms.Own); len(got) != 1 || got[0] != "vendor/propria" || ms.Own[0].Recommended {
		t.Fatalf("own = %v", got)
	}
	if got := ids(ms.Plan); len(got) != 2 || got[0] != "vendor/plano" || got[1] != "microsoft/mai-voice-2.1" || ms.Plan[0].Recommended || !ms.Plan[1].Recommended {
		t.Fatalf("plan = %v", got)
	}
	// A model the staff doesn't offer (or none): the recommended one, else
	// the list's first.
	if err := (config.Credentials{OpenRouterToken: "k"}).Save(); err != nil {
		t.Fatal(err)
	}
	voices, _ := NarrationVoices(ctx, "pt", "vendor/fora", "")
	for _, v := range voices {
		if v.Engine == "openrouter" && v.ID != "vendor/propria|pt-BR-Bia" || v.Engine == "blup" && v.ID != "blup:microsoft/mai-voice-2.1|pt-BR-Luana:MAI-Voice-2.1" {
			t.Fatalf("voice %+v", v)
		}
	}
	// Translating with one's own key: the staff's order (translate.Client.Models).
	if got := studioAI(ctx).Translate.Own; len(got) != 2 || got[0] != "x/um" || got[1] != "y/dois" {
		t.Fatalf("translate own = %v", got)
	}
}
