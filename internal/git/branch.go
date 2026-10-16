package git

import (
	"strconv"
	"strings"
	"time"
)

// Branch is a local or remote-tracking branch.
type Branch struct {
	Name         string // short name: "feat/x" or "origin/feat/x"
	Ref          string // full name: "refs/heads/feat/x"; pass it to git, it never starts with "-"
	SHA          string
	CommitTime   time.Time
	Upstream     string // "" when the branch has no upstream
	UpstreamGone bool   // upstream configured but deleted on the remote
	Remote       bool   // remote-tracking branch (refs/remotes/...)
	Current      bool   // checked out in this worktree
	Worktree     bool   // checked out in another worktree
}

// lstrip=2 rather than :short, which turns "main" into "heads/main" when a tag is also named main.
const branchFormat = "%(HEAD)%00%(refname)%00%(refname:lstrip=2)%00%(objectname)%00" +
	"%(committerdate:unix)%00%(upstream:lstrip=2)%00%(upstream:track)%00%(worktreepath)"

// Repo reads facts from a git repository and acts on it.
type Repo struct {
	run Runner
}

func New(r Runner) *Repo {
	return &Repo{run: r}
}

// Branches lists local branches, plus remote-tracking ones when withRemote is true.
// A remote branch is hidden when a local branch with the same name exists.
func (r *Repo) Branches(withRemote bool) ([]Branch, error) {
	args := []string{"for-each-ref", "--format=" + branchFormat, "refs/heads"}
	if withRemote {
		args = append(args, "refs/remotes")
	}
	out, err := r.run.Run(args...)
	if err != nil {
		return nil, err
	}
	return dedupe(parseBranches(out)), nil
}

func parseBranches(out string) []Branch {
	var branches []Branch
	for _, line := range strings.Split(out, "\n") {
		f := strings.Split(strings.TrimRight(line, "\r"), "\x00")
		if len(f) != 8 {
			continue
		}
		head, ref, short, sha, date, upstream, track, wtPath := f[0], f[1], f[2], f[3], f[4], f[5], f[6], f[7]
		if strings.HasSuffix(ref, "/HEAD") {
			continue // refs/remotes/origin/HEAD is an alias, not a branch
		}
		secs, _ := strconv.ParseInt(date, 10, 64)
		current := head == "*"
		branches = append(branches, Branch{
			Name:         short,
			Ref:          ref,
			SHA:          sha,
			CommitTime:   time.Unix(secs, 0),
			Upstream:     upstream,
			UpstreamGone: track == "[gone]",
			Remote:       strings.HasPrefix(ref, "refs/remotes/"),
			Current:      current,
			Worktree:     wtPath != "" && !current,
		})
	}
	return branches
}

func dedupe(branches []Branch) []Branch {
	local := map[string]bool{}
	for _, b := range branches {
		if !b.Remote {
			local[b.Name] = true
		}
	}
	var out []Branch
	for _, b := range branches {
		if b.Remote && local[LocalName(b.Name)] {
			continue
		}
		out = append(out, b)
	}
	return out
}

// LocalName strips the remote prefix: "origin/feat/x" -> "feat/x".
func LocalName(remoteName string) string {
	if _, rest, ok := strings.Cut(remoteName, "/"); ok {
		return rest
	}
	return remoteName
}
