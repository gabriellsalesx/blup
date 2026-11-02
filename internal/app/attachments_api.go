package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
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
	// Risky: a program or a script (see RiskyAttachment), which opens only
	// after the Studio asks.
	Risky bool `json:"risky"`
}

// riskyExts are the extensions a system runs instead of showing: programs,
// installers, scripts, shortcuts and documents with macros, on Windows,
// macOS and Linux.
var riskyExts = map[string]bool{}

func init() {
	for _, e := range strings.Fields(`
		.exe .msi .msp .mst .msix .msixbundle .appx .appxbundle .appinstaller .application .appref-ms
		.bat .cmd .com .scr .pif .cpl .dll .ocx .sys .drv .msc .gadget .inf .ins .isp .job
		.ps1 .ps1xml .ps2 .psc1 .psc2 .psd1 .psm1 .vb .vbs .vbe .js .jse .mjs .ws .wsf .wsc .wsh
		.hta .lnk .url .scf .shs .shb .reg .chm .hlp .xbap .xll .xlam .xla .ppam .docm .dotm .xlsm .xltm .pptm .potm .ppsm .sldm
		.settingcontent-ms .library-ms .search-ms .searchconnector-ms .diagcab .iso .img .vhd .vhdx .udl .rdp
		.jar .jnlp .py .pyw .pyz .pyc .pl .rb .php .tcl .lua
		.app .command .pkg .mpkg .dmg .terminal .workflow .action .scpt .scptd .applescript .tool .osx .prefpane .kext
		.sh .bash .zsh .ksh .csh .fish .run .bin .out .elf .appimage .deb .rpm .snap .flatpak .flatpakref .desktop .AppImage`) {
		riskyExts[strings.ToLower(e)] = true
	}
}

// RiskyAttachment tells whether opening the file name with the system
// would run it rather than show it (a program, a script, a shortcut). A
// vault downloaded or synced from someone else may bring one: the Studio
// opens it only after asking. Windows ignores dots and spaces at the end
// of a name ("x.exe." runs as x.exe), and a ":" picks a hidden stream of
// a file: both count as risky.
func RiskyAttachment(name string) bool {
	if strings.Contains(name, ":") {
		return true
	}
	n := strings.ToLower(strings.TrimRight(name, ". "))
	return riskyExts[filepath.Ext(n)]
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

// runnable tells whether a file has the permission to run (Linux and
// macOS: a program without an extension is still a program).
func runnable(info fs.FileInfo) bool {
	return runtime.GOOS != "windows" && info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0
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
		list = append(list, withTime{Attachment{Name: e.Name(), File: filepath.Join(dir, e.Name()), Size: info.Size(), Kind: attachmentKind(e.Name()), Uses: []string{}, Risky: RiskyAttachment(e.Name()) || runnable(info)}, info.ModTime().UnixNano()})
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
		if strings.EqualFold(filepath.Ext(p), ".md") && !isLink(d) {
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
	if !attachmentName(old) {
		return "", errors.New("not an attachment")
	}
	name, err := cleanName(strings.TrimSuffix(name, filepath.Ext(old)))
	if err != nil {
		return "", err
	}
	name += filepath.Ext(old)
	if name == old {
		return name, nil
	}
	dir := attachmentsPath(site)
	if !Within(site.ContentDir(), dir) {
		return "", errors.New("the attachments folder is outside the notes folder")
	}
	if _, err := os.Lstat(filepath.Join(dir, name)); err == nil {
		return "", errors.New(name + " already exists")
	}
	if err := renameInside(site.ContentDir(), filepath.Join(dir, old), filepath.Join(dir, name)); err != nil {
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
			WriteInside(site.ContentDir(), p, []byte(out))
		}
	})
	return name, nil
}

// DeleteAttachment moves a file to the Blup trash.
func DeleteAttachment(site *config.Site, name string) (string, error) {
	if !attachmentName(name) {
		return "", errors.New("not an attachment")
	}
	return toTrash(site, attachmentsFolder(site)+"/"+name)
}

// attachmentName tells whether name is a file name of the attachments
// folder as the Attachments screen lists it: no folders ("../x", "a/b"),
// not "." (the whole folder) and not hidden.
func attachmentName(name string) bool {
	return name != "" && name == filepath.Base(name) && !strings.ContainsAny(name, `/\`) && !strings.HasPrefix(name, ".")
}

// AddAttachments copies files into the attachments folder (a taken name
// gets a number) and returns their names.
func AddAttachments(site *config.Site, files []string) ([]string, error) {
	dir := attachmentsPath(site)
	if err := MkdirInside(site.ContentDir(), dir); err != nil {
		return nil, err
	}
	if !Within(site.ContentDir(), dir) {
		return nil, errors.New("the attachments folder is outside the notes folder")
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
			if _, err := os.Lstat(filepath.Join(dir, name)); errors.Is(err, fs.ErrNotExist) {
				break
			}
			name = fmt.Sprintf("%s %d%s", base, i, ext)
		}
		if err := writeNew(site.ContentDir(), filepath.Join(dir, name), b); err != nil {
			return out, err
		}
		out = append(out, name)
	}
	return out, nil
}
