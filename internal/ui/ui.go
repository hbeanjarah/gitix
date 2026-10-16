// Package ui formats output and asks questions in the terminal.
package ui

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"
	"unicode"
)

// ErrNoAnswer means the input ended before a valid answer was given.
var ErrNoAnswer = errors.New("no answer")

// RelTime renders how long ago t was, relative to now ("3 days ago").
func RelTime(t, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return ago(int(d/time.Minute), "minute")
	case d < 24*time.Hour:
		return ago(int(d/time.Hour), "hour")
	case d < 30*24*time.Hour:
		return ago(int(d/(24*time.Hour)), "day")
	case d < 365*24*time.Hour:
		return ago(int(d/(30*24*time.Hour)), "month")
	default:
		return ago(int(d/(365*24*time.Hour)), "year")
	}
}

func ago(n int, unit string) string {
	if n == 1 {
		return "1 " + unit + " ago"
	}
	return fmt.Sprintf("%d %ss ago", n, unit)
}

// Table writes rows as aligned columns, indented by two spaces.
func Table(w io.Writer, rows [][]string) {
	for _, line := range Columns(rows) {
		fmt.Fprintln(w, "  "+line)
	}
}

// Columns aligns rows into lines, with at least three spaces between columns.
func Columns(rows [][]string) []string {
	if len(rows) == 0 {
		return nil
	}
	var b strings.Builder
	tw := tabwriter.NewWriter(&b, 0, 0, 3, ' ', 0)
	for _, row := range rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	tw.Flush()
	return strings.Split(strings.TrimSuffix(b.String(), "\n"), "\n")
}

// Confirm asks a yes/no question whose default is no. Anything but y/yes is no.
func Confirm(in *bufio.Reader, out io.Writer, question string) (bool, error) {
	answer, err := ask(in, out, question+" [y/N] ")
	return answer == "y" || answer == "yes", err
}

// ConfirmYes asks a yes/no question whose default is yes: an empty answer is yes,
// anything but y/yes is no. The end of input is still no.
func ConfirmYes(in *bufio.Reader, out io.Writer, question string) (bool, error) {
	answer, err := ask(in, out, question+" [Y/n] ")
	return err == nil && (answer == "" || answer == "y" || answer == "yes"), err
}

// ask prints prompt and returns the answer, trimmed and lowercased.
func ask(in *bufio.Reader, out io.Writer, prompt string) (string, error) {
	fmt.Fprint(out, prompt)
	answer, err := in.ReadString('\n')
	if err != nil && answer == "" {
		fmt.Fprintln(out)
		return "", ErrNoAnswer
	}
	return strings.ToLower(strings.TrimSpace(answer)), nil
}

// Safe escapes control and invisible format characters, such as the bidi override
// U+202E, so a branch name cannot hide or reorder what the terminal shows.
func Safe(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			fmt.Fprintf(&b, "\\u%04X", r)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Painter adds ANSI colors when enabled.
type Painter struct {
	Enabled bool
}

// NewPainter enables colors when f is a terminal and NO_COLOR is not set.
func NewPainter(f *os.File) Painter {
	_, noColor := os.LookupEnv("NO_COLOR")
	return Painter{Enabled: IsTerminal(f) && !noColor}
}

func (p Painter) paint(code, s string) string {
	if !p.Enabled {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func (p Painter) Green(s string) string  { return p.paint("32", s) }
func (p Painter) Yellow(s string) string { return p.paint("33", s) }
func (p Painter) Dim(s string) string    { return p.paint("2", s) }
func (p Painter) Cyan(s string) string   { return p.paint("36", s) }

// IsTerminal reports whether f is an interactive terminal.
func IsTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
