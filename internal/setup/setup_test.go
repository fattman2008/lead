package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureShellIntegrationAppendsOnce(t *testing.T) {
	home := t.TempDir()
	rcPath, line, added, err := ensureShellIntegration(home, "zsh")
	if err != nil {
		t.Fatal(err)
	}
	if !added {
		t.Fatal("expected first call to add integration")
	}
	wantPath := filepath.Join(home, ".zshrc")
	if rcPath != wantPath {
		t.Fatalf("rc path = %q, want %q", rcPath, wantPath)
	}
	if !strings.Contains(line, `eval "$(pt shell init zsh)"`) {
		t.Fatalf("unexpected line: %q", line)
	}

	data, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "pt shell init") {
		t.Fatalf("rc missing marker:\n%s", data)
	}

	_, _, added, err = ensureShellIntegration(home, "zsh")
	if err != nil {
		t.Fatal(err)
	}
	if added {
		t.Fatal("expected second call to be a no-op")
	}

	data2, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(data2) {
		t.Fatal("rc file changed on second ensure")
	}
}

func TestEnsureShellIntegrationFish(t *testing.T) {
	home := t.TempDir()
	rcPath, line, added, err := ensureShellIntegration(home, "fish")
	if err != nil {
		t.Fatal(err)
	}
	if !added {
		t.Fatal("expected add")
	}
	wantPath := filepath.Join(home, ".config", "fish", "config.fish")
	if rcPath != wantPath {
		t.Fatalf("rc path = %q, want %q", rcPath, wantPath)
	}
	if line != `pt shell init fish | source` {
		t.Fatalf("unexpected line: %q", line)
	}
}

func TestEnsureWorktrunkPathIdempotent(t *testing.T) {
	home := t.TempDir()
	if err := ensureWorktrunkPath(home); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(home, ".config", "worktrunk", "config.toml")
	first, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(first), worktreePathLine) {
		t.Fatalf("missing worktree-path:\n%s", first)
	}
	if err := ensureWorktrunkPath(home); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("config changed on second ensure")
	}
}
