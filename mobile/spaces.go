package main

import (
	"cmp"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
)

// A phone can hold several vaults, one open at a time. Each lives in its own
// folder, with its notes, trash and sync state, and only the open one syncs.
// The first one keeps the app's own folder (ID ""), so nothing moves.

// SpaceService lists, opens and makes the vaults of this phone.
type SpaceService struct{}

// Space is a vault of this phone. Remote is its vault in Blup Cloud, if any.
type Space struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Remote  string `json:"remote"`
	Current bool   `json:"current"`
	Notes   int    `json:"notes"`
}

type spaceEntry struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type spaceFile struct {
	Current string       `json:"current"`
	List    []spaceEntry `json:"list"`
	// FirstGone: the first vault was deleted, so the app's folder is no vault.
	FirstGone bool `json:"firstGone,omitempty"`
	// FirstTitle: the first vault's name, once renamed.
	FirstTitle string `json:"firstTitle,omitempty"`
}

var (
	spaceLock sync.Mutex
	// syncLock keeps a sync and a change of vault from crossing.
	syncLock sync.Mutex
)

func spacesPath() (string, error) {
	root, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "spaces.json"), nil
}

func readSpaces() (spaceFile, error) {
	var f spaceFile
	file, err := spacesPath()
	if err != nil {
		return f, err
	}
	data, err := os.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return f, nil
	}
	if err != nil {
		return f, err
	}
	if json.Unmarshal(data, &f) != nil {
		return rebuildSpaces()
	}
	return f, nil
}

// rebuildSpaces finds the vaults by their folders when the list broke, so no
// notes are lost from sight. Their names come from the Cloud link, if any.
func rebuildSpaces() (spaceFile, error) {
	var f spaceFile
	root, err := dataDir()
	if err != nil {
		return f, err
	}
	entries, err := os.ReadDir(filepath.Join(root, "cofres"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return f, err
	}
	for i, e := range entries {
		if e.IsDir() {
			f.List = append(f.List, spaceEntry{ID: e.Name(), Title: tr("vaultN", i+2)})
		}
	}
	return f, nil
}

func writeSpaces(f spaceFile) error {
	file, err := spacesPath()
	if err != nil {
		return err
	}
	data, err := json.Marshal(f)
	if err != nil {
		return err
	}
	return writeSafe(file, data)
}

// spaceRoot is the folder of a vault of this phone.
func spaceRoot(id string) (string, error) {
	root, err := dataDir()
	if err != nil || id == "" {
		return root, err
	}
	return filepath.Join(root, "cofres", id), nil
}

// currentRoot is the folder of the open vault.
func currentRoot() (string, error) {
	spaceLock.Lock()
	defer spaceLock.Unlock()
	f, err := readSpaces()
	if err != nil {
		return "", err
	}
	return spaceRoot(f.Current)
}

// allSpaces are the vaults of this phone, the first one included.
func allSpaces(f spaceFile) []spaceEntry {
	if f.FirstGone {
		return f.List
	}
	return append([]spaceEntry{{Title: cmp.Or(f.FirstTitle, tr("firstVault"))}}, f.List...)
}

// List returns the vaults, by the name of their Cloud vault when linked.
func (s *SpaceService) List() ([]Space, error) {
	spaceLock.Lock()
	defer spaceLock.Unlock()
	f, err := readSpaces()
	if err != nil {
		return nil, err
	}
	out := []Space{}
	for _, e := range allSpaces(f) {
		root, err := spaceRoot(e.ID)
		if err != nil {
			return nil, err
		}
		l, err := linkAt(root)
		if err != nil {
			return nil, err
		}
		title := e.Title
		if l.Title != "" {
			title = l.Title
		}
		out = append(out, Space{ID: e.ID, Title: title, Remote: l.ID, Current: e.ID == f.Current, Notes: countNotes(filepath.Join(root, "cofre"))})
	}
	return out, nil
}

func countNotes(dir string) int {
	n := 0
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && path != dir && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		if !d.IsDir() && filepath.Ext(path) == ".md" {
			n++
		}
		return nil
	})
	return n
}

// Delete takes a vault off this phone, with its notes and trash, and, when
// it is linked, off Blup Cloud too (for every device). The Cloud goes
// first: if it can't, nothing is deleted. The phone always keeps one vault.
func (s *SpaceService) Delete(ctx context.Context, id string) error {
	if err := deleteRemote(ctx, id); err != nil {
		return err
	}
	if err := s.remove(id); err != nil {
		return err
	}
	return writeReminders()
}

// deleteRemote deletes the Cloud vault a vault of this phone is linked to.
func deleteRemote(ctx context.Context, id string) error {
	spaceLock.Lock()
	f, err := readSpaces()
	spaceLock.Unlock()
	if err != nil {
		return err
	}
	if len(allSpaces(f)) < 2 {
		return fail("needOneVault")
	}
	root, err := spaceRoot(id)
	if err != nil {
		return err
	}
	l, err := linkAt(root)
	if err != nil || l.ID == "" {
		return err
	}
	c, _, err := client()
	if errors.Is(err, blupcloud.ErrSignedOut) {
		return fail("signInDelete")
	}
	if err != nil {
		return err
	}
	switch err := c.DeleteVault(ctx, l.ID); {
	case errors.Is(err, blupcloud.ErrSyncBusy):
		return fail("vaultBusy")
	case errors.Is(err, blupcloud.ErrSignedOut):
		return fail("signInAgain")
	case errors.Is(err, blupcloud.ErrCantDelete):
		return fail("cloudNoDelete")
	case err != nil:
		return fail("cloudDeleteErr")
	}
	return nil
}

func (s *SpaceService) remove(id string) error {
	syncLock.Lock()
	defer syncLock.Unlock()
	spaceLock.Lock()
	defer spaceLock.Unlock()
	f, err := readSpaces()
	if err != nil {
		return err
	}
	all := allSpaces(f)
	if !slices.ContainsFunc(all, func(e spaceEntry) bool { return e.ID == id }) {
		return fail("vaultMissing")
	}
	if len(all) == 1 {
		return fail("needOneVault")
	}
	root, err := spaceRoot(id)
	if err != nil {
		return err
	}
	if id == "" {
		f.FirstGone = true
	} else {
		f.List = slices.DeleteFunc(f.List, func(e spaceEntry) bool { return e.ID == id })
	}
	if f.Current == id {
		f.Current = allSpaces(f)[0].ID
	}
	// The list first: a closed app must never open a vault half gone.
	if err := writeSpaces(f); err != nil {
		return err
	}
	if id != "" {
		return os.RemoveAll(root)
	}
	for _, name := range []string{"cofre", "site", "sync"} {
		if err := os.RemoveAll(filepath.Join(root, name)); err != nil {
			return err
		}
	}
	return nil
}

// Rename gives a vault of this phone a new name and, when it is linked, gives
// its Cloud vault the same one, so the other devices see it. It changes here
// anyway; false when the Cloud couldn't take it.
func (s *SpaceService) Rename(ctx context.Context, id, title string) (bool, error) {
	title = strings.Join(strings.Fields(title), " ")
	if title == "" || utf8.RuneCountInString(title) > 120 {
		return false, fail("vaultNameGive")
	}
	spaceLock.Lock()
	f, err := readSpaces()
	if err == nil {
		i := slices.IndexFunc(f.List, func(e spaceEntry) bool { return e.ID == id })
		switch {
		case id == "" && !f.FirstGone:
			f.FirstTitle = title
		case id != "" && i >= 0:
			f.List[i].Title = title
		default:
			err = fail("vaultMissing")
		}
	}
	if err == nil {
		err = writeSpaces(f)
	}
	spaceLock.Unlock()
	if err != nil {
		return false, err
	}
	root, err := spaceRoot(id)
	if err != nil {
		return false, err
	}
	l, err := linkAt(root)
	if err != nil || l.ID == "" {
		return true, err
	}
	l.Title = title
	if err := os.WriteFile(filepath.Join(root, "sync", "vault"), []byte(l.ID+"\n"+l.Title), 0o600); err != nil {
		return false, err
	}
	c, creds, err := client()
	if err != nil {
		return false, nil
	}
	k, err := keys(creds)
	if err != nil {
		return false, nil
	}
	return c.RenameSealed(ctx, k, l.ID, title) == nil, nil
}

// Open makes a vault the open one.
func (s *SpaceService) Open(id string) error {
	syncLock.Lock()
	defer syncLock.Unlock()
	spaceLock.Lock()
	defer spaceLock.Unlock()
	f, err := readSpaces()
	if err != nil {
		return err
	}
	if !slices.ContainsFunc(allSpaces(f), func(e spaceEntry) bool { return e.ID == id }) {
		return fail("vaultMissing")
	}
	f.Current = id
	return writeSpaces(f)
}

// Create makes an empty vault and opens it.
func (s *SpaceService) Create(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", fail("vaultName")
	}
	syncLock.Lock()
	defer syncLock.Unlock()
	spaceLock.Lock()
	defer spaceLock.Unlock()
	f, err := readSpaces()
	if err != nil {
		return "", err
	}
	for _, e := range allSpaces(f) {
		if strings.EqualFold(e.Title, title) {
			return "", fail("vaultTaken")
		}
	}
	raw := make([]byte, 6)
	rand.Read(raw)
	id := hex.EncodeToString(raw)
	root, err := spaceRoot(id)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(root, "cofre"), 0o700); err != nil {
		return "", err
	}
	f.List = append(f.List, spaceEntry{ID: id, Title: title})
	f.Current = id
	return id, writeSpaces(f)
}
