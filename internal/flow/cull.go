package flow

import (
	"fmt"
	"os"

	"github.com/fattman2008/lead/internal/gitutil"
	"github.com/fattman2008/lead/internal/wt"
)

// Sync unlocks other worktrees, runs gt sync, relocks survivors, then culls
// worktrees whose branches no longer exist.
func Sync(args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	before, err := wt.ListJSON(cwd)
	if err != nil {
		return fmt.Errorf("list worktrees before sync: %w", err)
	}

	if err := runGtUnlocked(cwd, append([]string{"sync"}, args...), UnlockAllOther, nil); err != nil {
		return err
	}

	return cullMissingBranches(cwd, before)
}

func cullMissingBranches(cwd string, before *wt.List) error {
	prefix := managedPrefix()
	for _, item := range before.Items {
		if !cullable(item, prefix) {
			continue
		}
		exists, err := gitutil.BranchExists(cwd, item.Branch)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if err := removeWorktreeItem(cwd, item); err != nil {
			return err
		}
	}
	return nil
}

func cullable(item wt.Item, prefix string) bool {
	if item.Worktree == nil || item.Branch == "" || item.Worktree.Main {
		return false
	}
	if wt.IsManaged(item.Worktree.Path, prefix) {
		return true
	}
	fmt.Fprintf(os.Stderr, "skipping unmanaged worktree for %s (%s)\n", item.Branch, item.Worktree.Path)
	return false
}

func removeWorktreeItem(cwd string, item wt.Item) error {
	force := false
	if item.Worktree != nil && item.Worktree.Changes.Dirty() {
		fmt.Fprintf(os.Stderr, "warning: worktree for deleted branch %s is dirty; forcing remove\n", item.Branch)
		force = true
	}
	fmt.Fprintf(os.Stderr, "removing worktree for deleted branch %s\n", item.Branch)
	code, err := wt.Remove(cwd, item.Branch, force)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("wt remove %s failed (exit %d)", item.Branch, code)
	}
	return nil
}
