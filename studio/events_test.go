package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Every event the Studio emits is registered: an unregistered one reaches
// the window untyped (any in the bindings). And the calendar's goes as a
// CalendarOpen: Wails cancels an event whose data isn't the registered type.
func TestEmittedEventsAreRegistered(t *testing.T) {
	files, _ := filepath.Glob("*.go")
	fset := token.NewFileSet()
	registered := map[string]bool{}
	emitted := map[string]ast.Expr{}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(fset, f, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			name, _ := strconv.Unquote(lit.Value)
			fun := call.Fun
			if ix, ok := fun.(*ast.IndexExpr); ok {
				fun = ix.X
			}
			sel, ok := fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			switch sel.Sel.Name {
			case "RegisterEvent":
				registered[name] = true
			case "Emit":
				var arg ast.Expr
				if len(call.Args) > 1 {
					arg = call.Args[1]
				}
				emitted[name] = arg
			}
			return true
		})
	}
	if len(emitted) == 0 {
		t.Fatal("no Emit found: the test reads the wrong folder")
	}
	for name := range emitted {
		if !registered[name] {
			t.Errorf("event %q is emitted but not registered (application.RegisterEvent)", name)
		}
	}
	call, ok := emitted["calendarOpen"].(*ast.CallExpr)
	if id, _ := func() (*ast.Ident, bool) {
		if !ok {
			return nil, false
		}
		id, ok := call.Fun.(*ast.Ident)
		return id, ok
	}(); id == nil || id.Name != "calendarOpenData" {
		t.Error(`calendarOpen must be emitted as calendarOpenData(...) (a CalendarOpen)`)
	}
}

func TestCalendarOpenData(t *testing.T) {
	got := calendarOpenData(map[string]any{"dir": `C:\sites\a`, "path": "agenda/x.md", "date": "2026-11-02", "extra": 1})
	if got != (CalendarOpen{Dir: `C:\sites\a`, Path: "agenda/x.md", Date: "2026-11-02"}) {
		t.Errorf("got %+v", got)
	}
	if got := calendarOpenData(map[string]any{"dir": 3, "date": nil}); got != (CalendarOpen{}) {
		t.Errorf("non-text values: got %+v", got)
	}
	if got := calendarOpenData(nil); got != (CalendarOpen{}) {
		t.Errorf("nil: got %+v", got)
	}
}
