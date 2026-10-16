package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hbeanjarah/gitix/internal/clean"
	"github.com/hbeanjarah/gitix/internal/git"
	"github.com/hbeanjarah/gitix/internal/pick"
	"github.com/hbeanjarah/gitix/internal/ui"
)

var defaultKeep = []string{"main", "master", "develop"}

type cleanFlags struct {
	keep, base                  []string
	dryRun, yes, force, noFetch bool
}

func (a *app) cleanCmd() *cobra.Command {
	var f cleanFlags
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Delete local branches, except the current, protected and kept ones",
		Example: "  gitix clean                 pick the branches to delete\n" +
			"  gitix clean --keep feat/x   also keep feat/x\n" +
			"  gitix clean --dry-run       show what would be deleted",
		Args: usageArgs(cobra.NoArgs),
		RunE: func(cmd *cobra.Command, args []string) error { return a.runClean(f) },
	}
	cmd.Flags().StringSliceVar(&f.keep, "keep", nil, "branches to keep (exact names, comma-separated or repeated)")
	cmd.Flags().StringSliceVar(&f.base, "base", nil, "branches to compare against, repeatable (default: origin/HEAD and the protected branches on origin)")
	cmd.Flags().BoolVar(&f.dryRun, "dry-run", false, "show the plan without deleting anything")
	cmd.Flags().BoolVar(&f.yes, "yes", false, "delete merged branches without asking (not merged ones are skipped)")
	cmd.Flags().BoolVar(&f.force, "force", false, "with --yes, also delete branches that are not merged")
	cmd.Flags().BoolVar(&f.noFetch, "no-fetch", false, "do not run git fetch origin first (or: git config gitix.fetch false)")
	cmd.RegisterFlagCompletionFunc("keep", a.completeBranches)
	cmd.RegisterFlagCompletionFunc("base", a.completeBranches)
	return cmd
}

func (a *app) runClean(f cleanFlags) error {
	if err := a.ready(); err != nil {
		return err
	}
	if err := a.fetch(f.noFetch); err != nil {
		return err
	}
	protected, err := a.protected()
	if err != nil {
		return fail("%v", err)
	}
	bases, err := a.bases(f.base, protected)
	if err != nil {
		return err
	}
	branches, err := a.repo.Branches(false)
	if err != nil {
		return fail("%v", err)
	}
	var progress func(done, total int)
	spin := a.spinner("Analyzing branches…")
	if spin != nil {
		progress = func(done, total int) { spin.Set(fmt.Sprintf("Analyzing branches  %d/%d", done, total)) }
	}
	plan, err := clean.Build(branches, keepList(protected, f.keep, bases), a.facts(bases), progress)
	if spin != nil {
		spin.Stop() // before the list and the questions
	}
	if err != nil {
		return fail("%v", err)
	}

	out := a.opts.Out
	label := "Base"
	if len(bases) > 1 {
		label = "Bases"
	}
	fmt.Fprintf(out, "%s: %s\n\n", label, ui.Safe(strings.Join(bases, ", ")))
	if len(plan.Merged) == 0 && len(plan.NotMerged) == 0 {
		fmt.Fprintln(out, "Nothing to clean.")
		return nil
	}
	if a.opts.Interactive && !f.yes && !f.dryRun {
		return a.pickAndDelete(plan)
	}

	// Both groups share one table so their columns line up.
	lines := ui.Columns(a.itemRows(deletable(plan), a.opts.Paint))
	a.printGroup(a.opts.Paint.Green("Merged"), lines[:len(plan.Merged)])
	a.printGroup(a.opts.Paint.Yellow("Not merged"), lines[len(plan.Merged):])
	switch {
	case f.dryRun:
		fmt.Fprintln(out, "Dry run: nothing deleted.")
		return nil
	case !f.yes:
		return fail("not a terminal, use --yes to delete")
	}
	toDelete := plan.Merged
	if f.force {
		toDelete = append(toDelete, plan.NotMerged...)
	} else if len(plan.NotMerged) > 0 {
		fmt.Fprintf(out, "Skipped %s (use --force to delete them).\n", count(len(plan.NotMerged), "not merged branch"))
	}
	return a.deleteAll(toDelete)
}

// pickAndDelete lets the user check the branches to delete. Merged ones are checked first.
func (a *app) pickAndDelete(plan clean.Plan) error {
	items := deletable(plan)
	labels := ui.Columns(a.itemRows(items, ui.Painter{}))
	choices := make([]pick.Choice, len(items))
	for i, label := range labels {
		choices[i] = pick.Choice{Label: label, Selected: i < len(plan.Merged)}
	}
	picked, err := a.opts.Picker.Many("Select branches to delete", choices)
	if errors.Is(err, pick.ErrCancelled) {
		return &ExitError{Code: 130}
	}
	if err != nil {
		return fail("%v", err)
	}
	if len(picked) == 0 {
		fmt.Fprintln(a.opts.Out, "Nothing selected (press space to check a branch).")
		return nil
	}
	var toDelete []clean.Item
	notMerged := 0
	for _, i := range picked {
		toDelete = append(toDelete, items[i])
		if i >= len(plan.Merged) {
			notMerged++
		}
	}
	// Always recap before deleting. Yes is the default only when everything is merged.
	question := fmt.Sprintf("Delete %s?", count(len(toDelete), "branch"))
	confirm := ui.ConfirmYes
	if notMerged > 0 {
		question = fmt.Sprintf("Delete %s, including %d not merged?", count(len(toDelete), "branch"), notMerged)
		confirm = ui.Confirm
	}
	if ok, _ := confirm(a.in, a.opts.Out, question); !ok {
		fmt.Fprintln(a.opts.Out, "Nothing deleted.")
		return nil
	}
	return a.deleteAll(toDelete)
}

// deletable lists the merged branches, then the not merged ones.
func deletable(plan clean.Plan) []clean.Item {
	return append(append([]clean.Item{}, plan.Merged...), plan.NotMerged...)
}

// printGroup writes "title (n)", the indented lines and a blank line; no lines print nothing.
func (a *app) printGroup(title string, lines []string) {
	if len(lines) == 0 {
		return
	}
	fmt.Fprintf(a.opts.Out, "%s (%d)\n", title, len(lines))
	for _, line := range lines {
		fmt.Fprintln(a.opts.Out, "  "+line)
	}
	fmt.Fprintln(a.opts.Out)
}

func (a *app) deleteAll(items []clean.Item) error {
	out := a.opts.Out
	var failed []string
	for _, it := range items {
		name := ui.Safe(it.Branch.Name)
		err := a.repo.Delete(it.Branch)
		switch {
		case errors.Is(err, git.ErrMoved):
			failed = append(failed, name)
			fmt.Fprintf(a.opts.Err, "%s changed since the analysis, kept\n", name)
			continue
		case err != nil:
			failed = append(failed, name)
			fmt.Fprintf(a.opts.Err, "Could not delete %s: %v\n", name, err)
			continue
		}
		fmt.Fprintf(out, "Deleted %s (was %s)\n", name, shortSHA(it.Branch.SHA))
	}
	if len(items) > len(failed) {
		fmt.Fprintln(out, a.opts.Paint.Dim("Restore with: git branch <name> <sha>"))
	}
	if len(failed) > 0 {
		return fail("could not delete: %s", strings.Join(failed, ", "))
	}
	return nil
}

// fetch runs `git fetch origin` so that merges made on the server are seen, unless
// --no-fetch or gitix.fetch=false. A failed fetch only warns: local data still works.
func (a *app) fetch(noFetch bool) error {
	enabled, err := a.repo.ConfigBool("gitix.fetch", true)
	if err != nil {
		return fail("%v", err)
	}
	if noFetch || !enabled || !a.repo.HasRemote("origin") {
		return nil
	}
	spin := a.spinner("Fetching origin…")
	err = a.repo.Fetch("origin")
	if spin != nil {
		spin.Stop()
	}
	if err != nil {
		reason, _, _ := strings.Cut(err.Error(), "\n")
		fmt.Fprintf(a.opts.Err, "gitix: could not fetch origin, using local data (%s)\n", reason)
	}
	return nil
}

// spinner shows an animated status line in a terminal, and returns nil elsewhere.
func (a *app) spinner(text string) *ui.Spinner {
	if !a.opts.Interactive {
		return nil
	}
	return ui.StartSpinner(a.opts.Err, a.opts.Paint, text)
}

// protected returns the gitix.keep config, or the default protected branches.
func (a *app) protected() ([]string, error) {
	keep, ok, err := a.repo.ConfigList("gitix.keep")
	if err != nil || !ok {
		return defaultKeep, err
	}
	return keep, nil
}

// bases returns the --base branches, else the gitix.base ones, else the detected ones.
// Given bases must exist.
func (a *app) bases(explicit, protected []string) ([]string, error) {
	from := ""
	if len(explicit) == 0 {
		configured, ok, err := a.repo.ConfigList("gitix.base")
		if err != nil {
			return nil, fail("%v", err)
		}
		explicit, from = configured, " from gitix.base"
		if !ok {
			bases, err := a.repo.Bases(protected)
			if errors.Is(err, git.ErrNotFound) {
				return nil, fail("no base branch found, use --base or set gitix.base")
			}
			if err != nil {
				return nil, fail("%v", err)
			}
			return bases, nil
		}
	}
	for _, base := range explicit {
		if !a.repo.Exists(base) {
			return nil, fail("base %q%s not found", base, from)
		}
	}
	return explicit, nil
}

// keepList merges the protected branches, --keep and the bases (with and without "origin/").
func keepList(protected, flagKeep, bases []string) []string {
	keep := append(append([]string{}, protected...), flagKeep...)
	for _, base := range bases {
		keep = append(keep, base, strings.TrimPrefix(base, "origin/"))
	}
	return keep
}

// facts tests the bases in order: the first one that contains the branch wins.
// A branch in none of them gets the fewest missing commits among the bases.
func (a *app) facts(bases []string) func(git.Branch) (clean.Fact, error) {
	return func(b git.Branch) (clean.Fact, error) {
		var f clean.Fact
		for i, base := range bases {
			merged, err := a.repo.IsAncestor(b.Ref, base)
			if err != nil {
				return clean.Fact{}, err
			}
			if merged {
				return clean.Fact{Base: base}, nil
			}
			squash, err := a.repo.SquashMerged(b.Ref, base)
			if err != nil {
				return clean.Fact{}, err
			}
			if squash {
				return clean.Fact{Base: base, Squash: true}, nil
			}
			rebase, err := a.repo.RebaseMerged(b.Ref, base)
			if err != nil {
				return clean.Fact{}, err
			}
			if rebase {
				return clean.Fact{Base: base, Rebase: true}, nil
			}
			notIn, err := a.repo.CommitsNotIn(b.Ref, base)
			if err != nil {
				return clean.Fact{}, err
			}
			if i == 0 || notIn < f.NotIn {
				f.NotIn, f.NotInBase = notIn, base
			}
		}
		return f, nil
	}
}

// itemRows renders items as [name, reason, age].
func (a *app) itemRows(items []clean.Item, paint ui.Painter) [][]string {
	now := a.opts.Now()
	rows := make([][]string, len(items))
	for i, it := range items {
		rows[i] = []string{ui.Safe(it.Branch.Name), it.Reason, paint.Dim(ui.RelTime(it.Branch.CommitTime, now))}
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
