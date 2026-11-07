package app

import (
	"os"
	"testing"
	"time"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/gabriellsalesx/blup/internal/narrate"
	"github.com/gabriellsalesx/blup/internal/translate"
)

// Tests never reach the real Blup Cloud: a test that needs it starts its
// own server and sets BLUP_CLOUD_URL; any other call fails at once here.
func TestMain(m *testing.M) {
	if os.Getenv("BLUP_CLOUD_URL") == "" {
		os.Setenv("BLUP_CLOUD_URL", "http://127.0.0.1:9")
	}
	// The AI lists start empty in the Cloud; these tests run as if the
	// owner had added the Studio's usual models (TestStudioAIFromStaff
	// changes them and puts them back).
	studioAIState.ai = &blupcloud.StudioAI{
		Voice:     blupcloud.VoiceLists{Plan: blupcloud.VoiceList{Models: []string{narrate.DefaultModel}}, Own: blupcloud.VoiceList{Models: []string{narrate.DefaultModel}}},
		Translate: blupcloud.ModelLists{Plan: []string{translate.Model}, Own: []string{translate.Model}},
	}
	studioAIState.at = time.Now().Add(24 * time.Hour) // fresh for the whole run
	os.Exit(m.Run())
}
