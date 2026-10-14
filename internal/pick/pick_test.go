package pick

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
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
