package main

import (
	"encoding/base64"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aymanbagabas/go-pty"
	"github.com/fsnotify/fsnotify"
	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// The terminal: the user's shell in the notes folder, in tabs, with
// shortcuts to the AI agents installed (Claude Code, Codex, OpenCode,
// Gemini, Antigravity). Output goes to the window as "term" events; the
// notes folder is watched so the tree and the editor follow what an agent
// changes ("notesChanged").

// TermData is a piece of a terminal's output (base64: it may split a
// character), or its end (Exit).
type TermData struct {
	ID   string `json:"id"`
	Data string `json:"data"`
	Exit bool   `json:"exit"`
}

func init() {
	application.RegisterEvent[TermData]("term")
	application.RegisterEvent[string]("notesChanged")
	application.RegisterEvent[app.LiveState]("cloudSync")
}

// Agent is an AI agent's command line.
type Agent struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	Command   string `json:"command"`
	Installed bool   `json:"installed"`
	Site      string `json:"site"` // where to install it
}

var agents = []Agent{
	{Key: "claude", Name: "Claude Code", Command: "claude", Site: "https://claude.com/claude-code"},
	{Key: "codex", Name: "Codex", Command: "codex", Site: "https://github.com/openai/codex"},
	{Key: "opencode", Name: "OpenCode", Command: "opencode", Site: "https://opencode.ai"},
	{Key: "gemini", Name: "Gemini CLI", Command: "gemini", Site: "https://github.com/google-gemini/gemini-cli"},
	{Key: "agy", Name: "Antigravity", Command: "agy", Site: "https://antigravity.google"},
}

// Agents lists the AI agents, and which are installed.
func (s *Studio) Agents() []Agent {
	out := make([]Agent, len(agents))
	for i, a := range agents {
		_, err := exec.LookPath(a.Command)
		a.Installed = err == nil
		out[i] = a
	}
	return out
}

type term struct {
	pty  pty.Pty
	cmd  *pty.Cmd
	snap string // the notes before an AI agent started (app.TakeSnapshot)
}

var (
	termsMu sync.Mutex
	terms   = map[string]*term{}
	termSeq int
)

func shell() (string, []string) {
	if runtime.GOOS == "windows" {
		return "powershell.exe", []string{"-NoLogo"}
	}
	if sh := os.Getenv("SHELL"); sh != "" {
		return sh, []string{"-l"}
	}
	return "/bin/bash", []string{"-l"}
}

// OpenTerminal starts a shell in the notes folder; with agent ("claude",
// "codex"...), the agent starts in it. It writes the guide for agents
// (AGENTS.md) the first time. It returns the terminal's id.
func (s *Studio) OpenTerminal(agent string, cols, rows int) (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	if err := app.WriteAgentGuides(site); err != nil {
		return "", err
	}
	run := ""
	if agent != "" {
		for _, a := range agents {
			if a.Key == agent {
				run = a.Command
			}
		}
		if run == "" {
			return "", errors.New("unknown agent " + agent)
		}
	}
	snap := ""
	if run != "" {
		if snap, err = app.TakeSnapshot(site); err != nil {
			return "", err
		}
	}
	p, err := pty.New()
	if err != nil {
		return "", err
	}
	if cols > 0 && rows > 0 {
		_ = p.Resize(cols, rows)
	}
	name, args := shell()
	c := p.Command(name, args...)
	c.Dir = site.ContentDir()
	c.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	if err := c.Start(); err != nil {
		p.Close()
		return "", err
	}
	termsMu.Lock()
	termSeq++
	id := strconv.Itoa(termSeq)
	terms[id] = &term{p, c, snap}
	termsMu.Unlock()

	emit := func(d TermData) { application.Get().Event.Emit("term", d) }
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := p.Read(buf)
			if n > 0 {
				emit(TermData{ID: id, Data: base64.StdEncoding.EncodeToString(buf[:n])})
			}
			if err != nil {
				break
			}
		}
	}()
	go func() {
		c.Wait()
		emit(TermData{ID: id, Exit: true})
		s.CloseTerminal(id)
	}()
	if run != "" {
		// The agent runs inside the shell: when it quits, the shell stays.
		_, _ = p.Write([]byte(run + "\r"))
	}
	return id, nil
}

func termOf(id string) *term {
	termsMu.Lock()
	defer termsMu.Unlock()
	return terms[id]
}

// WriteTerminal sends what the user types.
func (s *Studio) WriteTerminal(id, data string) error {
	t := termOf(id)
	if t == nil {
		return errors.New("terminal closed")
	}
	_, err := t.pty.Write([]byte(data))
	return err
}

// ResizeTerminal follows the panel's size.
func (s *Studio) ResizeTerminal(id string, cols, rows int) error {
	t := termOf(id)
	if t == nil || cols <= 0 || rows <= 0 {
		return nil
	}
	return t.pty.Resize(cols, rows)
}

// CloseTerminal stops a terminal (and what runs in it).
func (s *Studio) CloseTerminal(id string) {
	termsMu.Lock()
	t := terms[id]
	delete(terms, id)
	termsMu.Unlock()
	if t == nil {
		return
	}
	if t.cmd.Process != nil {
		_ = t.cmd.Process.Kill()
	}
	_ = t.pty.Close()
}

func closeTerminals(s *Studio) {
	termsMu.Lock()
	ids := make([]string, 0, len(terms))
	for id := range terms {
		ids = append(ids, id)
	}
	termsMu.Unlock()
	for _, id := range ids {
		s.CloseTerminal(id)
	}
}

// --- watching the notes ------------------------------------------------------

var (
	watchMu sync.Mutex
	watcher *fsnotify.Watcher
	// onNotesChanged is told too (the live sync).
	onNotesChanged func()
)

// watchNotes tells the window when notes change on disk (an agent, another
// editor), at most every half second. Hidden folders are left out.
func watchNotes(root string) {
	watchMu.Lock()
	defer watchMu.Unlock()
	if watcher != nil {
		watcher.Close()
		watcher = nil
	}
	if root == "" { // no site open: nothing to watch
		return
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	watcher = w
	add := func(dir string) {
		filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || !d.IsDir() {
				return nil
			}
			if p != dir && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			_ = w.Add(p)
			return nil
		})
	}
	add(root)
	go func() {
		var timer *time.Timer
		changed := map[string]bool{}
		var mu sync.Mutex
		for {
			select {
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				rel, err := filepath.Rel(root, ev.Name)
				// A change of permissions or access time only (a backup tool,
				// an indexer) changes nothing to read.
				if err != nil || strings.HasPrefix(filepath.Base(ev.Name), ".") || ev.Op == fsnotify.Chmod {
					continue
				}
				if ev.Has(fsnotify.Create) {
					if st, err := os.Stat(ev.Name); err == nil && st.IsDir() {
						add(ev.Name)
					}
				}
				mu.Lock()
				changed[filepath.ToSlash(rel)] = true
				if timer == nil {
					timer = time.AfterFunc(500*time.Millisecond, func() {
						mu.Lock()
						list := make([]string, 0, len(changed))
						for k := range changed {
							list = append(list, k)
						}
						changed = map[string]bool{}
						timer = nil
						mu.Unlock()
						application.Get().Event.Emit("notesChanged", strings.Join(list, "\n"))
						watchMu.Lock()
						f := onNotesChanged
						watchMu.Unlock()
						if f != nil {
							f()
						}
					})
				}
				mu.Unlock()
			case _, ok := <-w.Errors:
				if !ok {
					return
				}
			}
		}
	}()
}

func (s *Studio) snapshot(id string) (*config.Site, string, error) {
	site, err := s.current()
	if err != nil {
		return nil, "", err
	}
	termsMu.Lock()
	t := terms[id]
	termsMu.Unlock()
	if t == nil || t.snap == "" {
		return nil, "", errors.New("no snapshot for this terminal")
	}
	return site, t.snap, nil
}

// AIChanges lists what changed in the notes since the AI agent of
// terminal id started.
func (s *Studio) AIChanges(id string) ([]app.Change, error) {
	site, snap, err := s.snapshot(id)
	if err != nil {
		return nil, err
	}
	return app.SnapshotChanges(site, snap)
}

// UndoAI puts the notes back as they were when the AI agent of terminal id
// started; files it created go to the trash.
func (s *Studio) UndoAI(id string) error {
	site, snap, err := s.snapshot(id)
	if err != nil {
		return err
	}
	return app.UndoSnapshot(site, snap)
}
