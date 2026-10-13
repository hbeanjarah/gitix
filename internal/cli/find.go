package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hbeanjarah/gitix/internal/find"
	"github.com/hbeanjarah/gitix/internal/git"
	"github.com/hbeanjarah/gitix/internal/ui"
)

func (a *app) findCmd() *cobra.Command {
	var remote, switchTo bool
	cmd := &cobra.Command{
		Use:   "find [term]",
		Short: "Find branches by name, most recent first",
		Example: "  gitix find trash        list branches containing \"trash\"\n" +
			"  gitix find trash -s     switch to the match (menu if several)\n" +
			"  gitix find              list every branch, most recent first",
		Args: usageArgs(cobra.MaximumNArgs(1)),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return a.completeBranches(cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			term := ""
			if len(args) == 1 {
				term = args[0]
			}
			return a.runFind(term, remote, switchTo)
		},
	}
	cmd.Flags().BoolVarP(&remote, "remote", "r", false, "include remote branches")
	cmd.Flags().BoolVarP(&switchTo, "switch", "s", false, "switch to the matching branch")
	return cmd
}

func (a *app) runFind(term string, remote, switchTo bool) error {
	if err := a.ready(); err != nil {
		return err
	}
	branches, err := a.repo.Branches(remote)
	if err != nil {
		return fail("%v", err)
	}
	matches := find.Filter(branches, term)
	if len(matches) == 0 {
		if term == "" {
			return fail("no branches")
		}
		return fail("no branch matches %q", term)
	}
	if !switchTo {
		ui.Table(a.opts.Out, a.rows(matches))
		return nil
	}

	target := matches[0]
	if len(matches) > 1 {
		if !a.opts.Interactive {
			ui.Table(a.opts.Out, a.rows(matches))
			return fail("%d branches match %q, refine the search", len(matches), term)
		}
		i, err := ui.Choose(a.in, a.opts.Out, a.rows(matches))
		if err != nil {
			return fail("no branch chosen")
		}
		target = matches[i]
	}
	if target.Current {
		fmt.Fprintf(a.opts.Out, "Already on %s\n", target.Name)
		return nil
	}
	if err := a.repo.Switch(target); err != nil {
		var gitErr *git.Error
		if errors.As(err, &gitErr) {
			return &ExitError{Code: gitErr.ExitCode, Err: gitErr}
		}
		return fail("%v", err)
	}
	name := target.Name
	if target.Remote {
		name = git.LocalName(name)
	}
	fmt.Fprintf(a.opts.Out, "Switched to %s\n", a.opts.Paint.Green(name))
	return nil
}

// rows renders branches as [marker+name, age]; the current branch gets a "*".
func (a *app) rows(branches []git.Branch) [][]string {
	now := a.opts.Now()
	rows := make([][]string, len(branches))
	for i, b := range branches {
		marker := "  "
		if b.Current {
			marker = "* "
		}
		rows[i] = []string{marker + b.Name, a.opts.Paint.Dim(ui.RelTime(b.CommitTime, now))}
	}
	return rows
}
