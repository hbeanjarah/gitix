package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/spf13/cobra"

	"github.com/hbeanjarah/gitix/internal/git"
	"github.com/hbeanjarah/gitix/internal/ui"
)

// Options holds everything that differs between a real run and a test.
type Options struct {
	Dir         string // repository directory, "" for the current one
	In          io.Reader
	Out, Err    io.Writer
	Interactive bool // stdin and stdout are a terminal
	Paint       ui.Painter
	Now         func() time.Time
	Version     string
}

// ExitError carries a specific exit code. A nil Err prints nothing.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error { return e.Err }

func fail(format string, args ...any) error {
	return &ExitError{Code: 1, Err: fmt.Errorf(format, args...)}
}

func usageError(err error) error {
	return &ExitError{Code: 2, Err: fmt.Errorf("%w\nRun 'gitix --help' for usage.", err)}
}

func usageArgs(check cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := check(cmd, args); err != nil {
			return usageError(err)
		}
		return nil
	}
}

type app struct {
	opts Options
	repo *git.Repo
	in   *bufio.Reader
}

// Run executes gitix with args and returns the process exit code.
func Run(opts Options, args []string) int {
	root := newRoot(opts)
	root.SetArgs(args)
	err := root.Execute()
	if err == nil {
		return 0
	}
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		if exitErr.Err != nil {
			fmt.Fprintln(opts.Err, "gitix:", exitErr.Err)
		}
		return exitErr.Code
	}
	fmt.Fprintln(opts.Err, "gitix:", err)
	return 1
}

func newRoot(opts Options) *cobra.Command {
	a := &app{opts: opts, repo: git.New(git.ExecRunner{Dir: opts.Dir}), in: bufio.NewReader(opts.In)}
	root := &cobra.Command{
		Use:           "gitix",
		Short:         "Find and clean git branches. Identify and eXecute.",
		Version:       opts.Version,
		Args:          usageArgs(cobra.NoArgs),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE:          func(cmd *cobra.Command, args []string) error { return cmd.Help() },
	}
	root.SetIn(opts.In)
	root.SetOut(opts.Out)
	root.SetErr(opts.Err)
	root.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error { return usageError(err) })
	root.AddCommand(a.findCmd(), a.cleanCmd())
	return root
}

// ready checks that git is usable and that we are inside a repository.
func (a *app) ready() error {
	major, minor, err := a.repo.Version()
	if errors.Is(err, exec.ErrNotFound) {
		return fail("git not found in PATH. Install it from https://git-scm.com/downloads")
	}
	if err != nil {
		return fail("cannot run git: %v", err)
	}
	if major < 2 || (major == 2 && minor < 23) {
		return fail("gitix needs git 2.23 or newer (found %d.%d). Update it from https://git-scm.com/downloads", major, minor)
	}
	if !a.repo.InsideRepo() {
		return fail("not a git repository")
	}
	return nil
}

// completeBranches suggests local branch names for shell completion.
func (a *app) completeBranches(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	branches, err := a.repo.Branches(false)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var names []string
	for _, b := range branches {
		names = append(names, b.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
