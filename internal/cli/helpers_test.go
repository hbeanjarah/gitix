package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/hbeanjarah/gitix/internal/pick"
	"github.com/hbeanjarah/gitix/internal/testrepo"
)

type result struct {
	code     int
	out, err string
}

// fakePicker answers like a user would, and records what it was shown.
type fakePicker struct {
	many    []int // checked indexes returned by Many
	one     int   // index returned by One
	err     error // returned instead, e.g. pick.ErrCancelled
	calls   int
	choices []pick.Choice
	during  func() // runs while the list is open, e.g. to commit from "another terminal"
}

func (p *fakePicker) Many(title string, choices []pick.Choice) ([]int, error) {
	p.calls++
	p.choices = choices
	if p.during != nil {
		p.during()
	}
	return p.many, p.err
}

func (p *fakePicker) One(title string, choices []pick.Choice) (int, error) {
	p.calls++
	p.choices = choices
	return p.one, p.err
}

// labels returns the lines the picker was shown.
func (p *fakePicker) labels() string {
	var lines []string
	for _, c := range p.choices {
		lines = append(lines, c.Label)
	}
	return strings.Join(lines, "\n")
}

// gitix runs the CLI inside repo dir with stdin input; interactive simulates a terminal.
// It fails the test if a list opens: use gitixPick for that.
func gitix(t *testing.T, dir, input string, interactive bool, args ...string) result {
	t.Helper()
	p := &fakePicker{}
	res := gitixRun(t, dir, input, interactive, p, args...)
	if p.calls > 0 {
		t.Errorf("no list should open, got %d", p.calls)
	}
	return res
}

// gitixPick runs the CLI in a simulated terminal whose lists are answered by p.
func gitixPick(t *testing.T, dir, input string, p *fakePicker, args ...string) result {
	t.Helper()
	return gitixRun(t, dir, input, true, p, args...)
}

func gitixRun(t *testing.T, dir, input string, interactive bool, p *fakePicker, args ...string) result {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Run(Options{
		Dir:         dir,
		In:          strings.NewReader(input),
		Out:         &out,
		Err:         &errOut,
		Interactive: interactive,
		Picker:      p,
		Now:         time.Now,
		Version:     "test",
	}, args)
	return result{code, out.String(), errOut.String()}
}

func branchNames(r *testrepo.Repo) string {
	return r.Git("branch", "--format=%(refname:short)")
}

func hasBranch(r *testrepo.Repo, name string) bool {
	for _, b := range strings.Split(branchNames(r), "\n") {
		if b == name {
			return true
		}
	}
	return false
}
