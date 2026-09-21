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
| Restack / amend / submit | `pt restack` / `pt modify` / `pt submit` |
| Sync trunk + tidy worktrees | `pt sync` |
| Delete branch + worktree (cd to parent/trunk) | `pt delete` |

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
pt down          # parent worktree
pt up            # child worktree
pt submit --stack
pt sync          # unlocks clean worktrees, syncs, culls deleted/merged

pt delete        # deletes branch + worktree; if current, cds to parent (else trunk)
```

Restack/modify/sync detach clean parked worktrees so Graphite can move tips (dirty trees block unless `--force`).

## Commands (core)

- **Worktree-aware:** `create`, `checkout`/`switch`/`co`, `up`, `down`, `list`, `remove`, `sync`, `delete`, `modify`, `restack`, `continue`, `abort`, `undo`
- **Graphite-shaped:** `submit`, `log`, `info`, `track`, `init`, `auth`, …
- **Meta:** `setup`, `doctor`, `shell init`, `completion`

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
