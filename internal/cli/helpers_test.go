package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/hbeanjarah/gitix/internal/testrepo"
)

type result struct {
	code     int
	out, err string
}

// gitix runs the CLI inside repo dir with stdin input; interactive simulates a terminal.
func gitix(t *testing.T, dir, input string, interactive bool, args ...string) result {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Run(Options{
		Dir:         dir,
		In:          strings.NewReader(input),
		Out:         &out,
		Err:         &errOut,
		Interactive: interactive,
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
