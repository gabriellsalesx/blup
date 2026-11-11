package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aymanbagabas/go-pty"
	"github.com/fsnotify/fsnotify"
	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/proc"
	"github.com/gabriellsalesx/blup/internal/sandbox"
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
	if sandbox.Flatpak() {
		// SHELL may name a host-only shell. Do not load the user's host
		// startup files, which can launch programs unavailable in the sandbox.
		return "/bin/bash", []string{"--noprofile", "--norc"}
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
		if _, err := exec.LookPath(run); err != nil {
			return "", fmt.Errorf("%s is not available in this installation", run)
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
	// The full path: on Windows go-pty looks for a bare name inside c.Dir
	// (the site's folder), not in PATH, and finds no powershell.exe there.
	if full, err := exec.LookPath(name); err == nil {
		name = full
	}
	c := p.Command(name, args...)
	c.Dir = site.ContentDir()
	c.Env = append(proc.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	if sandbox.Flatpak() {
		c.Env = append(c.Env, "SHELL=/bin/bash")
	}
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

// InFlatpak lets the terminal explain that host-installed tools are outside
// this installation. It does not grant access to the host.
func (s *Studio) InFlatpak() bool { return sandbox.Flatpak() }

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
	// onFolderGone is told when a folder at the top of the notes is
	// deleted for good (rm, "delete permanently" in another app): Hugo's
	// watcher misses it, so the preview starts again (a Bento home that
	// was the section cards may now be the only section left).
	onFolderGone func()
	// watchReady closes when the current watcher has gone through the
	// notes' folders (the tests wait for it).
	watchReady chan struct{}
)

// notesChanged tells the window which notes changed.
var notesChanged = func(c NotesChanged) { emit("notesChanged", c) }

// siteFileChanged tells the window the site's settings file (hugo.toml)
// changed on disk.
var siteFileChanged = func(dir string) { emit("siteSettingsChanged", SiteSettingsChanged{Dir: dir}) }

// watchNotes tells the window when notes change on disk (an agent, another
// editor), at most every half second; and when the site's hugo.toml in
// dir (the site's folder) changes. Hidden folders are left out. The
// folders are gone through in the watcher's own goroutine, so opening a
// big vault doesn't wait for it; another site opened meanwhile closes this
// watcher, the walk stops at its next folder, and what was waiting to be
// told of it is dropped.
func watchNotes(dir, root string) {
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
		log.Printf("blup studio: watching the notes: %v", err)
		return
	}
	watcher = w
	ready := make(chan struct{})
	watchReady = ready
	// hugo.toml lives in the site's folder, outside the notes (most of
	// the time): that folder is watched too, alone.
	hugoFile := ""
	if dir != "" {
		hugoFile = filepath.Join(dir, "hugo.toml")
		if rel, err := filepath.Rel(root, dir); err != nil || strings.HasPrefix(rel, "..") {
			if err := w.Add(dir); err != nil {
				log.Printf("blup studio: watching %s: %v", dir, err)
			}
		}
	}
	// current tells whether w is still the watcher (not one closed by
	// another site opened).
	current := func() bool {
		watchMu.Lock()
		defer watchMu.Unlock()
		return watcher == w
	}
	// The folders at the top of the notes, to know a removed name was one.
	// Only the goroutine below touches it.
	top := map[string]bool{}
	add := func(d string) {
		filepath.WalkDir(d, func(p string, e fs.DirEntry, err error) error {
			if err != nil || !e.IsDir() {
				return nil
			}
			if p != d && strings.HasPrefix(e.Name(), ".") {
				return filepath.SkipDir
			}
			if filepath.Dir(p) == root {
				top[p] = true
			}
			if err := w.Add(p); errors.Is(err, fsnotify.ErrClosed) {
				return filepath.SkipAll // another site opened: this walk is over
			} else if err != nil {
				log.Printf("blup studio: watching %s: %v", p, err)
			}
			return nil
		})
	}
	go func() {
		// Changes during the walk wait in the system's queue, read below.
		add(root)
		close(ready)
		var (
			mu         sync.Mutex
			timer      *time.Timer
			settingsAt *time.Timer
			closed     bool
		)
		changed := map[string]bool{}
		defer func() {
			// Closed: nothing waiting goes out (the paths are of a site
			// no longer open).
			mu.Lock()
			closed = true
			if timer != nil {
				timer.Stop()
			}
			if settingsAt != nil {
				settingsAt.Stop()
			}
			mu.Unlock()
		}()
		for {
			select {
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				if hugoFile != "" && ev.Name == hugoFile {
					if ev.Op == fsnotify.Chmod {
						continue
					}
					mu.Lock()
					if settingsAt == nil {
						settingsAt = time.AfterFunc(500*time.Millisecond, func() {
							mu.Lock()
							settingsAt = nil
							gone := closed
							mu.Unlock()
							if !gone && current() {
								siteFileChanged(dir)
							}
						})
					}
					mu.Unlock()
					continue
				}
				rel, err := filepath.Rel(root, ev.Name)
				// A change of permissions or access time only (a backup tool,
				// an indexer) changes nothing to read.
				if err != nil || strings.HasPrefix(rel, "..") || strings.HasPrefix(filepath.Base(ev.Name), ".") || ev.Op == fsnotify.Chmod {
					continue
				}
				if ev.Has(fsnotify.Create) {
					if st, err := os.Stat(ev.Name); err == nil && st.IsDir() {
						add(ev.Name)
					}
				}
				if ev.Has(fsnotify.Remove) && top[ev.Name] {
					delete(top, ev.Name)
					watchMu.Lock()
					f := onFolderGone
					watchMu.Unlock()
					if f != nil {
						f()
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
						slices.Sort(list)
						changed = map[string]bool{}
						timer = nil
						gone := closed
						mu.Unlock()
						if gone || !current() {
							return
						}
						notesChanged(NotesChanged{Dir: dir, Paths: list})
						watchMu.Lock()
						f := onNotesChanged
						watchMu.Unlock()
						if f != nil {
							f()
						}
					})
				}
				mu.Unlock()
			case err, ok := <-w.Errors:
				if !ok {
					return
				}
				// The system's queue overflowed or a folder can't be read:
				// said, not hidden (changes may have been missed).
				log.Printf("blup studio: watching the notes: %v", err)
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
