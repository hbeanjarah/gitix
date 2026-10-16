package git_test

import (
	"strings"
	"testing"

	"github.com/hbeanjarah/gitix/internal/git"
	"github.com/hbeanjarah/gitix/internal/testrepo"
)

func TestBases(t *testing.T) {
	protected := []string{"main", "master", "develop"}
	cases := []struct {
		name  string
		setup func(r *testrepo.Repo)
		want  []string
	}{
		{"origin/HEAD only", func(r *testrepo.Repo) {}, []string{"origin/main"}},
		{"origin/HEAD and develop on origin", func(r *testrepo.Repo) {
			r.Branch("develop")
			r.Git("push", "origin", "develop")
		}, []string{"origin/main", "origin/develop"}},
		{"local develop is not a base", func(r *testrepo.Repo) {
			r.Branch("develop")
		}, []string{"origin/main"}},
		{"no origin/HEAD", func(r *testrepo.Repo) {
			r.Git("remote", "set-head", "origin", "--delete")
			r.Branch("develop")
			r.Git("push", "origin", "develop")
		}, []string{"origin/main", "origin/develop"}},
		{"dangling origin/HEAD", func(r *testrepo.Repo) {
			// The remote's default branch was renamed and pruned: origin/HEAD points to nothing.
			r.Git("update-ref", "-d", "refs/remotes/origin/main")
		}, []string{"main"}},
		{"no remote", func(r *testrepo.Repo) {
			r.Branch("develop")
			r.Git("remote", "remove", "origin")
		}, []string{"main", "develop"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := testrepo.New(t)
			c.setup(r)
			got, err := git.New(git.ExecRunner{Dir: r.Dir}).Bases(protected)
			if err != nil || strings.Join(got, ",") != strings.Join(c.want, ",") {
				t.Errorf("want %v, got %v, %v", c.want, got, err)
			}
		})
	}
}

func TestBasesNotFound(t *testing.T) {
	r := testrepo.New(t)
	r.Git("remote", "remove", "origin")
	_, err := git.New(git.ExecRunner{Dir: r.Dir}).Bases([]string{"develop"})
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

func TestRebaseMerged(t *testing.T) {
	r := testrepo.New(t)
	r.Branch("feat/rebased")
	r.Commit("second.txt", "two commits on this branch")
	r.RebaseMerge("feat/rebased")
	r.Branch("feat/half")
	r.Commit("half.txt", "only the first commit reaches main")
	r.Git("switch", "main")
	r.Git("cherry-pick", "feat/half~1")
	r.Git("push", "origin", "main")
	r.Branch("feat/wip")
	r.Git("fetch")
	repo := git.New(git.ExecRunner{Dir: r.Dir})

	for branch, want := range map[string]bool{"feat/rebased": true, "feat/half": false, "feat/wip": false} {
		got, err := repo.RebaseMerged(branch, "origin/main")
		if err != nil || got != want {
			t.Errorf("%s RebaseMerged: want %v, got %v, %v", branch, want, got, err)
		}
	}
	if sq, _ := repo.SquashMerged("feat/rebased", "origin/main"); sq {
		t.Errorf("feat/rebased: a rebase merge is not a squash merge")
	}
}
