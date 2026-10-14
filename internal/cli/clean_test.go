package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hbeanjarah/gitix/internal/pick"
	"github.com/hbeanjarah/gitix/internal/testrepo"
)

// scenario builds a repo with one branch of each kind and returns it on main.
// Merged, most recent first: feat/maintenance, feat/squashed, feat/merged.
// Not merged, most recent first: feat/abandoned, feat/wip.
func scenario(t *testing.T) *testrepo.Repo {
	r := testrepo.New(t)
	r.Branch("feat/merged")
	r.Backdate("2026-01-01T00:00:00")
	r.Git("push", "-u", "origin", "feat/merged")
	r.Merge("feat/merged")
	r.Branch("feat/squashed")
	r.Backdate("2026-01-02T00:00:00")
	r.Git("push", "-u", "origin", "feat/squashed")
	r.SquashMerge("feat/squashed")
	r.DeleteRemote("feat/squashed")
	r.Branch("feat/maintenance") // contains "main": must not be protected by it
	r.Merge("feat/maintenance")
	r.Branch("feat/wip") // never pushed
	r.Backdate("2026-01-03T00:00:00")
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

var (
	mergedBranches    = []string{"feat/maintenance", "feat/squashed", "feat/merged"}
	notMergedBranches = []string{"feat/abandoned", "feat/wip"}
	keptBranches      = []string{"main", "develop", "feat/in-worktree"}
)

func checkBranches(t *testing.T, r *testrepo.Repo, res result, gone, kept []string) {
	t.Helper()
	for _, name := range gone {
		if hasBranch(r, name) {
			t.Errorf("%s should be deleted\n%s", name, res.out)
		}
	}
	for _, name := range kept {
		if !hasBranch(r, name) {
			t.Errorf("%s should be kept\n%s", name, res.out)
		}
	}
}

func checkOutput(t *testing.T, got string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("output misses %q:\n%s", w, got)
		}
	}
}

func TestCleanListShowsDeletableBranchesMergedChecked(t *testing.T) {
	r := scenario(t)
	p := &fakePicker{err: pick.ErrCancelled}
	gitixPick(t, r.Dir, "", p, "clean")
	if len(p.choices) != 5 {
		t.Fatalf("want 5 choices, got:\n%s", p.labels())
	}
	for i, name := range append(append([]string{}, mergedBranches...), notMergedBranches...) {
		c := p.choices[i]
		if !strings.HasPrefix(c.Label, name+" ") || c.Selected != (i < len(mergedBranches)) {
			t.Errorf("choice %d: want %s (checked: %v), got %+v", i, name, i < len(mergedBranches), c)
		}
	}
	checkOutput(t, p.labels(),
		"merged → main", "merged (squash) → main", "1 commit, remote deleted", "1 commit, never pushed", "ago")
	if strings.Contains(p.labels(), "\x1b[") {
		t.Errorf("labels must not contain colors:\n%q", p.labels())
	}
}

func TestCleanDeletesCheckedMergedWithoutAsking(t *testing.T) {
	r := scenario(t)
	res := gitixPick(t, r.Dir, "", &fakePicker{many: []int{0, 1, 2}}, "clean")
	if res.code != 0 {
		t.Fatalf("got %+v", res)
	}
	checkBranches(t, r, res, mergedBranches, append(keptBranches, notMergedBranches...))
	checkOutput(t, res.out, "Base: origin/main", "Deleted feat/merged (was ", "Restore with: git branch <name> <sha>")
	if strings.Contains(res.out, "[y/N]") {
		t.Errorf("merged branches need no confirmation:\n%s", res.out)
	}
	checkOutput(t, res.err, "Analyzing 8 branches…")
}

func TestCleanAsksBeforeDeletingNotMerged(t *testing.T) {
	r := scenario(t)
	res := gitixPick(t, r.Dir, "y\n", &fakePicker{many: []int{0, 4}}, "clean")
	if res.code != 0 {
		t.Fatalf("got %+v", res)
	}
	checkOutput(t, res.out, "Delete 2 branches, including 1 not merged? [y/N]")
	checkBranches(t, r, res, []string{"feat/maintenance", "feat/wip"}, append(keptBranches, "feat/merged", "feat/abandoned"))
}

func TestCleanNotMergedRefused(t *testing.T) {
	r := scenario(t)
	before := branchNames(r)
	res := gitixPick(t, r.Dir, "n\n", &fakePicker{many: []int{0, 4}}, "clean")
	if res.code != 0 || branchNames(r) != before {
		t.Errorf("got %+v", res)
	}
	checkOutput(t, res.out, "Nothing deleted.")
}

func TestCleanNothingSelected(t *testing.T) {
	r := scenario(t)
	before := branchNames(r)
	res := gitixPick(t, r.Dir, "", &fakePicker{many: []int{}}, "clean")
	if res.code != 0 || branchNames(r) != before {
		t.Errorf("got %+v", res)
	}
	checkOutput(t, res.out, "Nothing selected.")
}

func TestCleanCancelled(t *testing.T) {
	r := scenario(t)
	before := branchNames(r)
	res := gitixPick(t, r.Dir, "", &fakePicker{err: pick.ErrCancelled}, "clean")
	if res.code != 130 || branchNames(r) != before || strings.Contains(res.err, "gitix:") {
		t.Errorf("got %+v", res)
	}
}

func TestCleanMergedIntoDevelop(t *testing.T) {
	r := testrepo.New(t)
	r.Branch("develop")
	r.Git("push", "-u", "origin", "develop")
	r.Branch("feat/done") // merged into develop only
	r.Git("push", "-u", "origin", "feat/done")
	r.Git("switch", "develop")
	r.Git("merge", "--no-ff", "-m", "merge feat/done", "feat/done")
	r.Commit("develop.txt", "more work on develop")
	r.Git("push", "origin", "develop")
	r.Branch("feat/next") // 1 commit ahead of develop, 3 ahead of main
	r.Git("push", "-u", "origin", "feat/next")
	r.Git("switch", "main")
	res := gitix(t, r.Dir, "", false, "clean", "--dry-run")
	if res.code != 0 {
		t.Fatalf("got %+v", res)
	}
	checkOutput(t, res.out,
		"Bases: origin/main, origin/develop",
		"merged → develop",
		"1 commit not merged",
	)
}

func TestCleanDryRunText(t *testing.T) {
	r := scenario(t)
	before := branchNames(r)
	res := gitix(t, r.Dir, "", false, "clean", "--dry-run")
	if res.code != 0 || branchNames(r) != before {
		t.Fatalf("got %+v", res)
	}
	lines := strings.Split(res.out, "\n")
	// Names and reasons line up across both groups; ages follow (they depend on today).
	want := []string{
		"Base: origin/main",
		"",
		"Merged (3)",
		"  feat/maintenance   merged → main              ",
		"  feat/squashed      merged (squash) → main     ",
		"  feat/merged        merged → main              ",
		"",
		"Not merged (2)",
		"  feat/abandoned     1 commit, remote deleted   ",
		"  feat/wip           1 commit, never pushed     ",
		"",
		"Dry run: nothing deleted.",
	}
	if len(lines) < len(want) {
		t.Fatalf("output too short:\n%s", res.out)
	}
	for i, w := range want {
		if !strings.HasPrefix(lines[i], w) {
			t.Errorf("line %d: want prefix %q, got %q", i, w, lines[i])
		}
	}
}

func TestCleanDryRunWinsOverYes(t *testing.T) {
	r := scenario(t)
	before := branchNames(r)
	res := gitix(t, r.Dir, "", true, "clean", "--dry-run", "--yes")
	if res.code != 0 || branchNames(r) != before {
		t.Errorf("got %+v", res)
	}
}

func TestCleanYesSkipsNotMerged(t *testing.T) {
	r := scenario(t)
	res := gitix(t, r.Dir, "", false, "clean", "--yes")
	if res.code != 0 {
		t.Fatalf("got %+v", res)
	}
	checkBranches(t, r, res, mergedBranches, append(keptBranches, notMergedBranches...))
	checkOutput(t, res.out, "Skipped 2 not merged branches (use --force to delete them).")
}

func TestCleanYesInTerminalDoesNotOpenTheList(t *testing.T) {
	r := scenario(t)
	res := gitix(t, r.Dir, "", true, "clean", "--yes")
	if res.code != 0 {
		t.Fatalf("got %+v", res)
	}
	checkBranches(t, r, res, mergedBranches, notMergedBranches)
}

func TestCleanYesForceDeletesNotMerged(t *testing.T) {
	r := scenario(t)
	res := gitix(t, r.Dir, "", false, "clean", "--yes", "--force")
	if res.code != 0 {
		t.Fatalf("got %+v", res)
	}
	checkBranches(t, r, res, append(mergedBranches, notMergedBranches...), keptBranches)
}

func TestCleanKeepFlagAndConfig(t *testing.T) {
	r := scenario(t)
	r.Git("config", "gitix.keep", "main feat/merged")
	res := gitix(t, r.Dir, "", false, "clean", "--yes", "--force", "--keep", "feat/wip")
	if res.code != 0 {
		t.Fatalf("got %+v", res)
	}
	checkBranches(t, r, res, []string{"develop"}, []string{"feat/merged", "feat/wip", "main"})
}

func TestCleanNotATerminal(t *testing.T) {
	r := scenario(t)
	before := branchNames(r)
	res := gitix(t, r.Dir, "y\n", false, "clean")
	if res.code != 1 || branchNames(r) != before || !strings.Contains(res.err, "not a terminal, use --yes to delete") {
		t.Errorf("got %+v", res)
	}
	if strings.Contains(res.err, "Analyzing") {
		t.Errorf("no progress line outside a terminal: %q", res.err)
	}
}

func TestCleanNothingToClean(t *testing.T) {
	r := testrepo.New(t)
	res := gitix(t, r.Dir, "", true, "clean")
	if res.code != 0 || !strings.Contains(res.out, "Nothing to clean.") {
		t.Errorf("got %+v", res)
	}
}

func TestCleanExplicitBases(t *testing.T) {
	r := scenario(t)
	res := gitix(t, r.Dir, "", false, "clean", "--dry-run", "--base", "main", "--base", "develop")
	if res.code != 0 {
		t.Fatalf("got %+v", res)
	}
	checkOutput(t, res.out, "Bases: main, develop")
}

func TestCleanUnknownBase(t *testing.T) {
	r := testrepo.New(t)
	res := gitix(t, r.Dir, "", false, "clean", "--base", "main", "--base", "nope")
	if res.code != 1 || !strings.Contains(res.err, `base "nope" not found`) {
		t.Errorf("got %+v", res)
	}
}

func TestCleanNoBase(t *testing.T) {
	r := testrepo.New(t)
	r.Git("remote", "remove", "origin")
	r.Git("branch", "-m", "main", "trunk")
	res := gitix(t, r.Dir, "", false, "clean")
	if res.code != 1 || !strings.Contains(res.err, "no base branch found, use --base") {
		t.Errorf("got %+v", res)
	}
}
