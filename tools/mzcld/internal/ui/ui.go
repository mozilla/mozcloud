// Package ui provides consistent terminal output helpers for mzcld commands.
package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var debug bool

// SetDebug enables or disables debug output.
func SetDebug(v bool) { debug = v }

// IsDebug reports whether debug mode is active.
func IsDebug() bool { return debug }

var (
	green  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	yellow = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	red    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	gray   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	bold   = lipgloss.NewStyle().Bold(true)
)

// Header prints a bold section heading.
func Header(msg string) {
	fmt.Println("\n" + bold.Render(msg))
}

// Success prints a green check line.
func Success(msg string) {
	fmt.Println(green.Render("  ✓ ") + msg)
}

// Warn prints a yellow warning line.
func Warn(msg string) {
	fmt.Println(yellow.Render("  ! ") + msg)
}

// Error prints a red error line to stderr.
func Error(msg string) {
	fmt.Fprintln(os.Stderr, red.Render("  ✗ ") + msg)
}

// Info prints a plain info line.
func Info(msg string) {
	fmt.Println("    " + msg)
}

// Dim prints a dimmed line.
func Dim(msg string) {
	fmt.Println(gray.Render("    " + msg))
}

// Debug prints a line only when debug mode is enabled.
func Debug(msg string) {
	if debug {
		fmt.Println(gray.Render("  … " + msg))
	}
}

// Status prints a transient status line to stderr that is cleared on the next
// call to ClearStatus or when a TUI form takes over the terminal. Safe for
// piping — never touches stdout.
func Status(msg string) {
	fmt.Fprintf(os.Stderr, "\r\033[K%s", gray.Render("  ⠋ "+msg))
}

// ClearStatus erases the current status line on stderr.
func ClearStatus() {
	fmt.Fprintf(os.Stderr, "\r\033[K")
}

// CheckResult represents the outcome of a single preflight check.
type CheckResult struct {
	Name    string
	Version string // populated on success
	Fix     string // install hint on failure
	OK      bool
	Warn    bool // true = optional, not a hard failure
}

// PrintChecks renders a table of check results and returns the number of failures.
func PrintChecks(checks []CheckResult) (failures int) {
	nameWidth := 0
	for _, c := range checks {
		if len(c.Name) > nameWidth {
			nameWidth = len(c.Name)
		}
	}

	fmt.Println()
	for _, c := range checks {
		pad := strings.Repeat(" ", nameWidth-len(c.Name)+2)
		switch {
		case c.OK:
			fmt.Printf("  %s %s%s%s\n",
				green.Render("✓"),
				bold.Render(c.Name), pad,
				gray.Render(c.Version))
		case c.Warn:
			fmt.Printf("  %s %s%s%s\n",
				yellow.Render("!"),
				bold.Render(c.Name), pad,
				yellow.Render("not found  →  "+c.Fix))
		default:
			fmt.Printf("  %s %s%s%s\n",
				red.Render("✗"),
				bold.Render(c.Name), pad,
				red.Render("not found  →  "+c.Fix))
			failures++
		}
	}
	fmt.Println()
	return failures
}
