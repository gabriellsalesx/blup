package obsidian

import "testing"

func TestUnpublished(t *testing.T) {
	for _, c := range []struct {
		props map[string]any
		want  bool
	}{
		{map[string]any{}, false},
		{map[string]any{"publish": false}, true},
		{map[string]any{"publish": "false"}, true},
		{map[string]any{"publish": "No"}, true},
		{map[string]any{"publish": "não"}, true},
		{map[string]any{"publish": true}, false},
		{map[string]any{"publish": "yes"}, false},
		{map[string]any{"publish": nil}, false},
		{map[string]any{"private": true}, true},
		{map[string]any{"private": "true"}, true},
		{map[string]any{"private": "yes"}, true},
		{map[string]any{"private": "sim"}, true},
		{map[string]any{"private": false}, false},
		{map[string]any{"private": "no"}, false},
		{map[string]any{"publish": true, "private": true}, true},
	} {
		if got := Unpublished(c.props); got != c.want {
			t.Errorf("Unpublished(%v) = %v, want %v", c.props, got, c.want)
		}
	}
}
