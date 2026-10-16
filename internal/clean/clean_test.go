package clean

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hbeanjarah/gitix/internal/git"
)

// factsFrom returns a facts func backed by a map; unknown branches are not merged.
func factsFrom(m map[string]Fact) func(git.Branch) (Fact, error) {
	return func(b git.Branch) (Fact, error) { return m[b.Name], nil }
}

func groupOf(p Plan, name string) (string, string) {
	for group, items := range map[string][]Item{"kept": p.Kept, "merged": p.Merged, "not merged": p.NotMerged} {
		for _, it := range items {
			if it.Branch.Name == name {
				return group, it.Reason
			}
		}
	}
	return "absent", ""
}

func names(items []Item) []string {
	var out []string
	for _, it := range items {
		out = append(out, it.Branch.Name)
	}
	return out
}

func TestKeepMatchesExactNamesOnly(t *testing.T) {
	bs := []git.Branch{{Name: "main"}, {Name: "feat/maintenance"}, {Name: "fix/domain-model"}}
	p, err := Build(bs, []string{"main"}, factsFrom(map[string]Fact{
		"feat/maintenance": {Base: "origin/main"},
		"fix/domain-model": {Base: "origin/main"},
	}), nil)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"main": "kept", "feat/maintenance": "merged", "fix/domain-model": "merged"} {
		if got, _ := groupOf(p, name); got != want {
			t.Errorf("%s: want %s, got %s", name, want, got)
		}
	}
}

func TestCurrentAndWorktreeAreAlwaysKept(t *testing.T) {
	bs := []git.Branch{{Name: "feat/here", Current: true}, {Name: "feat/there", Worktree: true}}
	called := false
	p, err := Build(bs, nil, func(git.Branch) (Fact, error) {
		called = true
		return Fact{Base: "main"}, nil
	}, nil)
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
		{Name: "in-develop", Upstream: "origin/in-develop"},
		{Name: "squashed", Upstream: "origin/squashed", UpstreamGone: true},
		{Name: "rebased", Upstream: "origin/rebased"},
		{Name: "local-base"},
		{Name: "local-only"},
		{Name: "one-commit"},
		{Name: "gone-unmerged", Upstream: "origin/gone-unmerged", UpstreamGone: true},
		{Name: "pushed-unmerged", Upstream: "origin/pushed-unmerged"},
		{Name: "origin/remote", Remote: true},
	}
	p, err := Build(bs, nil, factsFrom(map[string]Fact{
		"merged":          {Base: "origin/master"},
		"in-develop":      {Base: "origin/develop"},
		"squashed":        {Base: "origin/develop", Squash: true},
		"rebased":         {Base: "origin/develop", Rebase: true},
		"local-base":      {Base: "main"},
		"local-only":      {NotIn: 3, NotInBase: "origin/develop"},
		"one-commit":      {NotIn: 1, NotInBase: "origin/master"},
		"gone-unmerged":   {NotIn: 5, NotInBase: "origin/develop"},
		"pushed-unmerged": {NotIn: 4, NotInBase: "main"},
	}), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][2]string{
		"merged":          {"merged", "merged → master"},
		"in-develop":      {"merged", "merged → develop"},
		"squashed":        {"merged", "merged (squash) → develop"},
		"rebased":         {"merged", "merged (rebase) → develop"},
		"local-base":      {"merged", "merged → main"},
		"local-only":      {"not merged", "3 commits not in develop, never pushed"},
		"one-commit":      {"not merged", "1 commit not in master, never pushed"},
		"gone-unmerged":   {"not merged", "5 commits not in develop, remote deleted"},
		"pushed-unmerged": {"not merged", "4 commits not in main"},
		"origin/remote":   {"absent", ""},
	}
	for name, w := range want {
		group, reason := groupOf(p, name)
		if group != w[0] || reason != w[1] {
			t.Errorf("%s: want %v, got [%s %s]", name, w, group, reason)
		}
	}
}

func TestGroupsAreSortedMostRecentFirst(t *testing.T) {
	day := func(n int) time.Time { return time.Date(2026, 1, n, 0, 0, 0, 0, time.UTC) }
	bs := []git.Branch{
		{Name: "m-old", CommitTime: day(1)}, {Name: "m-new", CommitTime: day(3)},
		{Name: "n-old", CommitTime: day(1)}, {Name: "n-new", CommitTime: day(3)},
	}
	p, err := Build(bs, nil, factsFrom(map[string]Fact{
		"m-old": {Base: "main"}, "m-new": {Base: "main"},
	}), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := names(p.Merged); len(got) != 2 || got[0] != "m-new" {
		t.Errorf("merged: got %v", got)
	}
	if got := names(p.NotMerged); len(got) != 2 || got[0] != "n-new" {
		t.Errorf("not merged: got %v", got)
	}
}

func TestFactErrorStopsTheBuild(t *testing.T) {
	_, err := Build([]git.Branch{{Name: "x"}}, nil, func(git.Branch) (Fact, error) {
		return Fact{}, errors.New("boom")
	}, nil)
	if err == nil || err.Error() != "x: boom" {
		t.Errorf("got %v", err)
	}
}

func TestBuildReportsProgress(t *testing.T) {
	bs := []git.Branch{{Name: "main"}, {Name: "a"}, {Name: "b"}}
	var calls []string
	_, err := Build(bs, []string{"main"}, factsFrom(nil), func(done, total int) {
		calls = append(calls, fmt.Sprintf("%d/%d", done, total))
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(calls, " "); got != "0/2 1/2 2/2" {
		t.Errorf("kept branches are not analyzed: want 0/2 1/2 2/2, got %s", got)
	}
}
