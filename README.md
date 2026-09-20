# Lead (`pt`)

Lightweight CLI wrapper over [graphite](https://graphite.com/docs/command-reference) + [worktrunk](https://worktrunk.dev/) for a graphite-like CLI with automatic managed per-branch worktrees.

## Install

```bash
brew install fattman2008/tap/lead
pt setup
pt doctor
```

`brew install` pulls peer deps (`worktrunk`, `withgraphite/tap/graphite`). `pt setup` configures Worktrunk's worktree path layout and installs shell integration (required for auto-cd).

### From source

Peer dependencies must be on `PATH`: Graphite CLI (`gt`), Worktrunk (`wt`), and `git`.

```bash
go build -ldflags "-X github.com/fattman2008/lead/internal/cli.Version=0.1.0" -o bin/pt ./cmd/pt
# put bin/pt on your PATH, then:
pt setup
```

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
| Delete branch + worktree | `pt delete` |

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
pt sync          # also removes worktrees for deleted/merged branches
```

## Commands (core)

- **Worktree-aware:** `create`, `checkout`/`switch`/`co`, `up`, `down`, `list`, `remove`, `sync`, `delete`
- **Graphite-shaped:** `modify`, `submit`, `restack`, `log`, `info`, `track`, `init`, `auth`, …
- **Meta:** `setup`, `doctor`, `shell init`

## Development

```bash
go test ./...
go build -o bin/pt ./cmd/pt
./bin/pt doctor
```
