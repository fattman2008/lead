package shell

import (
	"path/filepath"
	"testing"
)

func TestInstallHint(t *testing.T) {
	if got := InstallHint("zsh"); got != `eval "$(pt shell init zsh)"` {
		t.Fatalf("zsh: %q", got)
	}
	if got := InstallHint("fish"); got != `pt shell init fish | source` {
		t.Fatalf("fish: %q", got)
	}
}

func TestRCPath(t *testing.T) {
	home := "/tmp/home"
	if got := RCPath(home, "zsh"); got != filepath.Join(home, ".zshrc") {
		t.Fatalf("zsh: %q", got)
	}
	if got := RCPath(home, "bash"); got != filepath.Join(home, ".bashrc") {
		t.Fatalf("bash: %q", got)
	}
	if got := RCPath(home, "fish"); got != filepath.Join(home, ".config", "fish", "config.fish") {
		t.Fatalf("fish: %q", got)
	}
}
