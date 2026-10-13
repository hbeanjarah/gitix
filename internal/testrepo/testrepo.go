package testrepo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Repo is a working clone at Dir whose "origin" is a local bare repository.
type Repo struct {
	t   testing.TB
	Dir string
}

// New creates a bare remote and a clone with one commit on main, pushed,
// with origin/HEAD pointing to origin/main. It isolates git from the user's config.
func New(t testing.TB) *Repo {
	t.Helper()
	root := t.TempDir()
	config := filepath.Join(root, "gitconfig")
	content := "[user]\n\tname = Test\n\temail = test@example.com\n[init]\n\tdefaultBranch = main\n"
	if err := os.WriteFile(config, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", config)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")

	remote := filepath.Join(root, "remote.git")
	r := &Repo{t: t, Dir: filepath.Join(root, "work")}
	run(t, root, "init", "--bare", remote)
	run(t, root, "clone", remote, r.Dir)
	r.Commit("README.md", "hello")
	r.Git("push", "-u", "origin", "main")
	r.Git("remote", "set-head", "origin", "main")
	return r
}

// Git runs git in the working clone and fails the test on error.
func (r *Repo) Git(args ...string) string {
	r.t.Helper()
	return run(r.t, r.Dir, args...)
}

// Commit writes file with content and commits it on the current branch.
func (r *Repo) Commit(file, content string) {
	r.t.Helper()
	path := filepath.Join(r.Dir, file)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		r.t.Fatal(err)
	}
	r.Git("add", file)
	r.Git("commit", "-m", "update "+file)
}

// Branch creates name from the current branch, adds one commit to it, and stays on it.
func (r *Repo) Branch(name string) {
	r.t.Helper()
	r.Git("switch", "-c", name)
	r.Commit(strings.ReplaceAll(name, "/", "-")+".txt", name)
}

// SquashMerge squash-merges name into main, pushes main and stays on main.
func (r *Repo) SquashMerge(name string) {
	r.t.Helper()
	r.Git("switch", "main")
	r.Git("merge", "--squash", name)
	r.Git("commit", "-m", "squash "+name)
	r.Git("push", "origin", "main")
}

// Merge merges name into main with a merge commit, pushes main and stays on main.
func (r *Repo) Merge(name string) {
	r.t.Helper()
	r.Git("switch", "main")
	r.Git("merge", "--no-ff", "-m", "merge "+name, name)
	r.Git("push", "origin", "main")
}

// Backdate rewrites the last commit's dates to date (e.g. "2020-01-01T00:00:00").
func (r *Repo) Backdate(date string) {
	r.t.Helper()
	cmd := exec.Command("git", "commit", "--amend", "--no-edit", "--date="+date)
	cmd.Dir = r.Dir
	cmd.Env = append(os.Environ(), "GIT_COMMITTER_DATE="+date)
	if out, err := cmd.CombinedOutput(); err != nil {
		r.t.Fatalf("backdate: %v\n%s", err, out)
	}
}

// DeleteRemote deletes name on origin and prunes it locally, like after a merged PR.
func (r *Repo) DeleteRemote(name string) {
	r.t.Helper()
	r.Git("push", "origin", "--delete", name)
	r.Git("fetch", "--prune")
}

func run(t testing.TB, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}
