package wt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/fattman2008/lead/internal/rewrite"
)

const Bin = "wt"

// Run executes wt with args, branding stdout/stderr as pt.
func Run(cwd string, args ...string) (int, error) {
	return run(cwd, os.Stdout, os.Stderr, true, args...)
}

// RunRaw executes wt without branding (for JSON / machine output).
func RunRaw(cwd string, args ...string) (int, error) {
	return run(cwd, os.Stdout, os.Stderr, false, args...)
}

func run(cwd string, stdout, stderr io.Writer, brand bool, args ...string) (int, error) {
	cmdArgs := args
	if cwd != "" {
		cmdArgs = append([]string{"-C", cwd}, args...)
	}
	cmd := exec.Command(Bin, cmdArgs...)
	cmd.Stdin = os.Stdin
	if brand {
		cmd.Stdout = rewrite.NewWriter(stdout)
		cmd.Stderr = rewrite.NewWriter(stderr)
	} else {
		cmd.Stdout = stdout
		cmd.Stderr = stderr
	}
	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return exit.ExitCode(), nil
	}
	return 1, fmt.Errorf("running %s: %w", Bin, err)
}

func output(cwd string, args ...string) (string, int, error) {
	var stdout, stderr bytes.Buffer
	code, err := run(cwd, &stdout, &stderr, false, args...)
	out := stdout.String()
	if err != nil {
		return out, code, err
	}
	if code != 0 {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(out)
		}
		return out, code, fmt.Errorf("wt %s: %s", strings.Join(args, " "), msg)
	}
	return out, code, nil
}

// ListJSON returns parsed worktree list.
// Pins Worktrunk JSON schema 2; falls back to schema 1 (bare array) if needed.
func ListJSON(cwd string) (*List, error) {
	out, _, err := output(cwd, "list", "--format", "json", "--config-set", "list.json-schema=2")
	if err != nil {
		return nil, err
	}
	return parseListJSON([]byte(out))
}

func parseListJSON(data []byte) (*List, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, fmt.Errorf("parse wt list json: empty output")
	}

	// Schema 2 envelope.
	var list List
	if data[0] == '{' {
		if err := json.Unmarshal(data, &list); err != nil {
			return nil, fmt.Errorf("parse wt list json: %w", err)
		}
		if list.Items == nil {
			list.Items = []Item{}
		}
		return &list, nil
	}

	// Schema 1 bare array.
	var raw []schema1Item
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse wt list json: %w", err)
	}
	list.Items = make([]Item, 0, len(raw))
	for _, r := range raw {
		list.Items = append(list.Items, r.toItem())
	}
	return &list, nil
}

// List is the wt list --format json schema (subset, normalized to schema 2 shape).
type List struct {
	Items []Item `json:"items"`
}

type Item struct {
	Branch   string    `json:"branch"`
	Worktree *Worktree `json:"worktree"`
}

type Worktree struct {
	Path    string  `json:"path"`
	Main    bool    `json:"main"`
	Current bool    `json:"current"`
	Changes Changes `json:"changes"`
}

type Changes struct {
	Staged     bool `json:"staged"`
	Modified   bool `json:"modified"`
	Untracked  bool `json:"untracked"`
	Conflicted bool `json:"conflicted"`
}

func (c Changes) Dirty() bool {
	return c.Staged || c.Modified || c.Untracked || c.Conflicted
}

// schema1Item is the Worktrunk list JSON schema 1 (bare-array) shape.
type schema1Item struct {
	Branch      string          `json:"branch"`
	Path        string          `json:"path"`
	IsCurrent   bool            `json:"is_current"`
	IsMain      bool            `json:"is_main"`
	WorkingTree *schema1Changes `json:"working_tree"`
}

type schema1Changes struct {
	Staged     bool `json:"staged"`
	Modified   bool `json:"modified"`
	Untracked  bool `json:"untracked"`
	Conflicted bool `json:"conflicted"`
}

func (r schema1Item) toItem() Item {
	item := Item{Branch: r.Branch}
	if r.Path == "" {
		return item
	}
	wt := &Worktree{
		Path:    r.Path,
		Main:    r.IsMain,
		Current: r.IsCurrent,
	}
	if r.WorkingTree != nil {
		wt.Changes = Changes{
			Staged:     r.WorkingTree.Staged,
			Modified:   r.WorkingTree.Modified,
			Untracked:  r.WorkingTree.Untracked,
			Conflicted: r.WorkingTree.Conflicted,
		}
	}
	item.Worktree = wt
	return item
}

// SwitchResult is the machine-readable result of wt switch --format json.
type SwitchResult struct {
	Action string `json:"action"`
	Branch string `json:"branch"`
	Path   string `json:"path"`
}

// SwitchJSON runs wt switch --no-cd --format json, brands human stderr as pt,
// and returns the structured result (including the worktree path to cd into).
func SwitchJSON(cwd string, switchArgs ...string) (*SwitchResult, int, error) {
	args := append([]string{"switch", "--no-cd", "--format", "json"}, switchArgs...)
	var stdout bytes.Buffer
	code, err := run(cwd, &stdout, rewrite.NewWriter(os.Stderr), false, args...)
	if err != nil {
		return nil, code, err
	}
	if code != 0 {
		return nil, code, nil
	}
	var res SwitchResult
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &res); err != nil {
		return nil, code, fmt.Errorf("parse wt switch json: %w", err)
	}
	if res.Path == "" {
		return nil, code, fmt.Errorf("wt switch returned empty path")
	}
	return &res, code, nil
}

// Remove removes a worktree for the given branch.
func Remove(cwd string, branch string, force bool) (int, error) {
	args := []string{"remove", "-y", branch}
	if force {
		args = append(args, "--force")
	}
	return Run(cwd, args...)
}

// LookPath checks that wt is on PATH.
func LookPath() (string, error) {
	return exec.LookPath(Bin)
}
