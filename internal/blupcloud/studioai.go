package blupcloud

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"slices"
)

// The AI models Blup Studio offers, chosen in Blup Cloud's staff area:
// Voice lists the voice models to choose from, in order, and the one the
// owner marked as recommended (none until marked); Translate is the order
// the models translate in (the main one first, the others take over).
// Plan is through Blup Cloud on the plan's AI allowance; Own, with the
// person's own OpenRouter key.

// ModelLists are a feature's models, through the plan and with one's own key.
type ModelLists struct {
	Plan []string `json:"plan"`
	Own  []string `json:"own"`
}

// VoiceList is a list of voice models and the recommended one ("" none).
type VoiceList struct {
	Models      []string `json:"models"`
	Recommended string   `json:"recommended"`
}

// VoiceLists are the voice models through the plan and with one's own key.
type VoiceLists struct {
	Plan VoiceList `json:"plan"`
	Own  VoiceList `json:"own"`
}

// StudioAI is what the staff picked for Blup Studio.
type StudioAI struct {
	Voice     VoiceLists `json:"voice"`
	Translate ModelLists `json:"translate"`
}

// modelID is an OpenRouter model id, as the cloud checks them.
var modelID = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}/[a-z0-9][a-z0-9._:-]{0,99}$`)

// maxModels is the longest list the cloud keeps.
const maxModels = 10

// StudioAI reads the models the staff picked. It needs no sign in: someone
// with only an OpenRouter key gets them too.
func (c *Client) StudioAI(ctx context.Context) (StudioAI, error) {
	var out StudioAI
	if err := c.do(ctx, http.MethodGet, "/studio-ai", nil, "", &out); err != nil {
		return StudioAI{}, err
	}
	for _, v := range []VoiceList{out.Voice.Plan, out.Voice.Own} {
		if v.Recommended != "" && !slices.Contains(v.Models, v.Recommended) {
			return StudioAI{}, fmt.Errorf("%w: studio ai recommended %q", ErrBadAnswer, v.Recommended)
		}
	}
	for _, l := range [][]string{out.Voice.Plan.Models, out.Voice.Own.Models, out.Translate.Plan, out.Translate.Own} {
		if len(l) > maxModels { // empty: the owner hasn't added one yet
			return StudioAI{}, fmt.Errorf("%w: studio ai", ErrBadAnswer)
		}
		for _, id := range l {
			if !modelID.MatchString(id) {
				return StudioAI{}, fmt.Errorf("%w: studio ai model %q", ErrBadAnswer, id)
			}
		}
	}
	return out, nil
}
