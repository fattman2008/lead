package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/fattman2008/lead/internal/complete"
	"github.com/fattman2008/lead/internal/doctor"
	"github.com/fattman2008/lead/internal/flow"
	"github.com/fattman2008/lead/internal/gt"
	"github.com/fattman2008/lead/internal/setup"
	"github.com/fattman2008/lead/internal/shell"
	"github.com/fattman2008/lead/internal/version"
	"github.com/fattman2008/lead/internal/wt"
	"github.com/spf13/cobra"
)

// Version is the CLI version from internal/version/VERSION.
var Version = version.String()

// Execute runs the pt CLI. Returns process exit code.
func Execute() int {
	root := newRoot()
	// Register before passthrough checks so `pt help` / `pt completion` are not
	// forwarded to gt (cobra normally adds these inside Execute).
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()

	args := os.Args[1:]
	if isCompletionRequest(args) {
		// Discover gt subcommands only when completing, so normal invocations
		// stay fast while tab-completion reaches full gt flag/command parity.
		complete.RegisterGtPassthroughs(root, passthrough)
	}

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

func isCompletionRequest(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "__complete", "__completeNoDesc":
		return true
	default:
		return false
	}
}

func shouldPassthrough(root *cobra.Command, args []string) bool {
	if len(args) == 0 {
		return false
	}
	// __complete is registered only inside cobra.Execute, so Find won't see it yet.
	if isCompletionRequest(args) {
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
		cmdRoot(),
		cmdPin(),
		cmdSetup(),
		cmdDoctor(),
		cmdShell(),
		unlocked("modify", "m", "Amend or commit on the current branch and restack descendants", flow.UnlockUpstack),
		passthrough("submit", "s", "Push the stack and create/update PRs"),
		unlocked("restack", "", "Rebase the stack onto correct parents", flow.UnlockUpstack),
		passthrough("log", "", "Show the current stack"),
		passthrough("info", "", "Show info about a branch"),
		passthrough("track", "", "Start tracking a branch with Graphite"),
		passthrough("init", "", "Initialize Graphite in this repository"),
		passthrough("auth", "", "Authenticate with Graphite"),
		unlocked("undo", "", "Undo the last Graphite command", flow.UnlockAllOther),
		unlocked("continue", "", "Continue after resolving conflicts", flow.UnlockAllOther),
		unlocked("abort", "", "Abort an in-progress Graphite operation", flow.UnlockAllOther),
	)

	return root
}

const longHelp = `Lead (pt) wraps Graphite and Worktrunk for stacked PRs with worktree-first parallelism.

Create and navigate in worktrees; restack, submit, and sync stay Graphite-shaped.

Clean parked worktrees may be detached briefly during restack/sync so Graphite
can move tips; dirty worktrees are treated as locked unless you pass --force.

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
If the working tree is clean, an empty commit is created with that message
so later pt modify -a can amend it.

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
	_ = cmd.RegisterFlagCompletionFunc("onto", complete.Branches)
	_ = cmd.RegisterFlagCompletionFunc("name", cobra.NoFileCompletions)
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
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: complete.BranchArg,
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
		ValidArgsFunction:  complete.FromWt("list"),
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
		ValidArgsFunction:  complete.FromWt("remove"),
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
		ValidArgsFunction:  complete.FromGt("sync"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return flow.Sync(args)
		},
	}
}

func cmdDelete() *cobra.Command {
	return &cobra.Command{
		Use:     "delete [name]",
		Aliases: []string{"dl"},
		Short:   "Delete a branch and its worktree",
		Long: `Delete a branch (via Graphite) and remove its worktree.

When the branch you are on is deleted, the shell cds into the parent
worktree (or trunk if there is no parent) — Graphite's post-delete
checkout, translated to worktrees.`,
		DisableFlagParsing: true,
		ValidArgsFunction:  complete.FromGt("delete"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return flow.Delete(args)
		},
	}
}

func cmdRoot() *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "root",
		Short: "Print the worktree path to use for local binaries",
		Long: `Print the checkout directory for a repository.

Resolution: current worktree if you are inside one, else the pin (pt pin),
else the main / canonical worktree. Prints only the directory; callers
append their own binary path.

Use --repo when wrapping a project from another directory:

  $(pt root --repo ~/Projects/lead)/bin/pt`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return flow.Root(repo)
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "Any path in the target repository")
	_ = cmd.MarkFlagDirname("repo")
	return cmd
}

func cmdPin() *cobra.Command {
	opts := flow.PinOpts{}
	cmd := &cobra.Command{
		Use:   "pin [branch]",
		Short: "Pin a worktree as the default for pt root outside the repo",
		Long: `Pin a worktree so pt root uses it when you are not inside the repository.

With no argument, pins the current worktree. Pass @ to pin the main clone.

  pt pin          # pin this worktree
  pt pin feature  # pin that branch's worktree
  pt pin @        # pin the main clone
  pt pin --show
  pt pin --clear`,
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: complete.BranchArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			o := opts
			if len(args) == 1 {
				o.Branch = args[0]
			}
			if (o.Show || o.Clear) && o.Branch != "" {
				return fmt.Errorf("branch argument is not valid with --show or --clear")
			}
			return flow.Pin(o)
		},
	}
	cmd.Flags().BoolVar(&opts.Show, "show", false, "Print the pinned worktree path")
	cmd.Flags().BoolVar(&opts.Clear, "clear", false, "Remove the pin for this repository")
	cmd.MarkFlagsMutuallyExclusive("show", "clear")
	return cmd
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
		Use:       "init [bash|zsh|fish]",
		Short:     "Print shell integration for eval",
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish"},
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
		ValidArgsFunction:  complete.FromGt(name),
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

func unlocked(name, alias, short string, scope flow.UnlockScope) *cobra.Command {
	cmd := &cobra.Command{
		Use:                name,
		Short:              short,
		DisableFlagParsing: true,
		ValidArgsFunction:  complete.FromGt(name),
		RunE: func(cmd *cobra.Command, args []string) error {
			return flow.RunGtUnlocked(append([]string{name}, args...), scope)
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
