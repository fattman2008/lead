package gt

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

const Bin = "gt"

// Run executes gt with args.
// Stdin/stdout/stderr are the process terminals when possible so interactive
// gt prompts (editors, etc.) keep working.
func Run(cwd string, args ...string) (int, error) {
	return run(cwd, os.Stdout, os.Stderr, args...)
}

func run(cwd string, stdout, stderr io.Writer, args ...string) (int, error) {
	cmdArgs := args
	if cwd != "" {
		cmdArgs = append([]string{"--cwd", cwd}, args...)
	}
	cmd := exec.Command(Bin, cmdArgs...)
	if cwd != "" {
		// Also set Dir: after removing the invoking worktree the process cwd
		// may be gone, and gt still resolves paths from process.cwd().
		cmd.Dir = cwd
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return exit.ExitCode(), nil
	}
	return 1, fmt.Errorf("running %s: %w", Bin, err)
}

// Parent returns the Graphite parent branch of the current branch in cwd.
func Parent(cwd string) (string, error) {
	return rawOutput(cwd, "parent", "--no-interactive")
}

// ParentOf returns the Graphite parent of branch from metadata (empty if none/unknown).
func ParentOf(cwd, branch string) (string, error) {
	if branch == "" {
		return "", nil
	}
	nodes, err := loadBranchMetadata(cwd)
	if err != nil {
		return "", err
	}
	n := nodes[branch]
	if n == nil {
		return "", nil
	}
	return n.Parent, nil
}

// Children returns child branch names of the current branch in cwd.
func Children(cwd string) ([]string, error) {
	out, err := rawOutput(cwd, "children", "--no-interactive")
	if err != nil {
		return nil, err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}
	var kids []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			kids = append(kids, line)
		}
	}
	return kids, nil
}

// Descendants returns Graphite upstack branches of branch (not including branch).
func Descendants(cwd, branch string) ([]string, error) {
	nodes, err := loadBranchMetadata(cwd)
	if err != nil {
		return nil, err
	}
	n := nodes[branch]
	if n == nil {
		return nil, nil
	}
	seen := map[string]bool{}
	var out []string
	queue := append([]string(nil), n.Kids...)
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
		if child := nodes[name]; child != nil {
			queue = append(queue, child.Kids...)
		}
	}
	return out, nil
}

// Trunk returns the configured Graphite trunk branch for the repo.
func Trunk(cwd string) (string, error) {
	cfg, err := readRepoConfig(cwd)
	if err != nil {
		return "", err
	}
	if cfg.Trunk == "" {
		names := cfg.trunkNames()
		if len(names) == 0 {
			return "", fmt.Errorf("graphite repo config has no trunk")
		}
		return names[0], nil
	}
	return cfg.Trunk, nil
}

func rawOutput(cwd string, args ...string) (string, error) {
	cmdArgs := args
	if cwd != "" {
		cmdArgs = append([]string{"--cwd", cwd}, args...)
	}
	cmd := exec.Command(Bin, cmdArgs...)
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
		return "", fmt.Errorf("gt %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// LookPath checks that gt is on PATH.
func LookPath() (string, error) {
	return exec.LookPath(Bin)
}
