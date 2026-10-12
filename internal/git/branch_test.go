package git

import (
	"strings"
	"testing"
)

type fakeRunner struct {
	out string
	err error
}

func (f fakeRunner) Run(args ...string) (string, error) {
	return f.out, f.err
}

func line(fields ...string) string { return strings.Join(fields, "\x00") }

func TestBranchesParsesForEachRef(t *testing.T) {
	out := strings.Join([]string{
		line("*", "refs/heads/main", "main", "aaa", "1700000000", "origin/main", "", "/repo"),
		line(" ", "refs/heads/feat/x", "feat/x", "bbb", "1700000100", "origin/feat/x", "[gone]", ""),
		line(" ", "refs/heads/wt", "wt", "ccc", "1700000200", "", "", "/other"),
	}, "\n")
	got, err := New(fakeRunner{out: out}).Branches(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 branches, got %d", len(got))
	}
	if !got[0].Current || got[0].Worktree {
		t.Errorf("main: want Current and not Worktree, got %+v", got[0])
	}
	if !got[1].UpstreamGone || got[1].Upstream != "origin/feat/x" {
		t.Errorf("feat/x: want gone upstream, got %+v", got[1])
	}
	if !got[2].Worktree {
		t.Errorf("wt: want Worktree, got %+v", got[2])
	}
	if got[1].CommitTime.Unix() != 1700000100 {
		t.Errorf("feat/x: wrong commit time %v", got[1].CommitTime)
	}
}

func TestBranchesSkipsRemoteHEADAndDuplicates(t *testing.T) {
	out := strings.Join([]string{
		line(" ", "refs/heads/feat/x", "feat/x", "a", "1", "", "", ""),
		line(" ", "refs/remotes/origin/HEAD", "origin", "b", "1", "", "", ""),
		line(" ", "refs/remotes/origin/feat/x", "origin/feat/x", "c", "1", "", "", ""),
		line(" ", "refs/remotes/origin/feat/y", "origin/feat/y", "d", "1", "", "", ""),
	}, "\n")
	got, err := New(fakeRunner{out: out}).Branches(true)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, b := range got {
		names = append(names, b.Name)
	}
	if strings.Join(names, ",") != "feat/x,origin/feat/y" {
		t.Errorf("got %v", names)
	}
	if !got[1].Remote {
		t.Errorf("origin/feat/y should be Remote")
	}
}

func TestLocalName(t *testing.T) {
	if got := LocalName("origin/feat/x"); got != "feat/x" {
		t.Errorf("got %q", got)
	}
}
