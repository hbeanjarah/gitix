package git

import (
	"strconv"
	"strings"
)

// ResolveBase returns the branch to compare against: explicit if given,
// otherwise origin/HEAD, then main, then master.
func (r *Repo) ResolveBase(explicit string) (string, error) {
	if explicit != "" {
		if !r.exists(explicit) {
			return "", ErrNotFound
		}
		return explicit, nil
	}
	if out, err := r.run.Run("symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD"); err == nil && out != "" {
		return out, nil
	}
	for _, name := range []string{"main", "master"} {
		if r.exists(name) {
			return name, nil
		}
	}
	return "", ErrNotFound
}

func (r *Repo) exists(ref string) bool {
	_, err := r.run.Run("rev-parse", "--verify", "--quiet", ref+"^{commit}")
	return err == nil
}

// IsAncestor reports whether branch is fully contained in base (classic merge).
func (r *Repo) IsAncestor(branch, base string) (bool, error) {
	_, err := r.run.Run("merge-base", "--is-ancestor", branch, base)
	if err == nil {
		return true, nil
	}
	if exitCode(err) == 1 {
		return false, nil
	}
	return false, err
}

// SquashMerged reports whether branch's combined changes already exist in base,
// which is what a "squash and merge" leaves behind. It builds a temporary commit
// squashing the branch onto its merge-base and asks `git cherry` if base has an
// equivalent patch. No ref is created or moved.
func (r *Repo) SquashMerged(branch, base string) (bool, error) {
	mergeBase, err := r.run.Run("merge-base", base, branch)
	if exitCode(err) == 1 {
		return false, nil // no common history, so base cannot contain the branch
	}
	if err != nil {
		return false, err
	}
	squash, err := r.run.Run("-c", "user.name=gitix", "-c", "user.email=gitix@localhost",
		"commit-tree", branch+"^{tree}", "-p", mergeBase, "-m", "gitix squash check")
	if err != nil {
		return false, err
	}
	out, err := r.run.Run("cherry", base, squash)
	if err != nil {
		return false, err
	}
	return strings.HasPrefix(out, "-"), nil
}

// CommitsNotIn counts commits on branch that are not reachable from base.
func (r *Repo) CommitsNotIn(branch, base string) (int, error) {
	out, err := r.run.Run("rev-list", "--count", base+".."+branch)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(out)
}
