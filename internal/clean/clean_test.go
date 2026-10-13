package clean

import (
	"errors"
	"testing"

	"github.com/hbeanjarah/gitix/internal/git"
)

// factsFrom returns a facts func backed by a map; unknown branches are not merged.
func factsFrom(m map[string]Fact) func(git.Branch) (Fact, error) {
	return func(b git.Branch) (Fact, error) { return m[b.Name], nil }
}

func groupOf(p Plan, name string) (string, string) {
	for group, items := range map[string][]Item{"kept": p.Kept, "safe": p.Safe, "risky": p.Risky} {
		for _, it := range items {
			if it.Branch.Name == name {
				return group, it.Reason
			}
		}
	}
	return "absent", ""
}

func TestKeepMatchesExactNamesOnly(t *testing.T) {
	bs := []git.Branch{{Name: "main"}, {Name: "feat/maintenance"}, {Name: "fix/domain-model"}}
	p, err := Build(bs, []string{"main"}, "origin/main", factsFrom(map[string]Fact{
		"feat/maintenance": {Merged: true},
		"fix/domain-model": {Merged: true},
	}))
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"main": "kept", "feat/maintenance": "safe", "fix/domain-model": "safe"} {
		if got, _ := groupOf(p, name); got != want {
			t.Errorf("%s: want %s, got %s", name, want, got)
		}
	}
}

func TestCurrentAndWorktreeAreAlwaysKept(t *testing.T) {
	bs := []git.Branch{{Name: "feat/here", Current: true}, {Name: "feat/there", Worktree: true}}
	called := false
	p, err := Build(bs, nil, "main", func(git.Branch) (Fact, error) {
		called = true
		return Fact{Merged: true}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Kept) != 2 || called {
		t.Errorf("want both kept without computing facts, got %+v (facts called: %v)", p, called)
	}
}

func TestClassification(t *testing.T) {
	bs := []git.Branch{
		{Name: "merged", Upstream: "origin/merged"},
		{Name: "squashed", Upstream: "origin/squashed", UpstreamGone: true},
		{Name: "local-only"},
		{Name: "gone-unmerged", Upstream: "origin/gone-unmerged", UpstreamGone: true},
		{Name: "pushed-unmerged", Upstream: "origin/pushed-unmerged"},
		{Name: "origin/remote", Remote: true},
	}
	p, err := Build(bs, nil, "origin/main", factsFrom(map[string]Fact{
		"merged":          {Merged: true},
		"squashed":        {Squash: true, NotIn: 2},
		"local-only":      {NotIn: 3},
		"gone-unmerged":   {NotIn: 1},
		"pushed-unmerged": {NotIn: 5},
	}))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][2]string{
		"merged":          {"safe", "merged"},
		"squashed":        {"safe", "merged (squash)"},
		"local-only":      {"risky", "3 commits never pushed"},
		"gone-unmerged":   {"risky", "remote deleted, 1 commit not in origin/main"},
		"pushed-unmerged": {"risky", "5 commits not in origin/main"},
		"origin/remote":   {"absent", ""},
	}
	for name, w := range want {
		group, reason := groupOf(p, name)
		if group != w[0] || reason != w[1] {
			t.Errorf("%s: want %v, got [%s %s]", name, w, group, reason)
		}
	}
}

func TestFactErrorStopsTheBuild(t *testing.T) {
	_, err := Build([]git.Branch{{Name: "x"}}, nil, "main", func(git.Branch) (Fact, error) {
		return Fact{}, errors.New("boom")
	})
	if err == nil || err.Error() != "x: boom" {
		t.Errorf("got %v", err)
	}
}
