package blupcloud

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/gabriellsalesx/blup/internal/cloudcrypt"
	"github.com/gabriellsalesx/blup/internal/notesync"
)

// What comes from Blup Cloud is checked before it is used: a manifest
// that opened (it is sealed, but a bug or an old client could have
// written anything) must still make sense, and the plain answers of the
// API (versions, head, account) must have their fields. A manifest that
// does not is an error, never an empty vault: an empty vault would make
// the sync delete the notes here.

// ErrBadAnswer: an answer of the cloud is missing fields or has values
// that cannot be.
var ErrBadAnswer = errors.New("blupcloud: unexpected answer from Blup Cloud")

// maxManifestFile bounds a file's size in a manifest: the largest file
// Blup syncs (notesync.MaxFileSize, 100 MiB; the cloud's plans allow up to
// 100 MB). A file is put together in memory, so nothing larger is ever
// downloaded.
const maxManifestFile = notesync.MaxFileSize

// maxManifestFiles bounds how many files a manifest lists.
const maxManifestFiles = 1_000_000

func validSHA1(s string) bool {
	if len(s) != 40 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// validNotePath is a path inside the vault as notesync writes it: forward
// slashes, relative, no "." or ".." part, no NUL or backslash, valid UTF-8.
func validNotePath(p string) bool {
	if p == "" || len(p) > 4096 || !utf8.ValidString(p) || strings.ContainsAny(p, "\\\x00") || p[0] == '/' {
		return false
	}
	for _, part := range strings.Split(p, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

// blocksFor is how many blocks cloudcrypt.Split makes of size bytes.
func blocksFor(size int64) int64 {
	if size == 0 {
		return 1
	}
	return (size + cloudcrypt.BlockSize - 1) / cloudcrypt.BlockSize
}

func (f manifestFile) check() error {
	switch {
	case !validSHA1(f.SHA):
		return errors.New("sha")
	case f.Size < 0 || f.Size > maxManifestFile:
		return errors.New("size")
	case f.Mode != "" && f.Mode != "100644" && f.Mode != "100755":
		return errors.New("mode")
	case int64(len(f.Blocks)) != blocksFor(f.Size):
		return errors.New("blocks do not match the size")
	}
	for _, id := range f.Blocks {
		if !validBlockID(id) {
			return errors.New("block id")
		}
	}
	return nil
}

// parseManifest reads an opened manifest: a JSON object with "files"
// (an object, possibly empty) whose every entry is valid.
func parseManifest(plain []byte) (manifest, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(plain, &root); err != nil || root == nil {
		return manifest{}, fmt.Errorf("%w: manifest is not an object", ErrBadAnswer)
	}
	raw, ok := root["files"]
	if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return manifest{}, fmt.Errorf("%w: manifest without files", ErrBadAnswer)
	}
	var m manifest
	if err := json.Unmarshal(raw, &m.Files); err != nil || m.Files == nil {
		return manifest{}, fmt.Errorf("%w: manifest files unreadable", ErrBadAnswer)
	}
	// The chain (chain.go): checked against the version by openManifest.
	if raw, ok := root["prev"]; ok {
		if err := json.Unmarshal(raw, &m.Prev); err != nil || m.Prev == nil {
			return manifest{}, fmt.Errorf("%w: manifest prev unreadable", ErrBadAnswer)
		}
	}
	if len(m.Files) > maxManifestFiles {
		return manifest{}, fmt.Errorf("%w: manifest too large", ErrBadAnswer)
	}
	for p, f := range m.Files {
		if !validNotePath(p) {
			return manifest{}, fmt.Errorf("%w: manifest path %q", ErrBadAnswer, p)
		}
		if err := f.check(); err != nil {
			return manifest{}, fmt.Errorf("%w: manifest file %q: %w", ErrBadAnswer, p, err)
		}
	}
	return m, nil
}

// checkVersions validates a page of versions: at most 50, numbers
// positive and strictly decreasing, below before (when not 0), sizes not
// negative and dates set.
func checkVersions(page []VersionInfo, before int64) error {
	if len(page) > versionsPage {
		return fmt.Errorf("%w: %d versions in a page", ErrBadAnswer, len(page))
	}
	last := before
	for _, v := range page {
		if v.N <= 0 || v.Size < 0 || v.CreatedAt.IsZero() || (last > 0 && v.N >= last) {
			return fmt.Errorf("%w: version %d", ErrBadAnswer, v.N)
		}
		last = v.N
	}
	return nil
}

// versionsPage is how many versions the cloud sends at a time.
const versionsPage = 50

func (a Account) check() error {
	l, u := a.Plan.Limits, a.Usage
	if !validOpaqueID(a.ID) || a.Plan.Name == "" || l.StorageMB < 0 || l.MaxFileMB < 0 || l.Sites < -1 || l.Devices < -1 ||
		l.HistoryDays < 0 || l.AIUsage < 0 || u.StorageBytes < 0 || u.StorageLimitBytes < 0 || (u.HistoryBytes != nil && *u.HistoryBytes < 0) || u.Vaults < 0 || u.Sites < 0 || u.Devices < 0 {
		return fmt.Errorf("%w: account", ErrBadAnswer)
	}
	return nil
}

func (u AIUsage) check() error {
	if u.MonthMicros < 0 || u.WeekCapMicros < 0 || u.UsedMonthMicros < 0 || u.UsedWeekMicros < 0 || u.PerCharMicros < 0 || u.Percent < 0 {
		return fmt.Errorf("%w: AI usage", ErrBadAnswer)
	}
	return nil
}
