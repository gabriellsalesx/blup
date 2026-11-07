package app

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
)

// The AI models the Studio offers come from Blup Cloud's staff area
// (blupcloud.StudioAI): which voice models show up, through the plan and
// with one's own OpenRouter key, and which models translate. They are read
// at most every 10 minutes. There are no defaults: a list the owner left
// empty offers nothing, and so does a Studio that never reached the Cloud
// (offline, the last answer read stays).

const studioAIEvery = 10 * time.Minute

var studioAIState struct {
	mu sync.Mutex
	at time.Time
	ai *blupcloud.StudioAI
}

// studioAI is the staff's choice, from the cache when fresh. A failed read
// keeps the last answer (or the defaults) and is tried again next time.
func studioAI(ctx context.Context) blupcloud.StudioAI {
	studioAIState.mu.Lock()
	defer studioAIState.mu.Unlock()
	if studioAIState.ai != nil && time.Since(studioAIState.at) < studioAIEvery {
		return *studioAIState.ai
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if ai, err := blupcloud.New("").StudioAI(ctx); err == nil {
		studioAIState.ai, studioAIState.at = &ai, time.Now()
		return ai
	}
	if studioAIState.ai != nil {
		return *studioAIState.ai
	}
	return blupcloud.StudioAI{}
}

// pickModel is model when the list offers it, else the recommended one
// (when the owner marked one), else the list's first; "" for an empty list.
func pickModel(model string, list blupcloud.VoiceList) string {
	switch {
	case slices.Contains(list.Models, model):
		return model
	case list.Recommended != "":
		return list.Recommended
	case len(list.Models) > 0:
		return list.Models[0]
	}
	return ""
}
