package flow

import (
	"fmt"
	"os"
	"strings"

	"github.com/fattman2008/lead/internal/gitutil"
	"github.com/fattman2008/lead/internal/gt"
	"github.com/manifoldco/promptui"
)

// CheckoutOpts mirrors gt checkout selection flags, then moves via worktree.
type CheckoutOpts struct {
	Branch        string // named branch or wt shortcut; empty → interactive
	Trunk         bool
	Stack         bool
	All           bool
	ShowUntracked bool
	Main          bool
	Extra         []string // forwarded to wt switch for named targets
}

// Checkout selects a branch Graphite-style, then moves into its worktree.
//
// Named targets and --trunk go straight to wt (create worktree if needed + cd).
// Interactive selection lists Graphite-tracked branches that still exist as
// local git refs (stack-aware) and then wt-switches — never checks the branch
// out in the current worktree.
//
// --main borrows the main worktree for the branch instead (see checkoutMain).
func Checkout(opts CheckoutOpts) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	if opts.Branch != "" {
		if opts.Main {
			return checkoutMain(cwd, opts.Branch)
		}
		return SwitchTo(opts.Branch, opts.Extra)
	}
	if opts.Trunk {
		trunk, err := gt.Trunk(cwd)
		if err != nil {
			return err
		}
		if opts.Main {
			return checkoutMain(cwd, trunk)
		}
		return SwitchTo(trunk, nil)
	}

	here, err := gitutil.CurrentBranch(cwd)
	if err != nil || here == "" {
		return fmt.Errorf("determine current branch: %w", err)
	}

	choices, err := gt.ListCheckoutBranches(cwd, gt.ListOpts{
		Current:       here,
		Stack:         opts.Stack,
		All:           opts.All,
		ShowUntracked: opts.ShowUntracked,
	})
	if err != nil {
		return err
	}

	selected, err := pickBranch(choices, here)
	if err != nil {
		return err
	}
	if opts.Main {
		return checkoutMain(cwd, selected)
	}
	return SwitchTo(selected, nil)
}

func pickBranch(choices []gt.BranchChoice, current string) (string, error) {
	labels := make([]string, len(choices))
	cursor := 0
	for i, c := range choices {
		pad := strings.Repeat("  ", c.Depth)
		label := pad + c.Name
		if c.Name == current {
			label += " (current)"
			cursor = i
		}
		labels[i] = label
	}

	prompt := promptui.Select{
		Label:     "Checkout a branch (autocomplete or arrow keys)",
		Items:     labels,
		CursorPos: cursor,
		Size:      15,
		Searcher: func(input string, index int) bool {
			input = strings.ToLower(strings.TrimSpace(input))
			if input == "" {
				return true
			}
			return strings.Contains(strings.ToLower(choices[index].Name), input)
		},
	}

	idx, _, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt || err == promptui.ErrEOF {
			return "", exitCodeError(1)
		}
		return "", err
	}
	return choices[idx].Name, nil
}
