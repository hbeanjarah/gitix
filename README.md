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
┃ > ✓ feat/trash-module   merged (squash) → develop   2 hours ago
┃   ✓ fix/login           merged → main               5 days ago
┃   • feat/wip            3 commits, never pushed     1 week ago

space toggle • ↑ up • ↓ down • / filter • enter confirm • a select all • esc cancel

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

Each branch is compared with the **bases**: `origin/HEAD`, plus each protected branch that
exists on `origin` (so with git flow, both `origin/main` and `origin/develop`):

- **merged**: already in one of the bases, including squash merges, which plain git does not
  detect. gitix shows which one: `merged (squash) → develop`.
- **not merged**: has work that is in none of the bases: `3 commits, never pushed`.

In a terminal, `gitix clean` opens a list where merged branches are checked and the others are
not. Check what you want, press enter: deleting a branch that is not merged asks for a
confirmation first. `esc` leaves without deleting anything.

| Flag | |
|---|---|
| `--keep a,b` | also keep these branches |
| `--base <branch>` | compare against this branch instead; repeat it for several bases |
| `--dry-run` | show the plan, delete nothing |
| `--yes` | delete merged branches without asking (not merged ones are skipped) |
| `--yes --force` | also delete not merged branches without asking |

Without a terminal (scripts, CI), gitix prints the plan and deletes only with `--yes`.

Every deleted branch prints its SHA, so `git branch <name> <sha>` brings it back.

### Configuration

Replace the default protected branches, globally or per repository:

```sh
git config --global gitix.keep "main master develop staging"
```

### Shell completion

```sh
gitix completion bash|zsh|fish|powershell --help
```

## Why not lazygit or git-trim?

They are great, and they do much more. `gitix` is for when you only want to
find a branch and clean the others: one binary, no config, nothing to learn.

## License

MIT
