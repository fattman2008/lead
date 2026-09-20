package gitutil

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
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
	return strings.TrimSpace(stdout.String()), nil
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

// CommonDir returns the absolute path to the shared git directory (.git).
func CommonDir(cwd string) (string, error) {
	dir, err := run(cwd, "rev-parse", "--git-common-dir")
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
