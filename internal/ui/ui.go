// Package ui wraps the terminal prompts so every question in the CLI looks
// the same and speaks the user's language.
package ui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"charm.land/huh/v2"
	"charm.land/huh/v2/spinner"
	"charm.land/lipgloss/v2"

	"github.com/gabriellsalesx/blup/internal/i18n"
)

// ErrAborted is returned when the user leaves a prompt with Ctrl+C or Esc.
var ErrAborted = huh.ErrUserAborted

var (
	accent  = lipgloss.Color("#a78bfa")
	green   = lipgloss.Color("#4ade80")
	yellow  = lipgloss.Color("#facc15")
	red     = lipgloss.Color("#f87171")
	muted   = lipgloss.Color("#a1a1aa")
	title   = lipgloss.NewStyle().Bold(true).Foreground(accent)
	okStyle = lipgloss.NewStyle().Foreground(green)
	warn    = lipgloss.NewStyle().Foreground(yellow)
	errS    = lipgloss.NewStyle().Foreground(red).Bold(true)
	hint    = lipgloss.NewStyle().Foreground(muted)
	box     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent).Padding(0, 2)
)

// Banner prints a framed heading.
func Banner(text string) { fmt.Println(box.Render(title.Render(text))) }

// Step prints a section heading.
func Step(text string) { fmt.Println("\n" + title.Render("› "+text)) }

// OK prints a success line.
func OK(text string) { fmt.Println(okStyle.Render("✓ ") + text) }

// Warn prints a warning line.
func Warn(text string) { fmt.Println(warn.Render("! ") + text) }

// Error prints an error line.
func Error(text string) { fmt.Fprintln(os.Stderr, errS.Render("✗ ")+text) }

// Hint prints muted helper text.
func Hint(text string) { fmt.Println(hint.Render(text)) }

// Highlight returns text in the accent colour.
func Highlight(text string) string { return title.Render(text) }

func keymap() *huh.KeyMap {
	k := huh.NewDefaultKeyMap()
	k.Select.Up.SetHelp("↑", i18n.T("key.up"))
	k.Select.Down.SetHelp("↓", i18n.T("key.down"))
	k.Select.Submit.SetHelp("enter", i18n.T("key.choose"))
	k.Select.Next.SetHelp("enter", i18n.T("key.choose"))
	k.Select.Filter.SetHelp("/", i18n.T("key.filter"))
	k.Input.Submit.SetHelp("enter", i18n.T("key.continue"))
	k.Input.Next.SetHelp("enter", i18n.T("key.continue"))
	k.Confirm.Toggle.SetHelp("←/→", i18n.T("key.toggle"))
	k.Confirm.Submit.SetHelp("enter", i18n.T("key.confirm"))
	k.Confirm.Next.SetHelp("enter", i18n.T("key.confirm"))
	k.MultiSelect.Toggle.SetHelp("x", i18n.T("key.tick"))
	k.MultiSelect.Up.SetHelp("↑", i18n.T("key.up"))
	k.MultiSelect.Down.SetHelp("↓", i18n.T("key.down"))
	k.MultiSelect.Filter.SetHelp("/", i18n.T("key.filter"))
	k.MultiSelect.SelectAll.SetHelp("ctrl+a", i18n.T("key.all"))
	k.MultiSelect.SelectNone.SetHelp("ctrl+a", i18n.T("key.none"))
	k.MultiSelect.Submit.SetHelp("enter", i18n.T("key.confirm"))
	k.MultiSelect.Next.SetHelp("enter", i18n.T("key.confirm"))
	k.Note.Submit.SetHelp("enter", i18n.T("key.continue"))
	k.Note.Next.SetHelp("enter", i18n.T("key.continue"))
	k.Quit.SetHelp("ctrl+c", i18n.T("key.quit"))
	return k
}

func run(fields ...huh.Field) error {
	return huh.NewForm(huh.NewGroup(fields...)).
		WithTheme(huh.ThemeFunc(huh.ThemeCharm)).
		WithKeyMap(keymap()).
		Run()
}

// Choose asks the user to pick one option.
func Choose[T comparable](question, help string, value *T, options ...huh.Option[T]) error {
	s := huh.NewSelect[T]().Title(question).Options(options...).Value(value)
	if help != "" {
		s.Description(help)
	}
	if len(options) > 8 {
		s.Height(10)
	}
	return run(s)
}

// ChooseMany asks the user to tick several options.
func ChooseMany[T comparable](question, help string, value *[]T, options ...huh.Option[T]) error {
	m := huh.NewMultiSelect[T]().Title(question).Options(options...).Value(value).Height(min(len(options), 10) + 6)
	if help != "" {
		m.Description(help)
	}
	return run(m)
}

// Option builds a choice.
func Option[T comparable](label string, value T) huh.Option[T] { return huh.NewOption(label, value) }

// Ask asks for a line of text. validate may be nil.
func Ask(question, help, placeholder string, value *string, validate func(string) error) error {
	in := huh.NewInput().Title(question).Placeholder(placeholder).Value(value)
	if help != "" {
		in.Description(help)
	}
	if validate != nil {
		in.Validate(validate)
	}
	return run(in)
}

// AskSecret asks for a token without echoing it.
func AskSecret(question, help string, value *string) error {
	in := huh.NewInput().Title(question).Description(help).Value(value).
		EchoMode(huh.EchoModePassword).Validate(huh.ValidateNotEmpty())
	return run(in)
}

// Confirm asks a yes/no question; def is the preselected answer.
func Confirm(question, help string, def bool) (bool, error) {
	v := def
	c := huh.NewConfirm().Title(question).Affirmative(i18n.T("yes")).Negative(i18n.T("no")).Value(&v)
	if help != "" {
		c.Description(help)
	}
	err := run(c)
	return v, err
}

// Explain shows a block of text and waits for Enter.
func Explain(heading, body string) error {
	return run(huh.NewNote().Title(heading).Description(body).Next(true).NextLabel(i18n.T("key.continue")))
}

// Spin runs fn with a spinner titled msg (a plain line without a terminal).
func Spin(ctx context.Context, msg string, fn func(ctx context.Context) error) error {
	if !Interactive() {
		fmt.Println("… " + msg)
		return fn(ctx)
	}
	return spinner.New().Context(ctx).Title(" " + msg).ActionWithErr(fn).Run()
}

// Interactive reports whether stdin and stdout are terminals.
func Interactive() bool {
	for _, f := range []*os.File{os.Stdin, os.Stdout} {
		st, err := f.Stat()
		if err != nil || st.Mode()&os.ModeCharDevice == 0 {
			return false
		}
	}
	return true
}

// Aborted reports whether err means the user gave up on a prompt.
func Aborted(err error) bool { return errors.Is(err, huh.ErrUserAborted) }

// OpenBrowser opens url in the default browser, ignoring failures.
func OpenBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
