package cli

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompletionNotPassthroughToGt(t *testing.T) {
	exe := buildTestPT(t)

	out, err := exec.Command(exe, "completion", "zsh").Output()
	if err != nil {
		t.Fatalf("pt completion zsh: %v", err)
	}
	if !bytes.Contains(out, []byte("compdef _pt pt")) {
		t.Fatalf("expected cobra pt completion, got:\n%s", out)
	}
	if bytes.Contains(out, []byte("_gt_yargs_completions")) {
		t.Fatal("pt completion incorrectly forwarded to gt")
	}
}

func TestHelpNotPassthroughToGt(t *testing.T) {
	exe := buildTestPT(t)
	out, err := exec.Command(exe, "help").Output()
	if err != nil {
		t.Fatalf("pt help: %v", err)
	}
	if !bytes.Contains(out, []byte("Lead (pt)")) {
		t.Fatalf("expected pt help, got:\n%s", out)
	}
}

func TestCompleteRootIncludesLeadAndGt(t *testing.T) {
	exe := buildTestPT(t)
	cmd := exec.Command(exe, "__complete", "")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run() // cobra __complete may exit 0 with directive on stdout
	got := stdout.String()
	for _, want := range []string{"create", "checkout", "submit", "absorb", "doctor", "root", "pin"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in completions:\n%s\nstderr:\n%s", want, got, stderr.String())
		}
	}
}

func TestCompleteSubmitFlags(t *testing.T) {
	exe := buildTestPT(t)
	cmd := exec.Command(exe, "__complete", "submit", "--dra")
	out, _ := cmd.Output()
	if !bytes.Contains(out, []byte("--draft")) {
		t.Fatalf("expected --draft, got:\n%s", out)
	}
}

func buildTestPT(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	exe := filepath.Join(dir, "pt")
	cmd := exec.Command("go", "build", "-o", exe, "./cmd/pt")
	cmd.Dir = repoRoot(t)
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		t.Fatalf("go build: %v", err)
	}
	return exe
}

func repoRoot(t *testing.T) string {
	t.Helper()
	// tests run with cwd = package dir; module root is ../.. from internal/cli
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "../.."))
}
