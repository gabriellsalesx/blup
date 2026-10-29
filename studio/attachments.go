package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// The Attachments screen: the files of the attachments folder (which the
// file tree hides), with small previews of the images.

// Attachments lists the site's attachments and the notes using each one.
func (s *Studio) Attachments() ([]app.Attachment, error) {
	site, err := s.current()
	if err != nil {
		return nil, err
	}
	return app.ListAttachments(site)
}

// RenameAttachment renames a file and fixes the notes that use it.
func (s *Studio) RenameAttachment(old, name string) (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	return app.RenameAttachment(site, old, name)
}

// DeleteAttachment moves a file to the trash.
// It returns where it went, to undo (RestoreSection).
func (s *Studio) DeleteAttachment(name string) (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	return app.DeleteAttachment(site, name)
}

// AddAttachments asks for files on the computer and copies them in.
func (s *Studio) AddAttachments() ([]string, error) {
	site, err := s.current()
	if err != nil {
		return nil, err
	}
	files, err := application.Get().Dialog.OpenFile().SetTitle("Blup Studio").CanChooseFiles(true).PromptForMultipleSelection()
	if err != nil || len(files) == 0 {
		return nil, err
	}
	return app.AddAttachments(site, files)
}

// OpenAttachment opens a file in the computer's own app for it.
func (s *Studio) OpenAttachment(name string) error {
	p, err := s.attachmentPath(name)
	if err != nil {
		return err
	}
	return openExternal(p)
}

func (s *Studio) attachmentPath(name string) (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	if name != filepath.Base(name) || strings.HasPrefix(name, ".") {
		return "", errors.New("not an attachment")
	}
	return filepath.Join(site.ContentDir(), app.AttachmentsFolder(site), name), nil
}

var thumbs sync.Map // file + modification time -> data URL

// Thumbnail is a small preview of an image attachment, as a data: URL ("" for
// files it can't draw). SVGs come as they are.
func (s *Studio) Thumbnail(name string) (string, error) {
	p, err := s.attachmentPath(name)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(p)
	if err != nil {
		return "", err
	}
	key := p + info.ModTime().String()
	if v, ok := thumbs.Load(key); ok {
		return v.(string), nil
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	out := ""
	if strings.EqualFold(filepath.Ext(p), ".svg") {
		out = "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(b)
	} else if img, _, err := image.Decode(bytes.NewReader(b)); err == nil {
		r := img.Bounds()
		w, h := r.Dx(), r.Dy()
		if max := 360; w > max || h > max {
			if w >= h {
				w, h = max, h*max/w
			} else {
				w, h = w*max/h, max
			}
		}
		dst := image.NewRGBA(image.Rect(0, 0, max(w, 1), max(h, 1)))
		draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, r, draw.Src, nil)
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 78}); err == nil {
			out = "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
		}
	}
	thumbs.Store(key, out)
	return out, nil
}
