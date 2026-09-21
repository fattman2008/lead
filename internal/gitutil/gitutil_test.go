package gitutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

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

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func write(t *testing.T, dir, name, contents string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
