package ui

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestRelTime(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	cases := map[time.Duration]string{
		10 * time.Second:     "just now",
		time.Minute:          "1 minute ago",
		5 * time.Hour:        "5 hours ago",
		3 * 24 * time.Hour:   "3 days ago",
		65 * 24 * time.Hour:  "2 months ago",
		800 * 24 * time.Hour: "2 years ago",
	}
	for d, want := range cases {
		if got := RelTime(now.Add(-d), now); got != want {
			t.Errorf("%v: want %q, got %q", d, want, got)
		}
	}
}

func TestConfirm(t *testing.T) {
	cases := map[string]bool{"y\n": true, "YES\n": true, "n\n": false, "\n": false, "whatever\n": false}
	for input, want := range cases {
		got, err := Confirm(bufio.NewReader(strings.NewReader(input)), &bytes.Buffer{}, "Delete?")
		if err != nil || got != want {
			t.Errorf("%q: want %v, got %v (%v)", input, want, got, err)
		}
	}
}

func TestConfirmEndOfInputIsNo(t *testing.T) {
	got, err := Confirm(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{}, "Delete?")
	if got || err != ErrNoAnswer {
		t.Errorf("want false, ErrNoAnswer; got %v, %v", got, err)
	}
}

func TestConfirmYes(t *testing.T) {
	cases := map[string]bool{"\n": true, "y\n": true, "YES\n": true, "n\n": false, "no\n": false, "whatever\n": false}
	for input, want := range cases {
		var out bytes.Buffer
		got, err := ConfirmYes(bufio.NewReader(strings.NewReader(input)), &out, "Delete?")
		if err != nil || got != want {
			t.Errorf("%q: want %v, got %v (%v)", input, want, got, err)
		}
		if out.String() != "Delete? [Y/n] " {
			t.Errorf("prompt: got %q", out.String())
		}
	}
}

func TestConfirmYesEndOfInputIsNo(t *testing.T) {
	got, err := ConfirmYes(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{}, "Delete?")
	if got || err != ErrNoAnswer {
		t.Errorf("want false, ErrNoAnswer; got %v, %v", got, err)
	}
}

func TestColumns(t *testing.T) {
	got := Columns([][]string{{"feat/a", "merged", "2 days ago"}, {"feat/longer", "3 commits", "1 year ago"}})
	want := []string{
		"feat/a        merged      2 days ago",
		"feat/longer   3 commits   1 year ago",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("want\n%s\ngot\n%s", strings.Join(want, "\n"), strings.Join(got, "\n"))
	}
	if Columns(nil) != nil {
		t.Errorf("no rows should give no lines")
	}
}

func TestPainterDisabledIsPlain(t *testing.T) {
	if got := (Painter{}).Yellow("x"); got != "x" {
		t.Errorf("got %q", got)
	}
}

func TestSafeEscapesInvisibleCharacters(t *testing.T) {
	cases := map[string]string{
		"feat/été":        "feat/été",
		"feat/x\u202eexe": `feat/x\u202Eexe`,
		"a\u200bb":        `a\u200Bb`,
		"bell\a":          `bell\u0007`,
	}
	for in, want := range cases {
		if got := Safe(in); got != want {
			t.Errorf("Safe(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSpinnerDrawsAtOnceAndErasesOnStop(t *testing.T) {
	var buf bytes.Buffer
	s := StartSpinner(&buf, Painter{}, "Fetching origin…")
	s.Set("Analyzing branches  1/2")
	s.Stop()
	out := buf.String()
	if !strings.Contains(out, frames[0]+" Fetching origin…") || !strings.Contains(out, "Analyzing branches  1/2") {
		t.Errorf("both texts must be drawn right away:\n%q", out)
	}
	if !strings.HasSuffix(out, "\r\x1b[2K") {
		t.Errorf("Stop must erase the line:\n%q", out)
	}
}

func TestSpinnerShowsElapsedTimeWhenSlow(t *testing.T) {
	var buf bytes.Buffer
	now := time.Unix(1_700_000_000, 0)
	s := startSpinner(&buf, Painter{}, "Fetching origin…", func() time.Time { return now })
	now = now.Add(2 * time.Second)
	s.Set("Fetching origin…")
	if strings.Contains(buf.String(), "2s") {
		t.Errorf("no time shown before 3s:\n%q", buf.String())
	}
	now = now.Add(2 * time.Second)
	s.Set("Fetching origin…")
	s.Stop()
	if !strings.Contains(buf.String(), "Fetching origin… 4s") {
		t.Errorf("want the elapsed time:\n%q", buf.String())
	}
}
