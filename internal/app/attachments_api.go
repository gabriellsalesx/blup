package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gabriellsalesx/blup/internal/config"
)

// The attachments folder (Anexos) as Blup Studio's Attachments screen shows
// it: images, PDFs, audio and video the notes use, with the notes that use
// each one. Renaming a file fixes the notes that point to it.

// Attachment is a file of the attachments folder.
type Attachment struct {
	Name string   `json:"name"`
	File string   `json:"file"` // full path
	Size int64    `json:"size"`
	Kind string   `json:"kind"` // "image", "pdf", "audio", "video" or "other"
	Uses []string `json:"uses"` // notes that point to it
}

func attachmentKind(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif", ".svg", ".avif", ".bmp":
		return "image"
	case ".pdf":
		return "pdf"
	case ".mp3", ".m4a", ".ogg", ".wav", ".flac":
		return "audio"
	case ".mp4", ".webm", ".mov":
		return "video"
	}
	return "other"
}

func attachmentsPath(site *config.Site) string {
	return filepath.Join(site.ContentDir(), attachmentsFolder(site))
}

// AttachmentsFolder is the name of the attachments folder in the notes.
func AttachmentsFolder(site *config.Site) string { return attachmentsFolder(site) }

// ListAttachments lists the attachments, newest first, with their uses.
func ListAttachments(site *config.Site) ([]Attachment, error) {
	dir := attachmentsPath(site)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return []Attachment{}, nil
	}
	if err != nil {
		return nil, err
	}
	type withTime struct {
		a Attachment
		t int64
	}
	var list []withTime
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		list = append(list, withTime{Attachment{Name: e.Name(), File: filepath.Join(dir, e.Name()), Size: info.Size(), Kind: attachmentKind(e.Name()), Uses: []string{}}, info.ModTime().UnixNano()})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].t > list[j].t })
	out := make([]Attachment, len(list))
	for i := range list {
		out[i] = list[i].a
	}
	// Which notes point to each file (by name: links, ![[embeds]] and
	// properties all carry it, maybe with %20 for spaces).
	root := site.ContentDir()
	eachNoteText(site, func(p, text string) {
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		for i := range out {
			if mentions(text, out[i].Name) {
				out[i].Uses = append(out[i].Uses, rel)
			}
		}
	})
	return out, nil
}

func mentions(text, name string) bool {
	for _, n := range []string{name, strings.ReplaceAll(name, " ", "%20")} {
		for _, before := range []string{"/", "[[", "(", "<"} {
			if strings.Contains(text, before+n) {
				return true
			}
		}
	}
	return false
}

// eachNoteText reads the text of every note (skipping hidden folders).
func eachNoteText(site *config.Site, fn func(path, text string)) {
	root := site.ContentDir()
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(p), ".md") {
			if b, err := os.ReadFile(p); err == nil {
				fn(p, string(b))
			}
		}
		return nil
	})
}

// RenameAttachment renames a file (keeping its extension) and fixes every
// note that points to it. It returns the new name.
func RenameAttachment(site *config.Site, old, name string) (string, error) {
	name, err := cleanName(strings.TrimSuffix(name, filepath.Ext(old)))
	if err != nil {
		return "", err
	}
	name += filepath.Ext(old)
	if name == old {
		return name, nil
	}
	dir := attachmentsPath(site)
	if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
		return "", errors.New(name + " already exists")
	}
	if err := os.Rename(filepath.Join(dir, filepath.Base(old)), filepath.Join(dir, name)); err != nil {
		return "", err
	}
	pairs := []string{}
	for _, before := range []string{"/", "[[", "(", "<"} {
		pairs = append(pairs, before+old, before+name)
		if strings.Contains(old, " ") {
			pairs = append(pairs, before+strings.ReplaceAll(old, " ", "%20"), before+strings.ReplaceAll(name, " ", "%20"))
		}
	}
	r := strings.NewReplacer(pairs...)
	eachNoteText(site, func(p, text string) {
		if out := r.Replace(text); out != text {
			os.WriteFile(p, []byte(out), 0o644)
		}
	})
	return name, nil
}

// DeleteAttachment moves a file to the Blup trash.
func DeleteAttachment(site *config.Site, name string) (string, error) {
	return toTrash(site, attachmentsFolder(site)+"/"+filepath.Base(name))
}

// AddAttachments copies files into the attachments folder (a taken name
// gets a number) and returns their names.
func AddAttachments(site *config.Site, files []string) ([]string, error) {
	dir := attachmentsPath(site)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	out := []string{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return out, err
		}
		ext := strings.ToLower(filepath.Ext(f))
		base := safeFileName(strings.TrimSuffix(filepath.Base(f), filepath.Ext(f)))
		if base == "" {
			base = "file"
		}
		name := base + ext
		for i := 2; ; i++ {
			if _, err := os.Stat(filepath.Join(dir, name)); errors.Is(err, fs.ErrNotExist) {
				break
			}
			name = fmt.Sprintf("%s %d%s", base, i, ext)
		}
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			return out, err
		}
		out = append(out, name)
	}
	return out, nil
}
