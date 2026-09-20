package complete

import (
	"os"
	"strings"

	"github.com/fattman2008/lead/internal/gitutil"
	"github.com/fattman2008/lead/internal/gt"
	"github.com/fattman2008/lead/internal/wt"
	"github.com/spf13/cobra"
)

// FromGt returns a ValidArgsFunction that delegates to Graphite's yargs completions
// for the given gt subcommand (e.g. "submit", "modify").
func FromGt(gtCmd string) cobra.CompletionFunc {
	return func(_ *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		words := make([]string, 0, 2+len(args)+1)
		words = append(words, gt.Bin, gtCmd)
		words = append(words, args...)
		words = append(words, toComplete)
		comps, err := gt.YargsComplete(words, len(words)-1)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return filterPrefix(comps, toComplete), cobra.ShellCompDirectiveNoFileComp
	}
}

// FromWt returns a ValidArgsFunction that delegates to Worktrunk's clap completions
// for the given wt subcommand (e.g. "list", "remove").
func FromWt(wtCmd string) cobra.CompletionFunc {
	return func(_ *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		words := make([]string, 0, 2+len(args)+1)
		words = append(words, wt.Bin, wtCmd)
		words = append(words, args...)
		words = append(words, toComplete)
		comps, err := wt.ClapComplete(words, len(words)-1)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return filterPrefixWt(comps, toComplete), cobra.ShellCompDirectiveNoFileComp
	}
}

// BranchArg completes a single positional branch argument (checkout).
func BranchArg(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return Branches(cmd, args, toComplete)
}

// Branches completes local/Graphite branch names (and Worktrunk shortcuts).
func Branches(_ *cobra.Command, _ []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}

	seen := map[string]bool{}
	var out []cobra.Completion

	add := func(name, desc string) {
		if name == "" || seen[name] || !strings.HasPrefix(name, toComplete) {
			return
		}
		seen[name] = true
		if desc != "" {
			out = append(out, cobra.CompletionWithDesc(name, desc))
		} else {
			out = append(out, cobra.Completion(name))
		}
	}

	for _, s := range []struct{ name, desc string }{
		{"@", "default / main worktree"},
		{"-", "previous worktree"},
		{"^", "source / repo root worktree"},
	} {
		add(s.name, s.desc)
	}

	choices, err := gt.ListCheckoutBranches(cwd, gt.ListOpts{
		ShowUntracked: true,
		All:           true,
	})
	if err == nil {
		for _, c := range choices {
			add(c.Name, "branch")
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	}

	branches, err := gitutil.LocalBranches(cwd)
	if err != nil {
		return out, cobra.ShellCompDirectiveNoFileComp
	}
	for _, b := range branches {
		add(b, "branch")
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func filterPrefix(comps []gt.Completion, toComplete string) []cobra.Completion {
	out := make([]cobra.Completion, 0, len(comps))
	for _, c := range comps {
		if toComplete != "" && !strings.HasPrefix(c.Value, toComplete) {
			continue
		}
		if c.Desc != "" {
			out = append(out, cobra.CompletionWithDesc(c.Value, c.Desc))
		} else {
			out = append(out, cobra.Completion(c.Value))
		}
	}
	return out
}

func filterPrefixWt(comps []wt.Completion, toComplete string) []cobra.Completion {
	out := make([]cobra.Completion, 0, len(comps))
	for _, c := range comps {
		if toComplete != "" && !strings.HasPrefix(c.Value, toComplete) {
			continue
		}
		if c.Desc != "" {
			out = append(out, cobra.CompletionWithDesc(c.Value, c.Desc))
		} else {
			out = append(out, cobra.Completion(c.Value))
		}
	}
	return out
}
