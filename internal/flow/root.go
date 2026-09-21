package flow

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fattman2008/lead/internal/gitutil"
	"github.com/fattman2008/lead/internal/pin"
	"github.com/fattman2008/lead/internal/wt"
)

// Root prints the worktree path to use for local binaries of a repo:
// cwd worktree if inside it, else pin, else the main worktree.
// repo is any path in the target repository (empty → cwd).
func Root(repo string) error {
	path, warn, err := resolveRoot(repo, "")
	if err != nil {
		return err
	}
	if warn != "" {
		fmt.Fprintln(os.Stderr, warn)
	}
	fmt.Println(path)
	return nil
}

// PinOpts controls pt pin.
type PinOpts struct {
	Branch string // empty → current worktree; "@" → main worktree
	Show   bool
	Clear  bool
}

// Pin records, shows, or clears the default worktree for pt root when outside the repo.
func Pin(opts PinOpts) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	list, err := wt.ListJSON(cwd)
	if err != nil {
		return fmt.Errorf("list worktrees: %w", err)
	}
	mainPath, err := mainWorktreePath(list)
	if err != nil {
		return err
	}

	file, err := pin.DefaultFile()
	if err != nil {
		return err
	}

	switch {
	case opts.Show:
		p, ok, err := pin.Get(file, mainPath)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("no pin for %s", mainPath)
		}
		fmt.Println(p)
		return nil
	case opts.Clear:
		if err := pin.Clear(file, mainPath); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "cleared pin for %s\n", mainPath)
		return nil
	}

	target, err := pinTarget(list, cwd, opts.Branch)
	if err != nil {
		return err
	}
	if err := pin.Set(file, mainPath, target); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "pinned %s\n", gitutil.CanonPath(target))
	return nil
}

func resolveRoot(repo, pinFile string) (path, warn string, err error) {
	if repo == "" {
		repo, err = os.Getwd()
		if err != nil {
			return "", "", err
		}
	}
	list, err := wt.ListJSON(repo)
	if err != nil {
		return "", "", fmt.Errorf("list worktrees: %w", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}
	if pinFile == "" {
		pinFile, err = pin.DefaultFile()
		if err != nil {
			return "", "", err
		}
	}
	mainPath, err := mainWorktreePath(list)
	if err != nil {
		return "", "", err
	}
	pinned, _, err := pin.Get(pinFile, mainPath)
	if err != nil {
		return "", "", err
	}
	return Resolve(list, cwd, pinned)
}

// Resolve picks a worktree path: cwd match, else a still-valid pin, else main.
func Resolve(list *wt.List, cwd, pinned string) (path, warn string, err error) {
	if wt := worktreeContaining(list, cwd); wt != nil {
		return gitutil.CanonPath(wt.Path), "", nil
	}
	mainPath, err := mainWorktreePath(list)
	if err != nil {
		return "", "", err
	}
	if pinned != "" {
		if wt := worktreeByPath(list, pinned); wt != nil {
			return gitutil.CanonPath(wt.Path), "", nil
		}
		warn = fmt.Sprintf("warning: pin %s is not a worktree of this repo; using main", pinned)
	}
	return gitutil.CanonPath(mainPath), warn, nil
}

func pinTarget(list *wt.List, cwd, branch string) (string, error) {
	switch branch {
	case "":
		wt := worktreeContaining(list, cwd)
		if wt == nil {
			return "", fmt.Errorf("not inside a listed worktree")
		}
		return wt.Path, nil
	case "@":
		return mainWorktreePath(list)
	default:
		wt := worktreeForBranch(list, branch)
		if wt == nil {
			return "", fmt.Errorf("no worktree for %s", branch)
		}
		return wt.Path, nil
	}
}

func mainWorktreePath(list *wt.List) (string, error) {
	if list == nil {
		return "", fmt.Errorf("no main worktree")
	}
	for _, item := range list.Items {
		if item.Worktree != nil && item.Worktree.Main && item.Worktree.Path != "" {
			return item.Worktree.Path, nil
		}
	}
	return "", fmt.Errorf("no main worktree")
}

func worktreeContaining(list *wt.List, dir string) *wt.Worktree {
	if list == nil {
		return nil
	}
	var best *wt.Worktree
	bestLen := -1
	for i := range list.Items {
		w := list.Items[i].Worktree
		if w == nil || w.Path == "" {
			continue
		}
		if !gitutil.ContainsPath(w.Path, dir) {
			continue
		}
		n := len(filepath.Clean(w.Path))
		if n > bestLen {
			bestLen = n
			best = w
		}
	}
	return best
}

func worktreeByPath(list *wt.List, path string) *wt.Worktree {
	if list == nil || path == "" {
		return nil
	}
	want := gitutil.CanonPath(path)
	for i := range list.Items {
		w := list.Items[i].Worktree
		if w == nil || w.Path == "" {
			continue
		}
		if gitutil.CanonPath(w.Path) == want {
			return w
		}
	}
	return nil
}

func worktreeForBranch(list *wt.List, branch string) *wt.Worktree {
	if list == nil || branch == "" {
		return nil
	}
	for i := range list.Items {
		if list.Items[i].Branch != branch {
			continue
		}
		if list.Items[i].Worktree != nil && list.Items[i].Worktree.Path != "" {
			return list.Items[i].Worktree
		}
	}
	return nil
}
