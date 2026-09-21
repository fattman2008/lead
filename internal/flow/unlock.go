package flow

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fattman2008/lead/internal/gitutil"
	"github.com/fattman2008/lead/internal/gt"
	"github.com/fattman2008/lead/internal/lock"
	"github.com/fattman2008/lead/internal/wt"
)

// UnlockScope selects which worktrees to detach before a Graphite rewrite.
type UnlockScope int

const (
	// UnlockUpstack detaches worktrees for Graphite descendants of the current branch.
	UnlockUpstack UnlockScope = iota
	// UnlockAllOther detaches every non-main worktree except the invoking one.
	UnlockAllOther
)

type detachedWT struct {
	Branch string
	Path   string
}

// RunGtUnlocked detaches unlocked target worktrees, runs gt with args, and
// relocks only on success. On nonzero exit, worktrees stay detached so
// continue/abort can proceed without re-hitting checkout exclusivity.
func RunGtUnlocked(args []string, scope UnlockScope) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return runGtUnlocked(cwd, args, scope, lock.Default())
}

func runGtUnlocked(cwd string, args []string, scope UnlockScope, check lock.Checker) error {
	if check == nil {
		check = lock.Default()
	}

	list, err := wt.ListJSON(cwd)
	if err != nil {
		return fmt.Errorf("list worktrees before unlock: %w", err)
	}

	here, herr := gitutil.CurrentBranch(cwd)
	if herr != nil {
		here = ""
	}

	cwdAbs, err := filepath.Abs(cwd)
	if err != nil {
		cwdAbs = cwd
	}

	targets, err := unlockTargetBranches(cwd, here, scope, list)
	if err != nil {
		return err
	}

	force := hasForceFlag(args)
	detached, err := detachTargets(list, cwdAbs, targets, check, force)
	if err != nil {
		_ = relockDetached(detached, false)
		return err
	}

	code, err := gt.Run(cwd, args...)
	if err != nil {
		_ = relockDetached(detached, false)
		return err
	}
	if code != 0 {
		// Leave detached for continue/abort.
		return exitCodeError(code)
	}

	return relockDetached(detached, true)
}

func unlockTargetBranches(cwd, here string, scope UnlockScope, list *wt.List) (map[string]bool, error) {
	want := map[string]bool{}
	switch scope {
	case UnlockUpstack:
		if here == "" {
			return want, nil
		}
		desc, err := gt.Descendants(cwd, here)
		if err != nil {
			return nil, fmt.Errorf("list upstack for %s: %w", here, err)
		}
		for _, b := range desc {
			want[b] = true
		}
	case UnlockAllOther:
		for _, item := range list.Items {
			if item.Branch != "" {
				want[item.Branch] = true
			}
		}
	}
	return want, nil
}

// detachTargets detaches worktrees for branches in want, skipping main and the
// invoking worktree (skipPath). Locked dirty trees error unless force.
// Validates all targets before detaching any.
func detachTargets(list *wt.List, skipPath string, want map[string]bool, check lock.Checker, force bool) ([]detachedWT, error) {
	cands, err := detachCandidates(list, skipPath, want, check, force)
	if err != nil {
		return nil, err
	}
	var detached []detachedWT
	for _, d := range cands {
		fmt.Fprintf(os.Stderr, "detaching worktree for %s\n", d.Branch)
		if err := gitutil.Detach(d.Path); err != nil {
			_ = relockDetached(detached, false)
			return detached, fmt.Errorf("detach %s: %w", d.Branch, err)
		}
		detached = append(detached, d)
	}
	return detached, nil
}

// detachCandidates returns worktrees that would be detached (no git side effects).
func detachCandidates(list *wt.List, skipPath string, want map[string]bool, check lock.Checker, force bool) ([]detachedWT, error) {
	if check == nil {
		check = lock.Default()
	}
	var cands []detachedWT
	for _, item := range list.Items {
		if item.Branch == "" || item.Worktree == nil {
			continue
		}
		if !want[item.Branch] {
			continue
		}
		if item.Worktree.Main {
			continue
		}
		if samePath(item.Worktree.Path, skipPath) {
			continue
		}
		dirty := item.Worktree.Changes.Dirty()
		if locked, reason := check.Locked(item.Branch, item.Worktree.Path, dirty); locked {
			if !force {
				return nil, fmt.Errorf("%s; commit/stash or pass --force", reason)
			}
			fmt.Fprintf(os.Stderr, "warning: %s; forcing detach\n", reason)
		}
		cands = append(cands, detachedWT{Branch: item.Branch, Path: item.Worktree.Path})
	}
	return cands, nil
}

// relockDetached reattaches detached worktrees to their branches.
// If skipMissing is true, branches that no longer exist are skipped (sync).
func relockDetached(detached []detachedWT, skipMissing bool) error {
	for _, d := range detached {
		if skipMissing {
			exists, err := gitutil.BranchExists(d.Path, d.Branch)
			if err != nil {
				return err
			}
			if !exists {
				continue
			}
		}
		if err := gitutil.Switch(d.Path, d.Branch); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to reattach worktree for %s: %v\n", d.Branch, err)
			continue
		}
	}
	return nil
}
