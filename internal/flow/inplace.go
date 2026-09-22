package flow

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fattman2008/lead/internal/cdfile"
	"github.com/fattman2008/lead/internal/gitutil"
	"github.com/fattman2008/lead/internal/gt"
	"github.com/fattman2008/lead/internal/wt"
)

// occupancy is the main worktree's current checkout. Borrowed means a feature
// occupies it; trunk lives there otherwise.
type occupancy struct {
	Path   string
	Branch string
	Trunk  string
	Dirty  bool
}

func (o occupancy) Borrowed() bool {
	return o.Branch != "" && o.Trunk != "" && o.Branch != o.Trunk
}

func (o occupancy) Contains(dir string) bool {
	return gitutil.ContainsPath(o.Path, dir)
}

type navAction int

const (
	navSwitch navAction = iota
	navExitWorktree
	navExitTrunk
	navJail
)

func mainOccupancy(list *wt.List, trunk string) (occupancy, error) {
	path, err := mainWorktreePath(list)
	if err != nil {
		return occupancy{}, err
	}
	var branch string
	dirty := false
	if list != nil {
		for _, item := range list.Items {
			if item.Worktree != nil && item.Worktree.Main && item.Worktree.Path != "" {
				branch = item.Branch
				dirty = item.Worktree.Changes.Dirty()
				break
			}
		}
	}
	return occupancy{Path: path, Branch: branch, Trunk: trunk, Dirty: dirty}, nil
}

func loadOccupancy(cwd string) (occupancy, *wt.List, error) {
	list, err := wt.ListJSON(cwd)
	if err != nil {
		return occupancy{}, nil, fmt.Errorf("list worktrees: %w", err)
	}
	trunk, err := gt.Trunk(cwd)
	if err != nil {
		return occupancy{}, list, err
	}
	occ, err := mainOccupancy(list, trunk)
	if err != nil {
		return occupancy{}, list, err
	}
	return occ, list, nil
}

func linkedWorktree(list *wt.List, branch string) *wt.Worktree {
	if list == nil || branch == "" {
		return nil
	}
	for i := range list.Items {
		if list.Items[i].Branch != branch {
			continue
		}
		w := list.Items[i].Worktree
		if w == nil || w.Path == "" || w.Main {
			continue
		}
		return w
	}
	return nil
}

func isWorktreeShortcut(name string) bool {
	return name == "@" || name == "-" || name == "^"
}

func classifyNav(occ occupancy, target string) navAction {
	if !occ.Borrowed() {
		return navSwitch
	}
	if target == occ.Trunk {
		return navExitTrunk
	}
	if target == occ.Branch {
		return navExitWorktree
	}
	return navJail
}

func mainJailError(occ occupancy, target string) error {
	return fmt.Errorf("%s is checked out on the main worktree. Use --main %s, checkout %s to restore a worktree, or checkout -t to return main to trunk",
		occ.Branch, target, occ.Branch)
}

func canRestoreMain(list *wt.List, occ occupancy) error {
	if !occ.Borrowed() {
		return nil
	}
	if occ.Dirty {
		return fmt.Errorf("main worktree has uncommitted changes; commit/stash before leaving")
	}
	if linked := linkedWorktree(list, occ.Trunk); linked != nil {
		return fmt.Errorf("trunk %s is checked out in a linked worktree (%s); cannot restore the main worktree", occ.Trunk, linked.Path)
	}
	return nil
}

func restoreMainToTrunk(list *wt.List, occ occupancy) error {
	if !occ.Borrowed() {
		return nil
	}
	if err := canRestoreMain(list, occ); err != nil {
		return err
	}
	if err := gitutil.Switch(occ.Path, occ.Trunk); err != nil {
		return fmt.Errorf("restore main worktree to %s: %w", occ.Trunk, err)
	}
	return nil
}

func shouldCreateStay(occ occupancy, cwd string) bool {
	return occ.Borrowed() && occ.Contains(cwd)
}

func containsBranch(names []string, branch string) bool {
	for _, n := range names {
		if n == branch {
			return true
		}
	}
	return false
}

func shouldReleaseMain(occ occupancy, strip []string) bool {
	return occ.Borrowed() && containsBranch(strip, occ.Branch)
}

func rejectMainRewrite(occ occupancy, cwd string, want map[string]bool) error {
	if !occ.Borrowed() || !want[occ.Branch] {
		return nil
	}
	if occ.Contains(cwd) {
		return nil
	}
	return fmt.Errorf("%s is checked out on the main worktree; run this from the main worktree, or: pt checkout %s", occ.Branch, occ.Branch)
}

func checkoutMain(cwd, branch string) error {
	if branch == "" {
		return fmt.Errorf("switch: branch required")
	}
	if isWorktreeShortcut(branch) {
		return fmt.Errorf("--main requires a branch name (not %s)", branch)
	}

	occ, list, err := loadOccupancy(cwd)
	if err != nil {
		return err
	}

	if branch == occ.Trunk {
		if err := restoreMainToTrunk(list, occ); err != nil {
			return err
		}
		return cdfile.Emit(occ.Path)
	}

	if occ.Branch == branch {
		return cdfile.Emit(occ.Path)
	}

	linked := linkedWorktree(list, branch)
	if linked != nil && linked.Changes.Dirty() {
		return fmt.Errorf("worktree for %s has uncommitted changes; commit/stash before --main", branch)
	}
	if occ.Dirty {
		return fmt.Errorf("main worktree has uncommitted changes; commit/stash before --main")
	}

	opCwd := cwd
	if linked != nil {
		cwdAbs, err := filepath.Abs(cwd)
		if err != nil {
			cwdAbs = cwd
		}
		if gitutil.ContainsPath(linked.Path, cwdAbs) {
			if err := os.Chdir(occ.Path); err != nil {
				return fmt.Errorf("chdir to main worktree: %w", err)
			}
			opCwd = occ.Path
		}
		// The linked worktree directory is discarded, including ignored files
		// in it. --main uses the main worktree's files, not a merge.
		fmt.Fprintf(os.Stderr, "removing worktree for %s (keeping branch)\n", branch)
		code, err := wt.RemoveKeepBranch(opCwd, branch, false)
		if err != nil {
			return err
		}
		if code != 0 {
			return fmt.Errorf("wt remove %s failed (exit %d)", branch, code)
		}
	}

	if err := gitutil.Switch(occ.Path, branch); err != nil {
		return fmt.Errorf("check out %s in the main worktree: %w", branch, err)
	}
	return cdfile.Emit(occ.Path)
}
