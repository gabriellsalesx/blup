//go:build linux && !android

// Command storepreview launches the real desktop UI with disposable example
// notes and isolated configuration. It never copies the person's vaults or
// credentials. Screenshot files are reviewed separately before publication.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/gabriellsalesx/blup/internal/app"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/i18n"
)

var sessionKeys = []string{"DISPLAY", "WAYLAND_DISPLAY", "XDG_RUNTIME_DIR", "XDG_SESSION_TYPE", "DBUS_SESSION_BUS_ADDRESS", "HYPRLAND_INSTANCE_SIGNATURE"}

type previewState struct {
	PID      int    `json:"pid"`
	Root     string `json:"root"`
	View     string `json:"view"`
	Instance string `json:"flatpakInstance,omitempty"`
}

func main() {
	studio := flag.String("studio", "bin/blup-studio-store-preview", "isolated preview executable")
	view := flag.String("view", envDefault("BLUP_PREVIEW_VIEW", "notes"), "notes, site or library")
	theme := flag.String("theme", envDefault("BLUP_PREVIEW_THEME", "light"), "light or dark")
	session := flag.String("session", os.Getenv("BLUP_PREVIEW_SESSION"), "optional local desktop-session file (never committed)")
	capture := flag.Bool("capture", false, "capture only the isolated preview window on Hyprland")
	inspect := flag.Bool("inspect", false, "inspect only Studio window identity and preview process metadata")
	exercise := flag.Bool("exercise", false, "exercise editing, note creation and the terminal in the isolated Flatpak")
	probe := flag.Bool("probe", false, "check sandbox features from inside the disposable Flatpak")
	probeDir := flag.String("probe-dir", "", "disposable examples directory for the sandbox probe")
	checkSandbox := flag.Bool("check-sandbox", false, "run the compiled sandbox probe in the preview installation")
	stop := flag.Bool("stop", false, "stop only this project's dedicated preview executable")
	bundle := flag.String("flatpak", os.Getenv("BLUP_PREVIEW_FLATPAK"), "launch this Flatpak bundle with disposable example data")
	runtimeDir := flag.String("runtime-dir", os.Getenv("BLUP_FLATPAK_RUNTIME_DIR"), "private runtime installation for the Flatpak preview")
	output := flag.String("output", envDefault("BLUP_PREVIEW_OUTPUT", "bin/store-preview.png"), "PNG capture destination")
	flag.Parse()
	var err error
	if *probe {
		err = probeSandbox(*probeDir)
	} else if *checkSandbox {
		err = checkPreviewSandbox(*session, *runtimeDir)
	} else if *stop {
		b, _ := os.ReadFile("bin/store-preview.json")
		var state previewState
		_ = json.Unmarshal(b, &state)
		if state.Instance != "" {
			err = stopFlatpakPreview(state, *session)
		} else {
			err = stopPreview(*studio)
		}
	} else if *exercise {
		err = exerciseFlatpak(*session)
	} else if *inspect {
		err = inspectPreview(*session)
	} else if *capture {
		err = captureWindow(*session, *output)
	} else if *bundle != "" {
		err = runFlatpak(*bundle, *runtimeDir, *view, *theme, *session)
	} else {
		err = run(*studio, *view, *theme, *session)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func envDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func desktopSession(file string) (map[string]string, error) {
	values := map[string]string{}
	for _, key := range sessionKeys {
		if v := os.Getenv(key); v != "" {
			values[key] = v
		}
	}
	if file != "" {
		b, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		var supplied map[string]string
		if err := json.Unmarshal(b, &supplied); err != nil {
			return nil, err
		}
		for _, key := range sessionKeys {
			if v := supplied[key]; v != "" {
				values[key] = v
			}
		}
	}
	if values["DISPLAY"] == "" && values["WAYLAND_DISPLAY"] == "" {
		return nil, errors.New("a Linux desktop session is required for native screenshots")
	}
	return values, nil
}

func run(binary, view, theme, session string) error {
	if view != "notes" && view != "site" && view != "library" {
		return errors.New("invalid preview view")
	}
	if theme != "light" && theme != "dark" {
		return errors.New("invalid preview theme")
	}
	desktop, err := desktopSession(session)
	if err != nil {
		return err
	}
	binary, err = filepath.Abs(binary)
	if err != nil {
		return err
	}
	if filepath.Base(binary) != "blup-studio-store-preview" {
		return errors.New("use the dedicated store-preview executable")
	}
	root, err := os.MkdirTemp("", "blup-store-preview-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(root)
	for _, key := range []string{"XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME"} {
		if err := os.Setenv(key, filepath.Join(root, strings.ToLower(key))); err != nil {
			return err
		}
	}
	if err := prepare(root, view, theme); err != nil {
		return err
	}
	// Start from an allowlist: tokens, proxies, reporting keys and production
	// configuration inherited by the launcher do not enter the demo process.
	env := []string{"PATH=" + os.Getenv("PATH"), "LANG=en_US.UTF-8", "BLUP_NO_UPDATE_CHECK=1"}
	for _, key := range []string{"XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME"} {
		env = append(env, key+"="+os.Getenv(key))
	}
	for _, key := range sessionKeys {
		if key == "HYPRLAND_INSTANCE_SIGNATURE" {
			continue
		}
		if value := desktop[key]; value != "" {
			env = append(env, key+"="+value)
		}
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary)
	cmd.Env, cmd.Dir = env, root
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	state := previewState{PID: cmd.Process.Pid, Root: root, View: view}
	b, err := json.MarshalIndent(state, "", "  ")
	if err == nil {
		err = os.WriteFile("bin/store-preview.json", b, 0o600)
	}
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return err
	}
	fmt.Println("Isolated Studio preview started; PID " + strconv.Itoa(state.PID))
	err = cmd.Wait()
	if ctx.Err() != nil {
		return nil
	}
	return err
}

func prepare(root, view, theme string) error {
	i18n.Set("en")
	notes := filepath.Join(root, "notes")
	if err := os.MkdirAll(notes, 0o700); err != nil {
		return err
	}
	for name, text := range exampleNotes {
		if err := os.WriteFile(filepath.Join(notes, name), []byte(text), 0o600); err != nil {
			return err
		}
	}
	g := config.Global{Lang: "en", ErrorReports: "off"}
	vault, err := app.CreateVault(&g, "Writing desk", notes)
	if err != nil {
		return err
	}
	blogNotes := filepath.Join(root, "journal")
	if err := os.MkdirAll(blogNotes, 0o700); err != nil {
		return err
	}
	article := "---\ntitle: A place for small ideas\ndate: 2026-11-10\ndescription: Turn a folder of notes into a journal of your own.\ntags: [writing, notes]\n---\n# A place for small ideas\n\nSome ideas begin as a sentence in a notebook. Others arrive as a list, a sketch or a question.\n\n## Keep the habit simple\n\nWrite a little, connect related notes and come back when an idea is ready to grow.\n\n> Your notes stay as Markdown files, ready for the next chapter.\n"
	if err := os.WriteFile(filepath.Join(blogNotes, "A place for small ideas.md"), []byte(article), 0o600); err != nil {
		return err
	}
	site, err := app.CreateSite(&g, app.NewSiteOptions{Vault: blogNotes, Dir: filepath.Join(root, "site"), Model: "nori", Title: "Field notes", Description: "A small journal about writing and making things", Author: "Blup", Lang: "en"})
	if err != nil {
		return err
	}
	last, note := vault.Dir(), "Write, connect and publish.md"
	if view == "site" {
		last, note = site.Dir(), "A place for small ideas.md"
	}
	if view == "library" {
		last = ""
	}
	// railNames off: the Studio's default rail (icons only), and no label
	// cut short at the screenshot's width.
	prefs := map[string]any{"theme": theme, "welcomed": true, "cloudTip": true, "shortcutTip": true, "toggleKey": "", "notifications": false, "errorReports": "off", "spellcheck": false, "lastSite": last, "lastNotes": map[string]string{last: note}, "sideWidth": 290, "railNames": false}
	b, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return err
	}
	dir, err := config.Dir()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "studio-prefs.json"), b, 0o600)
}

var exampleNotes = map[string]string{
	"Write, connect and publish.md": "---\ntags: [writing, ideas]\n---\n# Write, connect and publish\n\nA quiet place to turn small ideas into something worth sharing.\n\n## Start with a note\n\nKeep a reading list, plan a project or collect the things you notice along the way. Your words stay in plain Markdown files.\n\n- [x] Make room for a new idea\n- [x] Connect it to your [[Field notes]]\n- [ ] Share the next chapter\n\n## Follow the connections\n\nA link can bring two distant ideas together. Keep useful references in your [[Reading list]] and find them again when you need them.\n\n> The next good idea can begin with one sentence.\n",
	"Field notes.md":                "# Field notes\n\n## A small creative routine\n\n- Capture the idea while it is fresh.\n- Give it a clear title.\n- Link it to an existing note.\n\nReturn to [[Write, connect and publish]] when you are ready to share it.\n",
	"Reading list.md":               "# Reading list\n\n## On the desk\n\n- Essays about writing and attention\n- A notebook of useful questions\n- Stories worth reading twice\n\nKeep the ideas that resonate in [[Field notes]].\n",
}
