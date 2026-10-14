// Package ui formats output and asks questions in the terminal.
package ui

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
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

// Confirm asks a yes/no question. Anything but y/yes is no.
func Confirm(in *bufio.Reader, out io.Writer, question string) (bool, error) {
	fmt.Fprintf(out, "%s [y/N] ", question)
	answer, err := in.ReadString('\n')
	if err != nil && answer == "" {
		fmt.Fprintln(out)
		return false, ErrNoAnswer
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes", nil
}

// Choose shows a numbered menu and returns the chosen index (0-based).
// It asks again until the answer is valid, or returns ErrNoAnswer when input ends.
func Choose(in *bufio.Reader, out io.Writer, options [][]string) (int, error) {
	rows := make([][]string, len(options))
	for i, opt := range options {
		rows[i] = append([]string{strconv.Itoa(i+1) + ")"}, opt...)
	}
	Table(out, rows)
	for {
		fmt.Fprint(out, "  Choice: ")
		answer, err := in.ReadString('\n')
		n, convErr := strconv.Atoi(strings.TrimSpace(answer))
		if convErr == nil && n >= 1 && n <= len(options) {
			return n - 1, nil
		}
		if err != nil {
			fmt.Fprintln(out)
			return 0, ErrNoAnswer
		}
		fmt.Fprintf(out, "  Enter a number between 1 and %d.\n", len(options))
	}
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

// IsTerminal reports whether f is an interactive terminal.
func IsTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
