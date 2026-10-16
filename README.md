# gitix

**Find and clean git branches. Identify and eXecute.**

You work on many branches. You forget their names, and they pile up.
`gitix` does two things, simply, on Windows, macOS and Linux.

```console
$ gitix find trash
    feat/trash-module   2 hours ago
    fix/trash-icon      3 days ago

$ gitix find trash -s
┃ Switch to
┃ >   feat/trash-module   2 hours ago
┃     fix/trash-icon      3 days ago

$ gitix clean
Bases: origin/main, origin/develop

┃ Select branches to delete  2 of 3 selected
┃ > ✓ feat/trash-module   merged (squash) → develop             2 hours ago
┃   ✓ fix/login           merged → main                         5 days ago
┃   • feat/wip            3 commits not in main, never pushed   1 week ago

space toggle • ↑ up • ↓ down • / filter • enter confirm • a select all • esc cancel

Delete 2 branches? [Y/n]
Deleted feat/trash-module (was 3f2a1c9)
Deleted fix/login (was 8b7e002)
Restore with: git branch <name> <sha>
```

## Install

One command, nothing to configure:

```sh
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/hbeanjarah/gitix/main/install.sh | sh
```

```powershell
# Windows (PowerShell)
irm https://raw.githubusercontent.com/hbeanjarah/gitix/main/install.ps1 | iex
```

Or with a package manager:

| OS | Command |
|---|---|
| macOS / Linux (Homebrew) | `brew install --cask hbeanjarah/tap/gitix` |
| Windows (Scoop) | `scoop bucket add gitix https://github.com/hbeanjarah/scoop-bucket` then `scoop install gitix` |
| Debian / Ubuntu | download the `.deb` from [Releases](https://github.com/hbeanjarah/gitix/releases), then `sudo apt install ./gitix_*.deb` |
| Fedora / RHEL | download the `.rpm` from [Releases](https://github.com/hbeanjarah/gitix/releases), then `sudo dnf install ./gitix_*.rpm` |
| Go | `go install github.com/hbeanjarah/gitix/cmd/gitix@latest` |
| Any | download an archive from [Releases](https://github.com/hbeanjarah/gitix/releases) and put `gitix` in your `PATH` |

Requires git 2.23 or newer.

## Usage

### `gitix find [term]`

Lists branches whose name contains `term` (case-insensitive), most recent first.
Without a term, lists every branch: handy to remember what you worked on.

| Flag | |
|---|---|
| `-s, --switch` | switch to the match; if there are several, pick one in a list (type `/` to filter) |
| `-r, --remote` | include remote branches (switching creates the local tracking branch) |

### `gitix clean`

Deletes local branches, except:

- the current branch and branches checked out in a worktree,
- protected branches: `main`, `master`, `develop` by default,
- branches passed with `--keep`.

Names are matched exactly: protecting `main` does not protect `feat/maintenance`.
Remote branches are never touched.

First, gitix runs `git fetch origin`, so that pull requests merged on the server are seen.
It never deletes anything there (no `--prune`). Skip it with `--no-fetch`, or for good with
`git config gitix.fetch false`.

Each branch is compared with the **bases**: `origin/HEAD`, plus each protected branch that
exists on `origin` (so with git flow, both `origin/main` and `origin/develop`):

- **merged**: already in one of the bases, including squash and rebase merges, which plain git
  does not detect. gitix shows which one: `merged (squash) → develop`, `merged (rebase) → main`.
- **not merged**: has commits that are in none of the bases, counted against the closest one:
  `3 commits not in develop, never pushed`.

Wrong base? Pass `--base`, or set it for good with `git config gitix.base "origin/develop"`.

In a terminal, `gitix clean` opens a list where merged branches are checked and the others are
not. Check what you want and press enter: gitix recaps and asks before deleting. The answer
defaults to yes, or to no when a checked branch is not merged. `esc` leaves without deleting
anything.

| Flag | |
|---|---|
| `--keep a,b` | also keep these branches |
| `--base <branch>` | compare against this branch instead; repeat it for several bases |
| `--dry-run` | show the plan, delete nothing |
| `--no-fetch` | do not run `git fetch origin` first |
| `--yes` | delete merged branches without asking (not merged ones are skipped) |
| `--yes --force` | also delete not merged branches without asking |

Without a terminal (scripts, CI), gitix prints the plan and deletes only with `--yes`.

Every deleted branch prints its SHA, so `git branch <name> <sha>` brings it back.

### Configuration

Settings are git config keys, global (`--global`) or per repository:

```sh
git config --global gitix.keep "main master develop staging"   # protected branches
git config gitix.base "origin/develop"                          # bases to compare with
git config gitix.fetch false                                    # no git fetch before clean
```

`gitix.keep` **replaces** the default list (`main master develop`): list them too if you still
want them protected.

### Shell completion

```sh
gitix completion bash|zsh|fish|powershell --help
```

## Why not lazygit or git-trim?

They are great, and they do much more. `gitix` is for when you only want to
find a branch and clean the others: one binary, no config, nothing to learn.

## License

MIT
