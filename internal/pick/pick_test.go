package pick

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/charmbracelet/huh"
)

// Keys as a terminal sends them.
const (
	enter = "\r"
	down  = "\x1b[B"
	esc   = "\x1b"
	ctrlC = "\x03"
)

var three = []Choice{{"alpha", true}, {"beta", false}, {"gamma", false}}

func many(input string, choices []Choice) ([]int, error) {
	return Huh{In: strings.NewReader(input), Out: &bytes.Buffer{}}.Many("Select", choices)
}

func one(input string, choices []Choice) (int, error) {
	return Huh{In: strings.NewReader(input), Out: &bytes.Buffer{}}.One("Switch to", choices)
}

func TestManyKeys(t *testing.T) {
	cases := map[string][]int{
		enter:              {0},       // pre-checked choices
		"a" + enter:        {0, 1, 2}, // "a" checks all
		" " + enter:        {},        // space unchecks the first
		down + " " + enter: {0, 1},    // space checks the second
	}
	for input, want := range cases {
		got, err := many(input, three)
		if err != nil || fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("%q: want %v, got %v (%v)", input, want, got, err)
		}
	}
}

// keyReader returns one key per Read, like a terminal does when keys are typed one by one.
// A lone esc is only recognized when it arrives on its own.
type keyReader struct{ keys []string }

func (r *keyReader) Read(p []byte) (int, error) {
	if len(r.keys) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.keys[0])
	r.keys = r.keys[1:]
	return n, nil
}

func TestEscClearsAnAppliedFilterFirst(t *testing.T) {
	// Many: enter or esc ends the typing and keeps the filter; esc then clears it.
	got, err := Huh{In: &keyReader{[]string{"/", "b", "e", enter, esc, enter}}, Out: &bytes.Buffer{}}.Many("Select", three)
	if err != nil || fmt.Sprint(got) != "[0]" {
		t.Errorf("Many: want [0], got %v (%v)", got, err)
	}
	// One: esc ends the typing, a second esc clears the filter.
	i, err := Huh{In: &keyReader{[]string{"/", "g", esc, esc, enter}}, Out: &bytes.Buffer{}}.One("Switch to", three)
	if err != nil || i != 0 {
		t.Errorf("One: want 0, got %d (%v)", i, err)
	}
}

func TestEscAfterAnEmptyFilterCancels(t *testing.T) {
	_, err := Huh{In: &keyReader{[]string{"/", esc, esc}}, Out: &bytes.Buffer{}}.Many("Select", three)
	if err != ErrCancelled {
		t.Errorf("want ErrCancelled, got %v", err)
	}
}

func TestManyShowsTheCounterFromTheStart(t *testing.T) {
	// With nothing checked, huh never computes a dynamic title by itself.
	var picked []int
	field := manyField("Select", []Choice{{"alpha", false}, {"beta", false}}, &picked)
	field.WithTheme(huh.ThemeCharm())
	if view := field.View(); !strings.Contains(view, "Select  0 of 2 selected") {
		t.Errorf("title missing:\n%s", view)
	}
}

func TestOneKeys(t *testing.T) {
	got, err := one(down+enter, three)
	if err != nil || got != 1 {
		t.Errorf("want 1, got %d (%v)", got, err)
	}
}

func TestCancel(t *testing.T) {
	for _, input := range []string{esc, ctrlC} {
		if _, err := many(input, three); err != ErrCancelled {
			t.Errorf("Many %q: want ErrCancelled, got %v", input, err)
		}
		if _, err := one(input, three); err != ErrCancelled {
			t.Errorf("One %q: want ErrCancelled, got %v", input, err)
		}
	}
}
