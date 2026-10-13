// Package clean decides which local branches are kept, safe to delete, or risky.
package clean

import (
	"fmt"

	"github.com/hbeanjarah/gitix/internal/git"
)

// Fact describes how a branch relates to the base branch.
type Fact struct {
	Merged bool // branch is an ancestor of base
	Squash bool // an equivalent squashed patch exists in base
	NotIn  int  // commits on the branch that are not in base
}

// Item is a branch with the reason it landed in its group.
type Item struct {
	Branch git.Branch
	Reason string
}

// Plan groups local branches.
type Plan struct {
	Kept  []Item
	Safe  []Item
	Risky []Item
}

// Build classifies branches. keep holds exact names that must never be deleted.
// facts is only called for branches that are not kept.
func Build(branches []git.Branch, keep []string, base string, facts func(git.Branch) (Fact, error)) (Plan, error) {
	keepSet := map[string]bool{}
	for _, name := range keep {
		keepSet[name] = true
	}
	var p Plan
	for _, b := range branches {
		if b.Remote {
			continue
		}
		if reason, kept := keptReason(b, keepSet); kept {
			p.Kept = append(p.Kept, Item{b, reason})
			continue
		}
		f, err := facts(b)
		if err != nil {
			return Plan{}, fmt.Errorf("%s: %w", b.Name, err)
		}
		switch {
		case f.Merged:
			p.Safe = append(p.Safe, Item{b, "merged"})
		case f.Squash:
			p.Safe = append(p.Safe, Item{b, "merged (squash)"})
		default:
			p.Risky = append(p.Risky, Item{b, riskyReason(b, f.NotIn, base)})
		}
	}
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

func riskyReason(b git.Branch, notIn int, base string) string {
	commits := plural(notIn, "commit")
	switch {
	case b.Upstream == "":
		return commits + " never pushed"
	case b.UpstreamGone:
		return fmt.Sprintf("remote deleted, %s not in %s", commits, base)
	default:
		return fmt.Sprintf("%s not in %s", commits, base)
	}
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}
