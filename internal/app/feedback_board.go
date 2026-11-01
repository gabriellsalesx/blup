package app

import (
	"context"
	"errors"
	"regexp"
	"slices"
	"strings"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/deploy/github"
	"github.com/gabriellsalesx/blup/internal/i18n"
)

// The feedback board of a Bento site (a section turned on in Blup Studio):
// requests visitors open from the site as issues of its "-feedback"
// repository, marked in their text with their type and language. The site
// reads them live (theme/assets/js/feedback.js); here the Studio lists them
// and sets their status, a "status: ..." label (done and declined also
// close the issue).

// boardMark is the mark the site puts in a request's text.
var boardMark = regexp.MustCompile(`<!--\s*blup:feedback\s+kind=([a-z]+)(?:\s+lang=([a-z]{2}))?\s*-->`)

// BoardStatuses are a request's statuses, in order.
var BoardStatuses = []string{"new", "planned", "progress", "done", "declined"}

// statusColors are the colors of the status labels on GitHub.
var statusColors = map[string]string{"planned": "8b5cf6", "progress": "f59e0b", "done": "22c55e", "declined": "64748b"}

// FeedbackPost is a request of the board.
type FeedbackPost struct {
	Number   int    `json:"number"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Kind     string `json:"kind"` // feature, bug, improvement, other
	Lang     string `json:"lang"`
	Status   string `json:"status"`
	Author   string `json:"author"`
	Created  string `json:"created"`
	Votes    int    `json:"votes"`
	Comments int    `json:"comments"`
}

func boardClient(ctx context.Context, site *config.Site) (*github.Client, *config.Feedback, error) {
	if !site.Feedback.On() {
		return nil, nil, errors.New(i18n.T("fb.board_off"))
	}
	creds, err := config.LoadCredentials()
	if err != nil {
		return nil, nil, err
	}
	if creds.GitHubToken == "" {
		return nil, nil, errors.New(i18n.T("fb.no_token"))
	}
	c, _, err := githubClient(ctx)
	return c, &site.Feedback, err
}

// FeedbackPosts lists the board's requests, newest first.
func FeedbackPosts(ctx context.Context, site *config.Site) ([]FeedbackPost, error) {
	c, f, err := boardClient(ctx, site)
	if err != nil {
		return nil, err
	}
	issues, err := c.RecentIssues(ctx, f.Owner, f.Repo, 3)
	if err != nil {
		return nil, err
	}
	return boardPosts(issues), nil
}

// boardPosts are the requests among issues (the ones with the site's mark).
func boardPosts(issues []github.BoardIssue) []FeedbackPost {
	out := []FeedbackPost{}
	for _, i := range issues {
		m := boardMark.FindStringSubmatch(i.Body)
		if m == nil {
			continue
		}
		kind := m[1]
		if !slices.Contains([]string{"feature", "bug", "improvement"}, kind) {
			kind = "other"
		}
		out = append(out, FeedbackPost{
			Number: i.Number, Title: i.Title, URL: i.URL, Kind: kind, Lang: m[2],
			Status: boardStatus(i), Author: i.User.Login, Created: i.CreatedAt,
			Votes: i.Reactions.PlusOne, Comments: i.Comments,
		})
	}
	return out
}

func boardStatus(i github.BoardIssue) string {
	for _, l := range i.Labels {
		if s, ok := strings.CutPrefix(l.Name, "status:"); ok && slices.Contains(BoardStatuses, strings.TrimSpace(s)) {
			return strings.TrimSpace(s)
		}
	}
	if i.State == "closed" {
		return "done"
	}
	return "new"
}

// SetFeedbackStatus sets a request's status: its label (the other labels
// stay), and closes it when done or declined (opens it again otherwise).
func SetFeedbackStatus(ctx context.Context, site *config.Site, number int, status string) error {
	if number <= 0 {
		return errors.New("invalid feedback issue number")
	}
	if !slices.Contains(BoardStatuses, status) {
		return errors.New("unknown status " + status)
	}
	c, f, err := boardClient(ctx, site)
	if err != nil {
		return err
	}
	i, err := c.BoardIssue(ctx, f.Owner, f.Repo, number)
	if err != nil {
		return err
	}
	// The marker identifies a visitor request, never its authority: GitHub
	// permissions and the owner's credential authorize this change.
	if i.Number != number || i.PullRequest != nil || !boardMark.MatchString(i.Body) {
		return errors.New("not a feedback request")
	}
	name := "status: " + status
	if status != "new" {
		if err := c.EnsureLabel(ctx, f.Owner, f.Repo, name, statusColors[status], "blup"); err != nil {
			return err
		}
		if err := c.AddIssueLabels(ctx, f.Owner, f.Repo, number, []string{name}); err != nil {
			return err
		}
	}
	// Change only labels managed by this board. Replacing the complete
	// list would erase labels added by another tool since our read.
	for _, l := range i.Labels {
		if s, ok := strings.CutPrefix(l.Name, "status:"); ok && slices.Contains(BoardStatuses, strings.TrimSpace(s)) && (status == "new" || l.Name != name) {
			if err := c.RemoveIssueLabel(ctx, f.Owner, f.Repo, number, l.Name); err != nil {
				return err
			}
		}
	}
	switch status {
	case "done":
		return c.SetIssueState(ctx, f.Owner, f.Repo, number, false, "completed")
	case "declined":
		return c.SetIssueState(ctx, f.Owner, f.Repo, number, false, "not_planned")
	}
	return c.SetIssueState(ctx, f.Owner, f.Repo, number, true, "")
}
