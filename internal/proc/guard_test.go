package proc

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Programs that open no console of their own (they hand a file or link to
// the system), or that never run on Windows.
var noConsole = map[string]bool{
	"internal/ui/ui.go":              true, // open, rundll32, xdg-open
	"studio/files.go":                true, // open, rundll32, xdg-open
	"studio/hyprland.go":             true, // Linux only
	"internal/i18n/system_darwin.go": true, // macOS only
}

// Every program the Studio or the CLI starts goes through Hide, or a
// console window pops up on Windows in the middle of the screen.
func TestEveryCommandIsHidden(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, dir := range []string{"internal", "studio", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() && (d.Name() == "node_modules" || d.Name() == "frontend") {
				return filepath.SkipDir
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			if noConsole[rel] {
				return nil
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for i, line := range strings.Split(string(src), "\n") {
				if strings.Contains(line, "exec.Command") && !strings.Contains(line, "proc.Hide(") && !strings.HasPrefix(strings.TrimSpace(line), "//") {
					t.Errorf("%s:%d starts a program without proc.Hide", rel, i+1)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
