package flow

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestKeepLocalBranches(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "f")
	runGit(t, dir, "commit", "-m", "init")
	runGit(t, dir, "branch", "keep-me")

	got, err := keepLocalBranches(dir, []string{"keep-me", "gone", "main", ""})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"keep-me", "main"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}

	none, err := keepLocalBranches(dir, []string{"gone", "also-gone"})
	if err != nil {
		t.Fatal(err)
	}
	if len(none) != 0 {
		t.Fatalf("got %v want empty", none)
	}
}

func TestCreateSeedParent(t *testing.T) {
	cases := []struct {
		gtParent, onto, here, want string
	}{
		{"gt-parent", "onto", "here", "gt-parent"},
		{"", "onto", "here", "onto"},
		{"", "", "here", "here"},
		{"", "", "", ""},
	}
	for _, tc := range cases {
		got := createSeedParent(tc.gtParent, tc.onto, tc.here)
		if got != tc.want {
			t.Fatalf("createSeedParent(%q, %q, %q) = %q, want %q", tc.gtParent, tc.onto, tc.here, got, tc.want)
		}
	}
}

func TestSeedEmptyCommit(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "f")
	runGit(t, dir, "commit", "-m", "init")

	runGit(t, dir, "checkout", "-b", "feat")

	if err := seedEmptyCommit(dir, "main", nil); err != nil {
		t.Fatal(err)
	}
	if got := gitOut(t, dir, "log", "-1", "--format=%s"); got != "init" {
		t.Fatalf("no messages: subject = %q, want init", got)
	}

	if err := seedEmptyCommit(dir, "main", []string{"some thing"}); err != nil {
		t.Fatal(err)
	}
	if got := gitOut(t, dir, "log", "-1", "--format=%s"); got != "some thing" {
		t.Fatalf("seeded subject = %q, want %q", got, "some thing")
	}

	// Already ahead: do not add another commit.
	head := gitOut(t, dir, "rev-parse", "HEAD")
	if err := seedEmptyCommit(dir, "main", []string{"second"}); err != nil {
		t.Fatal(err)
	}
	if got := gitOut(t, dir, "rev-parse", "HEAD"); got != head {
		t.Fatal("already-ahead seed should be a no-op")
	}
	if got := gitOut(t, dir, "log", "-1", "--format=%s"); got != "some thing" {
		t.Fatalf("subject after no-op = %q, want %q", got, "some thing")
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimRight(string(out), "\r\n")
}
