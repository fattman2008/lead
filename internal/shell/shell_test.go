package shell

import (
	"path/filepath"
	"strings"
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

func TestInitScriptIncludesCompletions(t *testing.T) {
	zsh := InitScript("zsh")
	if !strings.Contains(zsh, "pt completion zsh") {
		t.Fatalf("zsh init missing completion:\n%s", zsh)
	}
	if !strings.Contains(zsh, "__complete") {
		t.Fatalf("zsh init missing __complete short-circuit:\n%s", zsh)
	}
	bash := InitScript("bash")
	if !strings.Contains(bash, "pt completion bash") {
		t.Fatalf("bash init missing completion:\n%s", bash)
	}
	fish := InitScript("fish")
	if !strings.Contains(fish, "pt completion fish") {
		t.Fatalf("fish init missing completion:\n%s", fish)
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
