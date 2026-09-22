package gitutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestToplevel(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	got, err := Toplevel(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := CanonPath(dir)
	if CanonPath(got) != want {
		t.Fatalf("Toplevel = %q, want %q", got, want)
	}
}

func TestContainsPath(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "sub", "dir")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	sibling := t.TempDir()

	if !ContainsPath(parent, parent) {
		t.Fatal("parent should contain itself")
	}
	if !ContainsPath(parent, child) {
		t.Fatal("parent should contain subdirectory")
	}
	if ContainsPath(parent, sibling) {
		t.Fatal("parent should not contain sibling temp dir")
	}
	if ContainsPath(parent, parent+"-extra") {
		t.Fatal("prefix match must not count as containment")
	}
	if ContainsPath("", child) || ContainsPath(parent, "") {
		t.Fatal("empty paths are not contained")
	}
}

func TestNeedsStageAll(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "test")
	write(t, dir, "file.txt", "base\n")
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "init")

	t.Run("clean", func(t *testing.T) {
		got, err := NeedsStageAll(dir)
		if err != nil {
			t.Fatal(err)
		}
		if got {
			t.Fatal("expected false on clean tree")
		}
	})

	t.Run("unstaged tracked modification", func(t *testing.T) {
		write(t, dir, "file.txt", "base\nchange\n")
		got, err := NeedsStageAll(dir)
		if err != nil {
			t.Fatal(err)
		}
		if !got {
			t.Fatal("expected true for unstaged modification (porcelain leading space)")
		}
		// restore for next cases
		runGit(t, dir, "checkout", "--", "file.txt")
	})

	t.Run("staged only", func(t *testing.T) {
		write(t, dir, "file.txt", "staged\n")
		runGit(t, dir, "add", "file.txt")
		got, err := NeedsStageAll(dir)
		if err != nil {
			t.Fatal(err)
		}
		if got {
			t.Fatal("expected false when changes are already staged")
		}
		runGit(t, dir, "checkout", "HEAD", "--", "file.txt")
	})

	t.Run("untracked", func(t *testing.T) {
		write(t, dir, "new.txt", "new\n")
		got, err := NeedsStageAll(dir)
		if err != nil {
			t.Fatal(err)
		}
		if !got {
			t.Fatal("expected true for untracked file")
		}
	})
}

func TestCommitsAhead(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "test")
	write(t, dir, "file.txt", "base\n")
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "init")

	n, err := CommitsAhead(dir, "main")
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("on main: ahead = %d, want 0", n)
	}

	runGit(t, dir, "checkout", "-b", "feat")
	n, err = CommitsAhead(dir, "main")
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("empty feat: ahead = %d, want 0", n)
	}

	write(t, dir, "file.txt", "change\n")
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "feat commit")
	n, err = CommitsAhead(dir, "main")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("after commit: ahead = %d, want 1", n)
	}

	if _, err := CommitsAhead(dir, ""); err == nil {
		t.Fatal("expected error for empty upstream")
	}
}

func TestCommitAllowEmpty(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "test")
	write(t, dir, "file.txt", "base\n")
	runGit(t, dir, "add", "file.txt")
	runGit(t, dir, "commit", "-m", "init")
	runGit(t, dir, "checkout", "-b", "feat")

	if err := CommitAllowEmpty(dir, nil); err == nil {
		t.Fatal("expected error when messages is empty")
	}

	if err := CommitAllowEmpty(dir, []string{"some thing"}); err != nil {
		t.Fatal(err)
	}
	subject := gitOut(t, dir, "log", "-1", "--format=%s")
	if subject != "some thing" {
		t.Fatalf("subject = %q, want %q", subject, "some thing")
	}
	n, err := CommitsAhead(dir, "main")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("ahead = %d, want 1", n)
	}

	if err := CommitAllowEmpty(dir, []string{"subject line", "body paragraph"}); err != nil {
		t.Fatal(err)
	}
	subject = gitOut(t, dir, "log", "-1", "--format=%s")
	if subject != "subject line" {
		t.Fatalf("subject = %q, want %q", subject, "subject line")
	}
	body := gitOut(t, dir, "log", "-1", "--format=%b")
	if !strings.Contains(body, "body paragraph") {
		t.Fatalf("body = %q, want to contain %q", body, "body paragraph")
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

func write(t *testing.T, dir, name, contents string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
