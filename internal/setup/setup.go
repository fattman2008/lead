package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fattman2008/lead/internal/shell"
)

const worktreePathLine = `worktree-path = "~/worktrees/{{ repo }}/{{ branch | sanitize }}"`

// Run configures shell integration instructions and Worktrunk worktree-path.
func Run(shellName string) error {
	if err := ensureWorktrunkPath(); err != nil {
		return err
	}
	fmt.Println("Worktrunk worktree-path set to:")
	fmt.Printf("  %s\n", worktreePathLine)
	fmt.Println()
	fmt.Println("Add Lead shell integration to your shell startup file:")
	fmt.Printf("  %s\n", shell.InstallHint(shellName))
	fmt.Println()
	fmt.Println("Or append now (zsh example):")
	fmt.Println(`  echo 'eval "$(pt shell init zsh)"' >> ~/.zshrc`)
	return nil
}

func ensureWorktrunkPath() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	cfgDir := filepath.Join(home, ".config", "worktrunk")
	cfgPath := filepath.Join(cfgDir, "config.toml")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		return err
	}

	var content string
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		content = "# Managed in part by lead (pt) setup\n\n" + worktreePathLine + "\n"
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
