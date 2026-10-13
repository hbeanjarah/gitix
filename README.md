# gitix

**Find and clean git branches. Identify and eXecute.**

You work on many branches. You forget their names, and they pile up.
`gitix` does two things, simply, on Windows, macOS and Linux.

```console
$ gitix find trash
    feat/trash-module   2 hours ago
    fix/trash-icon      3 days ago

$ gitix find trash -s
  1)     feat/trash-module   2 hours ago
  2)     fix/trash-icon      3 days ago
  Choice: 1
Switched to feat/trash-module

$ gitix clean
Base: origin/main

Safe to delete (already in origin/main):
  feat/trash-module   merged (squash)
  fix/login           merged
⚠ Risky (work not in origin/main):
  feat/wip   3 commits never pushed

Delete 2 safe branches? [y/N] y
Also delete 1 risky branch? [y/N] n
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
| `-s, --switch` | switch to the match; a numbered menu appears if there are several |
| `-r, --remote` | include remote branches (switching creates the local tracking branch) |

### `gitix clean`

Deletes local branches, except:

- the current branch and branches checked out in a worktree,
- protected branches: `main`, `master`, `develop` by default,
- branches passed with `--keep`.

Names are matched exactly: protecting `main` does not protect `feat/maintenance`.
Remote branches are never touched.

Each branch is classified against the base branch (`origin/HEAD`, else `main`, else `master`):

- **safe**: already in the base, including squash merges, which plain git does not detect;
- **risky**: has work that is not in the base. Asked separately, never deleted by `--yes` alone.

| Flag | |
|---|---|
| `--keep a,b` | also keep these branches |
| `--base <branch>` | compare against this branch |
| `--dry-run` | show the plan, delete nothing |
| `--yes` | delete safe branches without asking (risky ones are skipped) |
| `--yes --force` | also delete risky branches without asking |

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
find a branch and clean the others: one binary, no config, no UI to learn.

## License

MIT
