package complete

import (
	"strings"

	"github.com/fattman2008/lead/internal/gt"
	"github.com/spf13/cobra"
)

// gtPassthroughSkip are gt commands we intentionally do not register as
// passthroughs during completion (custom pt behavior, or meta commands).
var gtPassthroughSkip = map[string]bool{
	"completion":        true,
	"fish":              true,
	"mcp":               true,
	"list-repositories": true,
	// Worktree-aware overrides:
	"create":   true,
	"checkout": true,
	"up":       true,
	"down":     true,
	"sync":     true,
	"delete":   true,
	// Meta / lead-only (never from gt, but be safe):
	"setup":  true,
	"doctor": true,
	"shell":  true,
	"list":   true,
	"remove": true,
	"help":   true,
}

// RegisterGtPassthroughs adds unregistered gt subcommands onto root so
// `__complete` can offer them (and their flags via FromGt). Safe to call
// only on the completion path — normal execution still uses shouldPassthrough.
func RegisterGtPassthroughs(root *cobra.Command, addPassthrough func(name, alias, short string) *cobra.Command) {
	cmds, err := gt.ListCommands()
	if err != nil {
		return
	}

	existing := map[string]bool{}
	for _, c := range root.Commands() {
		existing[c.Name()] = true
		for _, a := range c.Aliases {
			existing[a] = true
		}
	}

	for _, c := range cmds {
		name := c.Name
		if name == "" || existing[name] || gtPassthroughSkip[name] {
			continue
		}
		desc := c.Desc
		if i := strings.Index(desc, "."); i > 0 && i < 80 {
			desc = desc[:i+1]
		}
		if len(desc) > 80 {
			desc = desc[:77] + "..."
		}
		cmd := addPassthrough(name, "", desc)
		cmd.ValidArgsFunction = FromGt(name)
		root.AddCommand(cmd)
		existing[name] = true
	}
}
