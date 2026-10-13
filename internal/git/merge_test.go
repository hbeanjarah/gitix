package git_test

import (
	"testing"

	"github.com/hbeanjarah/gitix/internal/git"
	"github.com/hbeanjarah/gitix/internal/testrepo"
)

func TestResolveBaseUsesOriginHEAD(t *testing.T) {
	r := testrepo.New(t)
	base, err := git.New(git.ExecRunner{Dir: r.Dir}).ResolveBase("")
	if err != nil || base != "origin/main" {
		t.Errorf("got %q, %v", base, err)
	}
}

func TestResolveBaseFallsBackToMain(t *testing.T) {
	r := testrepo.New(t)
	r.Git("remote", "set-head", "origin", "--delete")
	base, err := git.New(git.ExecRunner{Dir: r.Dir}).ResolveBase("")
	if err != nil || base != "main" {
		t.Errorf("got %q, %v", base, err)
	}
}

func TestResolveBaseRejectsUnknownBranch(t *testing.T) {
	r := testrepo.New(t)
	_, err := git.New(git.ExecRunner{Dir: r.Dir}).ResolveBase("nope")
	if err != git.ErrNotFound {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestMergeDetection(t *testing.T) {
	r := testrepo.New(t)
	r.Branch("feat/merged")
	r.Merge("feat/merged")
	r.Branch("feat/squashed")
	r.Commit("second.txt", "two commits on this branch")
	r.SquashMerge("feat/squashed")
	r.Branch("feat/wip")
	r.Git("fetch")
	repo := git.New(git.ExecRunner{Dir: r.Dir})

	cases := []struct {
		branch           string
		ancestor, squash bool
		notIn            int
	}{
		{"feat/merged", true, false, 0},
		{"feat/squashed", false, true, 2},
		{"feat/wip", false, false, 1},
	}
	for _, c := range cases {
		anc, err := repo.IsAncestor(c.branch, "origin/main")
		if err != nil || anc != c.ancestor {
			t.Errorf("%s IsAncestor: got %v, %v", c.branch, anc, err)
		}
		if !c.ancestor {
			sq, err := repo.SquashMerged(c.branch, "origin/main")
			if err != nil || sq != c.squash {
				t.Errorf("%s SquashMerged: got %v, %v", c.branch, sq, err)
			}
		}
		n, err := repo.CommitsNotIn(c.branch, "origin/main")
		if err != nil || n != c.notIn {
			t.Errorf("%s CommitsNotIn: got %d, %v", c.branch, n, err)
		}
	}
}

func TestSquashMergedUnrelatedHistory(t *testing.T) {
	r := testrepo.New(t)
	r.Git("switch", "--orphan", "unrelated")
	r.Commit("other.txt", "no common history with main")
	repo := git.New(git.ExecRunner{Dir: r.Dir})
	sq, err := repo.SquashMerged("unrelated", "origin/main")
	if err != nil || sq {
		t.Errorf("want false, nil; got %v, %v", sq, err)
	}
}
