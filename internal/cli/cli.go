package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/fattman2008/lead/internal/doctor"
	"github.com/fattman2008/lead/internal/flow"
	"github.com/fattman2008/lead/internal/gt"
	"github.com/fattman2008/lead/internal/setup"
	"github.com/fattman2008/lead/internal/shell"
	"github.com/fattman2008/lead/internal/wt"
	"github.com/spf13/cobra"
)

// Version is the CLI version. Overridden at link time via:
//
//	-ldflags "-X github.com/fattman2008/lead/internal/cli.Version=…"
var Version = "0.1.0"

// Execute runs the pt CLI. Returns process exit code.
func Execute() int {
	root := newRoot()

	args := os.Args[1:]
	if shouldPassthrough(root, args) {
		code, err := gt.Run("", args...)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return code
	}

	if err := root.Execute(); err != nil {
		var ec interface{ ExitCode() int }
		if errors.As(err, &ec) {
			return ec.ExitCode()
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func shouldPassthrough(root *cobra.Command, args []string) bool {
	if len(args) == 0 {
		return false
	}
	// Let cobra handle help/version flags at root.
	for _, a := range args {
		if a == "-h" || a == "--help" || a == "-v" || a == "--version" {
			return false
		}
		if strings.HasPrefix(a, "-") {
			continue
		}
		// First positional: if not a registered command, passthrough to gt.
		cmd, _, err := root.Find([]string{a})
		if err != nil || cmd == nil || cmd == root {
			return true
		}
		return false
	}
	return false
}

func newRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "pt",
		Short:         "Lead — Graphite stacking with first-class worktrees",
		Long:          longHelp,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       Version,
	}

	root.AddCommand(
		cmdCreate(),
		cmdSwitch(),
		cmdUp(),
		cmdDown(),
		cmdList(),
		cmdRemove(),
		cmdSync(),
		cmdDelete(),
		cmdSetup(),
		cmdDoctor(),
		cmdShell(),
		passthrough("modify", "m", "Amend or commit on the current branch and restack descendants"),
		passthrough("submit", "s", "Push the stack and create/update PRs"),
		passthrough("restack", "", "Rebase the stack onto correct parents"),
		passthrough("log", "", "Show the current stack"),
		passthrough("info", "", "Show info about a branch"),
		passthrough("track", "", "Start tracking a branch with Graphite"),
		passthrough("init", "", "Initialize Graphite in this repository"),
		passthrough("auth", "", "Authenticate with Graphite"),
		passthrough("undo", "", "Undo the last Graphite command"),
		passthrough("continue", "", "Continue after resolving conflicts"),
		passthrough("abort", "", "Abort an in-progress Graphite operation"),
	)

	return root
}

const longHelp = `Lead (pt) wraps Graphite and Worktrunk for stacked PRs with worktree-first parallelism.

Create and navigate in worktrees; restack, submit, and sync stay Graphite-shaped.

Peer dependencies: gt (Graphite) and wt (Worktrunk) must be on PATH.
Unknown commands are forwarded to gt (like gt forwards to git).`

func cmdCreate() *cobra.Command {
	opts := flow.CreateOpts{}
	cmd := &cobra.Command{
		Use:     "create [message]",
		Aliases: []string{"c"},
		Short:   "Create a stacked branch in a new worktree",
		Long: `Create a stacked branch in a new worktree.

Pass a commit message; the branch name is auto-generated (Graphite-style).
Unstaged/untracked changes are staged automatically when present.

Examples:
  pt create "add API"
  pt create -m "add API"          # same, Graphite flag style
  pt create "add API" --name api  # override branch name`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				// Positional message first; -m values append (multi -m like gt).
				opts.Message = append([]string{args[0]}, opts.Message...)
			}
			return flow.Create(opts)
		},
	}
	cmd.Flags().StringVar(&opts.Name, "name", "", "Branch name (default: auto-generated from message)")
	cmd.Flags().StringArrayVarP(&opts.Message, "message", "m", nil, "Commit message")
	cmd.Flags().BoolVarP(&opts.All, "all", "a", false, "Stage all changes including untracked (inferred when dirty)")
	cmd.Flags().BoolVarP(&opts.Update, "update", "u", false, "Stage updates to tracked files")
	cmd.Flags().StringVar(&opts.Onto, "onto", "", "Create on top of this branch")
	cmd.Flags().BoolVarP(&opts.Insert, "insert", "i", false, "Insert between current branch and child")
	cmd.Flags().BoolVar(&opts.AI, "ai", false, "AI-generate branch name and message")
	cmd.Flags().BoolVar(&opts.NoAI, "no-ai", false, "Do not AI-generate name/message")
	cmd.Flags().CountVarP(&opts.Verbose, "verbose", "v", "Show diffs in commit template")
	return cmd
}

func cmdSwitch() *cobra.Command {
	opts := flow.CheckoutOpts{}
	cmd := &cobra.Command{
		Use:     "checkout [branch]",
		Aliases: []string{"switch", "co"},
		Short:   "Switch to a branch worktree (Graphite-style select)",
		Long: `Select a branch like Graphite, then move into its worktree.

With no branch name, opens a stack-aware branch picker (same -s/-u/-a flags
as gt checkout). Selection never checks the branch out in the current
worktree — it only cds into that branch's worktree (creating it if needed).
Named targets and -t go straight to the worktree, including Worktrunk
shortcuts like @, -, and ^.

Examples:
  pt checkout              # interactive stack picker, then cd into worktree
  pt checkout -s           # only ancestors/descendants of current branch
  pt checkout feature      # move to feature's worktree
  pt checkout -t           # trunk worktree`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			o := opts
			o.Branch = ""
			o.Extra = nil
			if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
				o.Branch = args[0]
				o.Extra = args[1:]
			} else if len(args) > 0 {
				o.Extra = args
			}
			return flow.Checkout(o)
		},
	}
	cmd.Flags().BoolVarP(&opts.Trunk, "trunk", "t", false, "Checkout the current trunk")
	cmd.Flags().BoolVarP(&opts.Stack, "stack", "s", false, "Only show ancestors and descendants in interactive selection")
	cmd.Flags().BoolVarP(&opts.ShowUntracked, "show-untracked", "u", false, "Include untracked branches in interactive selection")
	cmd.Flags().BoolVarP(&opts.All, "all", "a", false, "Show branches across all configured trunks")
	return cmd
}

func cmdUp() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Switch to the upstack branch worktree",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return flow.Up()
		},
	}
}

func cmdDown() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Switch to the downstack (parent) branch worktree",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return flow.Down()
		},
	}
}

func cmdList() *cobra.Command {
	return &cobra.Command{
		Use:                "list",
		Aliases:            []string{"ls"},
		Short:              "List worktrees",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			code, err := wt.Run(cwd, append([]string{"list"}, args...)...)
			if err != nil {
				return err
			}
			if code != 0 {
				return flowExit(code)
			}
			return nil
		},
	}
}

func cmdRemove() *cobra.Command {
	return &cobra.Command{
		Use:                "remove [branches...]",
		Aliases:            []string{"rm"},
		Short:              "Remove a worktree",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			code, err := wt.Run(cwd, append([]string{"remove"}, args...)...)
			if err != nil {
				return err
			}
			if code != 0 {
				return flowExit(code)
			}
			return nil
		},
	}
}

func cmdSync() *cobra.Command {
	return &cobra.Command{
		Use:                "sync",
		Short:              "Sync stacks with trunk, then remove orphaned worktrees",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return flow.Sync(args)
		},
	}
}

func cmdDelete() *cobra.Command {
	return &cobra.Command{
		Use:                "delete [name]",
		Aliases:            []string{"dl"},
		Short:              "Delete a branch and its worktree",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return flow.Delete(args)
		},
	}
}

func cmdSetup() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Configure shell integration and Worktrunk worktree paths",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := shell.DetectFromEnv(os.Getenv("SHELL"))
			return setup.Run(name)
		},
	}
}

func cmdDoctor() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check that gt and wt are available",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return doctor.Run(os.Stdout)
		},
	}
}

func cmdShell() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shell",
		Short: "Shell integration helpers",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "init [bash|zsh|fish]",
		Short: "Print shell integration for eval",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := shell.DetectFromEnv(os.Getenv("SHELL"))
			if len(args) == 1 {
				name = args[0]
			}
			fmt.Print(shell.PrintInit(name))
			return nil
		},
	})
	return cmd
}

func passthrough(name, alias, short string) *cobra.Command {
	cmd := &cobra.Command{
		Use:                name,
		Short:              short,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			code, err := gt.Run("", append([]string{name}, args...)...)
			if err != nil {
				return err
			}
			if code != 0 {
				return flowExit(code)
			}
			return nil
		},
	}
	if alias != "" {
		cmd.Aliases = []string{alias}
	}
	return cmd
}

type exitErr int

func (e exitErr) Error() string { return fmt.Sprintf("exit status %d", int(e)) }
func (e exitErr) ExitCode() int { return int(e) }

func flowExit(code int) error { return exitErr(code) }
