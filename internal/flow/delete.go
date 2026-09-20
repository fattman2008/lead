package flow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fattman2008/lead/internal/cdfile"
	"github.com/fattman2008/lead/internal/gitutil"
	"github.com/fattman2008/lead/internal/gt"
	"github.com/fattman2008/lead/internal/wt"
)

// Delete runs gt delete, removes orphaned worktrees, and when the current
// branch is deleted cds into the parent worktree (else trunk) — Graphite's
// checkout-after-delete, worktree-translated.
//
// Graphite's post-delete `git switch <parent>` fails when parent is already
// checked out in another worktree. We remove the target worktree(s) first
// (keeping the branch), then run gt delete from a safe worktree.
func Delete(args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	before, err := wt.ListJSON(cwd)
	if err != nil {
		return fmt.Errorf("list worktrees before delete: %w", err)
	}

	here, err := gitutil.CurrentBranch(cwd)
	if err != nil {
		return fmt.Errorf("determine current branch: %w", err)
	}

	target := positionalBranch(args)
	branch := target
	if branch == "" {
		if here == "" {
			// Detached / unknown: let gt handle prompts or errors.
			code, err := gt.Run(cwd, append([]string{"delete"}, args...)...)
			if err != nil {
				return err
			}
			if code != 0 {
				return exitCodeError(code)
			}
			return cullMissingBranches(cwd, before)
		}
		branch = here
	}

	strip := []string{branch}
	if hasFlag(args, "--upstack") {
		desc, err := gt.Descendants(cwd, branch)
		if err != nil {
			return fmt.Errorf("list upstack for %s: %w", branch, err)
		}
		strip = append(strip, desc...)
	}

	parent, perr := gt.ParentOf(cwd, branch)
	if perr != nil {
		return fmt.Errorf("resolve parent of %s: %w", branch, perr)
	}

	opCwd := cwd
	destPath := ""

	// Relocate before strip whenever the active worktree is in the strip set
	// (delete current, or --upstack that includes the current descendant).
	if willStripHere(before, strip, here) {
		destBranch, err := resolveDeleteDest(cwd, parent)
		if err != nil {
			return err
		}
		res, code, err := wt.SwitchJSON(cwd, destBranch)
		if err != nil {
			return fmt.Errorf("ensure destination worktree %s: %w", destBranch, err)
		}
		if code != 0 {
			return fmt.Errorf("wt switch %s failed (exit %d)", destBranch, code)
		}
		destPath = res.Path
		opCwd = destPath
	}

	force := hasForceFlag(args)
	stripped, err := stripWorktrees(opCwd, before, strip, destPath, force)
	if err != nil {
		restoreStrippedWorktrees(opCwd, stripped, here)
		return err
	}

	// After stripping the current worktree, process cwd may be invalid.
	if destPath != "" {
		if err := os.Chdir(destPath); err != nil {
			restoreStrippedWorktrees(opCwd, stripped, here)
			return fmt.Errorf("chdir to destination worktree: %w", err)
		}
	}

	deleteArgs := ensureBranchArg(args, branch)
	code, err := gt.Run(opCwd, append([]string{"delete"}, deleteArgs...)...)
	if err != nil || code != 0 {
		restoreStrippedWorktrees(opCwd, stripped, here)
		if err != nil {
			return err
		}
		return exitCodeError(code)
	}

	hereGone := false
	if here != "" {
		exists, err := gitutil.BranchExists(opCwd, here)
		if err != nil {
			return err
		}
		hereGone = !exists
	}

	if !shouldNavigateAfterDelete(here, hereGone) {
		return cullMissingBranches(opCwd, before)
	}

	if destPath == "" {
		destBranch, err := resolveDeleteDest(opCwd, parent)
		if err != nil {
			return err
		}
		res, code, err := wt.SwitchJSON(opCwd, destBranch)
		if err != nil {
			return fmt.Errorf("ensure destination worktree %s: %w", destBranch, err)
		}
		if code != 0 {
			return fmt.Errorf("wt switch %s failed (exit %d)", destBranch, code)
		}
		destPath = res.Path
	}

	if err := cdfile.Emit(destPath); err != nil {
		return err
	}
	return cullMissingBranches(destPath, before)
}

// willStripHere reports whether stripWorktrees would remove the worktree for here.
func willStripHere(before *wt.List, strip []string, here string) bool {
	if here == "" {
		return false
	}
	for _, b := range strip {
		if b != here {
			continue
		}
		item := worktreeItem(before, here)
		if item == nil || item.Worktree == nil || item.Worktree.Main {
			return false
		}
		return true
	}
	return false
}

func stripWorktrees(opCwd string, before *wt.List, branches []string, destPath string, force bool) ([]string, error) {
	var stripped []string
	strippedSet := map[string]bool{}
	for _, b := range branches {
		item := worktreeItem(before, b)
		if item == nil || item.Worktree == nil {
			continue
		}
		if item.Worktree.Main {
			continue
		}
		if destPath != "" && samePath(item.Worktree.Path, destPath) {
			continue
		}
		dirty := item.Worktree.Changes.Dirty()
		if dirty && !force {
			return stripped, fmt.Errorf("worktree for %s has uncommitted changes; commit/stash or pass --force", b)
		}
		if dirty {
			fmt.Fprintf(os.Stderr, "warning: worktree for %s is dirty; forcing remove\n", b)
		}
		fmt.Fprintf(os.Stderr, "removing worktree for %s (keeping branch for delete)\n", b)
		code, err := wt.RemoveKeepBranch(opCwd, b, dirty)
		if err != nil {
			return stripped, err
		}
		if code != 0 {
			return stripped, fmt.Errorf("wt remove %s failed (exit %d)", b, code)
		}
		stripped = append(stripped, b)
		strippedSet[b] = true
	}
	// Avoid cull trying to remove worktrees we already stripped.
	if before != nil {
		for i := range before.Items {
			if strippedSet[before.Items[i].Branch] {
				before.Items[i].Worktree = nil
			}
		}
	}
	return stripped, nil
}

func restoreStrippedWorktrees(opCwd string, stripped []string, here string) {
	for _, b := range stripped {
		res, code, err := wt.SwitchJSON(opCwd, b)
		if err != nil || code != 0 || res == nil {
			fmt.Fprintf(os.Stderr, "warning: failed to restore worktree for %s after delete failure\n", b)
			continue
		}
		if b == here {
			if err := cdfile.Emit(res.Path); err != nil {
				fmt.Fprintf(os.Stderr, "warning: restore worktree for %s at %s (cd failed: %v)\n", b, res.Path, err)
			}
		}
	}
}

// resolveDeleteDest picks Graphite's post-delete checkout target: parent if it
// still exists, otherwise trunk.
func resolveDeleteDest(cwd, parent string) (string, error) {
	if parent != "" {
		exists, err := gitutil.BranchExists(cwd, parent)
		if err != nil {
			return "", err
		}
		if exists {
			return parent, nil
		}
	}
	return gt.Trunk(cwd)
}

func shouldNavigateAfterDelete(here string, hereGone bool) bool {
	return here != "" && hereGone
}

func worktreeItem(list *wt.List, branch string) *wt.Item {
	if list == nil {
		return nil
	}
	for i := range list.Items {
		if list.Items[i].Branch == branch && list.Items[i].Worktree != nil {
			return &list.Items[i]
		}
	}
	return nil
}

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return absA == absB
}

// ensureBranchArg makes the branch name explicit so gt delete run from another
// worktree cannot delete that worktree's current branch by accident.
func ensureBranchArg(args []string, branch string) []string {
	if branch == "" || positionalBranch(args) != "" {
		return args
	}
	out := make([]string, len(args)+1)
	copy(out, args)
	out[len(args)] = branch
	return out
}

func hasFlag(args []string, name string) bool {
	for _, a := range args {
		if a == name {
			return true
		}
	}
	return false
}

func hasForceFlag(args []string) bool {
	return hasFlag(args, "--force") || hasFlag(args, "-f")
}

// positionalBranch returns the first non-flag argument (the branch name), if any.
func positionalBranch(args []string) string {
	for _, a := range args {
		if a == "" {
			continue
		}
		if strings.HasPrefix(a, "-") {
			continue
		}
		return a
	}
	return ""
}
