// Package lock decides whether a worktree tip may be shifted during restack/sync.
//
// Tip freeze is best-effort: locked worktrees are not detached; unlocked ones
// are fair game. Default is dirty-working-tree only; Default can later return
// Any{Dirty{}, ...} for agent/file locks without changing call sites.
package lock

// Checker reports whether a worktree must not be shifted.
type Checker interface {
	Locked(branch, path string, dirty bool) (locked bool, reason string)
}

// Dirty locks any worktree with uncommitted changes.
type Dirty struct{}

func (Dirty) Locked(branch, _ string, dirty bool) (bool, string) {
	if !dirty {
		return false, ""
	}
	if branch == "" {
		return true, "worktree has uncommitted changes"
	}
	return true, "worktree for " + branch + " has uncommitted changes"
}

// Any ORs checkers: locked if any checker locks.
type Any []Checker

func (a Any) Locked(branch, path string, dirty bool) (bool, string) {
	for _, c := range a {
		if c == nil {
			continue
		}
		if locked, reason := c.Locked(branch, path, dirty); locked {
			return true, reason
		}
	}
	return false, ""
}

// Default returns the active lock policy (dirty-only for now).
func Default() Checker {
	return Dirty{}
}
