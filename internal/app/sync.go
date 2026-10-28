package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/deploy/github"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/notesync"
	"github.com/gabriellsalesx/blup/internal/ui"
)

func init() {
	registerErrors(
		knownError{notesync.ErrRemoteMoved, "sync.err_moved"},
		knownError{notesync.ErrLocalFolderEmpty, "sync.err_empty"},
	)
}

// syncNotes keeps the site's notes in a private GitHub repository, without
// git: what changed here is sent, what changed on another computer is
// received. It shows what will happen and asks before doing it.
func syncNotes(ctx context.Context, _ *config.Global, site *config.Site) error {
	ui.Step(i18n.T("sync.title"))
	c, login, err := githubClient(ctx)
	if err != nil {
		return err
	}
	s := &site.Sync
	if s.Repo == "" {
		if err := ui.Explain(i18n.T("sync.about_title"), i18n.T("sync.about")); err != nil {
			return err
		}
		s.Repo = Slug(site.Title) + i18n.T("sync.repo_suffix")
		if err := ui.Ask(i18n.T("sync.repo_q"), i18n.T("sync.repo_help", login), s.Repo, &s.Repo, notEmpty); err != nil {
			return err
		}
		s.Repo = strings.TrimSpace(s.Repo)
	}
	if s.Owner == "" {
		s.Owner = login
	}

	var repo github.Repo
	if err := ui.Spin(ctx, i18n.T("sync.preparing"), func(ctx context.Context) error {
		repo, err = notesync.EnsureRepo(ctx, c, s.Owner, s.Repo)
		return err
	}); err != nil {
		return err
	}
	if !repo.Private {
		ok, err := ui.Confirm(i18n.T("sync.public_q", repo.FullName), i18n.T("sync.public_help"), false)
		if err != nil || !ok {
			return err
		}
	}
	if err := site.Save(); err != nil {
		return err
	}

	sy := &notesync.Syncer{
		GH: c, Owner: s.Owner, Repo: s.Repo,
		Dir:       site.ContentDir(),
		StateFile: filepath.Join(site.Dir(), ".blup", "sync-state.json"),
	}
	var plan notesync.Report
	if err := ui.Spin(ctx, i18n.T("sync.checking"), func(ctx context.Context) error {
		plan, err = sy.Status(ctx, nil)
		return err
	}); err != nil {
		return err
	}
	if plan.UpToDate && len(plan.Conflicts) == 0 {
		ui.OK(i18n.T("sync.up_to_date", repo.FullName))
		return nil
	}
	summary := i18n.T("sync.plan", len(plan.Uploaded), len(plan.Downloaded), len(plan.Deleted), len(plan.DeletedLocal), len(plan.Conflicts))
	if ok, err := ui.Confirm(i18n.T("sync.confirm", repo.FullName), summary, true); err != nil || !ok {
		if err == nil {
			ui.Hint(i18n.T("cancelled"))
		}
		return err
	}

	var rep notesync.Report
	if err := ui.Spin(ctx, i18n.T("sync.running"), func(ctx context.Context) error {
		rep, err = sy.Sync(ctx, nil)
		return err
	}); err != nil {
		return err
	}
	ui.OK(i18n.T("sync.done", len(rep.Uploaded)+len(rep.Deleted), len(rep.Downloaded)+len(rep.DeletedLocal)))
	for i, p := range rep.Conflicts {
		copyName := ""
		if i < len(rep.ConflictCopies) {
			copyName = rep.ConflictCopies[i]
		}
		ui.Warn(i18n.T("sync.conflict", p, copyName))
	}
	if len(rep.DeletedLocal) > 0 {
		ui.Hint(i18n.T("sync.trash", len(rep.DeletedLocal), rep.Trash))
	}
	for _, p := range rep.Skipped {
		ui.Warn(i18n.T("sync.skipped", p, i18n.T(fmt.Sprintf("sync.skip_%s", rep.SkipReasons[p]))))
	}
	return nil
}
