package git

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrNotFound means a ref does not exist.
var ErrNotFound = errors.New("not found")

// ErrMoved means a branch no longer points to the commit it had when it was read.
var ErrMoved = errors.New("changed since it was read")

// Switch checks out b. A remote branch gets a local tracking branch.
// "--" keeps a name such as "-Cmain" from being read as an option.
func (r *Repo) Switch(b Branch) error {
	if b.Remote {
		_, err := r.run.Run("switch", "--track", "--", b.Name)
		return err
	}
	_, err := r.run.Run("switch", "--", b.Name)
	return err
}

// Delete force-deletes a local branch, unless it moved since b was read: work
// committed in the meantime is never lost. Other safety checks are the caller's job.
func (r *Repo) Delete(b Branch) error {
	sha, err := r.run.Run("rev-parse", "--verify", "--quiet", b.Ref)
	if err != nil || sha != b.SHA {
		return ErrMoved
	}
	_, err = r.run.Run("branch", "-D", "--", b.Name)
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

// ConfigBool returns `git config --type=bool <key>`, or def when unset.
func (r *Repo) ConfigBool(key string, def bool) (bool, error) {
	out, err := r.run.Run("config", "--type=bool", "--get", key)
	if exitCode(err) == 1 {
		return def, nil
	}
	if err != nil {
		return def, err
	}
	return out == "true", nil
}

// HasRemote reports whether a remote with this name is configured.
func (r *Repo) HasRemote(name string) bool {
	_, err := r.run.Run("config", "--get", "remote."+name+".url")
	return err == nil
}

// Fetch updates the remote-tracking branches of remote. It never prunes: what the
// user's own fetch.prune setting says still applies, gitix adds nothing.
func (r *Repo) Fetch(remote string) error {
	_, err := r.run.Run("fetch", "--quiet", "--", remote)
	return err
}

// InsideRepo reports whether the working directory is inside a git repository.
func (r *Repo) InsideRepo() bool {
	_, err := r.run.Run("rev-parse", "--git-dir")
	return err == nil
}

// ConfigList returns the names listed in `git config <key>`, separated by spaces or
// commas, or ok=false when unset or empty.
func (r *Repo) ConfigList(key string) (names []string, ok bool, err error) {
	out, err := r.run.Run("config", "--get", key)
	if exitCode(err) == 1 {
		return nil, false, nil // key not set
	}
	if err != nil {
		return nil, false, err
	}
	names = strings.FieldsFunc(out, func(c rune) bool { return c == ',' || c == ' ' || c == '\t' })
	return names, len(names) > 0, nil // an empty gitix.keep must not remove every protection
}
