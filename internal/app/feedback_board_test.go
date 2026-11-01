package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gabriellsalesx/blup/internal/config"
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

type boardTransport func(*http.Request) (*http.Response, error)

func (f boardTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSetFeedbackStatusAttacks(t *testing.T) {
	for _, tc := range []struct {
		name, body, status string
		number             int
		pr                 bool
		wantOK             bool
	}{
		{"negative", "<!-- blup:feedback kind=bug -->", "done", -1, false, false},
		{"zero", "<!-- blup:feedback kind=bug -->", "done", 0, false, false},
		{"status injection", "<!-- blup:feedback kind=bug -->", "done injected", 7, false, false},
		{"unrelated issue", "ordinary issue", "done", 7, false, false},
		{"pull request", "<!-- blup:feedback kind=bug -->", "done", 7, true, false},
		{"old request", "<!-- blup:feedback kind=bug -->", "planned", 7, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			if err := (&config.Credentials{GitHubToken: "test-only"}).Save(); err != nil {
				t.Fatal(err)
			}
			writes := 0
			old := http.DefaultTransport
			http.DefaultTransport = boardTransport(func(r *http.Request) (*http.Response, error) {
				body := `{}`
				switch {
				case r.URL.Path == "/user":
					body = `{"login":"owner"}`
				case r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/issues"):
					body = `[]` // older than the recent page
				case r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/issues/7"):
					i := map[string]any{"number": 7, "body": tc.body, "labels": []any{map[string]string{"name": "keep"}, map[string]string{"name": "status: security"}}}
					if tc.pr {
						i["pull_request"] = map[string]any{}
					}
					b, _ := json.Marshal(i)
					body = string(b)
				case r.Method != "GET":
					writes++
					if strings.HasSuffix(r.URL.Path, "/labels") && r.Method == "PUT" {
						t.Error("must not replace other labels using a stale recent page")
					}
					if strings.Contains(r.URL.Path, "security") || strings.HasSuffix(r.URL.Path, "/keep") {
						t.Error("removed unrelated label")
					}
				default:
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			})
			t.Cleanup(func() { http.DefaultTransport = old })
			site := &config.Site{Feedback: config.Feedback{Enabled: true, Owner: "owner", Repo: "site-feedback"}}
			err := SetFeedbackStatus(context.Background(), site, tc.number, tc.status)
			if (err == nil) != tc.wantOK {
				t.Errorf("error = %v, want success %v", err, tc.wantOK)
			}
			if !tc.wantOK && writes != 0 {
				t.Errorf("attack performed %d writes", writes)
			}
		})
	}
}

func TestBoardStatusRejectsUnmanagedLabel(t *testing.T) {
	var i github.BoardIssue
	_ = json.Unmarshal([]byte(`{"labels":[{"name":"status: injected hidden"}],"state":"open"}`), &i)
	if got := boardStatus(i); got != "new" {
		t.Fatalf("untrusted CSS status = %q", got)
	}
}
