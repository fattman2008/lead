# Lead (`pt`)

Lightweight CLI wrapper over [graphite](https://graphite.com/docs/command-reference) + [worktrunk](https://worktrunk.dev/) for a graphite-like CLI with automatic managed per-branch worktrees.

## Install

```bash
brew install fattman2008/tap/lead
pt setup
pt doctor
```

`brew install` pulls peer deps (`worktrunk`, `withgraphite/tap/graphite`). `pt setup` configures Worktrunk's worktree path layout and installs shell integration (required for auto-cd and tab completions).

### Upgrade

```bash
brew update && brew upgrade lead
```

### From source

Peer dependencies must be on `PATH`: Graphite CLI (`gt`), Worktrunk (`wt`), and `git`.

```bash
just build   # or: go build -o bin/pt ./cmd/pt
# put bin/pt on your PATH, then:
pt setup
```

Version lives in `internal/version/VERSION` (embedded at build time).

`pt setup` sets Worktrunk:

```toml
worktree-path = "~/worktrees/{{ repo }}/{{ branch | sanitize }}"
```

## Mental model

| You want… | Use |
| ----------- | ----- |
| New stacked branch in its own worktree | `pt create` |
| Move between branch worktrees | `pt checkout` / `pt switch` / `pt up` / `pt down` |
| Work on a branch in the main worktree | `pt checkout --main` |
| Restack / amend / submit | `pt restack` / `pt modify` / `pt submit` |
| Sync trunk + tidy worktrees | `pt sync` |
| Delete branch + worktree (cd to parent/trunk) | `pt delete` |
| Path of the checkout to run local binaries from | `pt root` |
| Pin a worktree for `pt root` outside the repo | `pt pin` |

Unknown `pt <cmd>` arguments are forwarded to `gt` (same idea as `gt` → `git`)

## Example

```bash
# on main (or any stack branch), with local changes
pt create "add API"
# → stages changes if needed; branch name auto-generated from message
# → parent worktree stays on parent; shell cds into the new worktree

pt create "add UI"
# → stacked on the previous branch, new worktree

pt checkout      # stack-aware picker → cd into that branch's worktree
pt checkout --main feature  # work on feature in the main worktree (no linked worktree)
pt checkout feature         # if it was on the main worktree, restore trunk and recreate the worktree
pt down          # parent worktree
pt up            # child worktree
pt submit --stack
pt sync          # unlocks clean worktrees, syncs, culls deleted/merged

pt delete        # deletes branch + worktree; if current, cds to parent (else trunk)
```

Restack/modify/sync detach clean parked worktrees so Graphite can move tips (dirty trees block unless `--force`).

## Local builds

The main worktree stays on trunk; feature work lives in linked worktrees. Hardcoded paths like `~/Projects/foo/bin/foo` therefore always hit trunk.

`pt checkout --main` checks the branch out on the main worktree and removes the linked worktree, when a repo's local tooling assumes that checkout. `pt checkout <branch>` restores a worktree; `pt checkout -t` returns the main worktree to trunk.

`pt root` prints the checkout to use instead:

1. The worktree you are in, if it belongs to that repo
2. Else the pin (`pt pin`)
3. Else the main / canonical worktree

```bash
# from inside a worktree of the project, or from anywhere after pinning
$(pt root --repo ~/Projects/lead)/bin/pt doctor

# dogfood a feature from other directories
pt pin              # in the worktree you want
pt pin --clear      # back to the main clone
```

`--repo` is required when the wrapper runs from another project (`pt root` without it would resolve *that* repo). Example wrapper:

```zsh
localpt() {
  local cd_file exit_code=0 root
  root="$(command pt root --repo ~/Projects/lead)" || return
  cd_file="$(mktemp)"
  LEAD_CD_FILE="$cd_file" "$root/bin/pt" "$@" || exit_code=$?
  if [[ -s "$cd_file" ]]; then
    builtin cd -- "$(<"$cd_file")"
    local cd_exit=$?
    if [[ $exit_code -eq 0 ]]; then
      exit_code=$cd_exit
    fi
  fi
  command rm -f "$cd_file"
  return "$exit_code"
}
```

## Commands (core)

- **Worktree-aware:** `create`, `checkout`/`switch`/`co`, `up`, `down`, `list`, `remove`, `sync`, `delete`, `modify`, `restack`, `continue`, `abort`, `undo`
- **Graphite-shaped:** `submit`, `log`, `info`, `track`, `init`, `auth`, …
- **Meta:** `root`, `pin`, `setup`, `doctor`, `shell init`, `completion`

Tab completion is installed via `pt setup` / `eval "$(pt shell init zsh)"` (bash/fish too). Completions cover Lead commands, branch names for `checkout`/`--onto`, and delegate to `gt`/`wt` for passthrough flags.

## Development

Requires [just](https://github.com/casey/just).

```bash
just test
just build
just doctor
# or: just check   # test + doctor
```

### Releasing (Homebrew)

1. Set `internal/version/VERSION` to `X.Y.Z` and commit
2. `just release` (or `just tag` then `just bump-formula`)
3. Commit and push in `homebrew-tap`
