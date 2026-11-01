package main

import (
	"encoding/base64"
	"errors"
	"mime"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gabriellsalesx/blup/internal/app"
)

// Boards: the two kinds of file a vault of notes has besides notes, both
// the Studio's own (drawn in its window, saved as JSON):
//
//   - a canvas (.canvas, in the open JSON Canvas format): cards, notes and
//     links on an endless board, joined by arrows;
//   - a drawing (.draw): shapes, arrows, text and free strokes;
//   - a database (.table): its records are notes (app/database.go).

// Kinds of a Node that isn't a note or a folder.
const (
	KindCanvas   = "canvas"
	KindDrawing  = "drawing"
	KindDatabase = "database"
)

// boardKind is the kind of a file name, "" for a note or anything else.
func boardKind(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.HasSuffix(n, ".canvas"):
		return KindCanvas
	case strings.HasSuffix(n, ".draw"):
		return KindDrawing
	case strings.HasSuffix(n, ".table"):
		return KindDatabase
	}
	return ""
}

// boardName is a board's name without its extension.
func boardName(name string) string {
	n := strings.ToLower(name)
	for _, ext := range []string{".draw", ".canvas", ".table"} {
		if strings.HasSuffix(n, ext) {
			return name[:len(name)-len(ext)]
		}
	}
	return name
}

// boardPath is a board of the open vault, refusing anything else.
func (s *Studio) boardPath(rel string) (string, error) {
	if boardKind(rel) == "" {
		return "", errors.New("not a canvas, a drawing or a database")
	}
	return s.notePath(rel)
}

// ReadBoard returns a canvas's or a drawing's file as it is.
func (s *Studio) ReadBoard(rel string) (string, error) {
	p, err := s.boardPath(rel)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(p)
	return string(b), err
}

// SaveBoard saves a canvas or a drawing.
func (s *Studio) SaveBoard(rel, text string) error {
	p, err := s.boardPath(rel)
	if err != nil {
		return err
	}
	return os.WriteFile(p, []byte(text), 0o644)
}

// An empty canvas and an empty drawing.
const (
	emptyCanvas  = "{\n\t\"nodes\":[],\n\t\"edges\":[]\n}"
	emptyDrawing = "{\n\t\"version\":1,\n\t\"elements\":[]\n}"
)

// CreateCanvas makes a canvas named name in folder and returns its path.
func (s *Studio) CreateCanvas(folder, name string) (string, error) {
	return s.createFile(folder, name, ".canvas", emptyCanvas)
}

// CreateDrawing makes a drawing named name in folder and returns its path.
func (s *Studio) CreateDrawing(folder, name string) (string, error) {
	return s.createFile(folder, name, ".draw", emptyDrawing)
}

// SaveBoardImage keeps an image pasted on a board (or a drawing exported
// as an image) in the attachments folder, its bytes in base64, and returns
// its path in the vault.
func (s *Studio) SaveBoardImage(board, name, data string) (string, error) {
	site, err := s.current()
	if err != nil {
		return "", err
	}
	b, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", err
	}
	md, err := app.PasteImage(site, board, name, b)
	if err != nil {
		return "", err
	}
	// ![name](link), the link relative to the board.
	link := mdLink.FindStringSubmatch(md)
	if link == nil {
		return "", errors.New("unexpected image link")
	}
	rel, err := url.PathUnescape(link[1])
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(path.Join(path.Dir("/"+board), rel), "/"), nil
}

var mdLink = regexp.MustCompile(`\]\(([^)]+)\)$`)

// BoardFile is a file of the vault as a data: URL: an image on a board.
func (s *Studio) BoardFile(rel string) (string, error) {
	p, err := s.notePath(rel)
	if err != nil {
		return "", err
	}
	kind := mime.TypeByExtension(strings.ToLower(filepath.Ext(p)))
	if !strings.HasPrefix(kind, "image/") {
		return "", errors.New("not an image")
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return "data:" + kind + ";base64," + base64.StdEncoding.EncodeToString(b), nil
}
