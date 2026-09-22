package gitutil

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func run(cwd string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	// Trim only trailing newlines — leading spaces are meaningful in
	// `git status --porcelain` (e.g. " M file" = unstaged-only change).
	return strings.TrimRight(stdout.String(), "\r\n"), nil
}

// CurrentBranch returns the current branch name for cwd (or ".").
func CurrentBranch(cwd string) (string, error) {
	return run(cwd, "branch", "--show-current")
}

// Switch checks out branch in cwd without creating a new branch.
func Switch(cwd, branch string) error {
	_, err := run(cwd, "switch", branch)
	return err
}

// Detach checks out a detached HEAD at the current commit.
func Detach(cwd string) error {
	_, err := run(cwd, "switch", "--detach")
	return err
}

// CommonDir returns the absolute path to the shared git directory (.git).
func CommonDir(cwd string) (string, error) {
	return absGitPath(cwd, "rev-parse", "--git-common-dir")
}

// Toplevel returns the absolute path to the current worktree root.
func Toplevel(cwd string) (string, error) {
	return absGitPath(cwd, "rev-parse", "--show-toplevel")
}

func absGitPath(cwd string, args ...string) (string, error) {
	dir, err := run(cwd, args...)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(dir) {
		return dir, nil
	}
	base := cwd
	if base == "" {
		base = "."
	}
	return filepath.Abs(filepath.Join(base, dir))
}

// CanonPath returns an absolute path, resolving symlinks when the path exists.
func CanonPath(p string) string {
	if p == "" {
		return ""
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.Clean(p)
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		return real
	}
	return abs
}

// ContainsPath reports whether child is parent or a subdirectory of parent.
func ContainsPath(parent, child string) bool {
	if parent == "" || child == "" {
		return false
	}
	rel, err := filepath.Rel(CanonPath(parent), CanonPath(child))
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

// BranchExists reports whether a local branch ref exists.
func BranchExists(cwd, branch string) (bool, error) {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	if cwd != "" {
		cmd.Dir = cwd
	}
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}

// LocalBranches returns local branch names (refs/heads/*).
func LocalBranches(cwd string) ([]string, error) {
	out, err := run(cwd, "for-each-ref", "--format=%(refname:short)", "refs/heads/")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	var branches []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			branches = append(branches, line)
		}
	}
	return branches, nil
}

// CommitsAhead returns the number of commits on HEAD that are not in upstream
// (git rev-list --count upstream..HEAD).
func CommitsAhead(cwd, upstream string) (int, error) {
	if upstream == "" {
		return 0, fmt.Errorf("commits ahead: empty upstream")
	}
	out, err := run(cwd, "rev-list", "--count", upstream+"..HEAD")
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return 0, fmt.Errorf("commits ahead: parse count %q: %w", out, err)
	}
	return n, nil
}

// CommitAllowEmpty creates a commit with --allow-empty and the given -m values
// (each extra message is a new paragraph, same as git commit -m ... -m ...).
func CommitAllowEmpty(cwd string, messages []string) error {
	if len(messages) == 0 {
		return fmt.Errorf("commit message required")
	}
	args := []string{"commit", "--allow-empty"}
	for _, m := range messages {
		args = append(args, "-m", m)
	}
	_, err := run(cwd, args...)
	return err
}

// NeedsStageAll reports whether the worktree has unstaged or untracked changes
// that gt create -a would pick up (already-staged-only changes return false).
func NeedsStageAll(cwd string) (bool, error) {
	out, err := run(cwd, "status", "--porcelain", "-u")
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 2 {
			continue
		}
		if line[0] == '?' && line[1] == '?' {
			return true, nil
		}
		if line[1] != ' ' {
			return true, nil
		}
	}
	return false, nil
}
