package git

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrNotFound means a ref does not exist.
var ErrNotFound = errors.New("not found")

// Switch checks out b. A remote branch gets a local tracking branch.
func (r *Repo) Switch(b Branch) error {
	if b.Remote {
		_, err := r.run.Run("switch", "--track", b.Name)
		return err
	}
	_, err := r.run.Run("switch", b.Name)
	return err
}

// Delete force-deletes a local branch. Safety checks are the caller's job.
func (r *Repo) Delete(name string) error {
	_, err := r.run.Run("branch", "-D", name)
	return err
}

// Version returns git's major and minor version ("git version 2.45.1.windows.1" -> 2, 45).
func (r *Repo) Version() (int, int, error) {
	out, err := r.run.Run("version")
	if err != nil {
		return 0, 0, err
	}
	fields := strings.Fields(out)
	if len(fields) < 3 {
		return 0, 0, fmt.Errorf("unexpected git version output: %q", out)
	}
	parts := strings.Split(fields[2], ".")
	if len(parts) < 2 {
		return 0, 0, fmt.Errorf("unexpected git version output: %q", out)
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return 0, 0, fmt.Errorf("unexpected git version output: %q", out)
	}
	return major, minor, nil
}

// InsideRepo reports whether the working directory is inside a git repository.
func (r *Repo) InsideRepo() bool {
	_, err := r.run.Run("rev-parse", "--git-dir")
	return err == nil
}

// KeepConfig returns the names listed in `git config gitix.keep`, or ok=false when unset.
func (r *Repo) KeepConfig() (names []string, ok bool, err error) {
	out, err := r.run.Run("config", "--get", "gitix.keep")
	if exitCode(err) == 1 {
		return nil, false, nil // key not set
	}
	if err != nil {
		return nil, false, err
	}
	return strings.FieldsFunc(out, func(c rune) bool { return c == ',' || c == ' ' || c == '\t' }), true, nil
}
