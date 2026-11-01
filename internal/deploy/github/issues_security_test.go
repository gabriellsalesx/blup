package github

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type issueTransport func(*http.Request) (*http.Response, error)

func (f issueTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestIssueMutationsRejectInvalidNumber(t *testing.T) {
	c := New("test-only")
	c.MinWriteInterval = -1
	c.HTTPClient = &http.Client{Transport: issueTransport(func(r *http.Request) (*http.Response, error) {
		t.Error("invalid issue number reached transport")
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`)), Request: r}, nil
	})}
	for _, n := range []int{0, -1} {
		if err := c.SetIssueLabels(context.Background(), "owner", "repo", n, []string{"keep"}); err == nil {
			t.Error("invalid number accepted for labels")
		}
		if err := c.SetIssueState(context.Background(), "owner", "repo", n, false, "completed"); err == nil {
			t.Error("invalid number accepted for state")
		}
	}
}

func TestIssueLabelChangesPreserveOtherLabels(t *testing.T) {
	c := New("test-only")
	c.MinWriteInterval = -1
	var calls []string
	c.HTTPClient = &http.Client{Transport: issueTransport(func(r *http.Request) (*http.Response, error) {
		calls = append(calls, r.Method+" "+r.URL.EscapedPath())
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`)), Request: r}, nil
	})}
	if err := c.AddIssueLabels(context.Background(), "owner", "repo", 7, []string{"status: done"}); err != nil {
		t.Fatal(err)
	}
	if err := c.RemoveIssueLabel(context.Background(), "owner", "repo", 7, "status: progress"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0] != "POST /repos/owner/repo/issues/7/labels" || calls[1] != "DELETE /repos/owner/repo/issues/7/labels/status:%20progress" {
		t.Fatalf("unexpected label mutations: %v", calls)
	}
}
