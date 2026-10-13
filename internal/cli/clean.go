package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hbeanjarah/gitix/internal/clean"
	"github.com/hbeanjarah/gitix/internal/git"
	"github.com/hbeanjarah/gitix/internal/ui"
)

var defaultKeep = []string{"main", "master", "develop"}

type cleanFlags struct {
	keep               []string
	base               string
	dryRun, yes, force bool
}

func (a *app) cleanCmd() *cobra.Command {
	var f cleanFlags
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Delete local branches, except the current, protected and kept ones",
		Example: "  gitix clean                 review and confirm before deleting\n" +
			"  gitix clean --keep feat/x   also keep feat/x\n" +
			"  gitix clean --dry-run       show what would be deleted",
		Args: usageArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, args []string) error { return a.runClean(f) },
	}
	cmd.Flags().StringSliceVar(&f.keep, "keep", nil, "branches to keep (exact names, comma-separated or repeated)")
	cmd.Flags().StringVar(&f.base, "base", "", "branch to compare against (default: origin/HEAD, main, then master)")
	cmd.Flags().BoolVar(&f.dryRun, "dry-run", false, "show the plan without deleting anything")
	cmd.Flags().BoolVar(&f.yes, "yes", false, "delete safe branches without asking (risky ones are skipped)")
	cmd.Flags().BoolVar(&f.force, "force", false, "with --yes, also delete risky branches")
	cmd.RegisterFlagCompletionFunc("keep", a.completeBranches)
	cmd.RegisterFlagCompletionFunc("base", a.completeBranches)
	return cmd
}

func (a *app) runClean(f cleanFlags) error {
	if err := a.ready(); err != nil {
		return err
	}
	base, err := a.repo.ResolveBase(f.base)
	if errors.Is(err, git.ErrNotFound) {
		if f.base != "" {
			return fail("base branch %q not found", f.base)
		}
		return fail("cannot find the base branch (no origin/HEAD, main or master); use --base <branch>")
	}
	if err != nil {
		return fail("%v", err)
	}
	keep, err := a.keepList(f.keep, base)
	if err != nil {
		return fail("%v", err)
	}
	branches, err := a.repo.Branches(false)
	if err != nil {
		return fail("%v", err)
	}

	plan, err := clean.Build(branches, keep, base, a.facts(base))

	if err != nil {
		return fail("%v", err)
	}

	out, paint := a.opts.Out, a.opts.Paint
	fmt.Fprintf(out, "Base: %s\n\n", base)
	if len(plan.Safe) == 0 && len(plan.Risky) == 0 {
		fmt.Fprintln(out, "Nothing to clean")
		return nil
	}
	if len(plan.Safe) > 0 {
		fmt.Fprintf(out, "%s\n", paint.Green(fmt.Sprintf("Safe to delete (already in %s):", base)))
		ui.Table(out, itemRows(plan.Safe))
	}
	if len(plan.Risky) > 0 {
		fmt.Fprintf(out, "%s\n", paint.Yellow(fmt.Sprintf("⚠ Risky (work not in %s):", base)))
		ui.Table(out, itemRows(plan.Risky))
	}
	fmt.Fprintln(out)
	if f.dryRun {
		fmt.Fprintln(out, "Dry run: nothing deleted")
		return nil
	}

	var toDelete []clean.Item
	if len(plan.Safe) > 0 {
		ok := f.yes
		if !ok {
			if ok, err = ui.Confirm(a.in, out, fmt.Sprintf("Delete %s?", count(len(plan.Safe), "safe branch"))); err != nil {
				return fail("no answer, nothing deleted")
			}
		}
		if ok {
			toDelete = append(toDelete, plan.Safe...)
		}
	}
	if len(plan.Risky) > 0 {
		ok := f.yes && f.force
		if !f.yes {
			if ok, err = ui.Confirm(a.in, out, fmt.Sprintf("Also delete %s?", count(len(plan.Risky), "risky branch"))); err != nil {
				return fail("no answer, nothing deleted")
			}
		} else if !f.force {
			fmt.Fprintf(out, "Skipped %s (add --force to delete them)\n", count(len(plan.Risky), "risky branch"))
		}
		if ok {
			toDelete = append(toDelete, plan.Risky...)
		}
	}
	return a.deleteAll(toDelete)
}

func (a *app) deleteAll(items []clean.Item) error {
	out := a.opts.Out
	var failed []string
	for _, it := range items {
		if err := a.repo.Delete(it.Branch.Name); err != nil {
			failed = append(failed, it.Branch.Name)
			fmt.Fprintf(a.opts.Err, "Could not delete %s: %v\n", it.Branch.Name, err)
			continue
		}
		fmt.Fprintf(out, "Deleted %s (was %s)\n", it.Branch.Name, shortSHA(it.Branch.SHA))
	}
	if len(items) > len(failed) {
		fmt.Fprintln(out, a.opts.Paint.Dim("Restore with: git branch <name> <sha>"))
	}
	if len(failed) > 0 {
		return fail("could not delete: %s", strings.Join(failed, ", "))
	}
	return nil
}

// keepList merges the gitix.keep config (or the defaults), --keep and the base branch.
func (a *app) keepList(flagKeep []string, base string) ([]string, error) {
	keep, ok, err := a.repo.KeepConfig()
	if err != nil {
		return nil, err
	}
	if !ok {
		keep = defaultKeep
	}
	keep = append(append([]string{}, keep...), flagKeep...)
	return append(keep, base, strings.TrimPrefix(base, "origin/")), nil
}

func (a *app) facts(base string) func(git.Branch) (clean.Fact, error) {
	return func(b git.Branch) (clean.Fact, error) {
		merged, err := a.repo.IsAncestor(b.Name, base)
		if err != nil || merged {
			return clean.Fact{Merged: merged}, err
		}
		squash, err := a.repo.SquashMerged(b.Name, base)
		if err != nil {
			return clean.Fact{}, err
		}
		notIn, err := a.repo.CommitsNotIn(b.Name, base)
		return clean.Fact{Squash: squash, NotIn: notIn}, err
	}
}

func itemRows(items []clean.Item) [][]string {
	rows := make([][]string, len(items))
	for i, it := range items {
		rows[i] = []string{it.Branch.Name, it.Reason}
	}
	return rows
}

func count(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	if strings.HasSuffix(noun, "ch") {
		return fmt.Sprintf("%d %ses", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
