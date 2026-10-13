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

func TestChooseRetriesUntilValid(t *testing.T) {
	var out bytes.Buffer
	in := bufio.NewReader(strings.NewReader("abc\n9\n2\n"))
	got, err := Choose(in, &out, [][]string{{"a"}, {"b"}})
	if err != nil || got != 1 {
		t.Errorf("want 1, got %d (%v)", got, err)
	}
	if strings.Count(out.String(), "Enter a number between 1 and 2.") != 2 {
		t.Errorf("want two retry messages, got:\n%s", out.String())
	}
}

func TestChooseEndOfInput(t *testing.T) {
	_, err := Choose(bufio.NewReader(strings.NewReader("")), &bytes.Buffer{}, [][]string{{"a"}})
	if err != ErrNoAnswer {
		t.Errorf("got %v", err)
	}
}

func TestPainterDisabledIsPlain(t *testing.T) {
	if got := (Painter{}).Yellow("x"); got != "x" {
		t.Errorf("got %q", got)
	}
}
