package cli

import (
	"strings"
	"testing"

	"github.com/hbeanjarah/gitix/internal/testrepo"
)

// Branch names that git itself refuses to create, but that update-ref or a clone can.

func TestFindSwitchToABranchNamedLikeAnOption(t *testing.T) {
	r := testrepo.New(t)
	r.Branch("feat/work")
	work := r.Git("rev-parse", "feat/work")
	r.Git("switch", "main")
	r.Git("update-ref", "refs/heads/-Cfeat/work", "main")

	res := gitix(t, r.Dir, "", false, "find", "Cfeat", "-s")
	if res.code != 0 {
		t.Fatalf("got %+v", res)
	}
	if got := r.Git("rev-parse", "feat/work"); got != work {
		t.Errorf("feat/work moved from %s to %s: the name was read as an option", work, got)
	}
	if got := r.Git("branch", "--show-current"); got != "-Cfeat/work" {
		t.Errorf("want to be on -Cfeat/work, on %q", got)
	}
}

func TestCleanWithABranchNamedLikeAnOption(t *testing.T) {
	r := scenario(t)
	r.Git("update-ref", "refs/heads/--force", "main")

	res := gitix(t, r.Dir, "", false, "clean", "--dry-run")
	if res.code != 0 {
		t.Fatalf("got %+v", res)
	}
	checkOutput(t, res.out, "--force ")

	res = gitix(t, r.Dir, "", false, "clean", "--yes")
	if res.code != 0 {
		t.Fatalf("got %+v", res)
	}
	checkOutput(t, res.out, "Deleted --force (was ")
	checkBranches(t, r, res, []string{"--force"}, keptBranches)
}

func TestCleanRefusesABaseNamedLikeAnOption(t *testing.T) {
	r := scenario(t)
	res := gitix(t, r.Dir, "", false, "clean", "--dry-run", "--base=--all")
	if res.code != 1 {
		t.Fatalf("got %+v", res)
	}
	checkOutput(t, res.err, `base "--all" not found`)
}

func TestCleanIsNotFooledByATagNamedLikeABranch(t *testing.T) {
	r := scenario(t)
	r.Git("tag", "develop")  // same name as a protected branch
	r.Git("tag", "feat/wip") // same name as a deletable branch
	res := gitix(t, r.Dir, "", false, "clean", "--dry-run")
	if res.code != 0 {
		t.Fatalf("got %+v", res)
	}
	if strings.Contains(res.out, "heads/") || strings.Contains(res.out, "develop ") {
		t.Errorf("develop must stay protected and names must be plain:\n%s", res.out)
	}
	checkOutput(t, res.out, "feat/wip ")
}

func TestCleanKeepsABranchThatMovedWhileTheListWasOpen(t *testing.T) {
	r := scenario(t)
	p := &fakePicker{many: []int{0, 2}, during: func() {
		r.Git("switch", "feat/merged")
		r.Commit("late.txt", "work done after the analysis")
		r.Git("switch", "main")
	}}
	res := gitixPick(t, r.Dir, "\n", p, "clean")
	if res.code != 1 {
		t.Fatalf("want exit 1, got %+v", res)
	}
	checkBranches(t, r, res, []string{"feat/maintenance"}, []string{"feat/merged"})
	checkOutput(t, res.err, "feat/merged changed since the analysis, kept")
	checkOutput(t, res.out, "Deleted feat/maintenance (was ")
}

func TestNamesWithInvisibleCharactersAreEscaped(t *testing.T) {
	r := testrepo.New(t)
	r.Branch("feat/x\u202eexe.txt")
	r.Git("switch", "main")
	res := gitix(t, r.Dir, "", false, "find", "feat")
	if strings.ContainsRune(res.out, '\u202e') {
		t.Errorf("raw bidi override printed:\n%q", res.out)
	}
	checkOutput(t, res.out, `feat/x\u202Eexe.txt`)
}

func TestEmptyKeepConfigKeepsTheDefaults(t *testing.T) {
	r := scenario(t)
	r.Git("config", "gitix.keep", "")
	res := gitix(t, r.Dir, "", false, "clean", "--dry-run")
	if strings.Contains(res.out, "develop ") {
		t.Errorf("develop must stay protected:\n%s", res.out)
	}
}
