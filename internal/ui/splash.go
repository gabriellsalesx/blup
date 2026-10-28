package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/term"
)

// The mascot drawn by its outline, with eyes and a small mouth, and the name
// in ASCII letters. @ marks an eye; the blink redraws those two rows.
var mascot = []string{
	`    ╭──────────────────╮    `,
	`  ╭─╯                  ╰─╮  `,
	`  │                      │  `,
	`  │          @    @      │  `,
	`  │          @    @      │  `,
	`  │           ╰──╯       │  `,
	`  ╰─╮                  ╭─╯  `,
	`    ╰──────────────────╯    `,
}

var wordmark = []string{
	` _     _             `,
	`| |__ | |_   _ _ __  `,
	`| '_ \| | | | | '_ \ `,
	`| |_) | | |_| | |_) |`,
	`|_.__/|_|\__,_| .__/ `,
	`              |_|    `,
}

const (
	splashWidth = 30
	eyeRow      = 3 // first of the two eye rows in mascot
)

var (
	blupBlue  = lipgloss.Color("#6fc0ee")
	outlineSt = lipgloss.NewStyle().Foreground(blupBlue)
	eyeSt     = lipgloss.NewStyle().Bold(true)
	wordSt    = lipgloss.NewStyle().Foreground(blupBlue).Bold(true)
)

// Splash opens the menu: the mascot, the name and the credit line. Small
// terminals get the plain banner instead.
func Splash(credit string) {
	out := os.Stdout
	tty := term.IsTerminal(out.Fd())
	if w, _, err := term.GetSize(out.Fd()); tty && err == nil && w < splashWidth+4 {
		Banner("blup · " + credit)
		return
	}

	center := func(s string, width int) string {
		pad := (splashWidth - width) / 2
		if pad < 0 {
			pad = 0
		}
		return strings.Repeat(" ", pad) + s
	}
	face := func(i int, closed bool) string {
		row := mascot[i]
		eye := eyeSt.Render("█")
		if closed {
			eye = eyeSt.Render("─")
			if i == eyeRow+1 {
				eye = " "
			}
		}
		parts := strings.Split(row, "@")
		for j := range parts {
			parts[j] = outlineSt.Render(parts[j])
		}
		return "  " + strings.Join(parts, eye)
	}

	fmt.Println()
	for i := range mascot {
		fmt.Println(face(i, false))
	}
	fmt.Println()
	for _, l := range wordmark {
		fmt.Println("  " + center(wordSt.Render(l), len(l)))
	}
	fmt.Println()
	fmt.Println("  " + center(hint.Render(credit), len([]rune(credit))))
	fmt.Println()

	if !tty || os.Getenv("NO_COLOR") != "" {
		return
	}
	// One blink: go back up to the eye rows, close, wait, open.
	below := len(mascot) - eyeRow + 1 + len(wordmark) + 3
	blink := func(closed bool) {
		fmt.Printf("\x1b[%dA\r", below)
		fmt.Print(face(eyeRow, closed) + "\x1b[K\n")
		fmt.Print(face(eyeRow+1, closed) + "\x1b[K\n")
		fmt.Printf("\x1b[%dB\r", below-2)
	}
	time.Sleep(450 * time.Millisecond)
	blink(true)
	time.Sleep(140 * time.Millisecond)
	blink(false)
}
