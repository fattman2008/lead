package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fattman2008/lead/internal/shell"
)

const worktreePathLine = `worktree-path = "~/worktrees/{{ repo }}/{{ branch | sanitize }}"`

const shellMarker = "pt shell init"

// Run configures Worktrunk worktree-path and installs shell integration into the
// user's shell rc file (idempotent).
func Run(shellName string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	if err := ensureWorktrunkPath(home); err != nil {
		return err
	}
	fmt.Println("Worktrunk worktree-path set to:")
	fmt.Printf("  %s\n", worktreePathLine)
	fmt.Println()

	rcPath, line, added, err := ensureShellIntegration(home, shellName)
	if err != nil {
		return err
	}
	if added {
		fmt.Printf("Shell integration added to %s:\n", rcPath)
		fmt.Printf("  %s\n", line)
		fmt.Println()
		fmt.Println("Reload your shell (or open a new terminal) for auto-cd to take effect.")
	} else {
		fmt.Printf("Shell integration already present in %s\n", rcPath)
	}
	return nil
}

func ensureWorktrunkPath(home string) error {
	cfgDir := filepath.Join(home, ".config", "worktrunk")
	cfgPath := filepath.Join(cfgDir, "config.toml")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		return err
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		content := "# Managed in part by lead (pt) setup\n\n" + worktreePathLine + "\n"
		return os.WriteFile(cfgPath, []byte(content), 0o644)
	}

	text := string(data)
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == worktreePathLine {
			return nil
		}
	}

	// Replace an existing active worktree-path assignment, or append.
	lines := strings.Split(text, "\n")
	replaced := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "worktree-path") && !strings.HasPrefix(trimmed, "#") {
			lines[i] = worktreePathLine
			replaced = true
			break
		}
	}
	if replaced {
		return os.WriteFile(cfgPath, []byte(strings.Join(lines, "\n")), 0o644)
	}

	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	text += "\n# Set by lead (pt) setup\n" + worktreePathLine + "\n"
	return os.WriteFile(cfgPath, []byte(text), 0o644)
}

// ensureShellIntegration appends the shell init line to the rc file if missing.
// Returns rc path, the install line, whether it was newly added, and any error.
func ensureShellIntegration(home, shellName string) (rcPath, line string, added bool, err error) {
	if shellName == "" {
		shellName = "zsh"
	}
	line = shell.InstallHint(shellName)
	rcPath = shell.RCPath(home, shellName)

	if err := os.MkdirAll(filepath.Dir(rcPath), 0o755); err != nil {
		return rcPath, line, false, err
	}

	data, err := os.ReadFile(rcPath)
	if err != nil && !os.IsNotExist(err) {
		return rcPath, line, false, err
	}
	text := string(data)
	if strings.Contains(text, shellMarker) {
		return rcPath, line, false, nil
	}

	var b strings.Builder
	if len(data) > 0 && !strings.HasSuffix(text, "\n") {
		b.WriteByte('\n')
	}
	if len(data) > 0 {
		b.WriteByte('\n')
	}
	b.WriteString("# Lead (pt) shell integration — directory switching\n")
	b.WriteString(line)
	b.WriteByte('\n')

	f, err := os.OpenFile(rcPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return rcPath, line, false, err
	}
	defer f.Close()
	if _, err := f.WriteString(b.String()); err != nil {
		return rcPath, line, false, err
	}
	return rcPath, line, true, nil
}
