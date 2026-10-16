package git

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
)

// Error is returned when git exits with a non-zero status.
type Error struct {
	Args     []string
	Stderr   string
	ExitCode int
}

type Runner interface {
	Run(args ...string) (string, error)
}

func (e *Error) Error() string {
	if e.Stderr != "" {
		return e.Stderr
	}
	return "git " + strings.Join(e.Args, " ") + " failed"
}

type ExecRunner struct {
	Dir string
	Env []string // added to the environment, e.g. "GIT_TERMINAL_PROMPT=0"
}

func (r ExecRunner) Run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	if len(r.Env) > 0 {
		cmd.Env = append(os.Environ(), r.Env...)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", &Error{Args: args, Stderr: strings.TrimSpace(stderr.String()), ExitCode: exitErr.ExitCode()}
		}
		return "", err
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

// exitCode returns git's exit code, or -1 if err is not a git exit error.
func exitCode(err error) int {
	var gitErr *Error
	if errors.As(err, &gitErr) {
		return gitErr.ExitCode
	}
	return -1
}
