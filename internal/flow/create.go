package flow

import (
	"fmt"
	"os"
	"strings"

	"github.com/fattman2008/lead/internal/cdfile"
	"github.com/fattman2008/lead/internal/gitutil"
	"github.com/fattman2008/lead/internal/gt"
	"github.com/fattman2008/lead/internal/wt"
)

// CreateOpts mirrors a subset of gt create flags.
// Prefer message-first: pass Message and let gt auto-generate the branch name.
type CreateOpts struct {
	Name    string // optional; empty → gt derives name from commit message
	Message []string
	All     bool
	Update  bool
	Onto    string
	Insert  bool
	AI      bool
	NoAI    bool
	Verbose int
	Extra   []string // any additional raw args to forward
}

// Create runs worktree-first branch create.
func Create(opts CreateOpts) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Branch checked out in this worktree before create — restore here even
	// when --onto points at a different stack parent.
	here, err := gitutil.CurrentBranch(cwd)
	if err != nil || here == "" {
		return fmt.Errorf("determine current branch: %w", err)
	}

	occ, _, occErr := loadOccupancy(cwd)
	if occErr != nil {
		occ = occupancy{}
	}

	// Graphite-style: if the tree has unstaged/untracked changes and the user
	// didn't already pick -a/-u, stage everything so create can commit them.
	if !opts.All && !opts.Update {
		needs, err := gitutil.NeedsStageAll(cwd)
		if err != nil {
			return fmt.Errorf("check working tree: %w", err)
		}
		if needs {
			opts.All = true
		}
	}

	args := []string{"create"}
	if opts.Name != "" {
		args = append(args, opts.Name)
	}
	for _, m := range opts.Message {
		args = append(args, "-m", m)
	}
	if opts.All {
		args = append(args, "-a")
	}
	if opts.Update {
		args = append(args, "-u")
	}
	if opts.Onto != "" {
		args = append(args, "--onto", opts.Onto)
	}
	if opts.Insert {
		args = append(args, "-i")
	}
	if opts.AI {
		args = append(args, "--ai")
	}
	if opts.NoAI {
		args = append(args, "--no-ai")
	}
	for i := 0; i < opts.Verbose; i++ {
		args = append(args, "-v")
	}
	args = append(args, opts.Extra...)

	code, err := gt.Run(cwd, args...)
	if err != nil {
		return err
	}
	if code != 0 {
		return exitCodeError(code)
	}

	newBranch, err := gitutil.CurrentBranch(cwd)
	if err != nil || newBranch == "" {
		return fmt.Errorf("after create, determine new branch: %w", err)
	}
	if newBranch == here {
		return fmt.Errorf("gt create did not leave a new branch (still on %s)", here)
	}

	// Clean-tree gt create uses -m only to name the branch. Seed an empty
	// commit so later pt modify -a amends that message instead of prompting.
	gtParent, _ := gt.Parent(cwd)
	if err := seedEmptyCommit(cwd, createSeedParent(gtParent, opts.Onto, here), opts.Message); err != nil {
		return err
	}

	if shouldCreateStay(occ, cwd) {
		// Occupying the main worktree: stay on the new branch (Graphite in a
		// single working tree). Do not restore parent or spawn a child worktree.
		return cdfile.Emit(occ.Path)
	}

	if err := gitutil.Switch(cwd, here); err != nil {
		return fmt.Errorf("restore worktree to %s (new branch is %s): %w\nrecover: stay here or switch manually", here, newBranch, err)
	}

	res, code, err := wt.SwitchJSON(cwd, newBranch)
	if err != nil {
		return fmt.Errorf("create worktree for %s (branch exists; worktree restored to %s): %w", newBranch, here, err)
	}
	if code != 0 {
		return fmt.Errorf("wt switch failed for %s (exit %d); worktree restored to %s", newBranch, code, here)
	}
	return cdfile.Emit(res.Path)
}

// createSeedParent prefers Graphite's parent, then --onto, then the branch
// we created from (here).
func createSeedParent(gtParent, onto, here string) string {
	if gtParent != "" {
		return gtParent
	}
	if onto != "" {
		return onto
	}
	return here
}

// seedEmptyCommit writes an empty commit with messages when the new branch
// has no unique commits yet. No-op if messages is empty or the branch is already ahead.
func seedEmptyCommit(cwd, parent string, messages []string) error {
	if len(messages) == 0 || parent == "" {
		return nil
	}
	ahead, err := gitutil.CommitsAhead(cwd, parent)
	if err != nil {
		return fmt.Errorf("count commits on new branch: %w", err)
	}
	if ahead > 0 {
		return nil
	}
	if err := gitutil.CommitAllowEmpty(cwd, messages); err != nil {
		return fmt.Errorf("create empty commit with message: %w", err)
	}
	return nil
}

type exitCodeError int

func (e exitCodeError) Error() string {
	return fmt.Sprintf("exit status %d", int(e))
}

func (e exitCodeError) ExitCode() int { return int(e) }

// SwitchTo moves to a branch's worktree (creating if needed) and emits cd.
// branch must be non-empty; interactive selection belongs in Checkout.
//
// If the main worktree is borrowed, only leaving to that branch's
// worktree or to trunk is allowed; other targets error.
func SwitchTo(branch string, extra []string) error {
	if branch == "" {
		return fmt.Errorf("switch: branch required")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	occ, list, err := loadOccupancy(cwd)
	if err != nil {
		return err
	}

	// @ and ^ name the main worktree, not a branch. While borrowed that is a
	// no-op cd; do not treat them as "other branch" jail.
	if branch == "@" || branch == "^" {
		if occ.Borrowed() {
			return cdfile.Emit(occ.Path)
		}
		return wtSwitch(cwd, branch, extra)
	}

	switch classifyNav(occ, branch) {
	case navJail:
		return mainJailError(occ, branch)
	case navExitTrunk:
		if err := restoreMainToTrunk(list, occ); err != nil {
			return err
		}
		return cdfile.Emit(occ.Path)
	case navExitWorktree:
		if err := restoreMainToTrunk(list, occ); err != nil {
			return err
		}
		return wtSwitch(cwd, branch, extra)
	default:
		return wtSwitch(cwd, branch, extra)
	}
}

func wtSwitch(cwd, branch string, extra []string) error {
	args := append([]string{branch}, extra...)
	res, code, err := wt.SwitchJSON(cwd, args...)
	if err != nil {
		return err
	}
	if code != 0 {
		return exitCodeError(code)
	}
	return cdfile.Emit(res.Path)
}

// Up moves to an upstack branch worktree.
func Up() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	kids, err := gt.Children(cwd)
	if err != nil {
		return err
	}
	if len(kids) == 0 {
		return fmt.Errorf("no upstack branch")
	}
	exist, err := keepLocalBranches(cwd, kids)
	if err != nil {
		return err
	}
	if len(exist) == 0 {
		return fmt.Errorf("upstack branch %s no longer exists", kids[0])
	}
	target := exist[0]
	if len(exist) > 1 {
		fmt.Fprintf(os.Stderr, "multiple upstack branches; using %s (also: %s)\n", target, strings.Join(exist[1:], ", "))
	}
	return SwitchTo(target, nil)
}

// Down moves to the downstack (parent) branch worktree.
func Down() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	parent, err := gt.Parent(cwd)
	if err != nil {
		return err
	}
	if parent == "" {
		return fmt.Errorf("switch: branch required")
	}
	ok, err := gitutil.BranchExists(cwd, parent)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("parent branch %s no longer exists", parent)
	}
	return SwitchTo(parent, nil)
}

// keepLocalBranches returns names that still exist as local git refs, in order.
func keepLocalBranches(cwd string, names []string) ([]string, error) {
	var out []string
	for _, name := range names {
		if name == "" {
			continue
		}
		ok, err := gitutil.BranchExists(cwd, name)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, name)
		}
	}
	return out, nil
}
