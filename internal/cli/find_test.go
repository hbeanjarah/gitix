package cli

import (
	"strings"
	"testing"

	"github.com/hbeanjarah/gitix/internal/testrepo"
)

func TestFindListsMatchesCaseInsensitive(t *testing.T) {
	r := testrepo.New(t)
	r.Branch("feat/trash-module")
	r.Branch("fix/login")
	res := gitix(t, r.Dir, "", false, "find", "TRASH")
	if res.code != 0 || !strings.Contains(res.out, "feat/trash-module") || strings.Contains(res.out, "fix/login") {
		t.Errorf("got %+v", res)
	}
}

func TestFindWithoutTermListsMostRecentFirst(t *testing.T) {
	r := testrepo.New(t)
	r.Branch("feat/older")
	r.Backdate("2020-01-01T00:00:00")
	r.Git("switch", "main")
	r.Branch("feat/newer")
	res := gitix(t, r.Dir, "", false, "find")
	newer, older := strings.Index(res.out, "feat/newer"), strings.Index(res.out, "feat/older")
	if res.code != 0 || newer == -1 || older == -1 || newer > older {
		t.Errorf("feat/newer should come before feat/older:\n%s", res.out)
	}
	if !strings.Contains(res.out, "years ago") {
		t.Errorf("feat/older should show its age:\n%s", res.out)
	}
}

func TestFindNoMatch(t *testing.T) {
	r := testrepo.New(t)
	res := gitix(t, r.Dir, "", false, "find", "nope")
	if res.code != 1 || !strings.Contains(res.err, `no branch matches "nope"`) {
		t.Errorf("got %+v", res)
	}
}

func TestFindSwitchSingleMatch(t *testing.T) {
	r := testrepo.New(t)
	r.Branch("feat/trash-module")
	r.Git("switch", "main")
	res := gitix(t, r.Dir, "", false, "find", "trash", "-s")
	if res.code != 0 || r.Git("branch", "--show-current") != "feat/trash-module" {
		t.Errorf("got %+v", res)
	}
}

func TestFindSwitchSeveralMatchesNonInteractive(t *testing.T) {
	r := testrepo.New(t)
	r.Branch("feat/trash-a")
	r.Git("switch", "main")
	r.Branch("feat/trash-b")
	r.Git("switch", "main")
	res := gitix(t, r.Dir, "", false, "find", "trash", "-s")
	if res.code != 1 || !strings.Contains(res.err, "refine the search") || r.Git("branch", "--show-current") != "main" {
		t.Errorf("got %+v", res)
	}
}

func TestFindSwitchMenu(t *testing.T) {
	r := testrepo.New(t)
	r.Branch("feat/trash-a")
	r.Backdate("2020-01-01T00:00:00")
	r.Git("switch", "main")
	r.Branch("feat/trash-b")
	r.Git("switch", "main")
	// feat/trash-b is more recent, so it is choice 1; pick 2.
	res := gitix(t, r.Dir, "2\n", true, "find", "trash", "-s")
	if res.code != 0 || r.Git("branch", "--show-current") != "feat/trash-a" {
		t.Errorf("got %+v", res)
	}
}

func TestFindSwitchRemoteCreatesTrackingBranch(t *testing.T) {
	r := testrepo.New(t)
	r.Branch("feat/shared")
	r.Git("push", "-u", "origin", "feat/shared")
	r.Git("switch", "main")
	r.Git("branch", "-D", "feat/shared")
	res := gitix(t, r.Dir, "", false, "find", "shared", "-r", "-s")
	if res.code != 0 || r.Git("branch", "--show-current") != "feat/shared" {
		t.Fatalf("got %+v", res)
	}
	if up := r.Git("rev-parse", "--abbrev-ref", "@{upstream}"); up != "origin/feat/shared" {
		t.Errorf("upstream: got %q", up)
	}
}

func TestFindOutsideRepo(t *testing.T) {
	res := gitix(t, t.TempDir(), "", false, "find")
	if res.code != 1 || !strings.Contains(res.err, "not a git repository") {
		t.Errorf("got %+v", res)
	}
}

func TestUnknownFlagIsUsageError(t *testing.T) {
	r := testrepo.New(t)
	res := gitix(t, r.Dir, "", false, "find", "--nope")
	if res.code != 2 {
		t.Errorf("got %+v", res)
	}
}
