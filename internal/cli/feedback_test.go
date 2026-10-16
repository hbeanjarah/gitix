package cli

import (
	"strings"
	"testing"

	"github.com/hbeanjarah/gitix/internal/pick"
	"github.com/hbeanjarah/gitix/internal/testrepo"
)

// Changes asked by the first testers.

func TestFindSuggestsRemoteBranchesWhenNoLocalOneMatches(t *testing.T) {
	r := testrepo.New(t)
	r.Branch("feat/invoice-8025")
	r.Git("push", "-u", "origin", "feat/invoice-8025")
	r.Git("switch", "main")
	r.Git("branch", "-D", "feat/invoice-8025")

	res := gitix(t, r.Dir, "", false, "find", "8025")
	if res.code != 1 {
		t.Fatalf("got %+v", res)
	}
	checkOutput(t, res.err, `no local branch matches "8025"; 1 remote branch does, run: gitix find 8025 -r`)
}

// developRepo has origin/main and origin/develop, so both are detected as bases.
func developRepo(t *testing.T) *testrepo.Repo {
	r := testrepo.New(t)
	r.Branch("develop")
	r.Git("push", "-u", "origin", "develop")
	r.Git("switch", "main")
	return r
}

func TestCleanUsesTheBaseConfig(t *testing.T) {
	r := developRepo(t)
	r.Git("config", "gitix.base", "origin/develop")
	res := gitix(t, r.Dir, "", false, "clean", "--dry-run", "--no-fetch")
	if res.code != 0 {
		t.Fatalf("got %+v", res)
	}
	checkOutput(t, res.out, "Base: origin/develop\n")
}

func TestCleanBaseFlagWinsOverTheConfig(t *testing.T) {
	r := developRepo(t)
	r.Git("config", "gitix.base", "origin/develop")
	res := gitix(t, r.Dir, "", false, "clean", "--dry-run", "--no-fetch", "--base", "origin/main")
	checkOutput(t, res.out, "Base: origin/main\n")
}

func TestCleanUnknownBaseInConfig(t *testing.T) {
	r := developRepo(t)
	r.Git("config", "gitix.base", "origin/nope")
	res := gitix(t, r.Dir, "", false, "clean", "--dry-run", "--no-fetch")
	if res.code != 1 {
		t.Fatalf("got %+v", res)
	}
	checkOutput(t, res.err, `base "origin/nope" from gitix.base not found`)
}

func TestCleanNoBaseMentionsTheConfig(t *testing.T) {
	r := testrepo.New(t)
	r.Git("remote", "remove", "origin")
	r.Git("branch", "-m", "main", "trunk")
	res := gitix(t, r.Dir, "", false, "clean")
	checkOutput(t, res.err, "no base branch found, use --base or set gitix.base")
}

// mergedOnTheServer returns a repo where feat/x was merged into main by a teammate,
// and the working clone has not fetched since.
func mergedOnTheServer(t *testing.T) *testrepo.Repo {
	r := testrepo.New(t)
	r.Branch("feat/x")
	r.Git("push", "-u", "origin", "feat/x")
	r.Git("switch", "main")
	r.TeammateMerges("feat/x")
	return r
}

func TestCleanFetchesFirst(t *testing.T) {
	r := mergedOnTheServer(t)
	res := gitix(t, r.Dir, "", false, "clean", "--dry-run")
	if res.code != 0 {
		t.Fatalf("got %+v", res)
	}
	checkOutput(t, res.out, "feat/x   merged → main")
}

func TestCleanFetchShowsProgressInATerminal(t *testing.T) {
	r := mergedOnTheServer(t)
	res := gitixPick(t, r.Dir, "", &fakePicker{err: pick.ErrCancelled}, "clean")
	checkOutput(t, res.err, "Fetching origin…")
}

func TestCleanWithoutFetch(t *testing.T) {
	for name, setup := range map[string]func(r *testrepo.Repo) []string{
		"--no-fetch":        func(r *testrepo.Repo) []string { return []string{"--no-fetch"} },
		"gitix.fetch false": func(r *testrepo.Repo) []string { r.Git("config", "gitix.fetch", "false"); return nil },
	} {
		r := mergedOnTheServer(t)
		args := append([]string{"clean", "--dry-run"}, setup(r)...)
		res := gitix(t, r.Dir, "", false, args...)
		if res.code != 0 || !strings.Contains(res.out, "feat/x   1 commit not in main") {
			t.Errorf("%s: the stale origin/main must be used\n%+v", name, res)
		}
	}
}

func TestCleanFetchNeverPrunes(t *testing.T) {
	r := testrepo.New(t)
	r.Branch("feat/gone")
	r.Git("push", "-u", "origin", "feat/gone")
	r.Git("switch", "main")
	r.Git("--git-dir", r.Remote, "branch", "-D", "feat/gone") // deleted on the server
	gitix(t, r.Dir, "", false, "clean", "--dry-run")
	r.Git("rev-parse", "--verify", "origin/feat/gone") // fails the test if it was pruned
}

func TestCleanGoesOnWhenTheFetchFails(t *testing.T) {
	r := testrepo.New(t)
	r.Branch("feat/x")
	r.Git("switch", "main")
	r.Git("remote", "set-url", "origin", r.Remote+"-unreachable")
	res := gitix(t, r.Dir, "", false, "clean", "--dry-run")
	if res.code != 0 {
		t.Fatalf("got %+v", res)
	}
	checkOutput(t, res.err, "could not fetch origin, using local data")
	checkOutput(t, res.out, "feat/x")
}

func TestCleanDoesNotFetchWithoutOrigin(t *testing.T) {
	r := testrepo.New(t)
	r.Git("remote", "remove", "origin")
	res := gitix(t, r.Dir, "", false, "clean", "--dry-run")
	if strings.Contains(res.err, "fetch") {
		t.Errorf("no origin, nothing to fetch:\n%+v", res)
	}
}

func TestCleanDetectsRebaseMerges(t *testing.T) {
	r := testrepo.New(t)
	r.Branch("feat/rebased")
	r.Commit("second.txt", "two commits, replayed on main with new SHAs")
	r.RebaseMerge("feat/rebased")
	res := gitix(t, r.Dir, "", false, "clean", "--dry-run")
	if res.code != 0 {
		t.Fatalf("got %+v", res)
	}
	checkOutput(t, res.out, "feat/rebased   merged (rebase) → main")
}
