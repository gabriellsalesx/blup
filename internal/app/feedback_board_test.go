package app

import (
	"encoding/json"
	"testing"

	"github.com/gabriellsalesx/blup/internal/deploy/github"
)

func TestBoardPosts(t *testing.T) {
	var issues []github.BoardIssue
	if err := json.Unmarshal([]byte(`[
		{"number": 3, "title": "Menu some", "state": "open", "body": "x\n\n<!-- blup:feedback kind=bug lang=es -->", "labels": [{"name": "status: planned"}], "reactions": {"+1": 4}, "comments": 2, "user": {"login": "ana"}},
		{"number": 2, "title": "Roadmap", "state": "open", "body": "from the CLI", "labels": [{"name": "blup"}]},
		{"number": 1, "title": "Velho", "state": "closed", "body": "<!-- blup:feedback kind=weird -->"}
	]`), &issues); err != nil {
		t.Fatal(err)
	}
	got := boardPosts(issues)
	if len(got) != 2 {
		t.Fatalf("posts = %+v", got)
	}
	if p := got[0]; p.Kind != "bug" || p.Lang != "es" || p.Status != "planned" || p.Votes != 4 || p.Comments != 2 || p.Author != "ana" {
		t.Fatalf("bug = %+v", p)
	}
	// An unknown type is "other"; closed without a status label is done.
	if p := got[1]; p.Kind != "other" || p.Status != "done" {
		t.Fatalf("old = %+v", p)
	}
}
