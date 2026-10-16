// Package clean decides which local branches are kept, merged, or not merged.
package clean

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hbeanjarah/gitix/internal/git"
)

// Fact describes how a branch relates to the bases.
type Fact struct {
	Base   string // first base that contains the branch, "" when not merged
	Squash bool   // the base contains it as a squashed patch, not as an ancestor
	Rebase bool   // the base contains each of its commits, replayed with new SHAs
	NotIn  int    // when not merged: commits missing from the closest base
	// NotInBase is that closest base.
	NotInBase string
}

// Item is a branch with the reason it landed in its group.
type Item struct {
	Branch git.Branch
	Reason string
}

// Plan groups local branches. Merged and NotMerged are sorted most recent first.
type Plan struct {
	Kept      []Item
	Merged    []Item
	NotMerged []Item
}

// Build classifies branches. keep holds exact names that must never be deleted.
// facts is only called for branches that are not kept. progress, when not nil, is
// called before the first analysis and after each one with (done, total).
func Build(branches []git.Branch, keep []string, facts func(git.Branch) (Fact, error), progress func(done, total int)) (Plan, error) {
	keepSet := map[string]bool{}
	for _, name := range keep {
		keepSet[name] = true
	}
	var p Plan
	var todo []git.Branch
	for _, b := range branches {
		if b.Remote {
			continue
		}
		if reason, kept := keptReason(b, keepSet); kept {
			p.Kept = append(p.Kept, Item{b, reason})
			continue
		}
		todo = append(todo, b)
	}
	if progress == nil {
		progress = func(int, int) {}
	}
	progress(0, len(todo))
	for i, b := range todo {
		f, err := facts(b)
		if err != nil {
			return Plan{}, fmt.Errorf("%s: %w", b.Name, err)
		}
		switch {
		case f.Base == "":
			p.NotMerged = append(p.NotMerged, Item{b, notMergedReason(b, f)})
		case f.Squash:
			p.Merged = append(p.Merged, Item{b, "merged (squash) → " + shortBase(f.Base)})
		case f.Rebase:
			p.Merged = append(p.Merged, Item{b, "merged (rebase) → " + shortBase(f.Base)})
		default:
			p.Merged = append(p.Merged, Item{b, "merged → " + shortBase(f.Base)})
		}
		progress(i+1, len(todo))
	}
	byDate(p.Merged)
	byDate(p.NotMerged)
	return p, nil
}

func keptReason(b git.Branch, keep map[string]bool) (string, bool) {
	switch {
	case b.Current:
		return "current branch", true
	case b.Worktree:
		return "checked out in a worktree", true
	case keep[b.Name]:
		return "protected", true
	}
	return "", false
}

// notMergedReason names the base: "4 commits not in develop" says what is counted.
func notMergedReason(b git.Branch, f Fact) string {
	commits := plural(f.NotIn, "commit") + " not in " + shortBase(f.NotInBase)
	switch {
	case b.Upstream == "":
		return commits + ", never pushed"
	case b.UpstreamGone:
		return commits + ", remote deleted"
	default:
		return commits
	}
}

// shortBase drops the remote: "origin/develop" -> "develop".
func shortBase(base string) string {
	return strings.TrimPrefix(base, "origin/")
}

func byDate(items []Item) {
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Branch.CommitTime.After(items[j].Branch.CommitTime)
	})
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}
