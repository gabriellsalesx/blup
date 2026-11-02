package app

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gabriellsalesx/blup/internal/config"
)

// PasteImage saves an image pasted or dropped on a note into the site's
// attachments folder (a taken name gets a number) and returns the Markdown
// to put in the note: ![name](../Anexos/name.png), relative to the note.
func PasteImage(site *config.Site, note, name string, data []byte) (string, error) {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif", ".svg", ".avif":
	default:
		return "", errors.New("not an image")
	}
	if len(data) == 0 || len(data) > 20<<20 {
		return "", errors.New("the image must be up to 20 MB")
	}
	base := safeFileName(strings.TrimSuffix(filepath.Base(name), filepath.Ext(name)))
	if base == "" || strings.EqualFold(base, "image") {
		base = map[string]string{"pt": "Imagem ", "es": "Imagen "}[site.Lang]
		if base == "" {
			base = "Image "
		}
		base += time.Now().Format("2006-01-02 150405")
	}
	folder := attachmentsFolder(site)
	dir := filepath.Join(site.ContentDir(), folder)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if !Within(site.ContentDir(), dir) {
		return "", errors.New("the attachments folder is outside the notes folder")
	}
	file := base + ext
	for i := 2; ; i++ {
		if _, err := os.Lstat(filepath.Join(dir, file)); errors.Is(err, os.ErrNotExist) {
			break
		}
		file = fmt.Sprintf("%s %d%s", base, i, ext)
	}
	if err := writeNew(filepath.Join(dir, file), data); err != nil {
		return "", err
	}
	rel, err := filepath.Rel(filepath.FromSlash(path.Dir("/"+note)), filepath.FromSlash("/"+folder+"/"+file))
	if err != nil {
		return "", err
	}
	link := strings.ReplaceAll(filepath.ToSlash(rel), " ", "%20")
	return "![" + base + "](" + link + ")", nil
}
