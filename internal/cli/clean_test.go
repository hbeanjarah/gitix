package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hbeanjarah/gitix/internal/testrepo"
)

// scenario builds a repo with one branch of each kind and returns it on main.
func scenario(t *testing.T) *testrepo.Repo {
	r := testrepo.New(t)
	r.Branch("feat/merged")
	r.Git("push", "-u", "origin", "feat/merged")
	r.Merge("feat/merged")
	r.Branch("feat/squashed")
	r.Git("push", "-u", "origin", "feat/squashed")
	r.SquashMerge("feat/squashed")
	r.DeleteRemote("feat/squashed")
	r.Branch("feat/maintenance") // contains "main": must not be protected by it
	r.Merge("feat/maintenance")
	r.Branch("feat/wip") // never pushed
	r.Git("switch", "main")
	r.Branch("feat/abandoned") // pushed, remote deleted, never merged
	r.Git("push", "-u", "origin", "feat/abandoned")
	r.DeleteRemote("feat/abandoned")
	r.Git("switch", "main")
	r.Branch("develop")
	r.Git("switch", "main")
	r.Git("worktree", "add", filepath.Join(t.TempDir(), "wt"), "-b", "feat/in-worktree")
	return r
}

func TestCleanDeletesSafeAndAsksForRisky(t *testing.T) {
	r := scenario(t)
	res := gitix(t, r.Dir, "y\nn\n", true, "clean")
	if res.code != 0 {
		t.Fatalf("got %+v", res)
	}
	for _, gone := range []string{"feat/merged", "feat/squashed", "feat/maintenance"} {
		if hasBranch(r, gone) {
			t.Errorf("%s should be deleted\n%s", gone, res.out)
		}
	}
	for _, kept := range []string{"main", "develop", "feat/in-worktree", "feat/wip", "feat/abandoned"} {
		if !hasBranch(r, kept) {
			t.Errorf("%s should be kept\n%s", kept, res.out)
		}
	}
	for _, want := range []string{
		"merged (squash)",
		"1 commit never pushed",
		"remote deleted, 1 commit not in origin/main",
		"Delete 3 safe branches? [y/N]",
		"Also delete 2 risky branches? [y/N]",
		"Deleted feat/merged (was ",
		"Restore with: git branch <name> <sha>",
	} {
		if !strings.Contains(res.out, want) {
			t.Errorf("output misses %q:\n%s", want, res.out)
		}
	}
}

func TestCleanYesSkipsRisky(t *testing.T) {
	r := scenario(t)
	res := gitix(t, r.Dir, "", false, "clean", "--yes")
	if res.code != 0 || hasBranch(r, "feat/merged") || !hasBranch(r, "feat/wip") {
		t.Errorf("got %+v\nbranches: %s", res, branchNames(r))
	}
	if !strings.Contains(res.out, "Skipped 2 risky branches (add --force to delete them)") {
		t.Errorf("output:\n%s", res.out)
	}
}

func TestCleanYesForceDeletesRisky(t *testing.T) {
	r := scenario(t)
	res := gitix(t, r.Dir, "", false, "clean", "--yes", "--force")
	if res.code != 0 || hasBranch(r, "feat/wip") || hasBranch(r, "feat/abandoned") || !hasBranch(r, "develop") {
		t.Errorf("got %+v\nbranches: %s", res, branchNames(r))
	}
}

func TestCleanDryRunDeletesNothing(t *testing.T) {
	r := scenario(t)
	before := branchNames(r)
	res := gitix(t, r.Dir, "", false, "clean", "--dry-run")
	if res.code != 0 || branchNames(r) != before || !strings.Contains(res.out, "Dry run: nothing deleted") {
		t.Errorf("got %+v", res)
	}
}

func TestCleanKeepFlagAndConfig(t *testing.T) {
	r := scenario(t)
	r.Git("config", "gitix.keep", "main feat/merged")
	res := gitix(t, r.Dir, "", false, "clean", "--yes", "--force", "--keep", "feat/wip")
	if res.code != 0 {
		t.Fatalf("got %+v", res)
	}
	for _, kept := range []string{"feat/merged", "feat/wip", "main"} {
		if !hasBranch(r, kept) {
			t.Errorf("%s should be kept", kept)
		}
	}
	if hasBranch(r, "develop") {
		t.Errorf("develop is not in gitix.keep, it should be deleted")
	}
}

func TestCleanNoAnswerDeletesNothing(t *testing.T) {
	r := scenario(t)
	before := branchNames(r)
	res := gitix(t, r.Dir, "", false, "clean")
	if res.code != 1 || branchNames(r) != before {
		t.Errorf("got %+v", res)
	}
}

func TestCleanNothingToClean(t *testing.T) {
	r := testrepo.New(t)
	res := gitix(t, r.Dir, "", false, "clean")
	if res.code != 0 || !strings.Contains(res.out, "Nothing to clean") {
		t.Errorf("got %+v", res)
	}
}

func TestCleanUnknownBase(t *testing.T) {
	r := testrepo.New(t)
	res := gitix(t, r.Dir, "", false, "clean", "--base", "nope")
	if res.code != 1 || !strings.Contains(res.err, `base branch "nope" not found`) {
		t.Errorf("got %+v", res)
	}
}
