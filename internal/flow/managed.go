package flow

import (
	"fmt"
	"os"

	"github.com/fattman2008/lead/internal/wt"
)

// managedPrefix is the Worktrunk worktree-path directory. Empty means manage
// every worktree (config missing or unresolvable).
func managedPrefix() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return wt.ManagedPrefixFromHome(home)
}

func skipUnmanaged(item wt.Item, prefix string) bool {
	if item.Worktree == nil {
		return false
	}
	if wt.IsManaged(item.Worktree.Path, prefix) {
		return false
	}
	fmt.Fprintf(os.Stderr, "skipping unmanaged worktree for %s (%s)\n", item.Branch, item.Worktree.Path)
	return true
}
